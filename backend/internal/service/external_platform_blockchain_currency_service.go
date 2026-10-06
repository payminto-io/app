package service

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ErrCurrencyNotEnabledForPlatform is returned when a withdrawal or payment
// targets a (platform, currency) pair that is not configured. Used by
// WithdrawalService.Create to enforce the per-project currency enable list.
var ErrCurrencyNotEnabledForPlatform = errors.New("currency not enabled for platform")

// EPBCConfig is the configurable subset of an
// ExternalPlatformBlockchainCurrency row. All fields are optional pointers
// because nil means "no limit".
type EPBCConfig struct {
	Enabled              bool
	AutoApproveThreshold *decimal.Decimal
	HourlyCap            *decimal.Decimal
	DailyCap             *decimal.Decimal
	MinAmount            *decimal.Decimal
	MaxAmount            *decimal.Decimal
}

// ExternalPlatformBlockchainCurrencyService manages the per-platform per-
// blockchain-currency enable list and limit configuration. WithdrawalService
// consults this service before authorising every payout to enforce the
// platform's allowed currencies and dollar caps.
type ExternalPlatformBlockchainCurrencyService struct {
	repo repository.ExternalPlatformBlockchainCurrencyRepository
}

// NewExternalPlatformBlockchainCurrencyService wires the EPBC service.
func NewExternalPlatformBlockchainCurrencyService(repo repository.ExternalPlatformBlockchainCurrencyRepository) *ExternalPlatformBlockchainCurrencyService {
	return &ExternalPlatformBlockchainCurrencyService{repo: repo}
}

// Enable upserts the row for the (platform, currency) pair with the given
// configuration and Enabled=true.
func (s *ExternalPlatformBlockchainCurrencyService) Enable(platformID, blockchainCurrencyID uint, cfg EPBCConfig) (*models.ExternalPlatformBlockchainCurrency, error) {
	row := &models.ExternalPlatformBlockchainCurrency{
		ExternalPlatformID:   platformID,
		BlockchainCurrencyID: blockchainCurrencyID,
		Enabled:              true,
		AutoApproveThreshold: cfg.AutoApproveThreshold,
		HourlyCap:            cfg.HourlyCap,
		DailyCap:             cfg.DailyCap,
		MinAmount:            cfg.MinAmount,
		MaxAmount:            cfg.MaxAmount,
	}
	if err := s.repo.Upsert(row); err != nil {
		return nil, fmt.Errorf("epbc upsert: %w", err)
	}
	return row, nil
}

// Disable flips the Enabled flag to false. The row is preserved so the
// configured limits survive a re-enable.
func (s *ExternalPlatformBlockchainCurrencyService) Disable(platformID, blockchainCurrencyID uint) error {
	return s.repo.SetEnabled(platformID, blockchainCurrencyID, false)
}

// GetByPair returns the EPBC row, mapping ErrRecordNotFound to
// ErrCurrencyNotEnabledForPlatform so callers can check the sentinel.
func (s *ExternalPlatformBlockchainCurrencyService) GetByPair(platformID, blockchainCurrencyID uint) (*models.ExternalPlatformBlockchainCurrency, error) {
	row, err := s.repo.GetByPair(platformID, blockchainCurrencyID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCurrencyNotEnabledForPlatform
		}
		return nil, err
	}
	return row, nil
}

// ListByPlatform returns every EPBC row for a platform.
func (s *ExternalPlatformBlockchainCurrencyService) ListByPlatform(platformID uint) ([]models.ExternalPlatformBlockchainCurrency, error) {
	return s.repo.ListByPlatform(platformID)
}

// UpdateLimits is a thin wrapper that re-upserts the row preserving Enabled.
func (s *ExternalPlatformBlockchainCurrencyService) UpdateLimits(platformID, blockchainCurrencyID uint, cfg EPBCConfig) (*models.ExternalPlatformBlockchainCurrency, error) {
	existing, err := s.GetByPair(platformID, blockchainCurrencyID)
	if err != nil {
		return nil, err
	}
	cfg.Enabled = existing.Enabled
	return s.Enable(platformID, blockchainCurrencyID, cfg)
}
