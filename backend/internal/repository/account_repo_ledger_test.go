package repository

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// newLedgerTestDB creates an in-memory SQLite database with all ledger tables.
func newLedgerTestDB(t *testing.T) *gorm.DB {
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

// ─── LedgerEntries.Validate ───────────────────────────────────────────────────

func TestLedgerEntries_Validate_Balanced(t *testing.T) {
	amount := decimal.NewFromFloat(100.0)
	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "crypto_assets", Debit: amount, Credit: decimal.Zero, CurrencyID: 1,
		}},
		Liabilities: []models.Liability{{
			Code: "merchant_balance", Debit: decimal.Zero, Credit: amount, CurrencyID: 1,
		}},
	}
	if err := entries.Validate(); err != nil {
		t.Errorf("expected valid balanced entries, got: %v", err)
	}
}

func TestLedgerEntries_Validate_Unbalanced(t *testing.T) {
	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "crypto_assets", Debit: decimal.NewFromFloat(100), Credit: decimal.Zero, CurrencyID: 1,
		}},
		// Missing corresponding credit — should fail validation.
	}
	if err := entries.Validate(); err == nil {
		t.Error("expected error for unbalanced entries, got nil")
	}
}

func TestLedgerEntries_Validate_MultiTable(t *testing.T) {
	// Genuinely unbalanced entries: total debits ≠ total credits.
	// Asset: debit=50, credit=0  → contributes 50 debit, 0 credit
	// Expense: debit=10, credit=0 → contributes 10 debit, 0 credit
	// Liability: debit=0, credit=50 → contributes 0 debit, 50 credit
	// TotalDebit=60, TotalCredit=50 → imbalanced.
	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "a", Debit: decimal.NewFromFloat(50), Credit: decimal.Zero, CurrencyID: 1,
		}},
		Expenses: []models.Expense{{
			Code: "g", Debit: decimal.NewFromFloat(10), Credit: decimal.Zero, CurrencyID: 1,
		}},
		Liabilities: []models.Liability{{
			Code: "l", Debit: decimal.Zero, Credit: decimal.NewFromFloat(50), CurrencyID: 1,
		}},
	}
	if err := entries.Validate(); err == nil {
		t.Error("expected error for multi-table unbalanced entries")
	}
}

func TestLedgerEntries_Validate_Empty(t *testing.T) {
	// Empty entries are trivially balanced (0 == 0).
	entries := LedgerEntries{}
	if err := entries.Validate(); err != nil {
		t.Errorf("empty entries should be valid: %v", err)
	}
}

// ─── AccountRepository.CreateLedgerEntries ───────────────────────────────────

func TestAccountRepo_CreateLedgerEntries_Balanced(t *testing.T) {
	db := newLedgerTestDB(t)
	repo := NewAccountRepository(db)

	amount := decimal.NewFromFloat(250.0)
	refID := uint(42)
	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "crypto_assets", Debit: amount, Credit: decimal.Zero,
			CurrencyID: 1, ReferenceID: &refID, Reference: "payment",
		}},
		Liabilities: []models.Liability{{
			Code: "merchant_balance", Debit: decimal.Zero, Credit: amount,
			CurrencyID: 1, ReferenceID: &refID, Reference: "payment",
		}},
	}

	if err := repo.CreateLedgerEntries(entries); err != nil {
		t.Fatalf("CreateLedgerEntries: %v", err)
	}

	// Verify rows landed in both tables.
	var assetCount, liabilityCount int64
	db.Model(&models.Asset{}).Where("reference = ?", "payment").Count(&assetCount)
	db.Model(&models.Liability{}).Where("reference = ?", "payment").Count(&liabilityCount)

	if assetCount != 1 {
		t.Errorf("expected 1 asset row, got %d", assetCount)
	}
	if liabilityCount != 1 {
		t.Errorf("expected 1 liability row, got %d", liabilityCount)
	}
}

func TestAccountRepo_CreateLedgerEntries_Unbalanced_Rejected(t *testing.T) {
	db := newLedgerTestDB(t)
	repo := NewAccountRepository(db)

	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "crypto_assets", Debit: decimal.NewFromFloat(100), Credit: decimal.Zero, CurrencyID: 1,
		}},
		// No corresponding credit entry — must be rejected.
	}

	if err := repo.CreateLedgerEntries(entries); err == nil {
		t.Error("expected error for unbalanced entries, got nil")
	}

	// Ensure no partial rows were written (transaction was rolled back).
	var count int64
	db.Model(&models.Asset{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 asset rows after rejected insert, got %d", count)
	}
}

func TestAccountRepo_CreateLedgerEntries_AllFourTables(t *testing.T) {
	db := newLedgerTestDB(t)
	repo := NewAccountRepository(db)

	// Balanced across all 4 tables:
	// Asset row:       debit=100, credit=0
	// Liability row:   debit=0,   credit=90
	// Revenue row:     debit=0,   credit=10
	// Expense row:     debit=0,   credit=0   (not used here; use Revenue instead)
	//
	// Actually let's use all 4:
	// Asset:     debit=100, credit=0   → +100D
	// Liability: debit=0,   credit=90  → +90C
	// Revenue:   debit=0,   credit=5   → +5C
	// Expense:   debit=0,   credit=5   → +5C
	// Wait, that's 100D vs 100C (90+5+5). ✓
	refID := uint(7)
	entries := LedgerEntries{
		Assets: []models.Asset{{
			Code: "crypto_assets", Debit: decimal.NewFromFloat(100), Credit: decimal.Zero,
			CurrencyID: 1, ReferenceID: &refID, Reference: "sweep",
		}},
		Liabilities: []models.Liability{{
			Code: "merchant_balance", Debit: decimal.Zero, Credit: decimal.NewFromFloat(90),
			CurrencyID: 1, ReferenceID: &refID, Reference: "sweep",
		}},
		Revenues: []models.Revenue{{
			Code: "fee_income", Debit: decimal.Zero, Credit: decimal.NewFromFloat(5),
			CurrencyID: 1, ReferenceID: &refID, Reference: "sweep",
		}},
		Expenses: []models.Expense{{
			Code: "sweep_gas", Debit: decimal.Zero, Credit: decimal.NewFromFloat(5),
			CurrencyID: 1, ReferenceID: &refID, Reference: "sweep",
		}},
	}

	if err := repo.CreateLedgerEntries(entries); err != nil {
		t.Fatalf("CreateLedgerEntries all-4-tables: %v", err)
	}

	var assetRows, liabRows, revRows, expRows int64
	db.Model(&models.Asset{}).Where("reference = ?", "sweep").Count(&assetRows)
	db.Model(&models.Liability{}).Where("reference = ?", "sweep").Count(&liabRows)
	db.Model(&models.Revenue{}).Where("reference = ?", "sweep").Count(&revRows)
	db.Model(&models.Expense{}).Where("reference = ?", "sweep").Count(&expRows)

	if assetRows != 1 {
		t.Errorf("expected 1 asset row, got %d", assetRows)
	}
	if liabRows != 1 {
		t.Errorf("expected 1 liability row, got %d", liabRows)
	}
	if revRows != 1 {
		t.Errorf("expected 1 revenue row, got %d", revRows)
	}
	if expRows != 1 {
		t.Errorf("expected 1 expense row, got %d", expRows)
	}
}
