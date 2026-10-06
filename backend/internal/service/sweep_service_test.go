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

func newSweepTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.Member{},
		&models.Account{},
		&models.AccountReward{},
		&models.Asset{},
		&models.Liability{},
		&models.Revenue{},
		&models.Expense{},
		&models.Sweep{},
		&models.SweepTransaction{},
		&models.BlockchainCurrency{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedBlockchain(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	bf := models.BlockchainFamily{Name: "EVM", Code: "EVM"}
	if err := db.Create(&bf).Error; err != nil {
		t.Fatalf("seed BlockchainFamily: %v", err)
	}
	bc := models.Blockchain{
		Code: "ETH", Name: "Ethereum", BlockchainFamilyID: bf.ID, MinConfirmations: 12,
	}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatalf("seed Blockchain: %v", err)
	}
	return bc.ID
}

func newSweepService(t *testing.T, db *gorm.DB) *SweepService {
	t.Helper()
	sweepRepo := repository.NewSweepRepository(db)
	sweepTxRepo := repository.NewSweepTransactionRepository(db)
	blockchainRepo := repository.NewBlockchainRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerSvc := NewLedgerService(accountRepo)
	return NewSweepService(db, sweepRepo, sweepTxRepo, blockchainRepo, ledgerSvc)
}

func TestSweepService_CreateAndGet(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	sweep, err := svc.CreateSweep(bcID)
	if err != nil {
		t.Fatalf("CreateSweep: %v", err)
	}
	if sweep.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if sweep.Status != SweepStatusPending {
		t.Errorf("status: got %s want %s", sweep.Status, SweepStatusPending)
	}

	got, err := svc.GetByID(sweep.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != sweep.ID {
		t.Errorf("id mismatch: got %d want %d", got.ID, sweep.ID)
	}
}

func TestSweepService_ListByStatus(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	// Create 2 pending, 1 completed.
	_, _ = svc.CreateSweep(bcID)
	_, _ = svc.CreateSweep(bcID)
	s3, _ := svc.CreateSweep(bcID)
	_ = svc.UpdateStatus(s3.ID, SweepStatusCompleted)

	pending, err := svc.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending sweeps, got %d", len(pending))
	}
}

func TestSweepService_MarkFailed(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	sweep, _ := svc.CreateSweep(bcID)
	if err := svc.MarkFailed(sweep.ID); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	got, _ := svc.GetByID(sweep.ID)
	if got.Status != SweepStatusFailed {
		t.Errorf("status: got %s want %s", got.Status, SweepStatusFailed)
	}
}

func TestSweepService_MarkStale(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	sweep, _ := svc.CreateSweep(bcID)
	if err := svc.MarkStale(sweep.ID); err != nil {
		t.Fatalf("MarkStale: %v", err)
	}

	got, _ := svc.GetByID(sweep.ID)
	if got.Status != SweepStatusStale {
		t.Errorf("status: got %s want %s", got.Status, SweepStatusStale)
	}
}

func TestSweepService_MarkCompleted(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	sweep, _ := svc.CreateSweep(bcID)
	amount := decimal.NewFromFloat(1.0)
	gas := decimal.NewFromFloat(0.001)
	if err := svc.MarkCompleted(sweep.ID, amount, gas, 1); err != nil {
		t.Fatalf("MarkCompleted: %v", err)
	}

	got, _ := svc.GetByID(sweep.ID)
	if got.Status != SweepStatusCompleted {
		t.Errorf("status: got %s want %s", got.Status, SweepStatusCompleted)
	}
	if !got.TotalAmount.Equal(amount) {
		t.Errorf("total amount: got %s want %s", got.TotalAmount, amount)
	}
}

// TestSweepService_MarkCompleted_Idempotent verifies that a second call to
// MarkCompleted for an already-completed sweep returns nil without writing a
// duplicate ledger entry (C3 regression).
func TestSweepService_MarkCompleted_Idempotent(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)
	accountRepo := repository.NewAccountRepository(db)

	sweep, _ := svc.CreateSweep(bcID)
	amount := decimal.NewFromFloat(1.0)
	gas := decimal.NewFromFloat(0.001)

	// First call must succeed.
	if err := svc.MarkCompleted(sweep.ID, amount, gas, 1); err != nil {
		t.Fatalf("first MarkCompleted: %v", err)
	}

	// Count ledger asset rows before the second call.
	var countBefore int64
	db.Model(&models.Asset{}).Count(&countBefore)

	// Second call must also return nil (idempotent).
	if err := svc.MarkCompleted(sweep.ID, amount, gas, 1); err != nil {
		t.Errorf("second MarkCompleted: expected nil, got: %v", err)
	}

	// Ledger entry count must not have increased — no second write.
	var countAfter int64
	db.Model(&models.Asset{}).Count(&countAfter)
	if countAfter != countBefore {
		t.Errorf("ledger rows: expected no new entries on second call, got %d before and %d after",
			countBefore, countAfter)
	}

	// Suppress unused import warning — accountRepo is used to verify indirectly
	// via the db queries above; keep the explicit reference for clarity.
	_ = accountRepo
}

// TestSweepService_MarkCompleted_LedgerFailureReturnsError verifies that when
// the ledger write fails, MarkCompleted returns an error (C3 regression: the
// original code silently swallowed ledger errors). The sweep status IS committed
// (sequential, not wrapped in one transaction) — a reconciliation job handles
// the rare window where sweep=completed but ledger is missing.
//
// We induce the ledger failure by dropping the assets table mid-test so that
// the INSERT into assets fails.
func TestSweepService_MarkCompleted_LedgerFailureReturnsError(t *testing.T) {
	db := newSweepTestDB(t)
	bcID := seedBlockchain(t, db)
	svc := newSweepService(t, db)

	sweep, _ := svc.CreateSweep(bcID)

	// Drop the assets table to force the ledger write to fail.
	if err := db.Exec("DROP TABLE IF EXISTS assets").Error; err != nil {
		t.Fatalf("drop assets: %v", err)
	}

	amount := decimal.NewFromFloat(1.0)
	gas := decimal.NewFromFloat(0.001)

	err := svc.MarkCompleted(sweep.ID, amount, gas, 1)
	if err == nil {
		t.Fatal("expected error when ledger write fails, got nil — C3: ledger failure must be surfaced")
	}
}
