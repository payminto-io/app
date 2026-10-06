package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMissedDepositDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.MissedDeposit{},
		&models.Blockchain{},
		&models.BlockchainCurrency{},
		&models.Currency{},
		&models.BlockchainFamily{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newMissedDepositService(t *testing.T) (*MissedDepositService, *gorm.DB) {
	t.Helper()
	db := setupMissedDepositDB(t)
	return NewMissedDepositService(repository.NewMissedDepositRepository(db)), db
}

func TestMissedDepositService_Create_RequiresFields(t *testing.T) {
	svc, _ := newMissedDepositService(t)

	if err := svc.Create(nil); err == nil {
		t.Error("expected error for nil")
	}
	if err := svc.Create(&models.MissedDeposit{TxHash: "0xabc"}); err == nil {
		t.Error("expected error for missing blockchain id")
	}
	if err := svc.Create(&models.MissedDeposit{BlockchainID: 1}); err == nil {
		t.Error("expected error for missing tx hash")
	}
}

func TestMissedDepositService_Create_DefaultsStatus(t *testing.T) {
	svc, _ := newMissedDepositService(t)

	d := &models.MissedDeposit{
		TxHash:       "0xabc",
		BlockchainID: 1,
		ToAddress:    "0xdest",
		Amount:       decimal.NewFromFloat(0.5),
		Reason:       "no_match",
	}
	if err := svc.Create(d); err != nil {
		t.Fatal(err)
	}
	if d.Status != models.MissedDepositStatusPending {
		t.Errorf("expected default status pending, got %s", d.Status)
	}
}

func TestMissedDepositService_Resolve_HappyPath(t *testing.T) {
	svc, _ := newMissedDepositService(t)

	d := &models.MissedDeposit{
		TxHash: "0xabc", BlockchainID: 1, ToAddress: "0x", Amount: decimal.NewFromInt(1), Reason: "no_match",
	}
	_ = svc.Create(d)

	if err := svc.Resolve(d.ID, MissedDepositActionRefund, "duplicate", 42); err != nil {
		t.Fatal(err)
	}

	got, _ := svc.GetByID(d.ID)
	if got.Status != models.MissedDepositStatusRefunded {
		t.Errorf("expected refunded, got %s", got.Status)
	}
	if got.ResolvedBy == nil || *got.ResolvedBy != 42 {
		t.Errorf("expected resolvedBy=42, got %v", got.ResolvedBy)
	}
}

func TestMissedDepositService_Resolve_AlreadyResolved(t *testing.T) {
	svc, _ := newMissedDepositService(t)

	d := &models.MissedDeposit{
		TxHash: "0xabc", BlockchainID: 1, ToAddress: "0x", Amount: decimal.NewFromInt(1), Reason: "no_match",
	}
	_ = svc.Create(d)
	_ = svc.Resolve(d.ID, MissedDepositActionRefund, "", 1)

	err := svc.Resolve(d.ID, MissedDepositActionRefund, "", 1)
	if !errors.Is(err, ErrMissedDepositAlreadyResolved) {
		t.Errorf("expected ErrMissedDepositAlreadyResolved, got %v", err)
	}
}

func TestMissedDepositService_Resolve_UnknownAction(t *testing.T) {
	svc, _ := newMissedDepositService(t)
	d := &models.MissedDeposit{TxHash: "0xabc", BlockchainID: 1, ToAddress: "0x", Amount: decimal.NewFromInt(1), Reason: "x"}
	_ = svc.Create(d)

	if err := svc.Resolve(d.ID, "deleted-from-disk", "", 1); err == nil {
		t.Error("expected error for unknown action")
	}
}

func TestMissedDepositService_GetByID_NotFound(t *testing.T) {
	svc, _ := newMissedDepositService(t)
	if _, err := svc.GetByID(9999); !errors.Is(err, ErrMissedDepositNotFound) {
		t.Errorf("expected ErrMissedDepositNotFound, got %v", err)
	}
}
