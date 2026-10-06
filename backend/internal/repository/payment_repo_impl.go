package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PaymentRepository defines all persistence operations for PaymentRequest records.
type PaymentRepository interface {
	Create(p *models.PaymentRequest) error
	Update(p *models.PaymentRequest) error
	GetByID(id uint) (*models.PaymentRequest, error)
	GetByReferenceID(refID string) (*models.PaymentRequest, error)
	GetByReferenceIDAndPlatform(refID string, platformID uint) (*models.PaymentRequest, error)
	ListByPlatform(platformID uint, opts ...QueryOption) ([]models.PaymentRequest, error)
	// ListByPlatformAndState returns payments for a platform filtered by state.
	// If state is empty, all payments are returned (equivalent to ListByPlatform).
	ListByPlatformAndState(platformID uint, state string, opts ...QueryOption) ([]models.PaymentRequest, error)
	CountByPlatform(platformID uint, state string) (int64, error)
	ExpireStale(cutoff time.Time) (int64, error)
	UpdateState(id uint, state string) error
	// FinalizeFromConfirmedDeposits runs an atomic read-sum-update transaction:
	//   1. SELECT … FOR UPDATE on the PaymentRequest row.
	//   2. If the payment is already in a terminal state (not OPEN or
	//      PARTIALLY_FILLED), it returns changed=false without touching anything.
	//   3. SUMs the amount of all confirmed deposits linked to the payment.
	//   4. Derives the new state (FILLED / PARTIALLY_FILLED / OVER_FILLED).
	//   5. Persists the new state (and sets confirmed_at for terminal fills).
	// Returns the new state, the summed total, and whether a state transition
	// actually occurred. Callers should treat changed=false as a no-op (another
	// goroutine already finalized the payment).
	FinalizeFromConfirmedDeposits(paymentRequestID uint) (newState string, total decimal.Decimal, changed bool, err error)
}

// PaymentRepositoryImpl is the GORM-backed implementation of PaymentRepository.
type PaymentRepositoryImpl struct {
	db *gorm.DB
}

// NewPaymentRepository constructs a new PaymentRepository backed by the provided *gorm.DB.
func NewPaymentRepository(db *gorm.DB) PaymentRepository {
	return &PaymentRepositoryImpl{db: db}
}

// Create inserts a new PaymentRequest record.
func (r *PaymentRepositoryImpl) Create(p *models.PaymentRequest) error {
	return r.db.Create(p).Error
}

// Update saves all fields of the given PaymentRequest.
func (r *PaymentRepositoryImpl) Update(p *models.PaymentRequest) error {
	return r.db.Save(p).Error
}

