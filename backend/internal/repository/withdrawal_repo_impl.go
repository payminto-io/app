package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// WithdrawalRepository defines all data access operations for the Withdrawal model.
type WithdrawalRepository interface {
	Create(w *models.Withdrawal) error
	// GetByID fetches by primary key — no tenant check.
	// INTERNAL USE ONLY — no tenant check. Use GetByIDForPlatform in handler context.
	GetByID(id uint) (*models.Withdrawal, error)
	// GetByIDForPlatform fetches by primary key scoped to the given platform.
	GetByIDForPlatform(id, platformID uint) (*models.Withdrawal, error)
	// GetByReferenceID fetches by reference ID — no tenant check.
	// INTERNAL USE ONLY — no tenant check. Use GetByReferenceIDForPlatform in handler context.
	GetByReferenceID(referenceID string) (*models.Withdrawal, error)
	// GetByReferenceIDForPlatform fetches by reference ID scoped to the given platform.
	GetByReferenceIDForPlatform(referenceID string, platformID uint) (*models.Withdrawal, error)
	Update(w *models.Withdrawal) error
	ListByState(state string, opts ...QueryOption) ([]models.Withdrawal, error)
	ListByPlatform(platformID uint, opts ...QueryOption) ([]models.Withdrawal, error)
	ListPendingForProcessing(limit int) ([]models.Withdrawal, error)
	UpdateState(id uint, fromState, toState string) error
	// Approve atomically transitions pending-approval → pending and sets approver fields.
	// Returns (rowsAffected, error). rowsAffected==0 means the row was not in
	// pending-approval state (concurrent modification guard).
	Approve(id, approverID uint) (int64, error)
	// ApproveForPlatform is the handler-safe approval operation. It adds tenant
	// ownership to the same atomic state transition.
	ApproveForPlatform(id, platformID, approverID uint) (int64, error)
	// AtomicCancel atomically cancels a withdrawal from any cancellable state.
	// Returns (rowsAffected, error). rowsAffected==0 means the row was not in a
	// cancellable state.
	AtomicCancel(id uint) (int64, error)
	// AtomicCancelForPlatform is the handler-safe cancellation operation.
	AtomicCancelForPlatform(id, platformID uint) (int64, error)
	// ClaimForProcessing atomically transitions pending → initiated.
	// Returns (rowsAffected, error). rowsAffected==0 means another worker already claimed it.
	ClaimForProcessing(id uint) (int64, error)
	// RevertClaim transitions initiated → pending, used for rollback on failure.
	RevertClaim(id uint) (int64, error)
	// MarkSent atomically transitions initiated → sent and records the tx hash.
	MarkSent(id uint, txHash string) (int64, error)
	// MarkProcessed atomically transitions sent → processed.
	MarkProcessed(id uint) (int64, error)
	RecordBroadcast(id uint, txHash string, initiatedAt time.Time) error
	// SumByPlatformAndCurrencyInWindow returns the total amount of withdrawals
	// for a (platform, blockchain_currency) pair created at or after `since`.
	// Excludes cancelled and failed withdrawals so caps reflect only money
	// that actually moved (or is still in flight).
	SumByPlatformAndCurrencyInWindow(platformID, blockchainCurrencyID uint, since time.Time) (decimal.Decimal, error)
}

// WithdrawalRepositoryImpl is the GORM-backed implementation.
type WithdrawalRepositoryImpl struct {
	db *gorm.DB
}

// NewWithdrawalRepository constructs a WithdrawalRepositoryImpl.
func NewWithdrawalRepository(db *gorm.DB) WithdrawalRepository {
	return &WithdrawalRepositoryImpl{db: db}
}

// Create inserts a new Withdrawal row.
func (r *WithdrawalRepositoryImpl) Create(w *models.Withdrawal) error {
	return r.db.Create(w).Error
}

