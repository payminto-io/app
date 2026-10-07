package ledger

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service is the only writer of ledger rows.
type Service struct {
	db          *gorm.DB
	maxBackdate time.Duration
	maxFuture   time.Duration
	// env is the fallback for keys and contexts that name no environment; guard is asked before every row.
	env   environment.Environment
	guard environment.Guard
}

type Option func(*Service)

// WithEnvironment sets the process environment: the fallback for unlabelled keys and the guard's reference.
func WithEnvironment(env environment.Environment) Option {
	return func(s *Service) {
		s.env = env
		if g, err := environment.NewGuard(env); err == nil {
			s.guard = g
		}
	}
}

// WithGuard replaces the guard built by WithEnvironment, e.g. with the shared process guard.
func WithGuard(guard environment.Guard) Option {
	return func(s *Service) { s.guard = guard }
}

// WithPostedAtWindow bounds how far PostedAt may sit behind or ahead of the clock.
// posted_at is the accounting date, so a closed period can only move within this window.
func WithPostedAtWindow(maxBackdate, maxFuture time.Duration) Option {
	return func(s *Service) {
		s.maxBackdate = maxBackdate
		s.maxFuture = maxFuture
	}
}

const (
	defaultMaxBackdate = 7 * 24 * time.Hour
	defaultMaxFuture   = 5 * time.Minute
)

// New defaults to the test environment so a service built without options can never touch live rows.
func New(db *gorm.DB, opts ...Option) *Service {
	s := &Service{db: db, maxBackdate: defaultMaxBackdate, maxFuture: defaultMaxFuture}
	WithEnvironment(environment.Test)(s)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Environment is the process environment this service serves.
func (s *Service) Environment() environment.Environment { return s.env }

// resolve picks the environment for one operation and asks the guard before any row is touched.
func (s *Service) resolve(ctx context.Context, explicit environment.Environment) (environment.Environment, error) {
	env := environment.Resolve(ctx, explicit, s.env)
	if s.guard != nil {
		if err := s.guard.Require(ctx, env); err != nil {
			return "", err
		}
	}
	return env, nil
}

// DB exposes the handle for callers that read ledger rows directly (tests, reports).
func (s *Service) DB() *gorm.DB { return s.db }

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
	explicit, _ := j.explicitEnvironment()
	env, err := s.resolve(ctx, explicit)
	if err != nil {
		return Receipt{}, err
	}
	tx = tx.WithContext(ctx)
	hash := j.requestHash()
	now := time.Now().UTC()
	postedAt := j.PostedAt
	if postedAt.IsZero() {
		postedAt = now
	}
	if postedAt.Before(now.Add(-s.maxBackdate)) || postedAt.After(now.Add(s.maxFuture)) {
		return Receipt{}, fmt.Errorf("%w: %s", ErrPostedAt, postedAt.Format(time.RFC3339))
	}

	// Accounts first, in key order, so two posters touching the same new accounts never lock in opposite order.
	accountIDs := make(map[AccountKey]AccountID, len(j.Lines))
	for _, key := range sortedAccountKeys(j.Lines) {
		id, err := ensureAccount(tx, key, env)
		if err != nil {
			return Receipt{}, err
		}
		accountIDs[key] = id
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
		lines = append(lines, LineRow{JournalID: row.ID, AccountID: accountIDs[l.Account], Asset: l.Account.Asset, Amount: l.Amount})
	}
	if err := tx.Create(&lines).Error; err != nil {
		return Receipt{}, fmt.Errorf("ledger: insert lines: %w", err)
	}
	return Receipt{ID: row.ID}, nil
}

func sortedAccountKeys(lines []Line) []AccountKey {
	seen := make(map[AccountKey]struct{}, len(lines))
	keys := make([]AccountKey, 0, len(lines))
	for _, l := range lines {
		if _, ok := seen[l.Account]; !ok {
			seen[l.Account] = struct{}{}
			keys = append(keys, l.Account)
		}
	}
	slices.SortFunc(keys, func(a, b AccountKey) int {
		return cmp.Or(
			cmp.Compare(a.OwnerType, b.OwnerType),
			cmp.Compare(a.OwnerID, b.OwnerID),
			cmp.Compare(a.Asset, b.Asset),
			cmp.Compare(a.Kind, b.Kind),
		)
	})
	return keys
}

