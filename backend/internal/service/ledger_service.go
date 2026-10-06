package service

import (
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// LedgerService is the ONLY place in the codebase that writes ledger rows.
// Other services call it with domain events; it builds the balanced
// LedgerEntries and passes them to AccountRepository.CreateLedgerEntries.
//
// Every method is idempotent against its reference (paymentID, sweepID,
// withdrawalID) — callers can safely retry on failure since GORM will return
// a unique-constraint error for duplicate reference pairs, which the caller
// should treat as a no-op.
type LedgerService struct {
	accountRepo repository.AccountRepository
}

// NewLedgerService constructs a LedgerService with the required AccountRepository.
func NewLedgerService(accountRepo repository.AccountRepository) *LedgerService {
	return &LedgerService{accountRepo: accountRepo}
}

// RecordPaymentDeposit records a confirmed deposit arriving into Payminto custody.
//
//	Debit: assets/crypto_assets    (coins we now hold)
//	Credit: liabilities/merchant_balance (what we owe the merchant)
func (s *LedgerService) RecordPaymentDeposit(paymentID uint, currencyID uint, amount decimal.Decimal) error {
	refID := paymentID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "payment",
		}},
		Liabilities: []models.Liability{{
			Code:        "merchant_balance",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "payment",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordSweep records a SmartSweep batch moving coins from deposit addresses
// to cold storage.
//
//	Assets stay in place (coins still ours, now in cold wallet).
//	Expense entry records the gas cost.
//	To keep entries balanced: debit expense for gas, credit asset for gas consumed.
func (s *LedgerService) RecordSweep(sweepID uint, currencyID uint, amount, gasCost decimal.Decimal) error {
	refID := sweepID

	// If no gas cost, produce the minimal balanced pair — asset moves from
	// hot-side to cold-side (same account bucket, we record it as a debit
	// to cold_wallet_assets and credit to deposit_assets).
	entries := repository.LedgerEntries{
		Assets: []models.Asset{
			{
				// Increase cold wallet asset balance
				Code:        "cold_wallet_assets",
				Debit:       amount,
				Credit:      decimal.Zero,
				CurrencyID:  currencyID,
				ReferenceID: &refID,
				Reference:   "sweep",
			},
			{
				// Decrease deposit-side asset balance
				Code:        "crypto_assets",
				Debit:       decimal.Zero,
				Credit:      amount.Add(gasCost),
				CurrencyID:  currencyID,
				ReferenceID: &refID,
				Reference:   "sweep",
			},
		},
		Expenses: []models.Expense{{
			// Gas cost as an operational expense
			Code:        "sweep_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "sweep",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordWithdrawal records a merchant withdrawal leaving the platform.
//
//	Liability decreases (we owe less to the merchant).
//	Asset decreases (coins left our custody).
//	Expense records the gas cost.
func (s *LedgerService) RecordWithdrawal(withdrawalID uint, currencyID uint, amount, gasCost decimal.Decimal) error {
	refID := withdrawalID
	total := amount.Add(gasCost)
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      total,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
		Liabilities: []models.Liability{{
			Code:        "merchant_balance",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
		Expenses: []models.Expense{{
			Code:        "withdrawal_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordGasFee records a standalone gas-fee transfer (e.g. funding a deposit
// address with ETH before sweeping an ERC-20).
//
//	Asset decreases by gas cost.
//	Expense increases by gas cost.
func (s *LedgerService) RecordGasFee(txID uint, currencyID uint, gasCost decimal.Decimal) error {
	refID := txID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      gasCost,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "gas_fee",
		}},
		Expenses: []models.Expense{{
			Code:        "gas_fee",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "gas_fee",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordDuplicateDeposit records the second occurrence of a deposit with the
// same tx_hash+address. The funds are unclaimed by the merchant (no
// PaymentRequest), so we book them as platform revenue.
//
//	Asset increases (coins we now hold).
//	Revenue increases (windfall income — will be refunded or absorbed per policy).
func (s *LedgerService) RecordDuplicateDeposit(depositID uint, currencyID uint, amount decimal.Decimal) error {
	refID := depositID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "duplicate_deposit",
		}},
		Revenues: []models.Revenue{{
			Code:        "unclaimed_deposit",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "duplicate_deposit",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordReferralPayout records a referral reward being paid out to a member.
// Mirrors RecordWithdrawal but uses referral_reward as the reference type.
//
//	Liability decreases (platform owes the member their reward).
//	Asset decreases (coins leave the platform).
func (s *LedgerService) RecordReferralPayout(rewardID uint, currencyID uint, amount decimal.Decimal) error {
	refID := rewardID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "referral_reward",
		}},
		Liabilities: []models.Liability{{
			Code:        "referral_rewards_payable",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "referral_reward",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}

// RecordAddressDeployment records the gas spent deploying a smart contract
// deposit wallet (CREATE2 SCW deploy).
//
//	Asset decreases by gas cost.
//	Expense records the deployment gas.
func (s *LedgerService) RecordAddressDeployment(deploymentID uint, currencyID uint, gasCost decimal.Decimal) error {
	refID := deploymentID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      gasCost,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "address_deployment",
		}},
		Expenses: []models.Expense{{
			Code:        "deployment_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  currencyID,
			ReferenceID: &refID,
			Reference:   "address_deployment",
		}},
	}
	return s.accountRepo.CreateLedgerEntries(entries)
}
