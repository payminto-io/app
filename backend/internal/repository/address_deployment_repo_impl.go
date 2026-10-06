package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// AddressDeploymentRepository defines persistence operations for
// AddressDeployment records — CREATE2 SCW deploy lifecycle tracking.
type AddressDeploymentRepository interface {
	Create(d *models.AddressDeployment) error
	Update(d *models.AddressDeployment) error
	GetByID(id uint) (*models.AddressDeployment, error)
	GetByAddress(address string) (*models.AddressDeployment, error)
	ListByStatus(status string, opts ...QueryOption) ([]models.AddressDeployment, error)
	UpdateStatus(id uint, status string) error
	UpdateStatusWithError(id uint, status, errMsg string) error
}

// AddressDeploymentRepositoryImpl is the GORM-backed implementation.
type AddressDeploymentRepositoryImpl struct {
	db *gorm.DB
}

// NewAddressDeploymentRepository constructs a new AddressDeploymentRepository.
func NewAddressDeploymentRepository(db *gorm.DB) AddressDeploymentRepository {
	return &AddressDeploymentRepositoryImpl{db: db}
}

// Create inserts a new AddressDeployment record.
func (r *AddressDeploymentRepositoryImpl) Create(d *models.AddressDeployment) error {
	return r.db.Create(d).Error
}

// Update saves all fields of the given AddressDeployment.
func (r *AddressDeploymentRepositoryImpl) Update(d *models.AddressDeployment) error {
	return r.db.Save(d).Error
}

// GetByID fetches an AddressDeployment by primary key.
func (r *AddressDeploymentRepositoryImpl) GetByID(id uint) (*models.AddressDeployment, error) {
	var d models.AddressDeployment
	if err := r.db.
		Preload("Blockchain").
		First(&d, id).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// GetByAddress fetches an AddressDeployment by the on-chain address of the SCW.
func (r *AddressDeploymentRepositoryImpl) GetByAddress(address string) (*models.AddressDeployment, error) {
	var d models.AddressDeployment
	if err := r.db.
		Where("address = ?", address).
		First(&d).Error; err != nil {
		return nil, err
	}
	return &d, nil
}

// ListByStatus returns AddressDeployment rows with the given status, with
// optional query options for pagination or ordering.
func (r *AddressDeploymentRepositoryImpl) ListByStatus(status string, opts ...QueryOption) ([]models.AddressDeployment, error) {
	q := Apply(r.db.Where("status = ?", status), opts...)
	var deployments []models.AddressDeployment
	if err := q.Find(&deployments).Error; err != nil {
		return nil, err
	}
	return deployments, nil
}

// UpdateStatus sets the status field on an AddressDeployment.
func (r *AddressDeploymentRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.AddressDeployment{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateStatusWithError sets the status and error_message on an AddressDeployment.
// Used to record a failed deploy attempt without losing the error context.
func (r *AddressDeploymentRepositoryImpl) UpdateStatusWithError(id uint, status, errMsg string) error {
	return r.db.Model(&models.AddressDeployment{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        status,
			"error_message": errMsg,
		}).Error
}
