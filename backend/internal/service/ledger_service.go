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
// Other services call it with domain events; it builds the balanced legacy
// LedgerEntries and, when a journal is configured, posts the same event to
// internal/ledger in the same transaction. The new ledger is the source of
// truth; the legacy tables are a V1 dual-write (see 01-ledger.md follow-ups).
//
// Every Record* method takes a blockchain_currencies id. The *In variants join
// the caller's transaction and context; the plain variants open their own.
// A replayed reference returns nil and writes nothing.
type LedgerService struct {
	accountRepo repository.AccountRepository
	journal     *ledger.Service
	assetsOf    AssetResolver
}

// Assets is what a blockchain_currencies row maps to: the asset itself and the chain's native asset gas is paid in.
// Codes are chain-qualified (USDC.BASE, ETH.BASE) because custody is per chain. Native is empty when the
// chain has no native row; that only fails a journal that actually books gas.
type Assets struct {
	Asset  string
	Native string
}

// AssetResolver maps a blockchain_currencies id to its ledger assets, reading through the posting
// transaction so it never needs a second connection. It must fail rather than guess.
type AssetResolver func(tx *gorm.DB, blockchainCurrencyID uint) (Assets, error)

// CurrencyCodeResolver maps a currencies id to its code (used by reconciliation of the chain-agnostic stored balance).
type CurrencyCodeResolver func(currencyID uint) (string, error)

// Option configures a LedgerService without changing NewLedgerService's shape for existing callers.
type Option func(*LedgerService)

// WithJournal enables dual-write into internal/ledger.
func WithJournal(journal *ledger.Service, resolver AssetResolver) Option {
	return func(s *LedgerService) {
		s.journal = journal
		s.assetsOf = resolver
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

// legacyAccount maps a legacy entry code to a ledger account; owner ids are the legacy codes so every old row maps to one new account.
func legacyAccount(code, asset string, kind ledger.AccountKind) ledger.AccountKey {
	return ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: code, Asset: asset, Kind: kind}
}

func line(code, asset string, kind ledger.AccountKind, amount decimal.Decimal) ledger.Line {
	return ledger.Line{Account: legacyAccount(code, asset, kind), Amount: amount}
}

// gasLines books gas in the chain's native asset: an expense against the platform's native holdings.
func gasLines(expenseCode string, a Assets, gas decimal.Decimal) ([]ledger.Line, error) {
	if gas.IsZero() {
		return nil, nil
	}
	if a.Native == "" {
		return nil, fmt.Errorf("ledger: %s has no native asset to book gas in; seed the chain's native currency row", a.Asset)
	}
	return []ledger.Line{
		line(expenseCode, a.Native, ledger.KindExpense, gas),
		line("crypto_assets", a.Native, ledger.KindAsset, gas.Neg()),
	}, nil
}

func newJournal(kind ledger.JournalKind, reference string, refID, blockchainCurrencyID uint, lines []ledger.Line, gas []ledger.Line, gasErr error) (ledger.Journal, error) {
	if gasErr != nil {
		return ledger.Journal{}, gasErr
	}
	id := strconv.FormatUint(uint64(refID), 10)
	return ledger.Journal{
		Kind:           kind,
		Reference:      ledger.Reference{Type: reference, ID: id},
		IdempotencyKey: "payminto:" + reference + ":" + id,
		Metadata:       map[string]any{"blockchain_currency_id": blockchainCurrencyID},
		Lines:          append(lines, gas...),
	}, nil
}

// txDB is implemented by repositories that can expose the handle the plain Record* methods open their own transaction on.
type txDB interface{ DB() *gorm.DB }

// InTransaction runs fn in one transaction on the ledger's database so callers can post alongside their own writes.
func (s *LedgerService) InTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if s.journal != nil {
		return s.journal.Transaction(ctx, fn)
	}
	if d, ok := s.accountRepo.(txDB); ok {
		return d.DB().WithContext(ctx).Transaction(fn)
	}
	return fmt.Errorf("ledger: no database handle to open a transaction")
}

