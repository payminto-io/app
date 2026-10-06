package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// SecretsVaultRepository defines the persistence interface for encrypted vault
// entries.
type SecretsVaultRepository interface {
	Create(v *models.SecretsVault) error
	Update(v *models.SecretsVault) error
	Delete(id uint) error
	GetByID(id uint) (*models.SecretsVault, error)
	GetByLabel(label string) (*models.SecretsVault, error)
	List(opts ...QueryOption) ([]models.SecretsVault, error)
	ListByType(secretType string) ([]models.SecretsVault, error)
	TouchLastUsed(id uint) error
	// BulkUpdateCiphertexts atomically replaces the ciphertext column for every
	// vault ID in the map. All updates are executed inside a single database
	// transaction; if any update fails the whole operation is rolled back.
	BulkUpdateCiphertexts(updates map[uint]string) error
}

// SecretsVaultRepositoryImpl is the GORM-backed implementation.
type SecretsVaultRepositoryImpl struct {
	db *gorm.DB
}

// NewSecretsVaultRepository returns a new SecretsVaultRepository backed by db.
func NewSecretsVaultRepository(db *gorm.DB) SecretsVaultRepository {
	return &SecretsVaultRepositoryImpl{db: db}
}

func (r *SecretsVaultRepositoryImpl) Create(v *models.SecretsVault) error {
	return r.db.Create(v).Error
}

func (r *SecretsVaultRepositoryImpl) Update(v *models.SecretsVault) error {
	return r.db.Save(v).Error
}

func (r *SecretsVaultRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.SecretsVault{}, id).Error
}

func (r *SecretsVaultRepositoryImpl) GetByID(id uint) (*models.SecretsVault, error) {
	var v models.SecretsVault
	if err := r.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SecretsVaultRepositoryImpl) GetByLabel(label string) (*models.SecretsVault, error) {
	var v models.SecretsVault
	if err := r.db.Where("label = ?", label).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *SecretsVaultRepositoryImpl) List(opts ...QueryOption) ([]models.SecretsVault, error) {
	var out []models.SecretsVault
	q := Apply(r.db, opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SecretsVaultRepositoryImpl) ListByType(secretType string) ([]models.SecretsVault, error) {
	var out []models.SecretsVault
	if err := r.db.Where("secret_type = ?", secretType).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SecretsVaultRepositoryImpl) TouchLastUsed(id uint) error {
	return r.db.Model(&models.SecretsVault{}).
		Where("id = ?", id).
		Update("last_used_at", gorm.Expr("CURRENT_TIMESTAMP")).Error
}

// BulkUpdateCiphertexts atomically updates the ciphertext for every vault
// entry in the map. All writes are wrapped in a single transaction; on any
// error the transaction is rolled back and the error is returned.
func (r *SecretsVaultRepositoryImpl) BulkUpdateCiphertexts(updates map[uint]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for id, ciphertext := range updates {
			if err := tx.Model(&models.SecretsVault{}).
				Where("id = ?", id).
				Update("ciphertext", ciphertext).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
