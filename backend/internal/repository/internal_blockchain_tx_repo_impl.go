package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// InternalBlockchainTransactionRepository defines persistence operations for
// InternalBlockchainTransaction records — gas-funding transfers and other
// internal moves between Payminto-controlled addresses.
type InternalBlockchainTransactionRepository interface {
	Create(tx *models.InternalBlockchainTransaction) error
	Update(tx *models.InternalBlockchainTransaction) error
	GetByID(id uint) (*models.InternalBlockchainTransaction, error)
	GetByTxHash(txHash string) (*models.InternalBlockchainTransaction, error)
	ListByStatus(status string, opts ...QueryOption) ([]models.InternalBlockchainTransaction, error)
	UpdateStatus(id uint, status string) error
	UpdateBlockNumber(id uint, blockNumber int64) error
}

// InternalBlockchainTransactionRepositoryImpl is the GORM-backed implementation.
type InternalBlockchainTransactionRepositoryImpl struct {
	db *gorm.DB
}

// NewInternalBlockchainTransactionRepository constructs a new
// InternalBlockchainTransactionRepository backed by the provided *gorm.DB.
func NewInternalBlockchainTransactionRepository(db *gorm.DB) InternalBlockchainTransactionRepository {
	return &InternalBlockchainTransactionRepositoryImpl{db: db}
}

// Create inserts a new InternalBlockchainTransaction record.
func (r *InternalBlockchainTransactionRepositoryImpl) Create(tx *models.InternalBlockchainTransaction) error {
	return r.db.Create(tx).Error
}

// Update saves all fields of the given InternalBlockchainTransaction.
func (r *InternalBlockchainTransactionRepositoryImpl) Update(tx *models.InternalBlockchainTransaction) error {
	return r.db.Save(tx).Error
}

// GetByID fetches an InternalBlockchainTransaction by primary key.
func (r *InternalBlockchainTransactionRepositoryImpl) GetByID(id uint) (*models.InternalBlockchainTransaction, error) {
	var tx models.InternalBlockchainTransaction
	if err := r.db.First(&tx, id).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

// GetByTxHash fetches an InternalBlockchainTransaction by on-chain transaction hash.
func (r *InternalBlockchainTransactionRepositoryImpl) GetByTxHash(txHash string) (*models.InternalBlockchainTransaction, error) {
	var tx models.InternalBlockchainTransaction
	if err := r.db.
		Where("tx_hash = ?", txHash).
		First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

// ListByStatus returns InternalBlockchainTransactions with the given status.
func (r *InternalBlockchainTransactionRepositoryImpl) ListByStatus(status string, opts ...QueryOption) ([]models.InternalBlockchainTransaction, error) {
	q := Apply(r.db.Where("status = ?", status), opts...)
	var txs []models.InternalBlockchainTransaction
	if err := q.Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// UpdateStatus sets the status field on an InternalBlockchainTransaction.
func (r *InternalBlockchainTransactionRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.InternalBlockchainTransaction{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateBlockNumber records the block number at which the transaction was mined.
func (r *InternalBlockchainTransactionRepositoryImpl) UpdateBlockNumber(id uint, blockNumber int64) error {
	return r.db.Model(&models.InternalBlockchainTransaction{}).
		Where("id = ?", id).
		Update("block_number", blockNumber).Error
}