// GetByID fetches a PaymentRequest by primary key, preloading its Deposits.
func (r *PaymentRepositoryImpl) GetByID(id uint) (*models.PaymentRequest, error) {
	var pr models.PaymentRequest
	err := r.db.Preload("Deposits").First(&pr, id).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetByReferenceID fetches a PaymentRequest by its business reference ID,
// preloading Deposits, Member, and ExternalPlatform associations.
func (r *PaymentRepositoryImpl) GetByReferenceID(refID string) (*models.PaymentRequest, error) {
	var pr models.PaymentRequest
	err := r.db.
		Preload("Deposits").
		Preload("Member").
		Preload("ExternalPlatform").
		Where("reference_id = ?", refID).
		First(&pr).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetByReferenceIDAndPlatform fetches a PaymentRequest scoped to a specific platform.
func (r *PaymentRepositoryImpl) GetByReferenceIDAndPlatform(refID string, platformID uint) (*models.PaymentRequest, error) {
	var pr models.PaymentRequest
	err := r.db.
		Where("reference_id = ? AND external_platform_id = ?", refID, platformID).
		First(&pr).Error
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

// ListByPlatform returns all PaymentRequests for a given platform, applying
// the provided QueryOptions. If no ordering option is supplied, results are
// ordered by created_at DESC.
func (r *PaymentRepositoryImpl) ListByPlatform(platformID uint, opts ...QueryOption) ([]models.PaymentRequest, error) {
	// Default ordering unless the caller supplies their own.
	if len(opts) == 0 {
		opts = append(opts, WithDescendingOrder("created_at"))
	} else {
		hasOrder := false
		for _, o := range opts {
			// Probe a no-op DB to detect whether any order was set.
			probe := r.db.Session(&gorm.Session{NewDB: true})
			after := o(probe)
			if after.Statement != nil && len(after.Statement.Clauses) > 0 {
				if _, ok := after.Statement.Clauses["ORDER BY"]; ok {
					hasOrder = true
					break
				}
			}
		}
		if !hasOrder {
			opts = append([]QueryOption{WithDescendingOrder("created_at")}, opts...)
		}
	}

	q := Apply(r.db.Where("external_platform_id = ?", platformID), opts...)
	var results []models.PaymentRequest
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// ListByPlatformAndState returns PaymentRequests for a platform optionally
// filtered by state. If state is empty it behaves identically to ListByPlatform.
func (r *PaymentRepositoryImpl) ListByPlatformAndState(platformID uint, state string, opts ...QueryOption) ([]models.PaymentRequest, error) {
	base := r.db.Where("external_platform_id = ?", platformID)
	if state != "" {
		base = base.Where("state = ?", state)
	}

	// Apply default ordering when caller does not specify one.
	hasOrder := false
	for _, o := range opts {
		probe := r.db.Session(&gorm.Session{NewDB: true})
		after := o(probe)
		if after.Statement != nil {
			if _, ok := after.Statement.Clauses["ORDER BY"]; ok {
				hasOrder = true
				break
			}
		}
	}
	if !hasOrder {
		opts = append([]QueryOption{WithDescendingOrder("created_at")}, opts...)
	}

	q := Apply(base, opts...)
	var results []models.PaymentRequest
	if err := q.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// CountByPlatform returns the number of PaymentRequests for a platform,
// optionally filtered by state (empty string = all states).
func (r *PaymentRepositoryImpl) CountByPlatform(platformID uint, state string) (int64, error) {
	q := r.db.Model(&models.PaymentRequest{}).Where("external_platform_id = ?", platformID)
	if state != "" {
		q = q.Where("state = ?", state)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ExpireStale transitions all OPEN PaymentRequests whose expiry has passed to
// CANCELLED. Returns the number of rows affected.
func (r *PaymentRepositoryImpl) ExpireStale(cutoff time.Time) (int64, error) {
	result := r.db.Model(&models.PaymentRequest{}).
		Where("state = ? AND expires_at < ?", models.PaymentStateOpen, cutoff).
		Update("state", models.PaymentStateCancelled)
	return result.RowsAffected, result.Error
}

// UpdateState sets the state field of a PaymentRequest without touching other fields.
func (r *PaymentRepositoryImpl) UpdateState(id uint, state string) error {
	return r.db.Model(&models.PaymentRequest{}).
		Where("id = ?", id).
		Update("state", state).Error
}

// FinalizeFromConfirmedDeposits atomically decides and applies the final
// payment state inside a single database transaction. The payment row is
// locked with SELECT … FOR UPDATE so concurrent block-processor goroutines
// cannot both apply the same transition.
func (r *PaymentRepositoryImpl) FinalizeFromConfirmedDeposits(paymentRequestID uint) (newState string, total decimal.Decimal, changed bool, err error) {
	err = r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Lock the payment row for the duration of the transaction.
		var p models.PaymentRequest
		if txErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&p, paymentRequestID).Error; txErr != nil {
			return txErr
		}

		// 2. Already in a terminal state — nothing to do.
		if p.State != models.PaymentStateOpen && p.State != models.PaymentStatePartiallyFilled {
			changed = false
			return nil
		}

		// 3. Sum confirmed deposits inside the same transaction so the read is
		//    consistent with the lock we hold on the payment row.
		// TODO(phase-I): convert deposit amounts to USD before comparison
		var rawTotal decimal.Decimal
		if txErr := tx.Model(&models.Deposit{}).
			Where("payment_request_id = ? AND status = ?", paymentRequestID, models.DepositStatusConfirmed).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&rawTotal).Error; txErr != nil {
			return txErr
		}
		total = rawTotal

		if total.IsZero() {
			changed = false
			return nil
		}

		// 4. Compute new state.
		diff := total.Sub(p.AmountInUSD)
		switch {
		case diff.IsZero():
			newState = models.PaymentStateFilled
		case diff.IsPositive():
			newState = models.PaymentStateOverFilled
		default:
			newState = models.PaymentStatePartiallyFilled
		}

		// 5. Persist: update state and, for terminal fills, set confirmed_at.
		now := time.Now()
		updates := map[string]any{"state": newState}
		if newState == models.PaymentStateFilled || newState == models.PaymentStateOverFilled {
			updates["confirmed_at"] = now
		}
		if txErr := tx.Model(&p).Updates(updates).Error; txErr != nil {
			return txErr
		}

		changed = true
		return nil
	})
	return newState, total, changed, err
}
