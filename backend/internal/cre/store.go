package cre

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresStore persists through GORM on the cre_* tables.
type PostgresStore struct {
	db *gorm.DB
}

func NewPostgresStore(db *gorm.DB) *PostgresStore { return &PostgresStore{db: db} }

var _ Store = (*PostgresStore)(nil)

type attestationRow struct {
	ID            string    `gorm:"primaryKey;type:uuid"`
	Kind          string    `gorm:"type:varchar(32)"`
	SubjectType   string    `gorm:"type:varchar(32)"`
	SubjectID     string    `gorm:"type:varchar(128)"`
	PayloadHash   []byte    `gorm:"type:bytea"`
	Payload       []byte    `gorm:"type:bytea"`
	Chain         string    `gorm:"type:varchar(64)"`
	TxHash        []byte    `gorm:"type:bytea"`
	BlockNumber   int64     `gorm:"type:bigint"`
	WorkflowID    []byte    `gorm:"type:bytea"`
	WorkflowOwner []byte    `gorm:"type:bytea"`
	ReportID      []byte    `gorm:"type:bytea"`
	ObservedAt    time.Time `gorm:"type:timestamptz"`
	RecordedAt    time.Time `gorm:"type:timestamptz"`
	Status        string    `gorm:"type:varchar(16)"`
	Provider      string    `gorm:"type:varchar(16)"`
	Simulated     bool
	Reason        string `gorm:"type:text"`
	Item          []byte `gorm:"type:jsonb"`
	ExecutionID   string `gorm:"type:varchar(128)"`
	ItemIndex     int16  `gorm:"type:smallint"`
}

func (attestationRow) TableName() string { return "cre_attestations" }

type subjectRow struct {
	Kind       string    `gorm:"primaryKey;type:varchar(32)"`
	SubjectKey []byte    `gorm:"primaryKey;type:bytea"`
	SubjectID  string    `gorm:"type:varchar(128)"`
	Facts      []byte    `gorm:"type:jsonb"`
	AskedAt    time.Time `gorm:"type:timestamptz"`
}

func (subjectRow) TableName() string { return "cre_subjects" }

type runRow struct {
	ID          uint64 `gorm:"primaryKey"`
	Kind        string `gorm:"type:varchar(32)"`
	Provider    string `gorm:"type:varchar(16)"`
	ExecutionID string `gorm:"type:varchar(128)"`
	Status      string `gorm:"type:varchar(16)"`
	Detail      string `gorm:"type:text"`
	StartedAt   time.Time
}

func (runRow) TableName() string { return "cre_runs" }

type cursorRow struct {
	Scope       string `gorm:"primaryKey;type:varchar(192)"`
	BlockNumber int64
	Seq         int64
	UpdatedAt   time.Time
}

func (cursorRow) TableName() string { return "cre_cursors" }

// ItemJSON is the stored, snake_case shape of a decoded item; it is also what the API returns.
func ItemJSON(item any) map[string]any {
	hex := func(b [32]byte) string { return "0x" + common.Bytes2Hex(b[:]) }
	str := func(v *big.Int) string {
		if v == nil {
			return "0"
		}
		return v.String()
	}
	switch it := item.(type) {
	case SolvencyItem:
		return map[string]any{
			"checkpoint_hash": hex(it.CheckpointHash), "asset": LabelFromKey(it.Asset),
			"liabilities_minor": str(it.Liabilities), "reserves_minor": str(it.Reserves), "decimals": it.Decimals,
		}
	case DepositItem:
		return map[string]any{
			"deposit_key": hex(it.DepositID), "chain": LabelFromKey(it.ChainID), "tx_ref": hex(it.TxRef), "token": LabelFromKey(it.Token),
			"amount_minor": str(it.Amount), "destination_key": hex(it.Destination), "slot_or_block": it.SlotOrBlock, "verdict": it.Verdict,
		}
	case ConversionItem:
		return map[string]any{
			"conversion_key": hex(it.ConversionID), "pair": LabelFromKey(it.Pair), "reference_rate": str(it.ReferenceRate),
			"reference_decimals": it.ReferenceDecimals, "deviation_bps": str(it.DeviationBps), "feed": it.Feed.Hex(), "round_id": str(it.RoundID),
		}
	case map[string]any:
		return it
	}
	return map[string]any{}
}

