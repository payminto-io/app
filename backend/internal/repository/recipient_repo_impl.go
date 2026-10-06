package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// RecipientRepository defines storage operations for Recipient (merchant payee book).
type RecipientRepository interface {
	Create(r *models.Recipient) error
	GetByID(id uint) (*models.Recipient, error)
	Update(r *models.Recipient) error
	Delete(id uint) error
	ListByMember(memberID uint) ([]models.Recipient, error)
	ListByPlatform(platformID uint) ([]models.Recipient, error)
}

// RecipientRepositoryImpl is the GORM-backed implementation.
type RecipientRepositoryImpl struct {
	db *gorm.DB
}

// NewRecipientRepository constructs a RecipientRepositoryImpl.
func NewRecipientRepository(db *gorm.DB) RecipientRepository {
	return &RecipientRepositoryImpl{db: db}
}

// Create inserts a new Recipient row.
func (r *RecipientRepositoryImpl) Create(rec *models.Recipient) error {
	return r.db.Create(rec).Error
}

// GetByID returns a Recipient by primary key.
func (r *RecipientRepositoryImpl) GetByID(id uint) (*models.Recipient, error) {
	var rec models.Recipient
	if err := r.db.First(&rec, id).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// Update saves all fields of the recipient.
func (r *RecipientRepositoryImpl) Update(rec *models.Recipient) error {
	return r.db.Save(rec).Error
}

// Delete soft-deletes the recipient with the given id.
func (r *RecipientRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.Recipient{}, id).Error
}

// ListByMember returns all non-deleted recipients belonging to a member.
func (r *RecipientRepositoryImpl) ListByMember(memberID uint) ([]models.Recipient, error) {
	var recs []models.Recipient
	err := r.db.Where("member_id = ?", memberID).Find(&recs).Error
	return recs, err
}

// ListByPlatform returns all non-deleted recipients for a given platform.
func (r *RecipientRepositoryImpl) ListByPlatform(platformID uint) ([]models.Recipient, error) {
	var recs []models.Recipient
	err := r.db.Where("external_platform_id = ?", platformID).Find(&recs).Error
	return recs, err
}
