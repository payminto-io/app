package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// PermissionRepository defines all data access operations for the Permission model.
type PermissionRepository interface {
	Create(p *models.Permission) error
	GetByID(id uint) (*models.Permission, error)
	GetByName(name string) (*models.Permission, error)
	GetByNames(names []string) ([]models.Permission, error)
	List() ([]models.Permission, error)
}

// PermissionRepositoryImpl is the GORM-backed implementation of PermissionRepository.
type PermissionRepositoryImpl struct {
	db *gorm.DB
}

// NewPermissionRepository constructs a PermissionRepositoryImpl bound to the given DB handle.
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &PermissionRepositoryImpl{db: db}
}

func (r *PermissionRepositoryImpl) Create(p *models.Permission) error {
	return r.db.Create(p).Error
}

func (r *PermissionRepositoryImpl) GetByID(id uint) (*models.Permission, error) {
	var p models.Permission
	err := r.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PermissionRepositoryImpl) GetByName(name string) (*models.Permission, error) {
	var p models.Permission
	err := r.db.Where("name = ?", name).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetByNames returns all permissions whose name is in the provided slice.
func (r *PermissionRepositoryImpl) GetByNames(names []string) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.Where("name IN ?", names).Find(&perms).Error
	return perms, err
}

func (r *PermissionRepositoryImpl) List() ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.Find(&perms).Error
	return perms, err
}
