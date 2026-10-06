package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// DepositRepository defines all persistence operations for Deposit records.
type DepositRepository interface {
	Create(d *models.Deposit) error
	Update(d *models.Deposit) error
	GetByID(id uint) (*models.Deposit, error)
	GetByTxID(txID string) ([]models.Deposit, error)
	GetByTxIDAndToAddress(txID, toAddr string, blockchainCurrencyID uint) (*models.Deposit, error)
	ListByStatus(status string, opts ...QueryOption) ([]models.Deposit, error)
	// ListByStatusAndMember returns deposits with the given status scoped to a
	// single merchant. Unlike ListByStatus it enforces tenant isolation via an
	// additional WHERE member_id = ? clause. The BlockchainCurrency and its
	// parent Blockchain are preloaded for callers that need chain metadata.
	ListByStatusAndMember(status string, memberID uint) ([]models.Deposit, error)
	ListByPaymentRequestID(paymentRequestID uint) ([]models.Deposit, error)
	ListConfirmingForChain(blockchainID uint, limit int) ([]models.Deposit, error)
	UpdateConfirmations(id uint, confirmations int) error
	// ConfirmIfPending atomically advances the confirmation count and deposit
	// status in a single UPDATE … WHERE id = ? AND status IN ('pending',
	// 'confirming'). Returns the number of rows affected (0 = already confirmed
	// or concurrently updated by another goroutine, so the caller should no-op).
	ConfirmIfPending(depositID uint, confirmations int) (int64, error)
	// ClaimForSweep atomically transitions a deposit confirmed → swept and
	// returns rows affected (0 = already claimed by another worker/round). This
	// claim-before-broadcast pattern guarantees a deposit is broadcast for
	// sweeping at most once, even if the post-broadcast DB write fails.
	ClaimForSweep(depositID uint) (int64, error)
	UpdateStatus(id uint, status string) error
	CountByStatus(status string) (int64, error)
}

// DepositRepositoryImpl is the GORM-backed implementation of DepositRepository.
type DepositRepositoryImpl struct {
	db *gorm.DB
}

// NewDepositRepository constructs a new DepositRepository backed by the provided *gorm.DB.
func NewDepositRepository(db *gorm.DB) DepositRepository {
	return &DepositRepositoryImpl{db: db}
}

// Create inserts a new Deposit record.
func (r *DepositRepositoryImpl) Create(d *models.Deposit) error {
	return r.db.Create(d).Error
}

// Update saves all fields of the given Deposit.
func (r *DepositRepositoryImpl) Update(d *models.Deposit) error {
	return r.db.Save(d).Error
}

