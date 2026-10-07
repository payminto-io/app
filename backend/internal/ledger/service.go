package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service is the only writer of ledger rows.
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// Receipt is what PostIn returns; Replayed is true when the key had already been posted.
type Receipt struct {
	ID       JournalID
	Replayed bool
}

// Post writes one journal in its own transaction. A replayed idempotency key returns the original id.
func (s *Service) Post(ctx context.Context, j Journal) (JournalID, error) {
	var receipt Receipt
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		receipt, err = s.PostIn(ctx, tx, j)
		return err
	})
	if err != nil {
		return 0, err
	}
	return receipt.ID, nil
}

// Transaction runs fn inside one database transaction so callers can post alongside their own writes.
func (s *Service) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

// PostIn posts within a caller-owned transaction; the deferred balance trigger still runs at that commit.
func (s *Service) PostIn(ctx context.Context, tx *gorm.DB, j Journal) (Receipt, error) {
	if err := j.Validate(); err != nil {
		return Receipt{}, err
	}
	tx = tx.WithContext(ctx)
	hash := j.requestHash()
	postedAt := j.PostedAt
	if postedAt.IsZero() {
		postedAt = time.Now().UTC()
	}
	row := JournalRow{
		Kind:           j.Kind,
		ReferenceType:  j.Reference.Type,
		ReferenceID:    j.Reference.ID,
		IdempotencyKey: j.IdempotencyKey,
		RequestHash:    hash,
		PostedAt:       postedAt,
		Metadata:       Metadata(j.Metadata),
	}
	res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "idempotency_key"}}, DoNothing: true}).Create(&row)
	if res.Error != nil {
		return Receipt{}, fmt.Errorf("ledger: insert journal: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		var existing JournalRow
		if err := tx.Where("idempotency_key = ?", j.IdempotencyKey).First(&existing).Error; err != nil {
			return Receipt{}, fmt.Errorf("ledger: load journal for key %q: %w", j.IdempotencyKey, err)
		}
		if existing.RequestHash != hash {
			return Receipt{}, fmt.Errorf("%w: key %q", ErrIdempotencyConflict, j.IdempotencyKey)
		}
		return Receipt{ID: existing.ID, Replayed: true}, nil
	}

	lines := make([]LineRow, 0, len(j.Lines))
	for _, l := range j.Lines {
		accountID, err := ensureAccount(tx, l.Account)
		if err != nil {
			return Receipt{}, err
		}
		lines = append(lines, LineRow{JournalID: row.ID, AccountID: accountID, Asset: l.Account.Asset, Amount: l.Amount})
	}
	if err := tx.Create(&lines).Error; err != nil {
		return Receipt{}, fmt.Errorf("ledger: insert lines: %w", err)
	}
	return Receipt{ID: row.ID}, nil
}

// EnsureAccount returns the account for key, creating it if absent.
func (s *Service) EnsureAccount(ctx context.Context, key AccountKey) (AccountID, error) {
	if err := key.validate(); err != nil {
		return 0, err
	}
	return ensureAccount(s.db.WithContext(ctx), key)
}

func ensureAccount(tx *gorm.DB, key AccountKey) (AccountID, error) {
	row := AccountRow{OwnerType: key.OwnerType, OwnerID: key.OwnerID, Asset: key.Asset, Kind: key.Kind}
	res := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "owner_type"}, {Name: "owner_id"}, {Name: "asset"}, {Name: "kind"}},
		DoNothing: true,
	}).Create(&row)
	if res.Error != nil {
		return 0, fmt.Errorf("ledger: ensure account %+v: %w", key, res.Error)
	}
	if res.RowsAffected > 0 {
		return row.ID, nil
	}
	id, err := lookupAccount(tx, key)
	if err != nil {
		return 0, fmt.Errorf("ledger: ensure account %+v: %w", key, err)
	}
	return id, nil
}

func lookupAccount(tx *gorm.DB, key AccountKey) (AccountID, error) {
	var row AccountRow
	err := tx.Where("owner_type = ? AND owner_id = ? AND asset = ? AND kind = ?", key.OwnerType, key.OwnerID, key.Asset, key.Kind).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrAccountNotFound
	}
	if err != nil {
		return 0, err
	}
	return row.ID, nil
}

// AccountID resolves key without creating anything.
func (s *Service) AccountID(ctx context.Context, key AccountKey) (AccountID, error) {
	return lookupAccount(s.db.WithContext(ctx), key)
}

