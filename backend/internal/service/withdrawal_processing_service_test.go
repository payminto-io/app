package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newProcessingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Withdrawal{},
		&models.Withdraw{},
		&models.Blockchain{},
		&models.BlockchainCurrency{},
		&models.ExternalPlatform{},
		&models.Account{},
		&models.Member{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

var pendingWithdrawalSeq int

func createPendingWithdrawal(t *testing.T, db *gorm.DB) *models.Withdrawal {
	t.Helper()
	pendingWithdrawalSeq++
	w := &models.Withdrawal{
		ReferenceID:          fmt.Sprintf("wd_test_%s_%d", t.Name(), pendingWithdrawalSeq),
		State:                models.WithdrawalStatePending,
		BlockchainCode:       "ETH",
		CurrencyCode:         "USDT",
		Amount:               decimal.NewFromInt(50),
		ToAddress:            "0xdeadbeef",
		MemberID:             1,
		ExternalPlatformID:   1,
		BlockchainCurrencyID: 0,
	}
	if err := db.Create(w).Error; err != nil {
		t.Fatalf("create withdrawal: %v", err)
	}
	return w
}

func newProcessingService(db *gorm.DB) *WithdrawalProcessingService {
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	withdrawRepo := repository.NewWithdrawRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerSvc := NewLedgerService(accountRepo)
	return NewWithdrawalProcessingService(withdrawalRepo, withdrawRepo, ledgerSvc, nil, nil, nil)
}

// TestWithdrawalProcessingService_Execute_Idempotent calls Execute twice on the
// same withdrawal. The second call must be a no-op because ClaimForProcessing
// will return rowsAffected==0 (the withdrawal is no longer in 'pending' state).
func TestWithdrawalProcessingService_Execute_Idempotent(t *testing.T) {
	db := newProcessingTestDB(t)
	svc := newProcessingService(db)

	w := createPendingWithdrawal(t, db)
	ctx := context.Background()

	// First call should succeed and advance the state machine.
	if err := svc.Execute(ctx, w); err != nil {
		t.Fatalf("Execute (first call): %v", err)
	}

	// Verify it reached processed state.
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	updated, err := withdrawalRepo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID after first execute: %v", err)
	}
	if updated.State != models.WithdrawalStateProcessed {
		t.Errorf("state after first execute = %q, want %q", updated.State, models.WithdrawalStateProcessed)
	}

	// Second call: withdrawal is in 'processed' state, not 'pending'.
	// ClaimForProcessing should return rowsAffected==0, Execute returns nil (no-op).
	if err := svc.Execute(ctx, w); err != nil {
		t.Errorf("Execute (second call, idempotent): expected nil, got %v", err)
	}

	// State must not have changed.
	stillUpdated, err := withdrawalRepo.GetByID(w.ID)
	if err != nil {
		t.Fatalf("GetByID after second execute: %v", err)
	}
	if stillUpdated.State != models.WithdrawalStateProcessed {
		t.Errorf("state changed after idempotent call: %q", stillUpdated.State)
	}
}

// TestWithdrawalProcessingService_ProcessPending_ContextCancellation ensures that
// ProcessPending returns ctx.Err() immediately when the context is cancelled between
// withdrawals (graceful shutdown path).
func TestWithdrawalProcessingService_ProcessPending_ContextCancellation(t *testing.T) {
	db := newProcessingTestDB(t)
	svc := newProcessingService(db)

	// Create two withdrawals.
	createPendingWithdrawal(t, db)
	createPendingWithdrawal(t, db)

	// Cancel immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.ProcessPending(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
