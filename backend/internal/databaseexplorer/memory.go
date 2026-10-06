package databaseexplorer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type MemoryRow struct {
	Dataset  Dataset
	TenantID string
	FreshAt  time.Time
	Values   Row
}

// MemoryAdapter is immutable after construction, making cursor pages stable
// and tests deterministic.
type MemoryAdapter struct {
	rows []MemoryRow
}

func NewMemoryAdapter(rows []MemoryRow) *MemoryAdapter {
	copyRows := make([]MemoryRow, len(rows))
	for i, row := range rows {
		copyRows[i] = row
		copyRows[i].Values = cloneRow(row.Values)
	}
	return &MemoryAdapter{rows: copyRows}
}

func (a *MemoryAdapter) read(ctx context.Context, query Query) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	cursor, err := decodeCursor(query)
	if err != nil {
		return Page{}, err
	}
	page := Page{Meta: PageMetadata{Source: "memory-v1"}}
	var matches []MemoryRow
	for _, row := range a.rows {
		if row.Dataset == query.Dataset && row.TenantID == query.TenantID && matchesFilters(row.Values, query.Filters) {
			matches = append(matches, row)
			if row.FreshAt.After(page.Meta.DataAsOf) {
				page.Meta.DataAsOf = row.FreshAt.UTC()
			}
		}
	}
	sortRows(matches, query.Sort)
	if cursor.Offset > len(matches) {
		return Page{}, ErrInvalidCursor
	}
	end := min(cursor.Offset+query.Limit, len(matches))
	for _, row := range matches[cursor.Offset:end] {
		page.Rows = append(page.Rows, cloneRow(row.Values))
	}
	page.Meta.HasMore = end < len(matches)
	if page.Meta.HasMore {
		page.Meta.NextCursor = encodeCursor(query, end)
	}
	return page, nil
}

func matchesFilters(row Row, filters []Filter) bool {
	for _, filter := range filters {
		value := row[filter.Field].Value
		switch filter.Operator {
		case OperatorEqual:
			if value != filter.Value {
				return false
			}
		case OperatorIn:
			found := false
			for _, candidate := range filter.Values {
				if value == candidate {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case OperatorGreaterThanOrEqual:
			if value < filter.Value {
				return false
			}
		case OperatorLessThan:
			if value >= filter.Value {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func sortRows(rows []MemoryRow, requested Sort) {
	field := requested.Field
	direction := requested.Direction
	if field == "" {
		field = FieldCreatedAt
		direction = SortDescending
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i].Values[field].Value, rows[j].Values[field].Value
		if left == right {
			left, right = rows[i].Values[FieldRecordID].Value, rows[j].Values[FieldRecordID].Value
		}
		if direction == SortAscending {
			return left < right
		}
		return left > right
	})
}

type memoryCursor struct {
	Version  int
	Dataset  Dataset
	TenantID string
	Scope    string
	Offset   int
}

func encodeCursor(query Query, offset int) string {
	raw, _ := json.Marshal(memoryCursor{Version: 1, Dataset: query.Dataset, TenantID: query.TenantID, Scope: queryScope(query), Offset: offset})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(query Query) (memoryCursor, error) {
	if query.Cursor == "" {
		return memoryCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(query.Cursor)
	if err != nil {
		return memoryCursor{}, ErrInvalidCursor
	}
	var cursor memoryCursor
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return memoryCursor{}, ErrInvalidCursor
	}
	if cursor.Version != 1 || cursor.Dataset != query.Dataset || cursor.TenantID != query.TenantID || cursor.Scope != queryScope(query) || cursor.Offset < 0 {
		return memoryCursor{}, fmt.Errorf("%w: scope mismatch", ErrInvalidCursor)
	}
	return cursor, nil
}

func queryScope(query Query) string {
	raw, _ := json.Marshal(struct {
		Filters []Filter
		Sort    Sort
	}{Filters: query.Filters, Sort: query.Sort})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}

func cloneRow(source Row) Row {
	result := make(Row, len(source))
	for field, value := range source {
		result[field] = value
	}
	return result
}
