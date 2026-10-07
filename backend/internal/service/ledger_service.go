package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LedgerService is the ONLY place in the codebase that writes ledger rows.
// Other services call it with domain events; it builds the balanced
// LedgerEntries and, when a journal is configured, posts the same event to
// internal/ledger in the same transaction. The new ledger is the source of
// truth; the legacy tables are a V1 dual-write (see 01-ledger.md follow-ups).
//
// Every method is idempotent against its reference (paymentID, sweepID,
// withdrawalID): a replay returns nil and writes nothing.
type LedgerService struct {
	accountRepo repository.AccountRepository
	journal     *ledger.Service
	assetOf     AssetResolver
}

// AssetResolver maps a Payminto currency id to the ledger's asset code.
type AssetResolver func(currencyID uint) (string, error)

// Option configures a LedgerService without changing NewLedgerService's shape for existing callers.
type Option func(*LedgerService)

// WithJournal enables dual-write into internal/ledger; resolver must fail rather than guess.
func WithJournal(journal *ledger.Service, resolver AssetResolver) Option {
	return func(s *LedgerService) {
		s.journal = journal
		s.assetOf = resolver
	}
}

// NewLedgerService constructs a LedgerService with the required AccountRepository.
func NewLedgerService(accountRepo repository.AccountRepository, opts ...Option) *LedgerService {
	s := &LedgerService{accountRepo: accountRepo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// legacyAccount maps a legacy entry table and code to a ledger account.
// Owner ids are the legacy codes so every old row maps to exactly one new account.
func legacyAccount(code, asset string, kind ledger.AccountKind) ledger.AccountKey {
	return ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: code, Asset: asset, Kind: kind}
}

var legacyJournalKinds = map[string]ledger.JournalKind{
	"payment":            ledger.KindPayment,
	"sweep":              ledger.KindTransfer,
	"withdrawal":         ledger.KindSettlement,
	"gas_fee":            ledger.KindFee,
	"duplicate_deposit":  ledger.KindAdjustment,
	"referral_reward":    ledger.KindSettlement,
	"address_deployment": ledger.KindFee,
}

// journalFor rebuilds the legacy entries as one signed journal (debit positive, credit negative).
func journalFor(reference string, refID, currencyID uint, asset string, entries repository.LedgerEntries) (ledger.Journal, error) {
	kind, ok := legacyJournalKinds[reference]
	if !ok {
		return ledger.Journal{}, fmt.Errorf("ledger: no journal kind for legacy reference %q", reference)
	}
	j := ledger.Journal{
		Kind:           kind,
		Reference:      ledger.Reference{Type: reference, ID: strconv.FormatUint(uint64(refID), 10)},
		IdempotencyKey: "payminto:" + reference + ":" + strconv.FormatUint(uint64(refID), 10),
		Metadata:       map[string]any{"currency_id": currencyID},
	}
	add := func(code string, kind ledger.AccountKind, debit, credit decimal.Decimal) {
		if amount := debit.Sub(credit); !amount.IsZero() {
			j.Lines = append(j.Lines, ledger.Line{Account: legacyAccount(code, asset, kind), Amount: amount})
		}
	}
	for _, e := range entries.Assets {
		add(e.Code, ledger.KindAsset, e.Debit, e.Credit)
	}
	for _, e := range entries.Liabilities {
		add(e.Code, ledger.KindLiability, e.Debit, e.Credit)
	}
	for _, e := range entries.Revenues {
		add(e.Code, ledger.KindIncome, e.Debit, e.Credit)
	}
	for _, e := range entries.Expenses {
		add(e.Code, ledger.KindExpense, e.Debit, e.Credit)
	}
	return j, nil
}

// record writes the legacy entries and, when configured, the journal in one transaction.
// A replayed journal key skips the legacy write too, so retries never duplicate rows.
func (s *LedgerService) record(reference string, refID, currencyID uint, entries repository.LedgerEntries) error {
	if s.journal == nil {
		return s.accountRepo.CreateLedgerEntries(entries)
	}
	if err := entries.Validate(); err != nil {
		return fmt.Errorf("unbalanced ledger entries: %w", err)
	}
	asset, err := s.assetOf(currencyID)
	if err != nil {
		return fmt.Errorf("ledger: resolve asset for currency %d: %w", currencyID, err)
	}
	j, err := journalFor(reference, refID, currencyID, asset, entries)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return s.journal.Transaction(ctx, func(tx *gorm.DB) error {
		receipt, err := s.journal.PostIn(ctx, tx, j)
		if err != nil {
			return err
		}
		if receipt.Replayed {
			return nil
		}
		repo := s.accountRepo
		if binder, ok := repo.(repository.TxBinder); ok {
			repo = binder.WithTx(tx)
		}
		return repo.CreateLedgerEntries(entries)
	})
}

// ReconcileStoredBalances compares each legacy accounts.balance with the member's derived
// ledger liability and reports the drift. It never writes; nothing is corrected silently.
func (s *LedgerService) ReconcileStoredBalances(ctx context.Context, accounts []models.Account) ([]ledger.Drift, error) {
	if s.journal == nil {
		return nil, fmt.Errorf("ledger: reconcile needs a journal")
	}
	expected := make([]ledger.Expected, 0, len(accounts))
	for _, a := range accounts {
		asset, err := s.assetOf(a.CurrencyID)
		if err != nil {
			return nil, fmt.Errorf("ledger: resolve asset for currency %d: %w", a.CurrencyID, err)
		}
		expected = append(expected, ledger.Expected{
			Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: strconv.FormatUint(uint64(a.MemberID), 10), Asset: asset, Kind: ledger.KindLiability},
			Stored:  a.Balance,
		})
	}
	return s.journal.Reconcile(ctx, expected)
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
	return s.record("payment", paymentID, currencyID, entries)
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
	return s.record("sweep", sweepID, currencyID, entries)
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
	return s.record("withdrawal", withdrawalID, currencyID, entries)
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
	return s.record("gas_fee", txID, currencyID, entries)
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
	return s.record("duplicate_deposit", depositID, currencyID, entries)
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
	return s.record("referral_reward", rewardID, currencyID, entries)
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
	return s.record("address_deployment", deploymentID, currencyID, entries)
}
