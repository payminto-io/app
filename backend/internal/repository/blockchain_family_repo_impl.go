package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// BlockchainFamilyRepository defines the persistence contract for blockchain family records.
type BlockchainFamilyRepository interface {
	Create(f *models.BlockchainFamily) error
	Update(f *models.BlockchainFamily) error
	GetByID(id uint) (*models.BlockchainFamily, error)
	GetByCode(code string) (*models.BlockchainFamily, error)
	GetByName(name string) (*models.BlockchainFamily, error)
	List() ([]models.BlockchainFamily, error)
	ListAll(opts ...QueryOption) ([]models.BlockchainFamily, error)
}

// BlockchainFamilyRepositoryImpl is the GORM-backed implementation of BlockchainFamilyRepository.
type BlockchainFamilyRepositoryImpl struct {
	db *gorm.DB
}

// NewBlockchainFamilyRepository constructs a BlockchainFamilyRepository backed by db.
func NewBlockchainFamilyRepository(db *gorm.DB) BlockchainFamilyRepository {
	return &BlockchainFamilyRepositoryImpl{db: db}
}

// Create inserts a new BlockchainFamily row.
func (r *BlockchainFamilyRepositoryImpl) Create(f *models.BlockchainFamily) error {
	return r.db.Create(f).Error
}

// Update saves all fields on the blockchain family (full save).
func (r *BlockchainFamilyRepositoryImpl) Update(f *models.BlockchainFamily) error {
	return r.db.Save(f).Error
}

// GetByID fetches a blockchain family by primary key.
func (r *BlockchainFamilyRepositoryImpl) GetByID(id uint) (*models.BlockchainFamily, error) {
	var f models.BlockchainFamily
	err := r.db.First(&f, id).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetByCode fetches a blockchain family by its unique code.
func (r *BlockchainFamilyRepositoryImpl) GetByCode(code string) (*models.BlockchainFamily, error) {
	var f models.BlockchainFamily
	err := r.db.Where("code = ?", code).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// GetByName fetches a blockchain family by its unique name.
func (r *BlockchainFamilyRepositoryImpl) GetByName(name string) (*models.BlockchainFamily, error) {
	var f models.BlockchainFamily
	err := r.db.Where("name = ?", name).First(&f).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// List returns all blockchain families.
func (r *BlockchainFamilyRepositoryImpl) List() ([]models.BlockchainFamily, error) {
	var families []models.BlockchainFamily
	err := r.db.Find(&families).Error
	return families, err
}

// ListAll returns all blockchain families, with optional query options.
func (r *BlockchainFamilyRepositoryImpl) ListAll(opts ...QueryOption) ([]models.BlockchainFamily, error) {
	var out []models.BlockchainFamily
	q := Apply(r.db, opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
