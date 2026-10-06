package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// SecretsVaultActivityRepository defines the persistence interface for the
// vault audit log.
type SecretsVaultActivityRepository interface {
	Create(a *models.SecretsVaultActivity) error
	ListByLabel(label string, opts ...QueryOption) ([]models.SecretsVaultActivity, error)
	ListByVaultID(vaultID uint, opts ...QueryOption) ([]models.SecretsVaultActivity, error)
	ListByMemberID(memberID uint, opts ...QueryOption) ([]models.SecretsVaultActivity, error)
	ListRecent(limit int) ([]models.SecretsVaultActivity, error)
}

// SecretsVaultActivityRepositoryImpl is the GORM-backed implementation.
type SecretsVaultActivityRepositoryImpl struct {
	db *gorm.DB
}

// NewSecretsVaultActivityRepository returns a new SecretsVaultActivityRepository
// backed by db.
func NewSecretsVaultActivityRepository(db *gorm.DB) SecretsVaultActivityRepository {
	return &SecretsVaultActivityRepositoryImpl{db: db}
}

func (r *SecretsVaultActivityRepositoryImpl) Create(a *models.SecretsVaultActivity) error {
	return r.db.Create(a).Error
}

func (r *SecretsVaultActivityRepositoryImpl) ListByLabel(label string, opts ...QueryOption) ([]models.SecretsVaultActivity, error) {
	var out []models.SecretsVaultActivity
	q := Apply(r.db.Where("label = ?", label).Order("created_at DESC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SecretsVaultActivityRepositoryImpl) ListByVaultID(vaultID uint, opts ...QueryOption) ([]models.SecretsVaultActivity, error) {
	var out []models.SecretsVaultActivity
	q := Apply(r.db.Where("vault_id = ?", vaultID).Order("created_at DESC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SecretsVaultActivityRepositoryImpl) ListByMemberID(memberID uint, opts ...QueryOption) ([]models.SecretsVaultActivity, error) {
	var out []models.SecretsVaultActivity
	q := Apply(r.db.Where("member_id = ?", memberID).Order("created_at DESC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SecretsVaultActivityRepositoryImpl) ListRecent(limit int) ([]models.SecretsVaultActivity, error) {
	var out []models.SecretsVaultActivity
	q := r.db.Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