func toRow(a Attestation) (attestationRow, error) {
	item, err := json.Marshal(ItemJSON(a.Item))
	if err != nil {
		return attestationRow{}, fmt.Errorf("cre: encode item: %w", err)
	}
	return attestationRow{
		ID: a.ID, Kind: string(a.Kind), SubjectType: a.SubjectType, SubjectID: a.SubjectID, PayloadHash: a.PayloadHash, Payload: a.Payload,
		Chain: a.Chain, TxHash: a.TxHash, BlockNumber: int64(a.BlockNumber), WorkflowID: a.WorkflowID[:], WorkflowOwner: a.WorkflowOwner[:], ReportID: a.ReportID[:],
		ObservedAt: a.ObservedAt, RecordedAt: a.RecordedAt, Status: string(a.Status), Provider: a.Provider, Simulated: a.Simulated, Reason: a.Reason, Item: item,
		ItemIndex: int16(a.ItemIndex),
	}, nil
}

func fromRow(r attestationRow) Attestation {
	a := Attestation{
		ID: r.ID, Kind: Kind(r.Kind), SubjectType: r.SubjectType, SubjectID: r.SubjectID, PayloadHash: r.PayloadHash, Payload: r.Payload,
		Chain: r.Chain, TxHash: r.TxHash, BlockNumber: uint64(r.BlockNumber), ObservedAt: r.ObservedAt.UTC(), RecordedAt: r.RecordedAt.UTC(),
		Status: Status(r.Status), Provider: r.Provider, Simulated: r.Simulated, Reason: r.Reason, ItemIndex: int(r.ItemIndex),
	}
	copy(a.WorkflowID[:], r.WorkflowID)
	copy(a.WorkflowOwner[:], r.WorkflowOwner)
	copy(a.ReportID[:], r.ReportID)
	var item map[string]any
	if json.Unmarshal(r.Item, &item) == nil {
		a.Item = item
	}
	return a
}

func (s *PostgresStore) SaveAttestations(ctx context.Context, rows []Attestation) ([]Attestation, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	out := make([]attestationRow, 0, len(rows))
	for _, a := range rows {
		r, err := toRow(a)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	// ON CONFLICT DO NOTHING ... RETURNING id: the ids that come back are the rows this call inserted.
	var ids []struct{ ID string }
	err := s.db.WithContext(ctx).Model(&attestationRow{}).Clauses(clause.OnConflict{DoNothing: true}, clause.Returning{Columns: []clause.Column{{Name: "id"}}}).Create(&out).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("cre: save attestations: %w", err)
	}
	kept := map[string]bool{}
	for _, id := range ids {
		kept[id.ID] = true
	}
	inserted := make([]Attestation, 0, len(rows))
	for _, a := range rows {
		if kept[a.ID] {
			inserted = append(inserted, a)
		}
	}
	return inserted, nil
}

func (s *PostgresStore) Seen(ctx context.Context, provider string, payloadHash []byte) (bool, error) {
	var n int64
	if err := s.db.WithContext(ctx).Model(&attestationRow{}).Where("provider = ? AND payload_hash = ? AND subject_type <> ?", provider, payloadHash, "report").Count(&n).Error; err != nil {
		return false, fmt.Errorf("cre: seen: %w", err)
	}
	return n > 0, nil
}

func (s *PostgresStore) ListAttestations(ctx context.Context, provider string, kind Kind, limit int) ([]Attestation, error) {
	q := s.db.WithContext(ctx).Where("provider = ?", provider).Order("recorded_at DESC, item_index, id").Limit(limit)
	if kind != "" {
		q = q.Where("kind = ?", string(kind))
	}
	var rows []attestationRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("cre: list attestations: %w", err)
	}
	out := make([]Attestation, len(rows))
	for i, r := range rows {
		out[i] = fromRow(r)
	}
	return out, nil
}

func (s *PostgresStore) GetAttestation(ctx context.Context, id string) (Attestation, bool, error) {
	var r attestationRow
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Attestation{}, false, nil
	}
	if err != nil {
		return Attestation{}, false, fmt.Errorf("cre: get attestation: %w", err)
	}
	return fromRow(r), true, nil
}

func (s *PostgresStore) LatestAttestation(ctx context.Context, provider string, kind Kind, status Status) (Attestation, bool, error) {
	var r attestationRow
	q := s.db.WithContext(ctx).Where("provider = ? AND kind = ?", provider, string(kind))
	if status != "" {
		q = q.Where("status = ?", string(status))
	}
	err := q.Order("recorded_at DESC, item_index").First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Attestation{}, false, nil
	}
	if err != nil {
		return Attestation{}, false, fmt.Errorf("cre: latest attestation: %w", err)
	}
	return fromRow(r), true, nil
}