type sumRow struct {
	Asset string
	Total decimal.Decimal
}

// Balance is the signed sum of an account's lines (debits positive).
func (s *Service) Balance(ctx context.Context, accountID AccountID) (decimal.Decimal, error) {
	db := s.db.WithContext(ctx)
	var exists int64
	if err := db.Model(&AccountRow{}).Where("id = ?", accountID).Count(&exists).Error; err != nil {
		return decimal.Zero, fmt.Errorf("ledger: balance: %w", err)
	}
	if exists == 0 {
		return decimal.Zero, fmt.Errorf("%w: id %d", ErrAccountNotFound, accountID)
	}
	var row sumRow
	err := db.Model(&LineRow{}).
		Select("COALESCE(SUM(amount), 0) AS total").
		Where("account_id = ?", accountID).
		Scan(&row).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("ledger: balance: %w", err)
	}
	return row.Total, nil
}

// Balances sums every account of one owner, grouped by asset across kinds.
func (s *Service) Balances(ctx context.Context, ownerType OwnerType, ownerID string) (map[string]decimal.Decimal, error) {
	var rows []sumRow
	err := s.db.WithContext(ctx).Model(&LineRow{}).
		Select("ledger_lines.asset AS asset, COALESCE(SUM(ledger_lines.amount), 0) AS total").
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Where("ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ?", ownerType, ownerID).
		Group("ledger_lines.asset").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: balances: %w", err)
	}
	out := make(map[string]decimal.Decimal, len(rows))
	for _, r := range rows {
		out[r.Asset] = r.Total
	}
	return out, nil
}

// StatementLine is one posted line with the owner's running balance for that asset after it.
type StatementLine struct {
	LineID      uint
	JournalID   JournalID
	JournalKind JournalKind
	Reference   Reference
	AccountID   AccountID
	AccountKind AccountKind
	Asset       string
	Amount      decimal.Decimal
	PostedAt    time.Time
	Running     decimal.Decimal
}

type statementRow struct {
	LineID        uint
	JournalID     JournalID
	JournalKind   JournalKind
	ReferenceType string
	ReferenceID   string
	AccountID     AccountID
	AccountKind   AccountKind
	Asset         string
	Amount        decimal.Decimal
	PostedAt      time.Time
}

// Statement lists an owner's lines posted in [from, to) with a per-asset running balance that starts from the lines before from.
func (s *Service) Statement(ctx context.Context, ownerType OwnerType, ownerID string, from, to time.Time) ([]StatementLine, error) {
	db := s.db.WithContext(ctx)
	var opening []sumRow
	err := db.Model(&LineRow{}).
		Select("ledger_lines.asset AS asset, COALESCE(SUM(ledger_lines.amount), 0) AS total").
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ? AND ledger_journals.posted_at < ?", ownerType, ownerID, from).
		Group("ledger_lines.asset").
		Scan(&opening).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: statement opening balance: %w", err)
	}
	running := make(map[string]decimal.Decimal, len(opening))
	for _, r := range opening {
		running[r.Asset] = r.Total
	}

	var rows []statementRow
	err = db.Model(&LineRow{}).
		Select(`ledger_lines.id AS line_id, ledger_lines.journal_id, ledger_journals.kind AS journal_kind,
			ledger_journals.reference_type, ledger_journals.reference_id,
			ledger_lines.account_id, ledger_accounts.kind AS account_kind, ledger_lines.asset, ledger_lines.amount,
			ledger_journals.posted_at`).
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ? AND ledger_journals.posted_at >= ? AND ledger_journals.posted_at < ?", ownerType, ownerID, from, to).
		Order("ledger_journals.posted_at, ledger_journals.id, ledger_lines.id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: statement: %w", err)
	}
	out := make([]StatementLine, 0, len(rows))
	for _, r := range rows {
		running[r.Asset] = running[r.Asset].Add(r.Amount)
		out = append(out, StatementLine{
			LineID:      r.LineID,
			JournalID:   r.JournalID,
			JournalKind: r.JournalKind,
			Reference:   Reference{Type: r.ReferenceType, ID: r.ReferenceID},
			AccountID:   r.AccountID,
			AccountKind: r.AccountKind,
			Asset:       r.Asset,
			Amount:      r.Amount,
			PostedAt:    r.PostedAt,
			Running:     running[r.Asset],
		})
	}
	return out, nil
}