// EnsureAccount returns the account for key, creating it if absent.
func (s *Service) EnsureAccount(ctx context.Context, key AccountKey) (AccountID, error) {
	if err := key.validate(); err != nil {
		return 0, err
	}
	env, err := s.resolve(ctx, key.Environment)
	if err != nil {
		return 0, err
	}
	return ensureAccount(s.db.WithContext(ctx), key, env)
}

func ensureAccount(tx *gorm.DB, key AccountKey, env environment.Environment) (AccountID, error) {
	row := AccountRow{Environment: env, OwnerType: key.OwnerType, OwnerID: key.OwnerID, Asset: key.Asset, Kind: key.Kind}
	res := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "environment"}, {Name: "owner_type"}, {Name: "owner_id"}, {Name: "asset"}, {Name: "kind"}},
		DoNothing: true,
	}).Create(&row)
	if res.Error != nil {
		return 0, fmt.Errorf("ledger: ensure account %+v: %w", key, res.Error)
	}
	if res.RowsAffected > 0 {
		return row.ID, nil
	}
	id, err := lookupAccount(tx, key, env)
	if err != nil {
		return 0, fmt.Errorf("ledger: ensure account %+v: %w", key, err)
	}
	return id, nil
}

func lookupAccount(tx *gorm.DB, key AccountKey, env environment.Environment) (AccountID, error) {
	var row AccountRow
	err := tx.Where("environment = ? AND owner_type = ? AND owner_id = ? AND asset = ? AND kind = ?", env, key.OwnerType, key.OwnerID, key.Asset, key.Kind).First(&row).Error
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
	env, err := s.resolve(ctx, key.Environment)
	if err != nil {
		return 0, err
	}
	return lookupAccount(s.db.WithContext(ctx), key, env)
}

type sumRow struct {
	Asset string
	Total decimal.Decimal
}

// Balance is the signed sum of an account's lines (debits positive). An account of the other environment is not found.
func (s *Service) Balance(ctx context.Context, accountID AccountID) (decimal.Decimal, error) {
	env, err := s.resolve(ctx, "")
	if err != nil {
		return decimal.Zero, err
	}
	db := s.db.WithContext(ctx)
	var exists int64
	if err := db.Model(&AccountRow{}).Where("id = ? AND environment = ?", accountID, env).Count(&exists).Error; err != nil {
		return decimal.Zero, fmt.Errorf("ledger: balance: %w", err)
	}
	if exists == 0 {
		return decimal.Zero, fmt.Errorf("%w: id %d", ErrAccountNotFound, accountID)
	}
	var row sumRow
	err = db.Model(&LineRow{}).
		Select("COALESCE(SUM(amount), 0) AS total").
		Where("account_id = ?", accountID).
		Scan(&row).Error
	if err != nil {
		return decimal.Zero, fmt.Errorf("ledger: balance: %w", err)
	}
	return row.Total, nil
}

// AccountBalance is one account of an owner with its signed and natural balance.
type AccountBalance struct {
	ID      AccountID
	Account AccountKey
	Signed  decimal.Decimal
	Natural decimal.Decimal
}

type accountSumRow struct {
	ID        AccountID
	OwnerType OwnerType
	OwnerID   string
	Asset     string
	Kind      AccountKind
	Total     decimal.Decimal
}

// AccountBalances lists every account of one owner; accounts with no lines are included at zero.
func (s *Service) AccountBalances(ctx context.Context, ownerType OwnerType, ownerID string) ([]AccountBalance, error) {
	env, err := s.resolve(ctx, "")
	if err != nil {
		return nil, err
	}
	var rows []accountSumRow
	err = s.db.WithContext(ctx).Model(&AccountRow{}).
		Select(`ledger_accounts.id, ledger_accounts.owner_type, ledger_accounts.owner_id, ledger_accounts.asset, ledger_accounts.kind,
			COALESCE(SUM(ledger_lines.amount), 0) AS total`).
		Joins("LEFT JOIN ledger_lines ON ledger_lines.account_id = ledger_accounts.id").
		Where("ledger_accounts.environment = ? AND ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ?", env, ownerType, ownerID).
		Group("ledger_accounts.id, ledger_accounts.owner_type, ledger_accounts.owner_id, ledger_accounts.asset, ledger_accounts.kind").
		Order("ledger_accounts.asset, ledger_accounts.kind").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: account balances: %w", err)
	}
	out := make([]AccountBalance, 0, len(rows))
	for _, r := range rows {
		key := AccountKey{OwnerType: r.OwnerType, OwnerID: r.OwnerID, Asset: r.Asset, Kind: r.Kind, Environment: env}
		out = append(out, AccountBalance{ID: r.ID, Account: key, Signed: r.Total, Natural: NaturalBalance(r.Kind, r.Total)})
	}
	return out, nil
}

