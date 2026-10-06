package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newLedgerSvcTestDB creates an in-memory SQLite database for ledger service tests.
func newLedgerSvcTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.Currency{},
		&models.Account{},
		&models.AccountReward{},
		&models.Asset{},
		&models.Liability{},
		&models.Revenue{},
		&models.Expense{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// assertBalance verifies that the sum of all debits equals the sum of all credits
// across all four ledger tables in the database. This is the fundamental double-entry
// invariant: every recorded event must leave the books balanced.
func assertBalance(t *testing.T, db *gorm.DB) {
	t.Helper()

	var totalDebit, totalCredit decimal.Decimal

	var assets []models.Asset
	db.Find(&assets)
	for _, a := range assets {
		totalDebit = totalDebit.Add(a.Debit)
		totalCredit = totalCredit.Add(a.Credit)
	}

	var liabs []models.Liability
	db.Find(&liabs)
	for _, l := range liabs {
		totalDebit = totalDebit.Add(l.Debit)
		totalCredit = totalCredit.Add(l.Credit)
	}

	var revs []models.Revenue
	db.Find(&revs)
	for _, r := range revs {
		totalDebit = totalDebit.Add(r.Debit)
		totalCredit = totalCredit.Add(r.Credit)
	}

	var exps []models.Expense
	db.Find(&exps)
	for _, e := range exps {
		totalDebit = totalDebit.Add(e.Debit)
		totalCredit = totalCredit.Add(e.Credit)
	}

	if !totalDebit.Equal(totalCredit) {
		t.Errorf("ledger imbalance: total debits %s != total credits %s", totalDebit, totalCredit)
	}
}

func newLedgerService(t *testing.T) (*LedgerService, *gorm.DB) {
	t.Helper()
	db := newLedgerSvcTestDB(t)
	repo := repository.NewAccountRepository(db)
	svc := NewLedgerService(repo)
	return svc, db
}

func TestLedgerService_RecordPaymentDeposit(t *testing.T) {
	svc, db := newLedgerService(t)

	amount := decimal.NewFromFloat(1.5)
	if err := svc.RecordPaymentDeposit(101, 1, amount); err != nil {
		t.Fatalf("RecordPaymentDeposit: %v", err)
	}

	assertBalance(t, db)

	// Verify asset row
	var asset models.Asset
	if err := db.Where("reference = ?", "payment").First(&asset).Error; err != nil {
		t.Fatalf("asset row not found: %v", err)
	}
	if !asset.Debit.Equal(amount) {
		t.Errorf("asset debit: got %s want %s", asset.Debit, amount)
	}

	// Verify liability row
	var liab models.Liability
	if err := db.Where("reference = ?", "payment").First(&liab).Error; err != nil {
		t.Fatalf("liability row not found: %v", err)
	}
	if !liab.Credit.Equal(amount) {
		t.Errorf("liability credit: got %s want %s", liab.Credit, amount)
	}
}

func TestLedgerService_RecordSweep(t *testing.T) {
	svc, db := newLedgerService(t)

	amount := decimal.NewFromFloat(2.0)
	gas := decimal.NewFromFloat(0.01)
	if err := svc.RecordSweep(55, 1, amount, gas); err != nil {
		t.Fatalf("RecordSweep: %v", err)
	}

	assertBalance(t, db)

	// Expense row should exist for gas
	var exp models.Expense
	if err := db.Where("code = ?", "sweep_gas").First(&exp).Error; err != nil {
		t.Fatalf("expense row not found: %v", err)
	}
	if !exp.Debit.Equal(gas) {
		t.Errorf("expense debit: got %s want %s", exp.Debit, gas)
	}
}

func TestLedgerService_RecordWithdrawal(t *testing.T) {
	svc, db := newLedgerService(t)

	amount := decimal.NewFromFloat(5.0)
	gas := decimal.NewFromFloat(0.02)
	if err := svc.RecordWithdrawal(77, 1, amount, gas); err != nil {
		t.Fatalf("RecordWithdrawal: %v", err)
	}

	assertBalance(t, db)

	// Liability should be debited (merchant owes us less)
	var liab models.Liability
	if err := db.Where("reference = ?", "withdrawal").First(&liab).Error; err != nil {
		t.Fatalf("liability row not found: %v", err)
	}
	if !liab.Debit.Equal(amount) {
		t.Errorf("liability debit: got %s want %s", liab.Debit, amount)
	}

	// Asset should be credited (coins left)
	var asset models.Asset
	if err := db.Where("reference = ?", "withdrawal").First(&asset).Error; err != nil {
		t.Fatalf("asset row not found: %v", err)
	}
	expectedCredit := amount.Add(gas)
	if !asset.Credit.Equal(expectedCredit) {
		t.Errorf("asset credit: got %s want %s", asset.Credit, expectedCredit)
	}
}

func TestLedgerService_RecordGasFee(t *testing.T) {
	svc, db := newLedgerService(t)

	gas := decimal.NewFromFloat(0.005)
	if err := svc.RecordGasFee(99, 1, gas); err != nil {
		t.Fatalf("RecordGasFee: %v", err)
	}

	assertBalance(t, db)

	var exp models.Expense
	if err := db.Where("code = ?", "gas_fee").First(&exp).Error; err != nil {
		t.Fatalf("expense row not found: %v", err)
	}
	if !exp.Debit.Equal(gas) {
		t.Errorf("gas expense debit: got %s want %s", exp.Debit, gas)
	}
}

func TestLedgerService_RecordDuplicateDeposit(t *testing.T) {
	svc, db := newLedgerService(t)

	amount := decimal.NewFromFloat(0.3)
	if err := svc.RecordDuplicateDeposit(200, 1, amount); err != nil {
		t.Fatalf("RecordDuplicateDeposit: %v", err)
	}

	assertBalance(t, db)

	// Revenue credit should exist
	var rev models.Revenue
	if err := db.Where("code = ?", "unclaimed_deposit").First(&rev).Error; err != nil {
		t.Fatalf("revenue row not found: %v", err)
	}
	if !rev.Credit.Equal(amount) {
		t.Errorf("revenue credit: got %s want %s", rev.Credit, amount)
	}
}

func TestLedgerService_RecordAddressDeployment(t *testing.T) {
	svc, db := newLedgerService(t)

	gas := decimal.NewFromFloat(0.0025)
	if err := svc.RecordAddressDeployment(300, 1, gas); err != nil {
		t.Fatalf("RecordAddressDeployment: %v", err)
	}

	assertBalance(t, db)

	var exp models.Expense
	if err := db.Where("code = ?", "deployment_gas").First(&exp).Error; err != nil {
		t.Fatalf("expense row not found: %v", err)
	}
	if !exp.Debit.Equal(gas) {
		t.Errorf("deployment expense debit: got %s want %s", exp.Debit, gas)
	}
}

// TestLedgerService_MultipleEvents verifies the cumulative balance invariant
// holds after recording several distinct events in sequence.
func TestLedgerService_MultipleEvents(t *testing.T) {
	svc, db := newLedgerService(t)

	if err := svc.RecordPaymentDeposit(1, 1, decimal.NewFromFloat(10)); err != nil {
		t.Fatalf("RecordPaymentDeposit: %v", err)
	}
	if err := svc.RecordSweep(1, 1, decimal.NewFromFloat(9.5), decimal.NewFromFloat(0.01)); err != nil {
		t.Fatalf("RecordSweep: %v", err)
	}
	if err := svc.RecordGasFee(1, 1, decimal.NewFromFloat(0.005)); err != nil {
		t.Fatalf("RecordGasFee: %v", err)
	}

	// After all events the books must still balance.
	assertBalance(t, db)
}
