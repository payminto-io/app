package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRecipientTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.ExternalPlatform{},
		&models.Recipient{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newRecipientSvc(t *testing.T, db *gorm.DB) *RecipientService {
	t.Helper()
	return NewRecipientService(repository.NewRecipientRepository(db))
}

func seedRecipientDB(t *testing.T, db *gorm.DB) (memberID, platformID uint) {
	t.Helper()
	email := "m@example.com"
	member := &models.Member{Name: "Merchant", Email: &email, State: "active", MemberType: "merchant"}
	db.Create(member)
	platform := &models.ExternalPlatform{Name: "Platform"}
	db.Create(platform)
	return member.ID, platform.ID
}

func TestRecipientService_CreateAndGet(t *testing.T) {
	db := newRecipientTestDB(t)
	svc := newRecipientSvc(t, db)
	memberID, platformID := seedRecipientDB(t, db)

	ctx := t.Context()
	rec, err := svc.Create(ctx, CreateRecipientInput{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		Name:               "Alice",
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Address:            "0x1234",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.ID == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := svc.GetByID(ctx, rec.ID, memberID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected Alice, got %s", got.Name)
	}
}

func TestRecipientService_TenantIsolation(t *testing.T) {
	db := newRecipientTestDB(t)
	svc := newRecipientSvc(t, db)
	memberID, platformID := seedRecipientDB(t, db)

	// Create a second member.
	email2 := "other@example.com"
	other := &models.Member{Name: "Other", Email: &email2, State: "active", MemberType: "merchant"}
	db.Create(other)

	ctx := t.Context()
	rec, _ := svc.Create(ctx, CreateRecipientInput{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		Name:               "Bob",
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Address:            "0xabcd",
	})

	// Other member should not see this recipient.
	_, err := svc.GetByID(ctx, rec.ID, other.ID)
	if !errors.Is(err, ErrRecipientNotOwned) {
		t.Errorf("expected ErrRecipientNotOwned, got %v", err)
	}
}

func TestRecipientService_ListByMember(t *testing.T) {
	db := newRecipientTestDB(t)
	svc := newRecipientSvc(t, db)
	memberID, platformID := seedRecipientDB(t, db)

	ctx := t.Context()
	for range 3 {
		_, _ = svc.Create(ctx, CreateRecipientInput{
			MemberID:           memberID,
			ExternalPlatformID: platformID,
			Name:               "Rec",
			BlockchainCode:     "ETH",
			CurrencyCode:       "USDT",
			Address:            "0xfff",
		})
	}
	recs, err := svc.ListByMember(ctx, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 3 {
		t.Errorf("expected 3, got %d", len(recs))
	}
}

func TestRecipientService_Update(t *testing.T) {
	db := newRecipientTestDB(t)
	svc := newRecipientSvc(t, db)
	memberID, platformID := seedRecipientDB(t, db)

	ctx := t.Context()
	rec, _ := svc.Create(ctx, CreateRecipientInput{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		Name:               "Old Name",
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Address:            "0x123",
	})

	updated, err := svc.Update(ctx, rec.ID, memberID, UpdateRecipientInput{Name: "New Name"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected New Name, got %s", updated.Name)
	}
}

func TestRecipientService_Delete(t *testing.T) {
	db := newRecipientTestDB(t)
	svc := newRecipientSvc(t, db)
	memberID, platformID := seedRecipientDB(t, db)

	ctx := t.Context()
	rec, _ := svc.Create(ctx, CreateRecipientInput{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		Name:               "ToDelete",
		BlockchainCode:     "ETH",
		CurrencyCode:       "USDT",
		Address:            "0xdead",
	})

	if err := svc.Delete(ctx, rec.ID, memberID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Expect not found.
	if _, err := svc.GetByID(ctx, rec.ID, memberID); !errors.Is(err, ErrRecipientNotOwned) {
		t.Errorf("expected ErrRecipientNotOwned after delete, got %v", err)
	}
}
