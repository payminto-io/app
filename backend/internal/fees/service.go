package fees

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/payminto/payminto/backend/internal/ledger"
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

func (s *Service) candidates(ctx context.Context, q Query) ([]Rule, error) {
	var rows []ruleRow
	err := s.db.WithContext(ctx).
		Where("method = ? AND currency = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", q.Method, q.Currency, q.At, q.At).
		Order("id").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("fees: load candidate rules: %w", err)
	}
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = r.rule()
	}
	return out, nil
}

func (s *Service) Resolve(ctx context.Context, q Query) (Rule, error) {
	q = q.normalize()
	if q.At.IsZero() {
		q.At = s.now()
	}
	rules, err := s.candidates(ctx, q)
	if err != nil {
		return Rule{}, err
	}
	return resolve(rules, q)
}

func (s *Service) Preview(ctx context.Context, req PreviewRequest) (Breakdown, error) {
	req.Query = req.Query.normalize()
	req.At = s.now()
	rules, err := s.candidates(ctx, req.Query)
	if err != nil {
		return Breakdown{}, err
	}
	return preview(rules, req, s.policy)
}

func (s *Service) CreateRule(ctx context.Context, in RuleInput, actor string) (Rule, error) {
	in, err := validateInput(in, s.policy, s.now())
	if err != nil {
		return Rule{}, err
	}
	row := newRuleRow(uuid.NewString(), 1, in.Scope, in.Pricing, actor)
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return Rule{}, fmt.Errorf("fees: create rule: %w", err)
	}
	return row.rule(), nil
}

// NewVersion inserts version n+1 and closes version n in one transaction; ruleID must be the lineage head.
func (s *Service) NewVersion(ctx context.Context, ruleID uint, p Pricing, actor string) (Rule, error) {
	var created ruleRow
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
		scope := prev.rule().Scope
		p, err = validatePricing(p, scope, s.policy, s.now())
		if err != nil {
			return err
		}
		closeAt := *p.EffectiveFrom
		if prev.EffectiveFrom.After(closeAt) {
			closeAt = prev.EffectiveFrom
		}
		if prev.EffectiveTo == nil || prev.EffectiveTo.After(closeAt) {
			if err := tx.Model(&ruleRow{}).Where("id = ?", prev.ID).Update("effective_to", closeAt).Error; err != nil {
				return fmt.Errorf("fees: close version %d: %w", prev.Version, err)
			}
		}
		created = newRuleRow(prev.LineageID, prev.Version+1, scope, p, actor)
		if err := tx.Create(&created).Error; err != nil {
			var pgErr *pgconn.PgError
			if errors.Is(err, gorm.ErrDuplicatedKey) || (errors.As(err, &pgErr) && pgErr.Code == "23505") {
				return ErrStaleVersion
			}
			return fmt.Errorf("fees: insert version %d: %w", created.Version, err)
		}
		return nil
	})
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
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = r.rule()
	}
	return out, nil
}

// ApplyToPayment writes the snapshot (idempotent for the same rule) and posts the fee journal in tx.
func (s *Service) ApplyToPayment(ctx context.Context, tx *gorm.DB, pf PaymentFee) error {
	j, post, err := feeJournal(pf)
	if err != nil {
		return err
	}
	tx = tx.WithContext(ctx)
	b := pf.Breakdown
	res := tx.Exec(`UPDATE payment_requests SET fee_rule_id = ?, fee_rule_version = ? WHERE id = ? AND fee_rule_id IS NULL`,
		b.RuleID, b.Version, pf.PaymentRequestID)
	if res.Error != nil {
		return fmt.Errorf("fees: snapshot on payment %d: %w", pf.PaymentRequestID, res.Error)
	}
	if res.RowsAffected == 0 {
		var snap struct {
			FeeRuleID      *uint
			FeeRuleVersion *int
		}
		err := tx.Raw(`SELECT fee_rule_id, fee_rule_version FROM payment_requests WHERE id = ?`, pf.PaymentRequestID).Scan(&snap).Error
		if err != nil {
			return fmt.Errorf("fees: read snapshot on payment %d: %w", pf.PaymentRequestID, err)
		}
		if snap.FeeRuleID == nil {
			return fmt.Errorf("%w: %d", ErrPaymentNotFound, pf.PaymentRequestID)
		}
		if *snap.FeeRuleID != b.RuleID || *snap.FeeRuleVersion != b.Version {
			return fmt.Errorf("%w: payment %d has rule %d v%d", ErrSnapshotConflict, pf.PaymentRequestID, *snap.FeeRuleID, *snap.FeeRuleVersion)
		}
	}
	if !post {
		return nil
	}
	if _, err := s.ledger.PostIn(ctx, tx, j); err != nil {
		return fmt.Errorf("fees: post fee journal for payment %d: %w", pf.PaymentRequestID, err)
	}
	return nil
}
