//go:build integration

package service

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// With one pool connection, a ledger post that opened its own transaction would wait forever for a second one.
func TestIntegration_RecordGasFundingNeedsOnlyTheCallerConnection(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	seedCurrencyCatalog(t, db)

	bcRepo := repository.NewBlockchainCurrencyRepository(db)
	ethBase, err := bcRepo.GetByBlockchainCodeAndCurrencyCode("BASE", "ETH")
	if err != nil {
		t.Fatal(err)
	}
	ledgerSvc := NewLedgerService(repository.NewAccountRepository(db), WithJournal(ledger.New(db), blockchainCurrencyAssetResolver()))
	svc := NewInternalBlockchainTransactionService(db, repository.NewInternalBlockchainTransactionRepository(db), ledgerSvc)

	done := make(chan error, 1)
	go func() {
		_, err := svc.RecordGasFunding(ethBase.ID, "0xfrom", "0xto", decimal.RequireFromString("0.1"), decimal.RequireFromString("0.0004"))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("RecordGasFunding did not finish with a single connection: the ledger post is not joining the caller transaction")
	}
	var journals int64
	db.Model(&ledger.JournalRow{}).Count(&journals)
	if journals != 1 {
		t.Fatalf("journals = %d, want 1", journals)
	}
}
