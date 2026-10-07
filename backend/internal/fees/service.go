package fees

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Poster is the slice of the ledger port fees needs.
type Poster interface {
	PostIn(ctx context.Context, tx *gorm.DB, j ledger.Journal) (ledger.Receipt, error)
}

// Service is the Postgres implementation of Port.
type Service struct {
	db     *gorm.DB
	ledger Poster
	policy Policy
	now    func() time.Time
}

var _ Port = (*Service)(nil)

func NewService(db *gorm.DB, poster Poster, policy Policy) *Service {
	return &Service{db: db, ledger: poster, policy: policy, now: time.Now}
}

func toRules(rows []ruleRow) []Rule {
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = r.rule()
	}
	return out
}

func candidates(ctx context.Context, db *gorm.DB, q Query) ([]Rule, error) {
	var rows []ruleRow
	err := db.WithContext(ctx).
		Where("method = ? AND currency = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", q.Method, q.Currency, q.At, q.At).
		Order("id").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("fees: load candidate rules: %w", err)
	}
	return toRules(rows), nil
}

func (s *Service) resolveIn(ctx context.Context, db *gorm.DB, q Query) (Rule, error) {
	q = q.normalize()
	if q.At.IsZero() {
		q.At = s.now()
	}
	rules, err := candidates(ctx, db, q)
	if err != nil {
		return Rule{}, err
	}
	return resolve(rules, q)
}

func (s *Service) Resolve(ctx context.Context, q Query) (Rule, error) {
	return s.resolveIn(ctx, s.db, q)
}

func (s *Service) Preview(ctx context.Context, req PreviewRequest) (Breakdown, error) {
	req.Query = req.Query.normalize()
	req.At = s.now()
	rules, err := candidates(ctx, s.db, req.Query)
	if err != nil {
		return Breakdown{}, err
	}
	return preview(rules, req, s.policy)
}

func isPgCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

// overlapError names the active rules of scope that intersect [from, to).
func overlapError(ctx context.Context, db *gorm.DB, sc Scope, from time.Time, to *time.Time) error {
	q := db.WithContext(ctx).Model(&ruleRow{}).
		Where("method = ? AND currency = ? AND coalesce(connector, '') = ? AND coalesce(card_type, '') = ? AND coalesce(region, '') = ?",
			sc.Method, sc.Currency, deref(sc.Connector), derefCard(sc.CardType), deref(sc.Region)).
		Where("effective_to IS NULL OR effective_to > ?", from).
		Where("effective_to IS NULL OR effective_to > effective_from")
	if to != nil {
		q = q.Where("effective_from < ?", *to)
	}
	var ids []uint
	if err := q.Order("id").Pluck("id", &ids).Error; err != nil {
		return fmt.Errorf("fees: find overlapping rules: %w", err)
	}
	return &OverlapError{RuleIDs: ids}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefCard(c *CardType) string {
	if c == nil {
		return ""
	}
	return string(*c)
}

func (s *Service) CreateRule(ctx context.Context, in RuleInput, actor string) (Rule, error) {
	v, err := validateInput(in, s.policy, s.now())
	if err != nil {
		return Rule{}, err
	}
	row := newRuleRow(uuid.NewString(), 1, v.Scope, v.minorUnits, v.Pricing, actor)
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.First(&row, row.ID).Error
	})
	if isPgCode(err, "23P01") {
		return Rule{}, overlapError(ctx, s.db, v.Scope, *v.EffectiveFrom, v.EffectiveTo)
	}
	if err != nil {
		return Rule{}, fmt.Errorf("fees: create rule: %w", err)
	}
	return row.rule(), nil
}

