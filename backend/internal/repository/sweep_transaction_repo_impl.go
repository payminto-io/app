package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// SweepTransactionRepository defines persistence operations for SweepTransaction rows.
type SweepTransactionRepository interface {
	Create(st *models.SweepTransaction) error
	Update(st *models.SweepTransaction) error
	GetByID(id uint) (*models.SweepTransaction, error)
	ListBySweep(sweepID uint) ([]models.SweepTransaction, error)
	ListByStatus(status string, opts ...QueryOption) ([]models.SweepTransaction, error)
	UpdateStatus(id uint, status string) error
	UpdateTxHash(id uint, txHash string) error
	// ListConfirmedAwaitingCompletion returns confirmed transactions whose sweep is not completed yet.
	ListConfirmedAwaitingCompletion() ([]models.SweepTransaction, error)
}

// SweepTransactionTxBinder is implemented by repositories that can run inside a caller's transaction.
type SweepTransactionTxBinder interface {
	WithTx(tx *gorm.DB) SweepTransactionRepository
}

// SweepTransactionRepositoryImpl is the GORM-backed implementation.
type SweepTransactionRepositoryImpl struct {
	db *gorm.DB
}

// NewSweepTransactionRepository constructs a new SweepTransactionRepository.
func NewSweepTransactionRepository(db *gorm.DB) SweepTransactionRepository {
	return &SweepTransactionRepositoryImpl{db: db}
}

// Create inserts a new SweepTransaction record.
func (r *SweepTransactionRepositoryImpl) Create(st *models.SweepTransaction) error {
	return r.db.Create(st).Error
}

// Update saves all fields of the given SweepTransaction.
func (r *SweepTransactionRepositoryImpl) Update(st *models.SweepTransaction) error {
	return r.db.Save(st).Error
}

// GetByID fetches a SweepTransaction by primary key.
func (r *SweepTransactionRepositoryImpl) GetByID(id uint) (*models.SweepTransaction, error) {
	var st models.SweepTransaction
	if err := r.db.
		Preload("Sweep").
		Preload("BlockchainCurrency").
		First(&st, id).Error; err != nil {
		return nil, err
	}
	return &st, nil
}

// ListBySweep returns all SweepTransactions belonging to a sweep batch.
func (r *SweepTransactionRepositoryImpl) ListBySweep(sweepID uint) ([]models.SweepTransaction, error) {
	var txs []models.SweepTransaction
	if err := r.db.
		Where("sweep_id = ?", sweepID).
		Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// ListByStatus returns SweepTransactions with the given status.
func (r *SweepTransactionRepositoryImpl) ListByStatus(status string, opts ...QueryOption) ([]models.SweepTransaction, error) {
	q := Apply(r.db.Where("status = ?", status), opts...)
	var txs []models.SweepTransaction
	if err := q.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// WithTx returns a copy bound to tx so the confirmed status commits with the sweep's ledger journal.
func (r *SweepTransactionRepositoryImpl) WithTx(tx *gorm.DB) SweepTransactionRepository {
	return &SweepTransactionRepositoryImpl{db: tx}
}

// ListConfirmedAwaitingCompletion returns confirmed transactions whose sweep is not completed.
func (r *SweepTransactionRepositoryImpl) ListConfirmedAwaitingCompletion() ([]models.SweepTransaction, error) {
	var txs []models.SweepTransaction
	err := r.db.
		Joins("JOIN sweeps ON sweeps.id = sweep_transactions.sweep_id").
		Where("sweep_transactions.status = ? AND sweeps.status <> ?", "confirmed", "completed").
		Find(&txs).Error
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// UpdateStatus sets the status field on a SweepTransaction.
func (r *SweepTransactionRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.SweepTransaction{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateTxHash sets the on-chain transaction hash on a SweepTransaction.
func (r *SweepTransactionRepositoryImpl) UpdateTxHash(id uint, txHash string) error {
	return r.db.Model(&models.SweepTransaction{}).
		Where("id = ?", id).
		Update("tx_hash", txHash).Error
}