// record writes the journal and then the legacy entries inside tx; with a nil tx it opens its own transaction.
// A replayed journal key skips the legacy write too, so retries never duplicate rows.
func (s *LedgerService) record(ctx context.Context, tx *gorm.DB, blockchainCurrencyID uint, entries repository.LedgerEntries, build func(Assets) (ledger.Journal, error)) error {
	if err := entries.Validate(); err != nil {
		return fmt.Errorf("unbalanced ledger entries: %w", err)
	}
	if s.journal == nil {
		if tx == nil {
			return s.accountRepo.CreateLedgerEntries(entries)
		}
		repo, err := s.boundRepo(tx)
		if err != nil {
			return err
		}
		return repo.CreateLedgerEntries(entries)
	}
	post := func(tx *gorm.DB) error {
		assets, err := s.assetsOf(tx, blockchainCurrencyID)
		if err != nil {
			return fmt.Errorf("ledger: resolve assets for blockchain currency %d: %w", blockchainCurrencyID, err)
		}
		j, err := build(assets)
		if err != nil {
			return err
		}
		receipt, err := s.journal.PostIn(ctx, tx, j)
		if err != nil {
			return err
		}
		if receipt.Replayed {
			return nil
		}
		repo, err := s.boundRepo(tx)
		if err != nil {
			return err
		}
		return repo.CreateLedgerEntries(entries)
	}
	if tx == nil {
		return s.journal.Transaction(ctx, post)
	}
	return post(tx)
}

// boundRepo refuses a repository that cannot join tx: a legacy write outside the journal transaction is a partial write.
func (s *LedgerService) boundRepo(tx *gorm.DB) (repository.AccountRepository, error) {
	binder, ok := s.accountRepo.(repository.TxBinder)
	if !ok {
		return nil, fmt.Errorf("ledger: account repository %T cannot join the transaction", s.accountRepo)
	}
	return binder.WithTx(tx), nil
}

// ReconcileStoredBalances compares each legacy accounts.balance (per member and currency, chain-agnostic)
// with the sum of that member's ledger liabilities in every chain instance of the currency, and reports drift.
// It never writes; nothing is corrected silently.
func (s *LedgerService) ReconcileStoredBalances(ctx context.Context, accounts []models.Account, codeOf CurrencyCodeResolver) ([]ledger.Drift, error) {
	if s.journal == nil {
		return nil, fmt.Errorf("ledger: reconcile needs a journal")
	}
	var drifts []ledger.Drift
	for _, a := range accounts {
		code, err := codeOf(a.CurrencyID)
		if err != nil {
			return nil, fmt.Errorf("ledger: resolve currency %d: %w", a.CurrencyID, err)
		}
		owner := strconv.FormatUint(uint64(a.MemberID), 10)
		balances, err := s.journal.AccountBalances(ctx, ledger.OwnerMember, owner)
		if err != nil {
			return nil, err
		}
		derived := decimal.Zero
		for _, b := range balances {
			if b.Account.Kind == ledger.KindLiability && ledger.CurrencyOfAsset(b.Account.Asset) == code {
				derived = derived.Add(b.Natural)
			}
		}
		if !derived.Equal(a.Balance) {
			key := ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: owner, Asset: code, Kind: ledger.KindLiability}
			drifts = append(drifts, ledger.Drift{Account: key, Stored: a.Balance, Derived: derived})
		}
	}
	return drifts, nil
}

// RecordPaymentDeposit records a confirmed deposit arriving into Payminto custody.
//
//	Debit: assets/crypto_assets    (coins we now hold)
//	Credit: liabilities/merchant_balance (what we owe the merchant)
func (s *LedgerService) RecordPaymentDeposit(paymentID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	return s.RecordPaymentDepositIn(context.Background(), nil, paymentID, blockchainCurrencyID, amount)
}

// RecordPaymentDepositIn is RecordPaymentDeposit inside the caller's transaction and context.
func (s *LedgerService) RecordPaymentDepositIn(ctx context.Context, tx *gorm.DB, paymentID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	refID := paymentID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "payment",
		}},
		Liabilities: []models.Liability{{
			Code:        "merchant_balance",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "payment",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		return newJournal(ledger.KindPayment, "payment", paymentID, blockchainCurrencyID, []ledger.Line{
			line("crypto_assets", a.Asset, ledger.KindAsset, amount),
			line("merchant_balance", a.Asset, ledger.KindLiability, amount.Neg()),
		}, nil, nil)
	})
}

