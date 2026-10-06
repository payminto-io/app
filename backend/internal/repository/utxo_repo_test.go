package repository

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newUTXOTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.UTXO{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

func seedUTXO(t *testing.T, db *gorm.DB) *models.UTXO {
	t.Helper()
	u := &models.UTXO{
		TxID:    "abc123",
		Vout:    0,
		Address: "bc1qtest",
		Spent:   false,
	}
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("seed UTXO: %v", err)
	}
	return u
}

// TestUTXORepo_MarkSpent_Idempotent_ReturnsSentinel verifies that the first call
// to MarkSpent succeeds and a second call for the same UTXO returns
// ErrUTXOAlreadySpent (C2 regression test).
func TestUTXORepo_MarkSpent_Idempotent_ReturnsSentinel(t *testing.T) {
	db := newUTXOTestDB(t)
	u := seedUTXO(t, db)
	repo := NewUTXORepository(db)

	// First call must succeed.
	if err := repo.MarkSpent(u.ID, "tx_first"); err != nil {
		t.Fatalf("first MarkSpent: unexpected error: %v", err)
	}

	// Second call for the same UTXO must return ErrUTXOAlreadySpent.
	err := repo.MarkSpent(u.ID, "tx_second")
	if !errors.Is(err, ErrUTXOAlreadySpent) {
		t.Errorf("second MarkSpent: got %v, want ErrUTXOAlreadySpent", err)
	}
}

// TestUTXORepo_MarkSpent_UpdatesFields verifies that a successful MarkSpent
// persists both spent=true and spent_tx_id.
func TestUTXORepo_MarkSpent_UpdatesFields(t *testing.T) {
	db := newUTXOTestDB(t)
	u := seedUTXO(t, db)
	repo := NewUTXORepository(db)

	if err := repo.MarkSpent(u.ID, "tx_abc"); err != nil {
		t.Fatalf("MarkSpent: %v", err)
	}

	got, err := repo.GetByID(u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.Spent {
		t.Error("expected Spent=true after MarkSpent")
	}
	if got.SpentTxID == nil || *got.SpentTxID != "tx_abc" {
		var got2 string
		if got.SpentTxID != nil {
			got2 = *got.SpentTxID
		}
		t.Errorf("SpentTxID: got %q want %q", got2, "tx_abc")
	}
}
