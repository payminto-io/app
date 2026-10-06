package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// BlockchainRepository defines the persistence contract for blockchain records.
type BlockchainRepository interface {
	Create(b *models.Blockchain) error
	Update(b *models.Blockchain) error
	GetByID(id uint) (*models.Blockchain, error)
	GetByCode(code string) (*models.Blockchain, error)
	List(opts ...QueryOption) ([]models.Blockchain, error)
	ListActive() ([]models.Blockchain, error)
	UpdateHeight(id uint, height int64) error
}

// BlockchainRepositoryImpl is the GORM-backed implementation of BlockchainRepository.
type BlockchainRepositoryImpl struct {
	db *gorm.DB
}

// NewBlockchainRepository constructs a BlockchainRepository backed by db.
func NewBlockchainRepository(db *gorm.DB) BlockchainRepository {
	return &BlockchainRepositoryImpl{db: db}
}

// Create inserts a new Blockchain row.
func (r *BlockchainRepositoryImpl) Create(b *models.Blockchain) error {
	return r.db.Create(b).Error
}

// Update saves all fields on the blockchain (full save).
func (r *BlockchainRepositoryImpl) Update(b *models.Blockchain) error {
	return r.db.Save(b).Error
}

// GetByID fetches a blockchain by primary key, preloading BlockchainFamily.
func (r *BlockchainRepositoryImpl) GetByID(id uint) (*models.Blockchain, error) {
	var b models.Blockchain
	err := r.db.Preload("BlockchainFamily").First(&b, id).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// GetByCode fetches a blockchain by its unique code, preloading BlockchainFamily.
func (r *BlockchainRepositoryImpl) GetByCode(code string) (*models.Blockchain, error) {
	var b models.Blockchain
	err := r.db.Preload("BlockchainFamily").Where("code = ?", code).First(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// List returns blockchains with optional query options, preloading BlockchainFamily.
func (r *BlockchainRepositoryImpl) List(opts ...QueryOption) ([]models.Blockchain, error) {
	var blockchains []models.Blockchain
	q := Apply(r.db.Preload("BlockchainFamily"), opts...)
	err := q.Find(&blockchains).Error
	return blockchains, err
}

// ListActive returns all blockchains with status='active', preloading BlockchainFamily.
func (r *BlockchainRepositoryImpl) ListActive() ([]models.Blockchain, error) {
	var blockchains []models.Blockchain
	err := r.db.Preload("BlockchainFamily").Where("status = ?", "active").Find(&blockchains).Error
	return blockchains, err
}

// UpdateHeight sets the block height and updated_at for the given blockchain ID.
func (r *BlockchainRepositoryImpl) UpdateHeight(id uint, height int64) error {
	return r.db.Model(&models.Blockchain{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"height":     height,
			"updated_at": time.Now(),
		}).Error
}