// GetByID fetches a Deposit by primary key, preloading BlockchainCurrency.
func (r *DepositRepositoryImpl) GetByID(id uint) (*models.Deposit, error) {
	var d models.Deposit
	err := r.db.Preload("BlockchainCurrency").First(&d, id).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// GetByTxID returns all Deposits that share a transaction ID.
// The same on-chain transaction may produce multiple deposit records (e.g. multi-output TXs).
func (r *DepositRepositoryImpl) GetByTxID(txID string) ([]models.Deposit, error) {
	var deposits []models.Deposit
	if err := r.db.Where("tx_id = ?", txID).Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// GetByTxIDAndToAddress fetches a single Deposit for duplicate detection.
// Matches on transaction ID, destination address (case-insensitive for EVM
// EIP-55 compat), and blockchain currency.
func (r *DepositRepositoryImpl) GetByTxIDAndToAddress(txID, toAddr string, blockchainCurrencyID uint) (*models.Deposit, error) {
	var d models.Deposit
	err := r.db.
		Where("tx_id = ? AND LOWER(to_address) = LOWER(?) AND blockchain_currency_id = ?", txID, toAddr, blockchainCurrencyID).
		First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// ListByStatus returns Deposits with the given status, preloading
// BlockchainCurrency and its parent Blockchain. Accepts additional QueryOptions.
func (r *DepositRepositoryImpl) ListByStatus(status string, opts ...QueryOption) ([]models.Deposit, error) {
	q := Apply(
		r.db.
			Preload("BlockchainCurrency.Blockchain").
			Where("status = ?", status),
		opts...,
	)
	var deposits []models.Deposit
	if err := q.Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// ListByStatusAndMember returns confirmed (or any given status) deposits
// scoped to a single merchant. Preloads BlockchainCurrency.Blockchain so
// callers have full chain metadata without an extra query.
func (r *DepositRepositoryImpl) ListByStatusAndMember(status string, memberID uint) ([]models.Deposit, error) {
	var deposits []models.Deposit
	err := r.db.
		Preload("BlockchainCurrency.Blockchain").
		Where("status = ? AND member_id = ?", status, memberID).
		Find(&deposits).Error
	if err != nil {
		return nil, err
	}
	return deposits, nil
}

// ConfirmIfPending atomically advances the confirmation count and deposit
// status with a single conditional UPDATE. The WHERE clause guards against
// concurrent updates by only touching rows still in pending or confirming
// state. Returns the number of rows affected (0 means someone else already
// advanced or confirmed this deposit — the caller must treat that as a no-op).
// ClaimForSweep atomically marks a confirmed deposit as swept. Returns rows
// affected: 1 if this caller won the claim, 0 if it was already swept.
func (r *DepositRepositoryImpl) ClaimForSweep(depositID uint) (int64, error) {
	result := r.db.Model(&models.Deposit{}).
		Where("id = ? AND status = ?", depositID, models.DepositStatusConfirmed).
		Updates(map[string]any{"status": models.DepositStatusSwept})
	return result.RowsAffected, result.Error
}

func (r *DepositRepositoryImpl) ConfirmIfPending(depositID uint, confirmations int) (int64, error) {
	result := r.db.Exec(`
		UPDATE deposits
		SET confirmations = ?,
		    status = CASE
		        WHEN ? >= required_confirmations THEN 'confirmed'
		        WHEN ? > 0 THEN 'confirming'
		        ELSE status
		    END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		  AND status IN ('pending', 'confirming')
	`, confirmations, confirmations, confirmations, depositID)
	return result.RowsAffected, result.Error
}

// ListByPaymentRequestID returns all Deposits linked to a specific payment request.
func (r *DepositRepositoryImpl) ListByPaymentRequestID(paymentRequestID uint) ([]models.Deposit, error) {
	var deposits []models.Deposit
	if err := r.db.Where("payment_request_id = ?", paymentRequestID).Find(&deposits).Error; err != nil {
		return nil, err
	}
	return deposits, nil
}

// ListConfirmingForChain returns up to limit Deposits that are in the
// 'confirming' status and belong to the given blockchain.
// Joins deposit → blockchain_currency → blockchain to filter by chain.
func (r *DepositRepositoryImpl) ListConfirmingForChain(blockchainID uint, limit int) ([]models.Deposit, error) {
	var deposits []models.Deposit
	err := r.db.
		Joins("JOIN blockchain_currencies ON blockchain_currencies.id = deposits.blockchain_currency_id").
		Where("deposits.status = ? AND blockchain_currencies.blockchain_id = ?", models.DepositStatusConfirming, blockchainID).
		Limit(limit).
		Find(&deposits).Error
	if err != nil {
		return nil, err
	}
	return deposits, nil
}

// UpdateConfirmations sets the confirmation count on a Deposit.
func (r *DepositRepositoryImpl) UpdateConfirmations(id uint, confirmations int) error {
	return r.db.Model(&models.Deposit{}).
		Where("id = ?", id).
		Update("confirmations", confirmations).Error
}

// UpdateStatus sets the status field on a Deposit.
func (r *DepositRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.Deposit{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// CountByStatus returns the number of Deposits in the given status.
func (r *DepositRepositoryImpl) CountByStatus(status string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Deposit{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
