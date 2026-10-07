package service

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// fakeChecker returns a fixed confirmation count for any tx.
type fakeChecker struct {
	confs uint64
	err   error
}

func (f fakeChecker) Confirmations(context.Context, string, string) (uint64, error) {
	return f.confs, f.err
}

func setupConfirmer(t *testing.T, checker confirmationChecker) (*EVMSweepConfirmer, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.SweepTransaction{}, &models.Sweep{}, &models.BlockchainCurrency{},
		&models.Blockchain{}, &models.Account{},
		&models.Asset{}, &models.Liability{}, &models.Revenue{}, &models.Expense{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	stRepo := repository.NewSweepTransactionRepository(db)
	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	ledger := NewLedgerService(repository.NewAccountRepository(db))
	sweepSvc := NewSweepService(db, repository.NewSweepRepository(db), stRepo, repository.NewBlockchainRepository(db), ledger)
	c := NewEVMSweepConfirmer(stRepo, bcRepo, sweepSvc, checker)
	return c, db
}

// seedSweepTx creates a Blockchain (12 confs), BlockchainCurrency, Sweep, and a
// broadcast SweepTransaction; returns the sweep tx ID.
func seedSweepTx(t *testing.T, db *gorm.DB, chainCode string) uint {
	t.Helper()
	chain := models.Blockchain{Code: chainCode, Name: chainCode, MinConfirmations: 12, Status: "active"}
	if err := db.Create(&chain).Error; err != nil {
		t.Fatalf("create chain: %v", err)
	}
	bc := models.BlockchainCurrency{CurrencyCode: chainCode, BlockchainCode: chainCode, BlockchainID: chain.ID}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatalf("create bc: %v", err)
	}
	sweep := models.Sweep{Status: SweepStatusPending, BlockchainID: chain.ID}
	if err := db.Create(&sweep).Error; err != nil {
		t.Fatalf("create sweep: %v", err)
	}
	st := models.SweepTransaction{
		TxHash: "0xsweeptx", Amount: decimal.NewFromFloat(1), GasFee: decimal.NewFromFloat(0.001),
		FromAddress: "0xfrom", ToAddress: "0xcold", Status: SweepTxStatusBroadcast,
		SweepID: sweep.ID, BlockchainCurrencyID: bc.ID,
	}
	if err := db.Create(&st).Error; err != nil {
		t.Fatalf("create sweep tx: %v", err)
	}
	return st.ID
}

func TestSweepConfirmer_EnoughConfs_MarksConfirmed(t *testing.T) {
	c, db := setupConfirmer(t, fakeChecker{confs: 12})
	stID := seedSweepTx(t, db, "ETH")

	n, err := c.TrackConfirmations(context.Background())
	if err != nil {
		t.Fatalf("TrackConfirmations: %v", err)
	}
	if n != 1 {
		t.Errorf("confirmed count = %d, want 1", n)
	}
	var st models.SweepTransaction
	db.First(&st, stID)
	if st.Status != SweepTxStatusConfirmed {
		t.Errorf("sweep tx status = %q, want confirmed", st.Status)
	}
	var sweep models.Sweep
	db.First(&sweep, st.SweepID)
	if sweep.Status != SweepStatusCompleted {
		t.Errorf("sweep batch status = %q, want completed", sweep.Status)
	}
}

func TestSweepConfirmer_PartialConfs_MarksConfirming(t *testing.T) {
	c, db := setupConfirmer(t, fakeChecker{confs: 3})
	stID := seedSweepTx(t, db, "ETH")

	n, _ := c.TrackConfirmations(context.Background())
	if n != 0 {
		t.Errorf("should not confirm with 3/12, n=%d", n)
	}
	var st models.SweepTransaction
	db.First(&st, stID)
	if st.Status != SweepTxStatusConfirming {
		t.Errorf("sweep tx status = %q, want confirming", st.Status)
	}
}

func TestSweepConfirmer_ZeroConfs_StaysBroadcast(t *testing.T) {
	c, db := setupConfirmer(t, fakeChecker{confs: 0})
	stID := seedSweepTx(t, db, "ETH")

	c.TrackConfirmations(context.Background())
	var st models.SweepTransaction
	db.First(&st, stID)
	if st.Status != SweepTxStatusBroadcast {
		t.Errorf("sweep tx status = %q, want still broadcast", st.Status)
	}
}

func TestSweepConfirmer_Idempotent(t *testing.T) {
	c, db := setupConfirmer(t, fakeChecker{confs: 30})
	seedSweepTx(t, db, "ETH")

	first, _ := c.TrackConfirmations(context.Background())
	if first != 1 {
		t.Fatalf("first round confirmed = %d, want 1", first)
	}
	// Second round: tx is now 'confirmed', not in broadcast/confirming list.
	second, _ := c.TrackConfirmations(context.Background())
	if second != 0 {
		t.Errorf("second round confirmed = %d, want 0 (idempotent)", second)
	}
}