// NewVersion inserts version n+1 and, in the same transaction, closes every version of the lineage whose
// window extends past the new start; ruleID must be the lineage head.
func (s *Service) NewVersion(ctx context.Context, ruleID uint, p Pricing, actor string) (Rule, error) {
	var created ruleRow
	var scope Scope
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var prev ruleRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&prev, ruleID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("fees: lock rule %d: %w", ruleID, err)
		}
		var newer int64
		if err := tx.Model(&ruleRow{}).Where("lineage_id = ? AND version > ?", prev.LineageID, prev.Version).Count(&newer).Error; err != nil {
			return fmt.Errorf("fees: check lineage head: %w", err)
		}
		if newer > 0 {
			return ErrStaleVersion
		}
		scope = prev.rule().Scope
		p, err = validatePricing(p, scope, prev.MinorUnits, s.policy, s.now())
		if err != nil {
			return err
		}
		if err := tx.Exec(`SET LOCAL fees.closing_version = 'on'`).Error; err != nil {
			return fmt.Errorf("fees: mark versioning transaction: %w", err)
		}
		if err := tx.Exec(`UPDATE fee_rules SET effective_to = GREATEST(effective_from, ?)
			WHERE lineage_id = ? AND (effective_to IS NULL OR effective_to > ?)`,
			*p.EffectiveFrom, prev.LineageID, *p.EffectiveFrom).Error; err != nil {
			return fmt.Errorf("fees: close lineage %s: %w", prev.LineageID, err)
		}
		created = newRuleRow(prev.LineageID, prev.Version+1, scope, prev.MinorUnits, p, actor)
		if err := tx.Create(&created).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || isPgCode(err, "23505") {
				return ErrStaleVersion
			}
			return fmt.Errorf("fees: insert version %d: %w", created.Version, err)
		}
		return tx.First(&created, created.ID).Error
	})
	if isPgCode(err, "23P01") {
		return Rule{}, overlapError(ctx, s.db, scope, *p.EffectiveFrom, p.EffectiveTo)
	}
	if err != nil {
		return Rule{}, err
	}
	return created.rule(), nil
}

func (s *Service) GetRule(ctx context.Context, id uint) (Rule, error) {
	var row ruleRow
	err := s.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Rule{}, ErrNotFound
	}
	if err != nil {
		return Rule{}, fmt.Errorf("fees: get rule %d: %w", id, err)
	}
	return row.rule(), nil
}

func (s *Service) ListRules(ctx context.Context, f RuleFilter) ([]Rule, error) {
	q := s.db.WithContext(ctx).Model(&ruleRow{})
	if f.Method != "" {
		q = q.Where("method = ?", strings.ToLower(strings.TrimSpace(string(f.Method))))
	}
	if f.Currency != "" {
		q = q.Where("currency = ?", strings.ToUpper(strings.TrimSpace(f.Currency)))
	}
	if f.LineageID != "" {
		if _, err := uuid.Parse(f.LineageID); err != nil {
			return nil, invalid("lineage_id", "must be a uuid")
		}
		q = q.Where("lineage_id = ?", f.LineageID)
	}
	var rows []ruleRow
	if err := q.Order("lineage_id, version").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("fees: list rules: %w", err)
	}
	return toRules(rows), nil
}

var attemptPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)

func (ref AttemptRef) validate() error {
	if ref.PaymentRequestID == 0 {
		return invalid("payment_request_id", "required")
	}
	if !attemptPattern.MatchString(ref.AttemptID) {
		return invalid("attempt_id", "must match %s", attemptPattern)
	}
	return nil
}

// lockPayment returns the merchant of a live (not soft-deleted) payment request, locking the row.
func lockPayment(tx *gorm.DB, id uint) (uint, error) {
	var row struct{ MemberID uint }
	res := tx.Raw(`SELECT member_id FROM payment_requests WHERE id = ? AND deleted_at IS NULL FOR UPDATE`, id).Scan(&row)
	if res.Error != nil {
		return 0, fmt.Errorf("fees: lock payment %d: %w", id, res.Error)
	}
	if res.RowsAffected == 0 || row.MemberID == 0 {
		return 0, fmt.Errorf("%w: %d", ErrPaymentNotFound, id)
	}
	return row.MemberID, nil
}

func loadSnapshot(tx *gorm.DB, attemptID string) (snapshotRow, bool, error) {
	var row snapshotRow
	res := tx.Where("attempt_id = ?", attemptID).Limit(1).Find(&row)
	if res.Error != nil {
		return row, false, fmt.Errorf("fees: load snapshot %s: %w", attemptID, res.Error)
	}
	return row, res.RowsAffected == 1, nil
}

