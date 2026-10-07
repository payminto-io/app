package repository

import (
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LedgerEntries is a balanced set of accounting entries produced by a single
// financial event (a deposit, a sweep, a withdrawal, etc.). Debits must equal
// credits across Asset + Liability + Revenue + Expense tables. The ledger
// service validates this before handing to the repo which persists everything
// inside a single database transaction.
type LedgerEntries struct {
	Assets      []models.Asset
	Liabilities []models.Liability
	Revenues    []models.Revenue
	Expenses    []models.Expense
}

// Validate asserts that sum(debits) == sum(credits) across all four tables.
// Uses decimal.Decimal throughout — never float64.
func (e LedgerEntries) Validate() error {
	var totalDebit, totalCredit decimal.Decimal

	for _, a := range e.Assets {
		totalDebit = totalDebit.Add(a.Debit)
		totalCredit = totalCredit.Add(a.Credit)
	}
	for _, l := range e.Liabilities {
		totalDebit = totalDebit.Add(l.Debit)
		totalCredit = totalCredit.Add(l.Credit)
	}
	for _, r := range e.Revenues {
		totalDebit = totalDebit.Add(r.Debit)
		totalCredit = totalCredit.Add(r.Credit)
	}
	for _, ex := range e.Expenses {
		totalDebit = totalDebit.Add(ex.Debit)
		totalCredit = totalCredit.Add(ex.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		return fmt.Errorf("ledger imbalance: total debits %s != total credits %s",
			totalDebit.String(), totalCredit.String())
	}
	return nil
}

// AccountRepository defines all persistence operations for Account, ledger
// entries, and related models.
type AccountRepository interface {
	// CreateLedgerEntries persists a balanced LedgerEntries inside a single
	// database transaction. If any insert fails the entire batch is rolled back.
	// This is the only supported path for writing ledger rows — direct Create
	// calls bypass the balance check enforced here.
	CreateLedgerEntries(entries LedgerEntries) error

	// GetOrCreateAccount returns the Account for (memberID, currencyID), creating
	// one with zero balances if it does not exist yet.
	GetOrCreateAccount(memberID, currencyID uint) (*models.Account, error)

	// UpdateAccountBalance atomically adjusts the balance for an account row.
	UpdateAccountBalance(memberID, currencyID uint, delta decimal.Decimal) error

	// GetAccountReward returns the AccountReward for (memberID, currencyID).
	GetAccountReward(memberID, currencyID uint) (*models.AccountReward, error)

	// ListPendingAccountRewards returns AccountReward rows with non-zero locked
	// amounts that are pending distribution.
	ListPendingAccountRewards() ([]models.AccountReward, error)
}

// TxBinder is implemented by repositories that can run inside a caller's transaction.
type TxBinder interface {
	WithTx(tx *gorm.DB) AccountRepository
}

// AccountRepositoryImpl is the GORM-backed implementation of AccountRepository.
type AccountRepositoryImpl struct {
	db *gorm.DB
}

// WithTx returns a copy bound to tx so ledger writes can share a transaction with the new ledger package.
func (r *AccountRepositoryImpl) WithTx(tx *gorm.DB) AccountRepository {
	return &AccountRepositoryImpl{db: tx}
}

// DB exposes the handle so the ledger service can open a transaction when no journal is configured.
func (r *AccountRepositoryImpl) DB() *gorm.DB { return r.db }

// NewAccountRepository constructs a new AccountRepository backed by the provided *gorm.DB.
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &AccountRepositoryImpl{db: db}
}

// CreateLedgerEntries persists a balanced LedgerEntries inside a single
// database transaction. If any insert fails the entire batch is rolled back.
func (r *AccountRepositoryImpl) CreateLedgerEntries(entries LedgerEntries) error {
	if err := entries.Validate(); err != nil {
		return fmt.Errorf("unbalanced ledger entries: %w", err)
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(entries.Assets) > 0 {
			if err := tx.Create(&entries.Assets).Error; err != nil {
				return err
			}
		}
		if len(entries.Liabilities) > 0 {
			if err := tx.Create(&entries.Liabilities).Error; err != nil {
				return err
			}
		}
		if len(entries.Revenues) > 0 {
			if err := tx.Create(&entries.Revenues).Error; err != nil {
				return err
			}
		}
		if len(entries.Expenses) > 0 {
			if err := tx.Create(&entries.Expenses).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetOrCreateAccount returns the Account for (memberID, currencyID), creating
// one with zero balances if it does not exist yet.
func (r *AccountRepositoryImpl) GetOrCreateAccount(memberID, currencyID uint) (*models.Account, error) {
	var acc models.Account
	err := r.db.
		Where("member_id = ? AND currency_id = ?", memberID, currencyID).
		First(&acc).Error
	if err == nil {
		return &acc, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	acc = models.Account{
		MemberID:   memberID,
		CurrencyID: currencyID,
		Balance:    decimal.Zero,
		Locked:     decimal.Zero,
	}
	if createErr := r.db.Create(&acc).Error; createErr != nil {
		return nil, createErr
	}
	return &acc, nil
}

// UpdateAccountBalance atomically adjusts the spendable balance for the account
// identified by (memberID, currencyID) using a SQL increment to avoid read-modify-write races.
func (r *AccountRepositoryImpl) UpdateAccountBalance(memberID, currencyID uint, delta decimal.Decimal) error {
	return r.db.Model(&models.Account{}).
		Where("member_id = ? AND currency_id = ?", memberID, currencyID).
		UpdateColumn("balance", gorm.Expr("balance + ?", delta)).Error
}

// GetAccountReward returns the AccountReward for (memberID, currencyID).
func (r *AccountRepositoryImpl) GetAccountReward(memberID, currencyID uint) (*models.AccountReward, error) {
	var ar models.AccountReward
	if err := r.db.
		Where("member_id = ? AND currency_id = ?", memberID, currencyID).
		First(&ar).Error; err != nil {
		return nil, err
	}
	return &ar, nil
}

// ListPendingAccountRewards returns AccountReward rows that have locked amounts
// waiting for distribution.
func (r *AccountRepositoryImpl) ListPendingAccountRewards() ([]models.AccountReward, error) {
	var rewards []models.AccountReward
	if err := r.db.
		Where("locked > 0").
		Find(&rewards).Error; err != nil {
		return nil, err
	}
	return rewards, nil
}
