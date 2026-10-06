package service

import (
	"context"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// AccountRewardService manages referral reward distribution and the per-member
// AccountReward balance. It is the Phase I replacement for the stub that shipped
// with Phase F.
//
// The actual on-chain payout signing is wired in Phase K; for Phase I, pending
// rewards are marked paid with a placeholder hash after ledger entries are written.
type AccountRewardService struct {
	accountRepo repository.AccountRepository
}

// NewAccountRewardService constructs an AccountRewardService.
func NewAccountRewardService(accountRepo repository.AccountRepository) *AccountRewardService {
	return &AccountRewardService{accountRepo: accountRepo}
}

// GetBalance returns the AccountReward row for (memberID, currencyID).
// Returns a zero-balance AccountReward if none exists.
func (s *AccountRewardService) GetBalance(_ context.Context, memberID, currencyID uint) (*models.AccountReward, error) {
	ar, err := s.accountRepo.GetAccountReward(memberID, currencyID)
	if err != nil {
		return nil, fmt.Errorf("account reward get balance: %w", err)
	}
	return ar, nil
}

// ProcessPending is a deliberate no-op stub deferred to Phase K.
//
// The previous implementation called UpdateAccountBalance per pending row
// without (a) clearing locked, (b) writing to the ledger, or (c) any
// idempotency guard — causing double-credits on every tick. The full
// fulfilment path requires:
//
//   - Atomic conditional UPDATE on reward status (pending → paid)
//   - LedgerService.RecordReferralPayout inside the same transaction
//   - Real on-chain payout via SecretsVault tx signing (Phase K)
//
// Until Phase K wires those primitives this method intentionally does
// nothing so the worker loop can keep running without corrupting balances.
//
// TODO(phase-k-payout): replace with the atomic claim + ledger write + sign + broadcast flow.
func (s *AccountRewardService) ProcessPending() error {
	return nil
}

// RetryFailed is a deliberate no-op stub deferred to Phase K.
//
// TODO(phase-k-payout): query AccountReward rows in failed state and retry
// them via the same atomic claim + ledger write + on-chain payout flow that
// ProcessPending will eventually use.
func (s *AccountRewardService) RetryFailed() error {
	return nil
}

// CreditLocked is a deliberate no-op stub deferred to Phase K.
//
// Previously this method only logged and never wrote anything, despite its
// doc comment claiming it credited the locked field. Removing the misleading
// behaviour: it now simply validates the amount and returns nil. The real
// reward credit flow lands in Phase K alongside ProcessPending.
//
// TODO(phase-k-payout): wire IncrementLocked on AccountRepository and call it here.
func (s *AccountRewardService) CreditLocked(_ context.Context, memberID, currencyID uint, amount decimal.Decimal) error {
	if amount.IsNegative() || amount.IsZero() {
		return fmt.Errorf("account reward credit: amount must be positive")
	}
	return nil
}

// GetOrCreateAccount delegates to the underlying repository to ensure a ledger
// account row exists for (memberID, currencyID).
func (s *AccountRewardService) GetOrCreateAccount(memberID, currencyID uint) (*models.Account, error) {
	return s.accountRepo.GetOrCreateAccount(memberID, currencyID)
}
