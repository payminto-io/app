package service

import (
	"errors"
	"fmt"
	"github.com/payminto/payminto/backend/internal/environment"
	"log"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ErrExternalPlatformNotFound signals an unknown external platform.
var ErrExternalPlatformNotFound = errors.New("external platform not found")

// ExternalPlatformInput carries the mutable fields a caller can set when
// creating or updating a platform. The status field is managed via the
// Activate / Deactivate methods.
type ExternalPlatformInput struct {
	MemberID         uint
	Name             string
	Website          string
	LogoPath         string
	BrandColor       string
	SuccessEndpoint  string
	FrontendEndpoint *string
	CancelEndpoint   *string
	SupportEmail     *string
}

// ExternalPlatformService is the admin-facing CRUD service for managing
// merchant projects (external platforms). Each platform owns one or more
// API keys, blockchain currency enable lists, and wallet→family mappings.
//
// API keys are persisted in a separate api_keys table; the service generates
// the key via the package-level GenerateAPIKey + HashAPIKey helpers and
// returns the plaintext to the caller exactly once.
type ExternalPlatformService struct {
	platformRepo repository.ExternalPlatformRepository
	apiKeyRepo   repository.APIKeyRepository
	environment  environment.Environment

	walletSvc            *WalletService
	addressPoolSvc       *AddressPoolService
	blockchainFamilyRepo repository.BlockchainFamilyRepository
	blockchainCurRepo    repository.BlockchainCurrencyRepository
	epbcRepo             repository.ExternalPlatformBlockchainCurrencyRepository
}

// NewExternalPlatformService wires the platform service.
func NewExternalPlatformService(
	platformRepo repository.ExternalPlatformRepository,
	apiKeyRepo repository.APIKeyRepository,
) *ExternalPlatformService {
	return &ExternalPlatformService{
		platformRepo: platformRepo,
		apiKeyRepo:   apiKeyRepo,
	}
}

// SetWalletService injects the HD wallet service (Pass 2 wiring).
func (s *ExternalPlatformService) SetWalletService(ws *WalletService) { s.walletSvc = ws }

// SetEnvironment sets the environment newly issued platform keys carry; unset means test.
func (s *ExternalPlatformService) SetEnvironment(env environment.Environment) { s.environment = env }

// SetAddressPoolService injects the address pool service (Pass 2 wiring).
func (s *ExternalPlatformService) SetAddressPoolService(aps *AddressPoolService) {
	s.addressPoolSvc = aps
}

// SetBlockchainFamilyRepo injects the blockchain family repository (Pass 2 wiring).
func (s *ExternalPlatformService) SetBlockchainFamilyRepo(r repository.BlockchainFamilyRepository) {
	s.blockchainFamilyRepo = r
}

// SetBlockchainCurrencyRepo injects the blockchain currency repository (Pass 2 wiring).
func (s *ExternalPlatformService) SetBlockchainCurrencyRepo(r repository.BlockchainCurrencyRepository) {
	s.blockchainCurRepo = r
}

// SetEPBCRepo injects the external-platform-blockchain-currency repository (Pass 2 wiring).
func (s *ExternalPlatformService) SetEPBCRepo(r repository.ExternalPlatformBlockchainCurrencyRepository) {
	s.epbcRepo = r
}

// Create persists a new external platform and issues an initial API key.
// Returns the platform and the plaintext API key — the only time the key
// is exposed in cleartext.
func (s *ExternalPlatformService) Create(input ExternalPlatformInput) (*models.ExternalPlatform, string, error) {
	if input.Name == "" {
		return nil, "", errors.New("platform name is required")
	}
	platform := &models.ExternalPlatform{
		Name:             input.Name,
		Website:          input.Website,
		LogoPath:         input.LogoPath,
		BrandColor:       input.BrandColor,
		SuccessEndpoint:  input.SuccessEndpoint,
		FrontendEndpoint: input.FrontendEndpoint,
		CancelEndpoint:   input.CancelEndpoint,
		SupportEmail:     input.SupportEmail,
	}
	if err := s.platformRepo.Create(platform); err != nil {
		return nil, "", fmt.Errorf("create platform: %w", err)
	}

	plain, err := s.issueAPIKey(platform.ID)
	if err != nil {
		// Roll back the platform on key creation failure.
		_ = s.platformRepo.Delete(platform.ID)
		return nil, "", fmt.Errorf("issue api key: %w", err)
	}

	// Phase D: auto-create HD wallets and enable all blockchain currencies.
	s.createDefaultWalletMappings(input.MemberID)
	s.enableAllBlockchainCurrencies(platform.ID)

	return platform, plain, nil
}

// GetByID returns a platform by primary key.
func (s *ExternalPlatformService) GetByID(id uint) (*models.ExternalPlatform, error) {
	p, err := s.platformRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExternalPlatformNotFound
		}
		return nil, err
	}
	return p, nil
}

