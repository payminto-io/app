package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// APIKeyRepository defines all data access operations for the APIKey model.
type APIKeyRepository interface {
	Create(k *models.APIKey) error
	Update(k *models.APIKey) error
	Delete(id uint) error
	// GetByKey looks up an API key by its already-hashed value.
	// Use Sha256Sum(rawKey) before calling this method.
	GetByKey(hashedKey string) (*models.APIKey, error)
	GetByID(id uint) (*models.APIKey, error)
	ListByMemberID(memberID uint, opts ...QueryOption) ([]models.APIKey, error)
	ListByExternalPlatformID(platformID uint, opts ...QueryOption) ([]models.APIKey, error)
	DeactivateByMemberID(memberID uint) error
}

// APIKeyRepositoryImpl is the GORM-backed implementation of APIKeyRepository.
type APIKeyRepositoryImpl struct {
	db *gorm.DB
}

// NewAPIKeyRepository constructs an APIKeyRepositoryImpl bound to the given DB handle.
func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &APIKeyRepositoryImpl{db: db}
}

func (r *APIKeyRepositoryImpl) Create(k *models.APIKey) error {
	return r.db.Create(k).Error
}

func (r *APIKeyRepositoryImpl) Update(k *models.APIKey) error {
	return r.db.Save(k).Error
}

func (r *APIKeyRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.APIKey{}, id).Error
}

// GetByKey returns the API key record matching the provided hashed key,
// with Member and ExternalPlatform associations preloaded.
func (r *APIKeyRepositoryImpl) GetByKey(hashedKey string) (*models.APIKey, error) {
	var k models.APIKey
	err := r.db.
		Preload("Member").
		Preload("ExternalPlatform").
		Where("key = ?", hashedKey).
		First(&k).Error
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepositoryImpl) GetByID(id uint) (*models.APIKey, error) {
	var k models.APIKey
	err := r.db.First(&k, id).Error
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *APIKeyRepositoryImpl) ListByMemberID(memberID uint, opts ...QueryOption) ([]models.APIKey, error) {
	var keys []models.APIKey
	q := Apply(r.db.Where("member_id = ?", memberID), opts...)
	err := q.Find(&keys).Error
	return keys, err
}

func (r *APIKeyRepositoryImpl) ListByExternalPlatformID(platformID uint, opts ...QueryOption) ([]models.APIKey, error) {
	var keys []models.APIKey
	q := Apply(r.db.Where("external_platform_id = ?", platformID), opts...)
	err := q.Find(&keys).Error
	return keys, err
}

// DeactivateByMemberID sets status='inactive' on all API keys belonging to a member.
func (r *APIKeyRepositoryImpl) DeactivateByMemberID(memberID uint) error {
	return r.db.Model(&models.APIKey{}).
		Where("member_id = ?", memberID).
		Update("status", "inactive").Error
}
