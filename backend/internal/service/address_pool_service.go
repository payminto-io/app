package service

import (
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// AddressPoolService keeps a warm pool of pre-generated deposit addresses
// per wallet so payment creation never blocks on key derivation. The
// background warmer worker calls EnsurePoolSize periodically; payment
// creation calls GetAvailable to claim one.
type AddressPoolService struct {
	poolRepo             repository.AddressPoolRepository
	walletRepo           repository.WalletRepository
	blockchainFamilyRepo repository.BlockchainFamilyRepository
	walletService        *WalletService
}

func NewAddressPoolService(
	poolRepo repository.AddressPoolRepository,
	walletRepo repository.WalletRepository,
	blockchainFamilyRepo repository.BlockchainFamilyRepository,
	walletService *WalletService,
) *AddressPoolService {
	return &AddressPoolService{
		poolRepo:             poolRepo,
		walletRepo:           walletRepo,
		blockchainFamilyRepo: blockchainFamilyRepo,
		walletService:        walletService,
	}
}

// GenerateBatch derives `count` fresh addresses for the wallet and creates
// them as `available` AddressPool rows in one bulk insert. Returns the
// number of addresses actually created.
func (s *AddressPoolService) GenerateBatch(walletID uint, count int) (int, error) {
	if count <= 0 {
		return 0, errors.New("count must be positive")
	}
	if s.walletService == nil {
		return 0, errors.New("wallet service not wired")
	}

	wallet, err := s.walletRepo.GetByID(walletID)
	if err != nil {
		return 0, fmt.Errorf("get wallet: %w", err)
	}

	family, err := s.blockchainFamilyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		return 0, fmt.Errorf("get blockchain family: %w", err)
	}

	chainCode, err := chainCodeForFamily(family.Code)
	if err != nil {
		return 0, err
	}

	pools := make([]models.AddressPool, 0, count)
	for range count {
		addr, idx, err := s.walletService.DeriveNextAddress(wallet.MemberID, walletID, chainCode)
		if err != nil {
			// Return what we have so far — partial success is better than none
			if len(pools) > 0 {
				_ = s.poolRepo.BulkCreate(pools)
			}
			return len(pools), fmt.Errorf("derive next address: %w", err)
		}
		pools = append(pools, models.AddressPool{
			Address:            addr,
			PathIndex:          idx,
			Status:             "available",
			WalletID:           walletID,
			BlockchainFamilyID: wallet.BlockchainFamilyID,
		})
	}

	if err := s.poolRepo.BulkCreate(pools); err != nil {
		return 0, fmt.Errorf("bulk create pool: %w", err)
	}
	return len(pools), nil
}

// GetAvailable returns the next available pool row for the wallet, or
// gorm.ErrRecordNotFound if the pool is empty. Callers that need to claim
// the address atomically should use ClaimNextAvailable instead.
func (s *AddressPoolService) GetAvailable(walletID uint) (*models.AddressPool, error) {
	return s.poolRepo.GetAvailableByWallet(walletID)
}

// ClaimNextAvailable atomically selects and marks used the lowest path_index
// available address for the wallet. Safe for concurrent callers — uses
// SELECT ... FOR UPDATE SKIP LOCKED under the hood.
func (s *AddressPoolService) ClaimNextAvailable(walletID uint) (*models.AddressPool, error) {
	return s.poolRepo.ClaimNextAvailable(walletID)
}

// MarkUsed marks an AddressPool row as used so it's excluded from future
// GetAvailable calls. Idempotent.
func (s *AddressPoolService) MarkUsed(poolID uint) error {
	return s.poolRepo.MarkUsed(poolID)
}

// CountAvailable returns the number of available pool rows for the wallet.
// Used by the warmer worker to decide whether to top up.
func (s *AddressPoolService) CountAvailable(walletID uint) (int64, error) {
	return s.poolRepo.CountAvailableByWallet(walletID)
}

// EnsurePoolSize tops up the pool if the current available count is below
// minSize. Returns the number of addresses generated (0 if the pool was
// already large enough).
func (s *AddressPoolService) EnsurePoolSize(walletID uint, minSize int) (int, error) {
	if minSize <= 0 {
		return 0, nil
	}
	current, err := s.poolRepo.CountAvailableByWallet(walletID)
	if err != nil {
		return 0, err
	}
	if current >= int64(minSize) {
		return 0, nil
	}
	return s.GenerateBatch(walletID, minSize-int(current))
}

// GetByAddress is a helper so DepositAddressService can look up the pool
// row it's about to reserve.
func (s *AddressPoolService) GetByAddress(address string) (*models.AddressPool, error) {
	return s.poolRepo.GetByAddress(address)
}

// chainCodeForFamily maps a BlockchainFamily code to the chain code that
// WalletService.DeriveNextAddress expects. Mainnet only — testnet runs in
// a separate Payminto instance per our dual-mode architecture.
func chainCodeForFamily(familyCode string) (string, error) {
	switch familyCode {
	case "evm", "EVM":
		return "ETH", nil
	case "btc", "BTC":
		return "BTC", nil
	case "trx", "TRX":
		return "TRX", nil
	case "sol", "SOL", "SOL_Family":
		return "SOLANA", nil
	}
	return "", fmt.Errorf("unsupported blockchain family %q", familyCode)
}