// ListAll returns every platform — admin-only.
func (s *ExternalPlatformService) ListAll() ([]models.ExternalPlatform, error) {
	return s.platformRepo.List()
}

// Update mutates the editable fields on a platform.
func (s *ExternalPlatformService) Update(id uint, input ExternalPlatformInput) (*models.ExternalPlatform, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if input.Name != "" {
		p.Name = input.Name
	}
	if input.Website != "" {
		p.Website = input.Website
	}
	if input.LogoPath != "" {
		p.LogoPath = input.LogoPath
	}
	if input.BrandColor != "" {
		p.BrandColor = input.BrandColor
	}
	if input.SuccessEndpoint != "" {
		p.SuccessEndpoint = input.SuccessEndpoint
	}
	if input.FrontendEndpoint != nil {
		p.FrontendEndpoint = input.FrontendEndpoint
	}
	if input.CancelEndpoint != nil {
		p.CancelEndpoint = input.CancelEndpoint
	}
	if input.SupportEmail != nil {
		p.SupportEmail = input.SupportEmail
	}
	if err := s.platformRepo.Update(p); err != nil {
		return nil, fmt.Errorf("update platform: %w", err)
	}
	return p, nil
}

// Delete soft-deletes the platform. Existing API keys, webhooks, payments
// remain readable but no new operations can be authenticated.
func (s *ExternalPlatformService) Delete(id uint) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}
	return s.platformRepo.Delete(id)
}

// RegenerateAPIKey deactivates every existing API key for the platform and
// issues a fresh one. Returns the new plaintext key.
func (s *ExternalPlatformService) RegenerateAPIKey(platformID uint) (string, error) {
	if _, err := s.GetByID(platformID); err != nil {
		return "", err
	}

	existing, err := s.apiKeyRepo.ListByExternalPlatformID(platformID)
	if err != nil {
		return "", fmt.Errorf("list existing keys: %w", err)
	}
	for i := range existing {
		k := &existing[i]
		k.Status = "inactive"
		if err := s.apiKeyRepo.Update(k); err != nil {
			return "", fmt.Errorf("deactivate key %d: %w", k.ID, err)
		}
	}

	return s.issueAPIKey(platformID)
}

// createDefaultWalletMappings iterates all blockchain families, creates an HD
// wallet per family, and pre-generates 20 deposit addresses in each wallet's pool.
// Errors are logged but do not fail the platform creation.
func (s *ExternalPlatformService) createDefaultWalletMappings(memberID uint) {
	if s.walletSvc == nil || s.addressPoolSvc == nil || s.blockchainFamilyRepo == nil {
		return
	}
	families, err := s.blockchainFamilyRepo.ListAll()
	if err != nil {
		log.Printf("[ExternalPlatformService] failed to list blockchain families: %v", err)
		return
	}
	for i := range families {
		family := &families[i]
		wallet, err := s.walletSvc.CreateHDWallet(memberID, family)
		if err != nil {
			log.Printf("[ExternalPlatformService] failed to create HD wallet for family %s (memberID=%d): %v", family.Code, memberID, err)
			continue
		}
		if _, err := s.addressPoolSvc.GenerateBatch(wallet.ID, 20); err != nil {
			log.Printf("[ExternalPlatformService] failed to pre-generate addresses for wallet %d (family %s): %v", wallet.ID, family.Code, err)
		}
	}
}

// enableAllBlockchainCurrencies iterates all blockchain currencies and upserts
// an EPBC row for the given platform so every currency is enabled by default.
// Errors are logged but do not fail the platform creation.
func (s *ExternalPlatformService) enableAllBlockchainCurrencies(platformID uint) {
	if s.epbcRepo == nil || s.blockchainCurRepo == nil {
		return
	}
	currencies, err := s.blockchainCurRepo.ListAll()
	if err != nil {
		log.Printf("[ExternalPlatformService] failed to list blockchain currencies: %v", err)
		return
	}
	for _, bc := range currencies {
		row := &models.ExternalPlatformBlockchainCurrency{
			ExternalPlatformID:   platformID,
			BlockchainCurrencyID: bc.ID,
			Enabled:              true,
		}
		if err := s.epbcRepo.Upsert(row); err != nil {
			log.Printf("[ExternalPlatformService] failed to upsert EPBC for platform %d, bc %d: %v", platformID, bc.ID, err)
		}
	}
}

// issueAPIKey creates a fresh hashed API key row for the platform and returns
// the plaintext value to the caller. Internal helper.
func (s *ExternalPlatformService) issueAPIKey(platformID uint) (string, error) {
	env := s.environment
	if env == "" {
		env = environment.Test
	}
	plain, err := GenerateAPIKeyFor(env)
	if err != nil {
		return "", err
	}
	if err := s.apiKeyRepo.Create(NewAPIKeyRow(plain, env, platformID)); err != nil {
		return "", err
	}
	return plain, nil
}