// Balances returns the natural balance per asset for an owner with one account kind per asset.
// Netting an asset account against a liability account would be a number nobody can read, so that case is ErrMixedKinds.
func (s *Service) Balances(ctx context.Context, ownerType OwnerType, ownerID string) (map[string]decimal.Decimal, error) {
	accounts, err := s.AccountBalances(ctx, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]decimal.Decimal, len(accounts))
	for _, a := range accounts {
		if _, dup := out[a.Account.Asset]; dup {
			return nil, fmt.Errorf("%w: %s/%s %s", ErrMixedKinds, ownerType, ownerID, a.Account.Asset)
		}
		out[a.Account.Asset] = a.Natural
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
	// Running is the account's signed balance after this line; RunningNatural flips it for credit-normal kinds.
	Running        decimal.Decimal
	RunningNatural decimal.Decimal
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

type openingRow struct {
	AccountID AccountID
	Total     decimal.Decimal
}

// Statement lists an owner's lines posted in [from, to) with a per-account running balance that starts from the lines before from.
func (s *Service) Statement(ctx context.Context, ownerType OwnerType, ownerID string, from, to time.Time) ([]StatementLine, error) {
	env, err := s.resolve(ctx, "")
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	from, to = from.UTC(), to.UTC()
	var opening []openingRow
	err = db.Model(&LineRow{}).
		Select("ledger_lines.account_id AS account_id, COALESCE(SUM(ledger_lines.amount), 0) AS total").
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_accounts.environment = ? AND ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ? AND ledger_journals.posted_at < ?", env, ownerType, ownerID, from).
		Group("ledger_lines.account_id").
		Scan(&opening).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: statement opening balance: %w", err)
	}
	running := make(map[AccountID]decimal.Decimal, len(opening))
	for _, r := range opening {
		running[r.AccountID] = r.Total
	}

	var rows []statementRow
	err = db.Model(&LineRow{}).
		Select(`ledger_lines.id AS line_id, ledger_lines.journal_id, ledger_journals.kind AS journal_kind,
			ledger_journals.reference_type, ledger_journals.reference_id,
			ledger_lines.account_id, ledger_accounts.kind AS account_kind, ledger_lines.asset, ledger_lines.amount,
			ledger_journals.posted_at`).
		Joins("JOIN ledger_accounts ON ledger_accounts.id = ledger_lines.account_id").
		Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_accounts.environment = ? AND ledger_accounts.owner_type = ? AND ledger_accounts.owner_id = ? AND ledger_journals.posted_at >= ? AND ledger_journals.posted_at < ?", env, ownerType, ownerID, from, to).
		Order("ledger_journals.posted_at, ledger_journals.id, ledger_lines.id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("ledger: statement: %w", err)
	}
	out := make([]StatementLine, 0, len(rows))
	for _, r := range rows {
		running[r.AccountID] = running[r.AccountID].Add(r.Amount)
		out = append(out, StatementLine{
			LineID:         r.LineID,
			JournalID:      r.JournalID,
			JournalKind:    r.JournalKind,
			Reference:      Reference{Type: r.ReferenceType, ID: r.ReferenceID},
			AccountID:      r.AccountID,
			AccountKind:    r.AccountKind,
			Asset:          r.Asset,
			Amount:         r.Amount,
			PostedAt:       r.PostedAt,
			Running:        running[r.AccountID],
			RunningNatural: NaturalBalance(r.AccountKind, running[r.AccountID]),
		})
	}
	return out, nil
}