// GetByID fetches a Withdrawal by primary key with BlockchainCurrency preloaded.
func (r *WithdrawalRepositoryImpl) GetByID(id uint) (*models.Withdrawal, error) {
	var w models.Withdrawal
	err := r.db.Preload("BlockchainCurrency").First(&w, id).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetByIDForPlatform fetches a Withdrawal by primary key scoped to the given platform.
func (r *WithdrawalRepositoryImpl) GetByIDForPlatform(id, platformID uint) (*models.Withdrawal, error) {
	var w models.Withdrawal
	err := r.db.Preload("BlockchainCurrency").
		Where("id = ? AND external_platform_id = ?", id, platformID).
		First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetByReferenceID fetches a Withdrawal by its unique reference ID.
// INTERNAL USE ONLY — no tenant check.
func (r *WithdrawalRepositoryImpl) GetByReferenceID(referenceID string) (*models.Withdrawal, error) {
	var w models.Withdrawal
	err := r.db.Where("reference_id = ?", referenceID).First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// GetByReferenceIDForPlatform fetches a Withdrawal by reference ID scoped to the given platform.
func (r *WithdrawalRepositoryImpl) GetByReferenceIDForPlatform(referenceID string, platformID uint) (*models.Withdrawal, error) {
	var w models.Withdrawal
	err := r.db.
		Where("reference_id = ? AND external_platform_id = ?", referenceID, platformID).
		First(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// Update saves all fields on the Withdrawal.
func (r *WithdrawalRepositoryImpl) Update(w *models.Withdrawal) error {
	return r.db.Save(w).Error
}

// ListByState returns withdrawals in the given state with optional query options.
func (r *WithdrawalRepositoryImpl) ListByState(state string, opts ...QueryOption) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	q := Apply(r.db.Where("state = ?", state), opts...)
	err := q.Find(&withdrawals).Error
	return withdrawals, err
}

// ListByPlatform returns withdrawals for a given platform with optional query options.
func (r *WithdrawalRepositoryImpl) ListByPlatform(platformID uint, opts ...QueryOption) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	q := Apply(r.db.Where("external_platform_id = ?", platformID), opts...)
	err := q.Order("created_at DESC").Find(&withdrawals).Error
	return withdrawals, err
}

// ListPendingForProcessing returns up to limit withdrawals in 'pending' state,
// ordered by created_at ASC (FIFO).
func (r *WithdrawalRepositoryImpl) ListPendingForProcessing(limit int) ([]models.Withdrawal, error) {
	var withdrawals []models.Withdrawal
	err := r.db.
		Where("state = ?", models.WithdrawalStatePending).
		Preload("BlockchainCurrency").
		Order("created_at ASC").
		Limit(limit).
		Find(&withdrawals).Error
	return withdrawals, err
}

// UpdateState atomically transitions a withdrawal from fromState to toState.
// Returns gorm.ErrRecordNotFound if the row was not in fromState (concurrent
// modification guard).
func (r *WithdrawalRepositoryImpl) UpdateState(id uint, fromState, toState string) error {
	result := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, fromState).
		Update("state", toState)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Approve atomically transitions pending-approval → pending and sets approver fields.
// Returns (rowsAffected, error). rowsAffected==0 means the row was not in
// pending-approval state.
func (r *WithdrawalRepositoryImpl) Approve(id, approverID uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, models.WithdrawalStatePendingApproval).
		Updates(map[string]any{
			"state":                 models.WithdrawalStatePending,
			"approved_by_member_id": approverID,
			"approved_at":           now,
			"updated_at":            now,
		})
	return res.RowsAffected, res.Error
}

// ApproveForPlatform atomically transitions a tenant-owned withdrawal from
// pending-approval to pending and records the approving operator.
func (r *WithdrawalRepositoryImpl) ApproveForPlatform(id, platformID, approverID uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND external_platform_id = ? AND state = ?", id, platformID, models.WithdrawalStatePendingApproval).
		Updates(map[string]any{
			"state":                 models.WithdrawalStatePending,
			"approved_by_member_id": approverID,
			"approved_at":           now,
			"updated_at":            now,
		})
	return res.RowsAffected, res.Error
}

// AtomicCancel atomically cancels a withdrawal from any cancellable state
// (pending-otp, pending-approval, or pending). Returns (rowsAffected, error).
// rowsAffected==0 means the row was not in a cancellable state.
func (r *WithdrawalRepositoryImpl) AtomicCancel(id uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state IN ?", id, []string{
			models.WithdrawalStatePendingOTP,
			models.WithdrawalStatePendingApproval,
			models.WithdrawalStatePending,
		}).
		Updates(map[string]any{
			"state":      models.WithdrawalStateCancelled,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// AtomicCancelForPlatform atomically cancels a tenant-owned withdrawal from a
// cancellable state.
func (r *WithdrawalRepositoryImpl) AtomicCancelForPlatform(id, platformID uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND external_platform_id = ? AND state IN ?", id, platformID, []string{
			models.WithdrawalStatePendingOTP,
			models.WithdrawalStatePendingApproval,
			models.WithdrawalStatePending,
		}).
		Updates(map[string]any{
			"state":      models.WithdrawalStateCancelled,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// ClaimForProcessing atomically transitions pending → initiated so only one
// worker processes the withdrawal. Returns (rowsAffected, error).
// rowsAffected==0 means another worker already claimed it.
func (r *WithdrawalRepositoryImpl) ClaimForProcessing(id uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, models.WithdrawalStatePending).
		Updates(map[string]any{
			"state":      models.WithdrawalStateInitiated,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// RevertClaim transitions initiated → pending, for rollback on withdraw-row creation failure.
func (r *WithdrawalRepositoryImpl) RevertClaim(id uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, models.WithdrawalStateInitiated).
		Updates(map[string]any{
			"state":      models.WithdrawalStatePending,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// MarkSent atomically transitions initiated → sent and records the tx hash.
// Returns (rowsAffected, error).
func (r *WithdrawalRepositoryImpl) MarkSent(id uint, txHash string) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, models.WithdrawalStateInitiated).
		Updates(map[string]any{
			"state":      models.WithdrawalStateSent,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// MarkProcessed atomically transitions sent → processed.
// Returns (rowsAffected, error).
func (r *WithdrawalRepositoryImpl) MarkProcessed(id uint) (int64, error) {
	now := time.Now()
	res := r.db.Model(&models.Withdrawal{}).
		Where("id = ? AND state = ?", id, models.WithdrawalStateSent).
		Updates(map[string]any{
			"state":      models.WithdrawalStateProcessed,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// RecordBroadcast sets the tx_hash and initiated_at on a withdrawal row,
// transitioning it to the 'initiated' state.
func (r *WithdrawalRepositoryImpl) RecordBroadcast(id uint, txHash string, initiatedAt time.Time) error {
	return r.db.Model(&models.Withdrawal{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"state":        models.WithdrawalStateInitiated,
			"initiated_at": initiatedAt,
		}).Error
}

// SumByPlatformAndCurrencyInWindow returns the total withdrawal amount for a
// (platform, blockchain_currency) pair created at or after `since`. Excludes
// cancelled and failed withdrawals so caps reflect funds that actually moved
// or are still in flight.
func (r *WithdrawalRepositoryImpl) SumByPlatformAndCurrencyInWindow(platformID, blockchainCurrencyID uint, since time.Time) (decimal.Decimal, error) {
	var total decimal.NullDecimal
	err := r.db.Model(&models.Withdrawal{}).
		Where("external_platform_id = ? AND blockchain_currency_id = ? AND created_at >= ?", platformID, blockchainCurrencyID, since).
		Where("state NOT IN ?", []string{models.WithdrawalStateCancelled, models.WithdrawalStateFailed}).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error
	if err != nil {
		return decimal.Zero, err
	}
	if !total.Valid {
		return decimal.Zero, nil
	}
	return total.Decimal, nil
}

// WithdrawRepository defines data access for Withdraw (on-chain tx) rows.
type WithdrawRepository interface {
	Create(w *models.Withdraw) error
	GetByID(id uint) (*models.Withdraw, error)
	ListByWithdrawalID(withdrawalID uint) ([]models.Withdraw, error)
	Update(w *models.Withdraw) error
}

// WithdrawRepositoryImpl is the GORM-backed implementation.
type WithdrawRepositoryImpl struct {
	db *gorm.DB
}

// NewWithdrawRepository constructs a WithdrawRepositoryImpl.
func NewWithdrawRepository(db *gorm.DB) WithdrawRepository {
	return &WithdrawRepositoryImpl{db: db}
}

// Create inserts a new Withdraw row.
func (r *WithdrawRepositoryImpl) Create(w *models.Withdraw) error {
	return r.db.Create(w).Error
}

// GetByID fetches a Withdraw by primary key.
func (r *WithdrawRepositoryImpl) GetByID(id uint) (*models.Withdraw, error) {
	var w models.Withdraw
	err := r.db.First(&w, id).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// ListByWithdrawalID returns all Withdraw rows for the given parent Withdrawal.
func (r *WithdrawRepositoryImpl) ListByWithdrawalID(withdrawalID uint) ([]models.Withdraw, error) {
	var rows []models.Withdraw
	err := r.db.Where("withdrawal_id = ?", withdrawalID).Find(&rows).Error
	return rows, err
}

// Update saves all fields on a Withdraw row.
func (r *WithdrawRepositoryImpl) Update(w *models.Withdraw) error {
	return r.db.Save(w).Error
}