// RecordDepositIn is RecordPaymentDepositIn keyed by the deposit rather than the payment, so a
// payment filled by several deposits gets one journal per deposit instead of an idempotency conflict.
func (s *LedgerService) RecordDepositIn(ctx context.Context, tx *gorm.DB, depositID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	refID := depositID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "deposit",
		}},
		Liabilities: []models.Liability{{
			Code:        "merchant_balance",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "deposit",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		return newJournal(ledger.KindPayment, "deposit", depositID, blockchainCurrencyID, []ledger.Line{
			line("crypto_assets", a.Asset, ledger.KindAsset, amount),
			line("merchant_balance", a.Asset, ledger.KindLiability, amount.Neg()),
		}, nil, nil)
	})
}

// RecordSweep records a SmartSweep batch moving coins from deposit addresses
// to cold storage.
//
//	Assets stay in place (coins still ours, now in cold wallet).
//	Expense entry records the gas cost.
//	To keep entries balanced: debit expense for gas, credit asset for gas consumed.
func (s *LedgerService) RecordSweep(sweepID uint, blockchainCurrencyID uint, amount, gasCost decimal.Decimal) error {
	return s.RecordSweepIn(context.Background(), nil, sweepID, blockchainCurrencyID, amount, gasCost)
}

// RecordSweepIn is RecordSweep inside the caller's transaction and context.
func (s *LedgerService) RecordSweepIn(ctx context.Context, tx *gorm.DB, sweepID uint, blockchainCurrencyID uint, amount, gasCost decimal.Decimal) error {
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
				CurrencyID:  blockchainCurrencyID,
				ReferenceID: &refID,
				Reference:   "sweep",
			},
			{
				// Decrease deposit-side asset balance
				Code:        "crypto_assets",
				Debit:       decimal.Zero,
				Credit:      amount.Add(gasCost),
				CurrencyID:  blockchainCurrencyID,
				ReferenceID: &refID,
				Reference:   "sweep",
			},
		},
		Expenses: []models.Expense{{
			// Gas cost as an operational expense
			Code:        "sweep_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "sweep",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		gas, err := gasLines("sweep_gas", a, gasCost)
		return newJournal(ledger.KindTransfer, "sweep", sweepID, blockchainCurrencyID, []ledger.Line{
			line("cold_wallet_assets", a.Asset, ledger.KindAsset, amount),
			line("crypto_assets", a.Asset, ledger.KindAsset, amount.Neg()),
		}, gas, err)
	})
}

// RecordWithdrawal records a merchant withdrawal leaving the platform.
//
//	Liability decreases (we owe less to the merchant).
//	Asset decreases (coins left our custody).
//	Expense records the gas cost.
func (s *LedgerService) RecordWithdrawal(withdrawalID uint, blockchainCurrencyID uint, amount, gasCost decimal.Decimal) error {
	return s.RecordWithdrawalIn(context.Background(), nil, withdrawalID, blockchainCurrencyID, amount, gasCost)
}

// RecordWithdrawalIn is RecordWithdrawal inside the caller's transaction and context.
func (s *LedgerService) RecordWithdrawalIn(ctx context.Context, tx *gorm.DB, withdrawalID uint, blockchainCurrencyID uint, amount, gasCost decimal.Decimal) error {
	refID := withdrawalID
	total := amount.Add(gasCost)
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      total,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
		Liabilities: []models.Liability{{
			Code:        "merchant_balance",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
		Expenses: []models.Expense{{
			Code:        "withdrawal_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "withdrawal",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		gas, err := gasLines("withdrawal_gas", a, gasCost)
		return newJournal(ledger.KindSettlement, "withdrawal", withdrawalID, blockchainCurrencyID, []ledger.Line{
			line("merchant_balance", a.Asset, ledger.KindLiability, amount),
			line("crypto_assets", a.Asset, ledger.KindAsset, amount.Neg()),
		}, gas, err)
	})
}

// RecordGasFee records a standalone gas-fee transfer (e.g. funding a deposit
// address with ETH before sweeping an ERC-20).
//
//	Asset decreases by gas cost.
//	Expense increases by gas cost.
func (s *LedgerService) RecordGasFee(txID uint, blockchainCurrencyID uint, gasCost decimal.Decimal) error {
	return s.RecordGasFeeIn(context.Background(), nil, txID, blockchainCurrencyID, gasCost)
}

// RecordGasFeeIn is RecordGasFee inside the caller's transaction and context.
func (s *LedgerService) RecordGasFeeIn(ctx context.Context, tx *gorm.DB, txID uint, blockchainCurrencyID uint, gasCost decimal.Decimal) error {
	refID := txID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      gasCost,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "gas_fee",
		}},
		Expenses: []models.Expense{{
			Code:        "gas_fee",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "gas_fee",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		gas, err := gasLines("gas_fee", a, gasCost)
		return newJournal(ledger.KindFee, "gas_fee", txID, blockchainCurrencyID, nil, gas, err)
	})
}

