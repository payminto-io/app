package service

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAccountRewardTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
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
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newAccountRewardSvc(t *testing.T) (*AccountRewardService, *gorm.DB) {
	t.Helper()
	db := newAccountRewardTestDB(t)
	repo := repository.NewAccountRepository(db)
	return NewAccountRewardService(repo), db
}

func TestAccountRewardService_ProcessPending_EmptyDB(t *testing.T) {
	svc, _ := newAccountRewardSvc(t)

	// Should not error on empty database.
	if err := svc.ProcessPending(); err != nil {
		t.Fatalf("ProcessPending on empty DB: %v", err)
	}
}

func TestAccountRewardService_RetryFailed_NoOp(t *testing.T) {
	svc, _ := newAccountRewardSvc(t)

	if err := svc.RetryFailed(); err != nil {
		t.Fatalf("RetryFailed: %v", err)
	}
}

func TestAccountRewardService_GetOrCreateAccount(t *testing.T) {
	svc, db := newAccountRewardSvc(t)

	acc, err := svc.GetOrCreateAccount(10, 1)
	if err != nil {
		t.Fatalf("GetOrCreateAccount: %v", err)
	}
	if acc.MemberID != 10 {
		t.Errorf("member ID: got %d want 10", acc.MemberID)
	}

	// Second call should return same row.
	acc2, err := svc.GetOrCreateAccount(10, 1)
	if err != nil {
		t.Fatalf("second GetOrCreateAccount: %v", err)
	}
	if acc.ID != acc2.ID {
		t.Error("expected same account ID on second call")
	}

	var count int64
	db.Model(&models.Account{}).Where("member_id = ? AND currency_id = ?", 10, 1).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 account row, got %d", count)
	}
}

func TestAccountRewardService_GetBalance_NoRecord(t *testing.T) {
	svc, _ := newAccountRewardSvc(t)

	// No record exists — should return an error (not create silently).
	_, err := svc.GetBalance(context.Background(), 99, 1)
	if err == nil {
		t.Error("expected error for non-existent account reward")
	}
}

// TestAccountRewardService_ProcessPending_StubBehavior asserts that the
// Phase I-review-fixed ProcessPending is a true no-op stub: it must not
// touch DB state, must not credit balances, must not consume locked rows.
// The full fulfilment flow lands in Phase K alongside on-chain payout
// signing — this test guards against accidental re-introduction of the
// double-credit bug fixed in the Phase I review.
func TestAccountRewardService_ProcessPending_StubBehavior(t *testing.T) {
	svc, db := newAccountRewardSvc(t)

	// Pre-seed an AccountReward with locked > 0 plus the underlying Account.
	ar := &models.AccountReward{
		MemberID:   55,
		CurrencyID: 1,
		Balance:    decimal.Zero,
		Locked:     decimal.NewFromFloat(10),
	}
	if err := db.Create(ar).Error; err != nil {
		t.Fatalf("create account reward: %v", err)
	}
	acc := &models.Account{
		MemberID:   55,
		CurrencyID: 1,
		Balance:    decimal.Zero,
		Locked:     decimal.Zero,
	}
	if err := db.Create(acc).Error; err != nil {
		t.Fatalf("create account: %v", err)
	}

	if err := svc.ProcessPending(); err != nil {
		t.Fatalf("ProcessPending: %v", err)
	}

	// The stub MUST NOT touch the underlying Account.
	var updatedAcc models.Account
	if err := db.Where("member_id = ? AND currency_id = ?", 55, 1).First(&updatedAcc).Error; err != nil {
		t.Fatalf("re-read account: %v", err)
	}
	if !updatedAcc.Balance.Equal(decimal.Zero) {
		t.Errorf("ProcessPending must be a no-op; balance changed to %s", updatedAcc.Balance)
	}
	if !updatedAcc.Locked.Equal(decimal.Zero) {
		t.Errorf("ProcessPending must be a no-op; locked changed to %s", updatedAcc.Locked)
	}

	// And MUST NOT touch the AccountReward row either.
	var refreshedAR models.AccountReward
	if err := db.First(&refreshedAR, ar.ID).Error; err != nil {
		t.Fatalf("re-read account reward: %v", err)
	}
	if !refreshedAR.Locked.Equal(decimal.NewFromFloat(10)) {
		t.Errorf("ProcessPending must be a no-op; account_reward.locked changed to %s", refreshedAR.Locked)
	}
}