func (s *Service) Snapshot(ctx context.Context, tx *gorm.DB, ref AttemptRef, q Query) (Snapshot, error) {
	if err := ref.validate(); err != nil {
		return Snapshot{}, err
	}
	tx = tx.WithContext(ctx)
	merchant, err := lockPayment(tx, ref.PaymentRequestID)
	if err != nil {
		return Snapshot{}, err
	}
	if existing, ok, err := loadSnapshot(tx, ref.AttemptID); err != nil {
		return Snapshot{}, err
	} else if ok {
		if existing.PaymentRequestID != ref.PaymentRequestID {
			return Snapshot{}, fmt.Errorf("%w: attempt %s belongs to payment %d", ErrSnapshotConflict, ref.AttemptID, existing.PaymentRequestID)
		}
		return existing.snapshot(), nil
	}
	q = q.normalize()
	r, err := s.resolveIn(ctx, tx, q)
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.policy.checkBearer(r.Method, r.FeeBearer); err != nil {
		return Snapshot{}, err
	}
	asset, err := ledgerAsset(s.policy.Precision, r.Currency, q.Chain)
	if err != nil {
		return Snapshot{}, err
	}
	row := snapshotRow{
		AttemptID: ref.AttemptID, PaymentRequestID: ref.PaymentRequestID, MerchantID: merchant,
		FeeRuleID: r.ID, FeeRuleVersion: r.Version, Currency: r.Currency, LedgerAsset: asset, FeeBearer: string(r.FeeBearer),
	}
	if err := tx.Create(&row).Error; err != nil {
		return Snapshot{}, fmt.Errorf("fees: record snapshot for attempt %s: %w", ref.AttemptID, err)
	}
	return row.snapshot(), nil
}

func (s *Service) PostFee(ctx context.Context, tx *gorm.DB, ref AttemptRef, captured decimal.Decimal) (Breakdown, error) {
	if err := ref.validate(); err != nil {
		return Breakdown{}, err
	}
	tx = tx.WithContext(ctx)
	row, ok, err := loadSnapshot(tx, ref.AttemptID)
	if err != nil {
		return Breakdown{}, err
	}
	if !ok {
		return Breakdown{}, fmt.Errorf("%w: %s", ErrSnapshotNotFound, ref.AttemptID)
	}
	if row.PaymentRequestID != ref.PaymentRequestID {
		return Breakdown{}, fmt.Errorf("%w: attempt %s belongs to payment %d", ErrSnapshotConflict, ref.AttemptID, row.PaymentRequestID)
	}
	merchant, err := lockPayment(tx, row.PaymentRequestID)
	if err != nil {
		return Breakdown{}, err
	}
	if merchant != row.MerchantID {
		return Breakdown{}, fmt.Errorf("%w: payment %d changed merchant since the snapshot", ErrSnapshotConflict, row.PaymentRequestID)
	}
	var rr ruleRow
	if err := tx.Where("id = ? AND version = ?", row.FeeRuleID, row.FeeRuleVersion).First(&rr).Error; err != nil {
		return Breakdown{}, fmt.Errorf("fees: load snapshotted rule %d v%d: %w", row.FeeRuleID, row.FeeRuleVersion, err)
	}
	r := rr.rule()
	r.FeeBearer = FeeBearer(row.FeeBearer)
	if err := checkAmount(captured, r.MinorUnits, r.Currency); err != nil {
		return Breakdown{}, err
	}
	b, err := computeChecked(r, captured)
	if err != nil {
		return Breakdown{}, err
	}
	snap := row.snapshot()
	if j, post, err := feeJournal(snap, b); err != nil {
		return Breakdown{}, err
	} else if post {
		if _, err := s.ledger.PostIn(ctx, tx, j); err != nil {
			return Breakdown{}, fmt.Errorf("fees: post fee journal for attempt %s: %w", ref.AttemptID, err)
		}
	}
	if err := tx.Exec(`UPDATE payment_requests SET fee_rule_id = ?, fee_rule_version = ? WHERE id = ?`,
		row.FeeRuleID, row.FeeRuleVersion, row.PaymentRequestID).Error; err != nil {
		return Breakdown{}, fmt.Errorf("fees: legacy snapshot on payment %d: %w", row.PaymentRequestID, err)
	}
	return b, nil
}