// RecordDuplicateDeposit records the second occurrence of a deposit with the
// same tx_hash+address. The funds are unclaimed by the merchant (no
// PaymentRequest), so we book them as platform revenue.
//
//	Asset increases (coins we now hold).
//	Revenue increases (windfall income — will be refunded or absorbed per policy).
func (s *LedgerService) RecordDuplicateDeposit(depositID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	return s.RecordDuplicateDepositIn(context.Background(), nil, depositID, blockchainCurrencyID, amount)
}

// RecordDuplicateDepositIn is RecordDuplicateDeposit inside the caller's transaction and context.
func (s *LedgerService) RecordDuplicateDepositIn(ctx context.Context, tx *gorm.DB, depositID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	refID := depositID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "duplicate_deposit",
		}},
		Revenues: []models.Revenue{{
			Code:        "unclaimed_deposit",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "duplicate_deposit",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		return newJournal(ledger.KindAdjustment, "duplicate_deposit", depositID, blockchainCurrencyID, []ledger.Line{
			line("crypto_assets", a.Asset, ledger.KindAsset, amount),
			line("unclaimed_deposit", a.Asset, ledger.KindIncome, amount.Neg()),
		}, nil, nil)
	})
}

// RecordReferralPayout records a referral reward being paid out to a member.
// Mirrors RecordWithdrawal but uses referral_reward as the reference type.
//
//	Liability decreases (platform owes the member their reward).
//	Asset decreases (coins leave the platform).
func (s *LedgerService) RecordReferralPayout(rewardID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	return s.RecordReferralPayoutIn(context.Background(), nil, rewardID, blockchainCurrencyID, amount)
}

// RecordReferralPayoutIn is RecordReferralPayout inside the caller's transaction and context.
func (s *LedgerService) RecordReferralPayoutIn(ctx context.Context, tx *gorm.DB, rewardID uint, blockchainCurrencyID uint, amount decimal.Decimal) error {
	refID := rewardID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      amount,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "referral_reward",
		}},
		Liabilities: []models.Liability{{
			Code:        "referral_rewards_payable",
			Debit:       amount,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "referral_reward",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		return newJournal(ledger.KindSettlement, "referral_reward", rewardID, blockchainCurrencyID, []ledger.Line{
			line("referral_rewards_payable", a.Asset, ledger.KindLiability, amount),
			line("crypto_assets", a.Asset, ledger.KindAsset, amount.Neg()),
		}, nil, nil)
	})
}

// RecordAddressDeployment records the gas spent deploying a smart contract
// deposit wallet (CREATE2 SCW deploy).
//
//	Asset decreases by gas cost.
//	Expense records the deployment gas.
func (s *LedgerService) RecordAddressDeployment(deploymentID uint, blockchainCurrencyID uint, gasCost decimal.Decimal) error {
	return s.RecordAddressDeploymentIn(context.Background(), nil, deploymentID, blockchainCurrencyID, gasCost)
}

// RecordAddressDeploymentIn is RecordAddressDeployment inside the caller's transaction and context.
func (s *LedgerService) RecordAddressDeploymentIn(ctx context.Context, tx *gorm.DB, deploymentID uint, blockchainCurrencyID uint, gasCost decimal.Decimal) error {
	refID := deploymentID
	entries := repository.LedgerEntries{
		Assets: []models.Asset{{
			Code:        "crypto_assets",
			Debit:       decimal.Zero,
			Credit:      gasCost,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "address_deployment",
		}},
		Expenses: []models.Expense{{
			Code:        "deployment_gas",
			Debit:       gasCost,
			Credit:      decimal.Zero,
			CurrencyID:  blockchainCurrencyID,
			ReferenceID: &refID,
			Reference:   "address_deployment",
		}},
	}
	return s.record(ctx, tx, blockchainCurrencyID, entries, func(a Assets) (ledger.Journal, error) {
		gas, err := gasLines("deployment_gas", a, gasCost)
		return newJournal(ledger.KindFee, "address_deployment", deploymentID, blockchainCurrencyID, nil, gas, err)
	})
}
