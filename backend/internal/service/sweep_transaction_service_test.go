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

func newSweepTxTestDB(t *testing.T) *gorm.DB {
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
		&models.BlockchainCurrency{},
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
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func newSweepTxService(t *testing.T, db *gorm.DB) *SweepTransactionService {
	t.Helper()
	sweepTxRepo := repository.NewSweepTransactionRepository(db)
	sweepRepo := repository.NewSweepRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerSvc := NewLedgerService(accountRepo)
	return NewSweepTransactionService(sweepTxRepo, sweepRepo, ledgerSvc)
}

// seedSweep creates a Sweep row to satisfy the SweepTransaction FK.
func seedSweep(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	bf := models.BlockchainFamily{Name: "EVM-tx", Code: "EVM-tx"}
	db.Create(&bf)
	bc := models.Blockchain{Code: "ETH-tx", Name: "Ethereum", BlockchainFamilyID: bf.ID}
	db.Create(&bc)
	sweep := models.Sweep{Status: SweepStatusPending, BlockchainID: bc.ID}
	if err := db.Create(&sweep).Error; err != nil {
		t.Fatalf("seed sweep: %v", err)
	}
	return sweep.ID
}

func TestSweepTransactionService_CreatePayload(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	amount := decimal.NewFromFloat(0.5)
	st, err := svc.CreateSweepTransactionPayload(sweepID, 1, "0xFrom", "0xTo", amount)
	if err != nil {
		t.Fatalf("CreateSweepTransactionPayload: %v", err)
	}
	if st.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if st.Status != SweepTxStatusPending {
		t.Errorf("status: got %s want %s", st.Status, SweepTxStatusPending)
	}
	if !st.Amount.Equal(amount) {
		t.Errorf("amount: got %s want %s", st.Amount, amount)
	}
}

func TestSweepTransactionService_EthAutoSweepTransaction(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	amount := decimal.NewFromFloat(1.0)
	gas := decimal.NewFromFloat(0.002)
	st, err := svc.EthAutoSweepTransaction(sweepID, 1, "0xDepositAddr", "0xColdWallet", amount, gas)
	if err != nil {
		t.Fatalf("EthAutoSweepTransaction: %v", err)
	}

	// Must produce a pending row — no broadcast yet.
	if st.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if st.Status != SweepTxStatusPending {
		t.Errorf("status: got %s want %s", st.Status, SweepTxStatusPending)
	}
	if st.TxHash != "" {
		t.Errorf("expected empty TxHash before Phase-K signing, got %s", st.TxHash)
	}

	// Confirm the row is persisted in the DB.
	var count int64
	db.Model(&models.SweepTransaction{}).Where("id = ?", st.ID).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 sweep_transaction row, got %d", count)
	}
}

func TestSweepTransactionService_EthAutoSweepTransaction_ZeroAmount(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	_, err := svc.EthAutoSweepTransaction(sweepID, 1, "0xFrom", "0xTo", decimal.Zero, decimal.Zero)
	if err == nil {
		t.Error("expected error for zero amount, got nil")
	}
}

func TestSweepTransactionService_ProcessERC20Sweep(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	tokenAmt := decimal.NewFromFloat(500.0) // 500 USDC
	gasAmt := decimal.NewFromFloat(0.005)   // ETH for gas
	st, err := svc.ProcessERC20Sweep(sweepID, 1, "0xDeposit", "0xCold", tokenAmt, gasAmt)
	if err != nil {
		t.Fatalf("ProcessERC20Sweep: %v", err)
	}

	if st.Status != SweepTxStatusPending {
		t.Errorf("status: got %s want %s", st.Status, SweepTxStatusPending)
	}
	if !st.Amount.Equal(tokenAmt) {
		t.Errorf("amount: got %s want %s", st.Amount, tokenAmt)
	}
}

func TestSweepTransactionService_ListPending(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	// Create 2 pending, then update one to broadcast.
	st1, _ := svc.CreateSweepTransactionPayload(sweepID, 1, "0xA", "0xZ", decimal.NewFromFloat(1))
	_, _ = svc.CreateSweepTransactionPayload(sweepID, 1, "0xB", "0xZ", decimal.NewFromFloat(2))
	_ = svc.UpdateStatus(st1.ID, SweepTxStatusBroadcast)

	pending, err := svc.ListPending()
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}
}

func TestSweepTransactionService_UpdateTxHash(t *testing.T) {
	db := newSweepTxTestDB(t)
	svc := newSweepTxService(t, db)
	sweepID := seedSweep(t, db)

	st, _ := svc.CreateSweepTransactionPayload(sweepID, 1, "0xA", "0xZ", decimal.NewFromFloat(1))
	if err := svc.UpdateTxHash(st.ID, "0xdeadbeef"); err != nil {
		t.Fatalf("UpdateTxHash: %v", err)
	}

	var got models.SweepTransaction
	db.First(&got, st.ID)
	if got.TxHash != "0xdeadbeef" {
		t.Errorf("TxHash: got %s want 0xdeadbeef", got.TxHash)
	}
}
