package repository

import (
	"errors"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BlockchainCurrencyRepository defines the persistence contract for blockchain-currency configuration records.
type BlockchainCurrencyRepository interface {
	Create(bc *models.BlockchainCurrency) error
	Update(bc *models.BlockchainCurrency) error
	GetByID(id uint) (*models.BlockchainCurrency, error)
	GetByBlockchainAndCurrency(blockchainID, currencyID uint) (*models.BlockchainCurrency, error)
	GetByBlockchainCodeAndCurrencyCode(blockchainCode, currencyCode string) (*models.BlockchainCurrency, error)
	ListByBlockchainID(blockchainID uint, opts ...QueryOption) ([]models.BlockchainCurrency, error)
	ListByCurrencyID(currencyID uint, opts ...QueryOption) ([]models.BlockchainCurrency, error)
	ListDepositEnabled() ([]models.BlockchainCurrency, error)
	ListAll(opts ...QueryOption) ([]models.BlockchainCurrency, error)
	CreateOrUpdate(bc *models.BlockchainCurrency) error
}

// BlockchainCurrencyRepositoryImpl is the GORM-backed implementation of BlockchainCurrencyRepository.
type BlockchainCurrencyRepositoryImpl struct {
	db *gorm.DB
}

// NewBlockchainCurrencyRepository constructs a BlockchainCurrencyRepository backed by db.
func NewBlockchainCurrencyRepository(db *gorm.DB) BlockchainCurrencyRepository {
	return &BlockchainCurrencyRepositoryImpl{db: db}
}

// Create inserts a new BlockchainCurrency row.
// Uses Select("*") to ensure zero-value bool fields (e.g. DepositEnabled=false, Visible=false) are written explicitly.
func (r *BlockchainCurrencyRepositoryImpl) Create(bc *models.BlockchainCurrency) error {
	return r.db.Select("*").Create(bc).Error
}

// Update saves all fields on the blockchain currency (full save).
func (r *BlockchainCurrencyRepositoryImpl) Update(bc *models.BlockchainCurrency) error {
	return r.db.Save(bc).Error
}

// GetByID fetches a blockchain currency by primary key, preloading Currency and Blockchain.
func (r *BlockchainCurrencyRepositoryImpl) GetByID(id uint) (*models.BlockchainCurrency, error) {
	var bc models.BlockchainCurrency
	err := r.db.Preload("Currency").Preload("Blockchain").First(&bc, id).Error
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// GetByBlockchainAndCurrency fetches the config for a specific blockchain+currency pair, preloading both.
func (r *BlockchainCurrencyRepositoryImpl) GetByBlockchainAndCurrency(blockchainID, currencyID uint) (*models.BlockchainCurrency, error) {
	var bc models.BlockchainCurrency
	err := r.db.Preload("Currency").Preload("Blockchain").
		Where("blockchain_id = ? AND currency_id = ?", blockchainID, currencyID).
		First(&bc).Error
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// GetByBlockchainCodeAndCurrencyCode fetches config by blockchain code and currency code, preloading both.
func (r *BlockchainCurrencyRepositoryImpl) GetByBlockchainCodeAndCurrencyCode(blockchainCode, currencyCode string) (*models.BlockchainCurrency, error) {
	var bc models.BlockchainCurrency
	err := r.db.Preload("Currency").Preload("Blockchain").
		Where("blockchain_code = ? AND currency_code = ?", blockchainCode, currencyCode).
		First(&bc).Error
	if err != nil {
		return nil, err
	}
	return &bc, nil
}

// ListByBlockchainID returns all blockchain-currency configs for a given blockchain, preloading Currency.
func (r *BlockchainCurrencyRepositoryImpl) ListByBlockchainID(blockchainID uint, opts ...QueryOption) ([]models.BlockchainCurrency, error) {
	var bcs []models.BlockchainCurrency
	q := Apply(r.db.Preload("Currency").Where("blockchain_id = ?", blockchainID), opts...)
	err := q.Find(&bcs).Error
	return bcs, err
}

// ListByCurrencyID returns all blockchain-currency configs for a given currency, preloading Blockchain.
func (r *BlockchainCurrencyRepositoryImpl) ListByCurrencyID(currencyID uint, opts ...QueryOption) ([]models.BlockchainCurrency, error) {
	var bcs []models.BlockchainCurrency
	q := Apply(r.db.Preload("Blockchain").Where("currency_id = ?", currencyID), opts...)
	err := q.Find(&bcs).Error
	return bcs, err
}

// ListDepositEnabled returns all blockchain-currency configs where deposit_enabled = true and visible = true,
// preloading both Currency and Blockchain.
func (r *BlockchainCurrencyRepositoryImpl) ListDepositEnabled() ([]models.BlockchainCurrency, error) {
	var bcs []models.BlockchainCurrency
	err := r.db.Preload("Currency").Preload("Blockchain").
		Where("deposit_enabled = ? AND visible = ?", true, true).
		Find(&bcs).Error
	return bcs, err
}

// ListAll returns all blockchain-currency configs, with optional query options.
func (r *BlockchainCurrencyRepositoryImpl) ListAll(opts ...QueryOption) ([]models.BlockchainCurrency, error) {
	var out []models.BlockchainCurrency
	q := Apply(r.db, opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// CreateOrUpdate upserts a BlockchainCurrency by (blockchain_id, currency_id).
// It tries to find an existing record first; if found it updates, otherwise it creates.
func (r *BlockchainCurrencyRepositoryImpl) CreateOrUpdate(bc *models.BlockchainCurrency) error {
	var existing models.BlockchainCurrency
	err := r.db.Where("blockchain_id = ? AND currency_id = ?", bc.BlockchainID, bc.CurrencyID).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No existing record — insert fresh.
		return r.db.Create(bc).Error
	}
	if err != nil {
		return err
	}

	// Record exists — update all fields.
	bc.ID = existing.ID
	bc.CreatedAt = existing.CreatedAt
	return r.db.Clauses(clause.OnConflict{UpdateAll: true}).Save(bc).Error
}
