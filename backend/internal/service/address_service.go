package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// AddressService is the query/facade layer over AddressRepository. It
// exposes balance queries, sweep eligibility, and balance refresh operations
// that the sweep processor and dashboard call.
//
// It does NOT own address derivation — that's WalletService. It does NOT own
// address assignment — that's DepositAddressService. It only answers
// "what's the current state of an address" and "which addresses can I act
// on right now".
type AddressService struct {
	addressRepo       repository.AddressRepository
	blockchainCurRepo repository.BlockchainCurrencyRepository
}

func NewAddressService(
	addressRepo repository.AddressRepository,
	blockchainCurRepo repository.BlockchainCurrencyRepository,
) *AddressService {
	return &AddressService{
		addressRepo:       addressRepo,
		blockchainCurRepo: blockchainCurRepo,
	}
}

// Balances returns a per-currency balance summary for a member. Sums across
// all deposit addresses the member controls.
func (s *AddressService) Balances(memberID uint) ([]repository.BalanceEntry, error) {
	return s.addressRepo.Balances(memberID)
}

// PendingForApproval returns addresses that are pending sweep approval for
// this member. Used by the admin dashboard to show "X sweeps awaiting
// approval" cards.
func (s *AddressService) PendingForApproval(memberID uint) ([]repository.EligibleAddress, error) {
	return s.addressRepo.PendingForApproval(memberID)
}

// GetEligibleAddressesToSweep returns addresses whose balance is at or above
// the given threshold and are NOT currently locked. The sweep processor
// calls this periodically per blockchain_currency.
//
// Returns an empty slice (not nil, not error) when nothing is eligible —
// callers can safely range over the result without nil checks.
func (s *AddressService) GetEligibleAddressesToSweep(blockchainCurrencyID uint, threshold decimal.Decimal) ([]repository.EligibleAddress, error) {
	if threshold.IsNegative() {
		return nil, errors.New("threshold must be non-negative")
	}
	addrs, err := s.addressRepo.GetEligibleAddressesToSweep(blockchainCurrencyID, threshold)
	if err != nil {
		return nil, err
	}
	if addrs == nil {
		return []repository.EligibleAddress{}, nil
	}
	return addrs, nil
}

// GetEligibleAddressesToTransferFees returns addresses that hold non-zero
// token balances but no native balance (ETH/TRX) — candidates for gas
// top-up before ERC-20 sweep can proceed.
func (s *AddressService) GetEligibleAddressesToTransferFees(blockchainCurrencyID uint) ([]repository.EligibleAddress, error) {
	addrs, err := s.addressRepo.GetEligibleAddressesToTransferFees(blockchainCurrencyID)
	if err != nil {
		return nil, err
	}
	if addrs == nil {
		return []repository.EligibleAddress{}, nil
	}
	return addrs, nil
}

// RefreshBalancesForChain walks every active deposit address on the given
// blockchain and updates `account_addresses.balance` with fresh on-chain
// data fetched via the provided adapter. Returns the count of addresses
// whose balance changed.
//
// This is a long-running operation — the caller should run it in a
// background goroutine. The operation is idempotent and safe to call
// concurrently with the sweep processor (address-level updates).
func (s *AddressService) RefreshBalancesForChain(ctx context.Context, blockchainID uint, adapter blockchain.ChainAdapter) (int64, error) {
	if adapter == nil {
		return 0, errors.New("adapter is nil")
	}

	// The repo method takes a callback so the repo layer stays
	// adapter-agnostic. The callback resolves blockchain_currency -> token
	// address -> adapter.GetBalance.
	fetchBalance := func(addr string, blockchainCurrencyID uint) (decimal.Decimal, error) {
		bc, err := s.blockchainCurRepo.GetByID(blockchainCurrencyID)
		if err != nil {
			return decimal.Zero, err
		}
		token := ""
		if bc.Address != "" {
			token = bc.Address
		}
		bigBal, err := adapter.GetBalance(ctx, addr, token)
		if err != nil {
			return decimal.Zero, err
		}
		if bigBal == nil {
			return decimal.Zero, nil
		}
		// Convert big.Int to decimal, scaled by wallet precision
		raw := decimal.NewFromBigInt(bigBal, 0)
		if bc.WalletPrecision > 0 {
			divisor := decimal.New(1, int32(bc.WalletPrecision))
			return raw.Div(divisor), nil
		}
		return raw, nil
	}

	return s.addressRepo.ProcessAddressBalanceAndUpdateStatus(blockchainID, fetchBalance)
}

// MarkLocked marks an address as locked (a sweep is in flight against it)
// for the given duration. Returns an error if the address pool row doesn't
// exist. Uses the address string (not ID) because callers typically have
// the string.
func (s *AddressService) MarkLocked(addressPoolID uint, lockDurationSeconds int) error {
	if lockDurationSeconds <= 0 {
		return fmt.Errorf("lock duration must be positive, got %d", lockDurationSeconds)
	}
	return s.addressRepo.MarkLocked(addressPoolID, lockDurationSeconds)
}

// MarkUnlocked clears the lock.
func (s *AddressService) MarkUnlocked(addressPoolID uint) error {
	return s.addressRepo.MarkUnlocked(addressPoolID)
}
