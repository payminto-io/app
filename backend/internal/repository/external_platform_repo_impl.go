package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ExternalPlatformRepository defines all data access operations for the ExternalPlatform model.
type ExternalPlatformRepository interface {
	Create(p *models.ExternalPlatform) error
	Update(p *models.ExternalPlatform) error
	Delete(id uint) error
	GetByID(id uint, opts ...QueryOption) (*models.ExternalPlatform, error)
	GetByName(name string) (*models.ExternalPlatform, error)
	List(opts ...QueryOption) ([]models.ExternalPlatform, error)
	Count() (int64, error)
}

// ExternalPlatformRepositoryImpl is the GORM-backed implementation of ExternalPlatformRepository.
type ExternalPlatformRepositoryImpl struct {
	db *gorm.DB
}

// NewExternalPlatformRepository constructs an ExternalPlatformRepositoryImpl bound to the given DB handle.
func NewExternalPlatformRepository(db *gorm.DB) ExternalPlatformRepository {
	return &ExternalPlatformRepositoryImpl{db: db}
}

func (r *ExternalPlatformRepositoryImpl) Create(p *models.ExternalPlatform) error {
	return r.db.Create(p).Error
}

func (r *ExternalPlatformRepositoryImpl) Update(p *models.ExternalPlatform) error {
	return r.db.Save(p).Error
}

func (r *ExternalPlatformRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.ExternalPlatform{}, id).Error
}

// GetByID returns a platform by ID, allowing callers to pass extra query options
// (e.g., WithPreload) to eager-load associations.
func (r *ExternalPlatformRepositoryImpl) GetByID(id uint, opts ...QueryOption) (*models.ExternalPlatform, error) {
	var p models.ExternalPlatform
	q := Apply(r.db, opts...)
	err := q.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ExternalPlatformRepositoryImpl) GetByName(name string) (*models.ExternalPlatform, error) {
	var p models.ExternalPlatform
	err := r.db.Where("name = ?", name).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ExternalPlatformRepositoryImpl) List(opts ...QueryOption) ([]models.ExternalPlatform, error) {
	var platforms []models.ExternalPlatform
	q := Apply(r.db.Model(&models.ExternalPlatform{}), opts...)
	err := q.Find(&platforms).Error
	return platforms, err
}

func (r *ExternalPlatformRepositoryImpl) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.ExternalPlatform{}).Count(&count).Error
	return count, err
}
