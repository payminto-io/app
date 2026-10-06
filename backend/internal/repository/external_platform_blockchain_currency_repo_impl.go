package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ExternalPlatformBlockchainCurrencyRepository persists the per-platform
// per-blockchain-currency enable list and limit configuration.
type ExternalPlatformBlockchainCurrencyRepository interface {
	// Upsert creates or updates the row for the (platformID, bcID) pair.
	Upsert(epbc *models.ExternalPlatformBlockchainCurrency) error
	// GetByPair returns the row for the (platformID, bcID) pair, or
	// gorm.ErrRecordNotFound when no row exists.
	GetByPair(platformID, blockchainCurrencyID uint) (*models.ExternalPlatformBlockchainCurrency, error)
	// ListByPlatform returns every enabled+disabled row for a platform.
	ListByPlatform(platformID uint) ([]models.ExternalPlatformBlockchainCurrency, error)
	// SetEnabled flips the Enabled flag with a single conditional UPDATE.
	SetEnabled(platformID, blockchainCurrencyID uint, enabled bool) error
}

// ExternalPlatformBlockchainCurrencyRepositoryImpl is the GORM implementation.
type ExternalPlatformBlockchainCurrencyRepositoryImpl struct {
	db *gorm.DB
}

// NewExternalPlatformBlockchainCurrencyRepository constructs the repo.
func NewExternalPlatformBlockchainCurrencyRepository(db *gorm.DB) ExternalPlatformBlockchainCurrencyRepository {
	return &ExternalPlatformBlockchainCurrencyRepositoryImpl{db: db}
}

// Upsert finds an existing row by (platformID, bcID) and updates it, or
// creates a new row when none exists. The check + write happen in a single
// transaction to avoid the read-modify-write race.
func (r *ExternalPlatformBlockchainCurrencyRepositoryImpl) Upsert(epbc *models.ExternalPlatformBlockchainCurrency) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.ExternalPlatformBlockchainCurrency
		err := tx.Where(
			"external_platform_id = ? AND blockchain_currency_id = ?",
			epbc.ExternalPlatformID, epbc.BlockchainCurrencyID,
		).First(&existing).Error
		if err == nil {
			existing.Enabled = epbc.Enabled
			existing.AutoApproveThreshold = epbc.AutoApproveThreshold
			existing.HourlyCap = epbc.HourlyCap
			existing.DailyCap = epbc.DailyCap
			existing.MinAmount = epbc.MinAmount
			existing.MaxAmount = epbc.MaxAmount
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			*epbc = existing
			return nil
		}
		return tx.Create(epbc).Error
	})
}

// GetByPair returns the per-platform per-currency row.
func (r *ExternalPlatformBlockchainCurrencyRepositoryImpl) GetByPair(platformID, blockchainCurrencyID uint) (*models.ExternalPlatformBlockchainCurrency, error) {
	var epbc models.ExternalPlatformBlockchainCurrency
	if err := r.db.Where(
		"external_platform_id = ? AND blockchain_currency_id = ?",
		platformID, blockchainCurrencyID,
	).First(&epbc).Error; err != nil {
		return nil, err
	}
	return &epbc, nil
}

// ListByPlatform returns every row for a platform regardless of enabled state.
func (r *ExternalPlatformBlockchainCurrencyRepositoryImpl) ListByPlatform(platformID uint) ([]models.ExternalPlatformBlockchainCurrency, error) {
	var rows []models.ExternalPlatformBlockchainCurrency
	if err := r.db.Where("external_platform_id = ?", platformID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SetEnabled atomically flips the Enabled flag.
func (r *ExternalPlatformBlockchainCurrencyRepositoryImpl) SetEnabled(platformID, blockchainCurrencyID uint, enabled bool) error {
	return r.db.Model(&models.ExternalPlatformBlockchainCurrency{}).
		Where("external_platform_id = ? AND blockchain_currency_id = ?", platformID, blockchainCurrencyID).
		Update("enabled", enabled).Error
}
