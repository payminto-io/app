package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var errInjected = errors.New("injected broadcast error")

// fakeNativeSweeper records calls and returns a canned result.
type fakeNativeSweeper struct {
	txHash string
	amount *big.Int
	gasFee *big.Int
	err    error
	calls  int
}

func (f *fakeNativeSweeper) SweepNative(_ context.Context, _, _, _ string, _ *big.Int) (string, *big.Int, *big.Int, error) {
	f.calls++
	return f.txHash, f.amount, f.gasFee, f.err
}

func setupEVMSweep(t *testing.T, sweeper nativeSweeper) (*EVMSweepService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Deposit{}, &models.BlockchainCurrency{},
		&models.Sweep{}, &models.SweepTransaction{}, &models.Account{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	depRepo := repository.NewDepositRepository(db)
	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	ledger := NewLedgerService(repository.NewAccountRepository(db))
	sweepSvc := NewSweepService(db, repository.NewSweepRepository(db), repository.NewSweepTransactionRepository(db), repository.NewBlockchainRepository(db), ledger)
	sweepTxSvc := NewSweepTransactionService(repository.NewSweepTransactionRepository(db), repository.NewSweepRepository(db), ledger)
	svc := NewEVMSweepService(depRepo, bcRepo, sweepSvc, sweepTxSvc, sweeper, "0xColdWallet", big.NewInt(0))
	return svc, db
}

// seedDeposit creates a BlockchainCurrency + a confirmed deposit and returns the deposit ID.
func seedDeposit(t *testing.T, db *gorm.DB, code, contract string) uint {
	t.Helper()
	bc := models.BlockchainCurrency{CurrencyCode: code, BlockchainCode: code, Address: contract, BlockchainID: 1}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatalf("create bc: %v", err)
	}
	d := models.Deposit{
		TxID: "0xdep", ToAddress: "0xDeposit", Amount: decimal.NewFromFloat(1),
		Status: models.DepositStatusConfirmed, BlockchainCurrencyID: bc.ID, MemberID: 1,
	}
	if err := db.Create(&d).Error; err != nil {
		t.Fatalf("create deposit: %v", err)
	}
	return d.ID
}

func TestEVMSweep_NativeDeposit_SweepsAndMarks(t *testing.T) {
	sweeper := &fakeNativeSweeper{txHash: "0xsweeptx", amount: big.NewInt(1_000_000_000_000_000_000), gasFee: big.NewInt(420000000000000)}
	svc, db := setupEVMSweep(t, sweeper)
	depID := seedDeposit(t, db, "ETH", "") // native (no contract)

	n, err := svc.SweepConfirmedNative(context.Background())
	if err != nil {
		t.Fatalf("SweepConfirmedNative: %v", err)
	}
	if n != 1 {
		t.Errorf("swept count = %d, want 1", n)
	}

	var d models.Deposit
	db.First(&d, depID)
	if d.Status != models.DepositStatusSwept {
		t.Errorf("deposit status = %q, want swept", d.Status)
	}
	var txs []models.SweepTransaction
	db.Find(&txs)
	if len(txs) != 1 || txs[0].TxHash != "0xsweeptx" || txs[0].Status != SweepTxStatusBroadcast {
		t.Errorf("expected 1 broadcast sweep tx with hash, got %+v", txs)
	}

	// Idempotent: a second run does nothing (deposit no longer confirmed).
	n2, _ := svc.SweepConfirmedNative(context.Background())
	if n2 != 0 {
		t.Errorf("second run swept = %d, want 0 (idempotent)", n2)
	}
}

func TestEVMSweep_ERC20Deposit_Skipped(t *testing.T) {
	sweeper := &fakeNativeSweeper{txHash: "0xshould-not-be-used"}
	svc, db := setupEVMSweep(t, sweeper)
	depID := seedDeposit(t, db, "ETH", "0xTokenContract") // has contract → ERC-20

	n, _ := svc.SweepConfirmedNative(context.Background())
	if n != 0 || sweeper.calls != 0 {
		t.Errorf("ERC-20 deposit must be skipped by native sweeper (n=%d calls=%d)", n, sweeper.calls)
	}
	var d models.Deposit
	db.First(&d, depID)
	if d.Status != models.DepositStatusConfirmed {
		t.Errorf("ERC-20 deposit should remain confirmed, got %q", d.Status)
	}
}

func TestEVMSweep_BroadcastFailure_RevertsClaim(t *testing.T) {
	// Broadcast errors → the claim must be released so the deposit is retried.
	sweeper := &fakeNativeSweeper{err: errInjected}
	svc, db := setupEVMSweep(t, sweeper)
	depID := seedDeposit(t, db, "ETH", "")

	n, _ := svc.SweepConfirmedNative(context.Background())
	if n != 0 {
		t.Errorf("failed broadcast should not count as swept, n=%d", n)
	}
	var d models.Deposit
	db.First(&d, depID)
	if d.Status != models.DepositStatusConfirmed {
		t.Errorf("deposit must revert to confirmed after broadcast failure, got %q", d.Status)
	}
	var txCount int64
	db.Model(&models.SweepTransaction{}).Count(&txCount)
	if txCount != 0 {
		t.Errorf("failed broadcast must not record a sweep tx, got %d", txCount)
	}
}

func TestEVMSweep_NonEVMDeposit_Skipped(t *testing.T) {
	sweeper := &fakeNativeSweeper{}
	svc, db := setupEVMSweep(t, sweeper)
	seedDeposit(t, db, "BTC", "")

	n, _ := svc.SweepConfirmedNative(context.Background())
	if n != 0 || sweeper.calls != 0 {
		t.Errorf("non-EVM deposit must be skipped (n=%d calls=%d)", n, sweeper.calls)
	}
}

func TestEVMSweep_DustDeposit_MarkedSweptNoTx(t *testing.T) {
	// Broadcaster returns empty hash (below dust) → mark swept, no sweep tx.
	sweeper := &fakeNativeSweeper{txHash: "", amount: nil}
	svc, db := setupEVMSweep(t, sweeper)
	depID := seedDeposit(t, db, "ETH", "")

	n, _ := svc.SweepConfirmedNative(context.Background())
	if n != 0 {
		t.Errorf("dust deposit should not count as swept tx, n=%d", n)
	}
	var d models.Deposit
	db.First(&d, depID)
	if d.Status != models.DepositStatusSwept {
		t.Errorf("dust deposit should be marked swept to avoid reprocessing, got %q", d.Status)
	}
	var txCount int64
	db.Model(&models.SweepTransaction{}).Count(&txCount)
	if txCount != 0 {
		t.Errorf("dust sweep must not create a sweep tx, got %d", txCount)
	}
}
