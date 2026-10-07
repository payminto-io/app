package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// DepositAddressService assigns pre-generated pool addresses to payment
// requests. It's called by PaymentService.CreatePayment when the caller
// provides a blockchain preference.
//
// Pipeline:
//  1. Look up the BlockchainCurrency by (chainCode, currencyCode) — this
//     gives us the family ID
//  2. Find the merchant's Wallet for that family
//  3. Pull an available AddressPool row from AddressPoolService.GetAvailable
//  4. Mark the pool row used
//  5. Create a DepositAddress row linking (address, blockchain_currency,
//     member, payment_request)
//  6. Return the DepositAddress
//
// If the pool is empty, the service auto-tops it up by calling
// EnsurePoolSize(50). If pool generation also fails, it returns an error
// — callers should retry or escalate.
type DepositAddressService struct {
	depositAddressRepo repository.DepositAddressRepository
	walletRepo         repository.WalletRepository
	blockchainCurRepo  repository.BlockchainCurrencyRepository
	walletService      *WalletService
	addressPoolService *AddressPoolService
	// solanaAccounts is set by WithSolanaDepositAccounts; SOLANA tokens deposit into an ATA, not the owner.
	solanaAccounts   repository.SolanaDepositAccountRepository
	solanaDB         *gorm.DB
	solanaLateWindow time.Duration
}

func NewDepositAddressService(
	depositAddressRepo repository.DepositAddressRepository,
	walletRepo repository.WalletRepository,
	blockchainCurRepo repository.BlockchainCurrencyRepository,
	walletService *WalletService,
	addressPoolService *AddressPoolService,
) *DepositAddressService {
	return &DepositAddressService{
		depositAddressRepo: depositAddressRepo,
		walletRepo:         walletRepo,
		blockchainCurRepo:  blockchainCurRepo,
		walletService:      walletService,
		addressPoolService: addressPoolService,
	}
}

// AssignForPayment reserves a deposit address for a payment. chainCode is
// the blockchain (e.g. "ETH", "BASE"); currencyCode is the token
// (e.g. "ETH", "USDC", "USDT"). Empty currencyCode defaults to the chain's
// native currency (matching chainCode).
//
// Returns the created DepositAddress with BlockchainCurrency preloaded so
// callers can build a QR code or explorer link without a second lookup.
func (s *DepositAddressService) AssignForPayment(
	payment *models.PaymentRequest,
	chainCode, currencyCode string,
) (*models.DepositAddress, error) {
	if payment == nil || payment.ID == 0 {
		return nil, errors.New("payment is nil or has no ID")
	}
	if chainCode == "" {
		return nil, errors.New("chainCode is required")
	}
	if currencyCode == "" {
		currencyCode = chainCode
	}

	// Step 1: resolve the BlockchainCurrency
	bc, err := s.blockchainCurRepo.GetByBlockchainCodeAndCurrencyCode(chainCode, currencyCode)
	if err != nil {
		return nil, fmt.Errorf("blockchain_currency %s/%s not found: %w", chainCode, currencyCode, err)
	}
	if bc.Blockchain == nil {
		return nil, fmt.Errorf("blockchain_currency missing Blockchain preload")
	}

	// Step 2: find the merchant's wallet for this family
	wallet, err := s.walletRepo.GetByMemberAndFamily(payment.MemberID, bc.Blockchain.BlockchainFamilyID)
	if err != nil {
		return nil, fmt.Errorf("wallet not found for member %d family %d: %w",
			payment.MemberID, bc.Blockchain.BlockchainFamilyID, err)
	}

	// Step 3: atomically claim an available pool row; auto-top-up if empty.
	pool, err := s.addressPoolService.ClaimNextAvailable(wallet.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Pool empty — try to top it up once, then claim again.
		if _, topErr := s.addressPoolService.EnsurePoolSize(wallet.ID, 50); topErr != nil {
			return nil, fmt.Errorf("pool empty and top-up failed: %w", topErr)
		}
		pool, err = s.addressPoolService.ClaimNextAvailable(wallet.ID)
		if err != nil {
			return nil, fmt.Errorf("pool still empty after top-up: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("claim available pool entry: %w", err)
	}

	// Step 5: create the DepositAddress row
	paymentID := payment.ID
	if isSolanaToken(bc) {
		da, err := s.createSolanaDepositAddress(bc, pool, payment)
		if err != nil {
			return nil, err
		}
		return s.depositAddressRepo.GetByAddress(da.Address, bc.ID)
	}
	da := &models.DepositAddress{
		Address:              pool.Address,
		BlockchainCurrencyID: bc.ID,
		MemberID:             payment.MemberID,
		PaymentRequestID:     &paymentID,
	}
	if err := s.depositAddressRepo.Create(da); err != nil {
		return nil, fmt.Errorf("create deposit_address: %w", err)
	}

	// Re-fetch so the returned object includes the BlockchainCurrency preload
	return s.depositAddressRepo.GetByAddress(pool.Address, bc.ID)
}

// GetByAddress looks up a DepositAddress by its on-chain address and
// blockchain_currency_id. Used by the block processor to match incoming
// transactions to known addresses.
func (s *DepositAddressService) GetByAddress(address string, blockchainCurrencyID uint) (*models.DepositAddress, error) {
	return s.depositAddressRepo.GetByAddress(address, blockchainCurrencyID)
}

// ListForPayment returns every deposit address currently assigned to a
// payment (one per chain the merchant accepted). Used by the dashboard
// payment detail view.
func (s *DepositAddressService) ListForPayment(paymentRequestID uint) ([]models.DepositAddress, error) {
	return s.depositAddressRepo.GetByPaymentRequestID(paymentRequestID)
}

// ListByMember returns all deposit addresses owned by a member. Used by
// the wallets page on the dashboard.
func (s *DepositAddressService) ListByMember(memberID uint, opts ...repository.QueryOption) ([]models.DepositAddress, error) {
	return s.depositAddressRepo.ListByMember(memberID, opts...)
}
