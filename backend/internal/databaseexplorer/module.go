// Package databaseexplorer is a read-only, allowlisted operational data
// module. Its sole public behavior is Explorer.Query; there is deliberately no
// raw SQL or mutation interface.
package databaseexplorer

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrTenantDenied   = errors.New("database explorer: tenant access denied")
	ErrTenantRequired = errors.New("database explorer: tenant is required")
	ErrRoleDenied     = errors.New("database explorer: role is not allowed")
	ErrUnknownDataset = errors.New("database explorer: unknown dataset")
	ErrUnknownFilter  = errors.New("database explorer: filter is not allowed")
	ErrUnknownSort    = errors.New("database explorer: sort is not allowed")
	ErrLimitExceeded  = errors.New("database explorer: page limit exceeds 100 rows")
	ErrInvalidCursor  = errors.New("database explorer: invalid cursor")
)

type readAdapter interface {
	read(context.Context, Query) (Page, error)
}

type Explorer struct {
	adapter readAdapter
	config  Config
}

func New(adapter readAdapter, config Config) *Explorer {
	return &Explorer{adapter: adapter, config: config}
}

func (e *Explorer) Query(ctx context.Context, query Query) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	definition, known := datasets[query.Dataset]
	if !known {
		return Page{}, ErrUnknownDataset
	}
	if query.Actor.Role != RoleMerchant && query.Actor.Role != RoleOperator {
		return Page{}, ErrRoleDenied
	}
	if query.TenantID == "" {
		return Page{}, ErrTenantRequired
	}
	if query.Actor.Role == RoleMerchant {
		if query.Actor.TenantID == "" {
			return Page{}, ErrTenantRequired
		}
		if query.Actor.TenantID != query.TenantID {
			return Page{}, ErrTenantDenied
		}
	}
	for _, filter := range query.Filters {
		operators, allowed := definition.filters[filter.Field]
		if !allowed || !operators[filter.Operator] {
			return Page{}, ErrUnknownFilter
		}
	}
	if query.Sort.Field != "" {
		if !definition.sorts[query.Sort.Field] ||
			(query.Sort.Direction != SortAscending && query.Sort.Direction != SortDescending) {
			return Page{}, ErrUnknownSort
		}
	}

	maxRows := e.config.MaxRows
	if maxRows <= 0 || maxRows > 100 {
		maxRows = 100
	}
	if query.Limit <= 0 {
		query.Limit = min(50, maxRows)
	}
	if query.Limit > maxRows || query.Limit > 100 {
		return Page{}, ErrLimitExceeded
	}

	timeout := e.config.QueryTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	queryContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	page, err := e.adapter.read(queryContext, query)
	if err != nil {
		return Page{}, err
	}
	page.Rows = redactRows(page.Rows, definition.fields)
	now := time.Now
	if e.config.Now != nil {
		now = e.config.Now
	}
	page.Meta.Dataset = query.Dataset
	page.Meta.Limit = query.Limit
	page.Meta.Returned = len(page.Rows)
	page.Meta.ReadAt = now().UTC()
	return page, nil
}

type datasetDefinition struct {
	fields  map[Field]Classification
	filters map[Field]map[Operator]bool
	sorts   map[Field]bool
}

var datasets = map[Dataset]datasetDefinition{
	DatasetPaymentLifecycle: {
		fields: map[Field]Classification{
			FieldPaymentReference:          ClassificationInternal,
			FieldInvoiceID:                 ClassificationInternal,
			FieldInvoiceState:              ClassificationInternal,
			FieldInvoiceAmountFiat:         ClassificationFinancial,
			FieldFiatCurrency:              ClassificationInternal,
			FieldQuoteRequiredAtomicAmount: ClassificationFinancial,
			FieldDepositAddress:            ClassificationPII,
			FieldLifecycleStatus:           ClassificationInternal,
			FieldOutboxStatus:              ClassificationInternal,
			FieldIdempotencyFingerprint:    ClassificationInternal,
			FieldCustomerEmail:             ClassificationPII,
			FieldCreatedAt:                 ClassificationInternal,
			FieldRecordID:                  ClassificationInternal,
		},
		filters: map[Field]map[Operator]bool{
			FieldPaymentReference: {OperatorEqual: true},
			FieldInvoiceState:     {OperatorEqual: true, OperatorIn: true},
			FieldOutboxStatus:     {OperatorEqual: true, OperatorIn: true},
			FieldCreatedAt:        {OperatorGreaterThanOrEqual: true, OperatorLessThan: true},
		},
		sorts: map[Field]bool{
			FieldCreatedAt: true,
			FieldRecordID:  true,
		},
	},
}

func redactRows(rows []Row, allowed map[Field]Classification) []Row {
	result := make([]Row, 0, len(rows))
	for _, source := range rows {
		row := make(Row, len(source))
		for field, value := range source {
			classification, exists := allowed[field]
			if !exists || classification == ClassificationSecret {
				continue
			}
			value.Classification = classification
			if classification == ClassificationPII && !value.Null {
				value.Value = maskPII(field, value.Value)
				value.Redacted = true
			}
			row[field] = value
		}
		result = append(result, row)
	}
	return result
}

func maskPII(field Field, value string) string {
	if field == FieldCustomerEmail {
		local, domain, ok := strings.Cut(value, "@")
		if ok && local != "" && domain != "" {
			return local[:1] + "***@" + domain
		}
	}
	return "***"
}
