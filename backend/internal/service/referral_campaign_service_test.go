package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newCampaignTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.ReferralCampaign{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newCampaignSvc(t *testing.T) (*ReferralCampaignService, *gorm.DB) {
	t.Helper()
	db := newCampaignTestDB(t)
	return NewReferralCampaignService(repository.NewReferralCampaignRepository(db)), db
}

func TestReferralCampaignService_Create(t *testing.T) {
	svc, _ := newCampaignSvc(t)

	c, err := svc.CreateCampaign(CreateCampaignInput{
		Project:      "payminto",
		Name:         "Q2 Referral",
		RewardType:   "fixed",
		RewardValue:  decimal.NewFromFloat(10),
		CurrencyCode: "USDT",
	})
	if err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if c.Status != "active" {
		t.Errorf("status: got %s want active", c.Status)
	}
}

func TestReferralCampaignService_Create_MissingName(t *testing.T) {
	svc, _ := newCampaignSvc(t)

	_, err := svc.CreateCampaign(CreateCampaignInput{Project: "payminto"})
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestReferralCampaignService_GetCampaign(t *testing.T) {
	svc, _ := newCampaignSvc(t)

	c, _ := svc.CreateCampaign(CreateCampaignInput{Project: "p", Name: "n"})
	got, err := svc.GetCampaign(c.ID)
	if err != nil {
		t.Fatalf("GetCampaign: %v", err)
	}
	if got.ID != c.ID {
		t.Errorf("ID mismatch: got %d want %d", got.ID, c.ID)
	}
}

func TestReferralCampaignService_GetCampaign_NotFound(t *testing.T) {
	svc, _ := newCampaignSvc(t)
	_, err := svc.GetCampaign(9999)
	if !errors.Is(err, ErrCampaignNotFound) {
		t.Errorf("expected ErrCampaignNotFound, got %v", err)
	}
}

func TestReferralCampaignService_ListCampaigns(t *testing.T) {
	svc, _ := newCampaignSvc(t)

	svc.CreateCampaign(CreateCampaignInput{Project: "p1", Name: "A"})
	svc.CreateCampaign(CreateCampaignInput{Project: "p1", Name: "B"})
	svc.CreateCampaign(CreateCampaignInput{Project: "p2", Name: "C"})

	list, err := svc.ListCampaigns("p1")
	if err != nil {
		t.Fatalf("ListCampaigns: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 campaigns for p1, got %d", len(list))
	}
}

func TestReferralCampaignService_Activate_Deactivate(t *testing.T) {
	svc, db := newCampaignSvc(t)

	c, _ := svc.CreateCampaign(CreateCampaignInput{Project: "p", Name: "X"})

	if err := svc.DeactivateCampaign(c.ID); err != nil {
		t.Fatalf("DeactivateCampaign: %v", err)
	}
	var updated models.ReferralCampaign
	db.First(&updated, c.ID)
	if updated.Status != "inactive" {
		t.Errorf("status after deactivate: got %s want inactive", updated.Status)
	}

	if err := svc.ActivateCampaign(c.ID); err != nil {
		t.Fatalf("ActivateCampaign: %v", err)
	}
	db.First(&updated, c.ID)
	if updated.Status != "active" {
		t.Errorf("status after activate: got %s want active", updated.Status)
	}
}

func TestReferralCampaignService_DeleteCampaign(t *testing.T) {
	svc, db := newCampaignSvc(t)

	c, _ := svc.CreateCampaign(CreateCampaignInput{Project: "p", Name: "Del"})

	if err := svc.DeleteCampaign(c.ID); err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}

	// Soft-deleted — should not be found via standard query.
	_, err := svc.GetCampaign(c.ID)
	if !errors.Is(err, ErrCampaignNotFound) {
		t.Errorf("expected ErrCampaignNotFound after delete, got %v", err)
	}

	// Paranoid check: row exists with deleted_at set.
	var raw models.ReferralCampaign
	db.Unscoped().First(&raw, c.ID)
	if raw.DeletedAt.Time.IsZero() {
		t.Error("expected deleted_at to be set after soft delete")
	}
}
