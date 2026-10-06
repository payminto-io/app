package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// SweepRepository defines all persistence operations for Sweep batches.
type SweepRepository interface {
	Create(s *models.Sweep) error
	Update(s *models.Sweep) error
	GetByID(id uint) (*models.Sweep, error)
	ListByStatus(status string) ([]models.Sweep, error)
	ListByBlockchain(blockchainID uint, opts ...QueryOption) ([]models.Sweep, error)
	UpdateStatus(id uint, status string) error
}

// SweepRepositoryImpl is the GORM-backed implementation of SweepRepository.
type SweepRepositoryImpl struct {
	db *gorm.DB
}

// NewSweepRepository constructs a new SweepRepository backed by the provided *gorm.DB.
func NewSweepRepository(db *gorm.DB) SweepRepository {
	return &SweepRepositoryImpl{db: db}
}

// Create inserts a new Sweep record.
func (r *SweepRepositoryImpl) Create(s *models.Sweep) error {
	return r.db.Create(s).Error
}

// Update saves all fields of the given Sweep.
func (r *SweepRepositoryImpl) Update(s *models.Sweep) error {
	return r.db.Save(s).Error
}

// GetByID fetches a Sweep by primary key, preloading SweepTransactions.
func (r *SweepRepositoryImpl) GetByID(id uint) (*models.Sweep, error) {
	var s models.Sweep
	if err := r.db.
		Preload("SweepTransactions").
		Preload("Blockchain").
		First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

// ListByStatus returns all Sweeps with the given status.
func (r *SweepRepositoryImpl) ListByStatus(status string) ([]models.Sweep, error) {
	var sweeps []models.Sweep
	if err := r.db.
		Preload("Blockchain").
		Where("status = ?", status).
		Find(&sweeps).Error; err != nil {
		return nil, err
	}
	return sweeps, nil
}

// ListByBlockchain returns Sweeps for a specific blockchain, with optional query options.
func (r *SweepRepositoryImpl) ListByBlockchain(blockchainID uint, opts ...QueryOption) ([]models.Sweep, error) {
	q := Apply(r.db.Where("blockchain_id = ?", blockchainID), opts...)
	var sweeps []models.Sweep
	if err := q.Find(&sweeps).Error; err != nil {
		return nil, err
	}
	return sweeps, nil
}

// UpdateStatus sets the status field on a Sweep.
func (r *SweepRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.Sweep{}).
		Where("id = ?", id).
		Update("status", status).Error
}