func (s *PostgresStore) RememberSubjects(ctx context.Context, subjects []Subject) error {
	if len(subjects) == 0 {
		return nil
	}
	rows := make([]subjectRow, 0, len(subjects))
	for _, sub := range subjects {
		facts, err := json.Marshal(sub.Facts)
		if err != nil {
			return fmt.Errorf("cre: encode subject: %w", err)
		}
		rows = append(rows, subjectRow{Kind: string(sub.Kind), SubjectKey: sub.Key[:], SubjectID: sub.ID, Facts: facts, AskedAt: sub.AskedAt})
	}
	// The first facts served stand; a later ask must not rewrite what the verifier compares against.
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "kind"}, {Name: "subject_key"}}, DoUpdates: clause.AssignmentColumns([]string{"asked_at"})}).Create(&rows).Error
	if err != nil {
		return fmt.Errorf("cre: remember subjects: %w", err)
	}
	return nil
}

func subjectFromRow(r subjectRow) Subject {
	s := Subject{Kind: Kind(r.Kind), ID: r.SubjectID, AskedAt: r.AskedAt.UTC()}
	copy(s.Key[:], r.SubjectKey)
	_ = json.Unmarshal(r.Facts, &s.Facts)
	return s
}

func (s *PostgresStore) LookupSubject(ctx context.Context, kind Kind, key [32]byte) (Subject, bool, error) {
	var r subjectRow
	err := s.db.WithContext(ctx).Where("kind = ? AND subject_key = ?", string(kind), key[:]).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Subject{}, false, nil
	}
	if err != nil {
		return Subject{}, false, fmt.Errorf("cre: lookup subject: %w", err)
	}
	return subjectFromRow(r), true, nil
}

func (s *PostgresStore) LatestSubject(ctx context.Context, kind Kind) (Subject, bool, error) {
	var r subjectRow
	err := s.db.WithContext(ctx).Where("kind = ?", string(kind)).Order("asked_at DESC").First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Subject{}, false, nil
	}
	if err != nil {
		return Subject{}, false, fmt.Errorf("cre: latest subject: %w", err)
	}
	return subjectFromRow(r), true, nil
}

func (s *PostgresStore) RecordRun(ctx context.Context, run Run) error {
	row := runRow{Kind: string(run.Kind), Provider: run.Provider, ExecutionID: run.ExecutionID, Status: run.Status, Detail: run.Detail, StartedAt: run.StartedAt}
	if row.StartedAt.IsZero() {
		row.StartedAt = time.Now().UTC()
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("cre: record run: %w", err)
	}
	return nil
}

func (s *PostgresStore) LatestRun(ctx context.Context, kind Kind) (Run, bool, error) {
	var r runRow
	err := s.db.WithContext(ctx).Where("kind = ?", string(kind)).Order("started_at DESC, id DESC").First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Run{}, false, nil
	}
	if err != nil {
		return Run{}, false, fmt.Errorf("cre: latest run: %w", err)
	}
	return Run{Kind: Kind(r.Kind), Provider: r.Provider, ExecutionID: r.ExecutionID, Status: r.Status, Detail: r.Detail, StartedAt: r.StartedAt.UTC()}, true, nil
}

func (s *PostgresStore) GetCursor(ctx context.Context, scope CursorScope) (Cursor, error) {
	var r cursorRow
	err := s.db.WithContext(ctx).Where("scope = ?", scope.String()).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Cursor{}, nil
	}
	if err != nil {
		return Cursor{}, fmt.Errorf("cre: get cursor: %w", err)
	}
	return Cursor{Block: uint64(r.BlockNumber), Seq: uint64(r.Seq)}, nil
}

func (s *PostgresStore) SetCursor(ctx context.Context, scope CursorScope, c Cursor) error {
	row := cursorRow{Scope: scope.String(), BlockNumber: int64(c.Block), Seq: int64(c.Seq), UpdatedAt: time.Now().UTC()}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scope"}}, DoUpdates: clause.AssignmentColumns([]string{"block_number", "seq", "updated_at"})}).Create(&row).Error
	if err != nil {
		return fmt.Errorf("cre: set cursor: %w", err)
	}
	return nil
}
