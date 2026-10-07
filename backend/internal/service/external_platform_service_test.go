package service

import (
	"errors"
	"github.com/payminto/payminto/backend/internal/environment"
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupExternalPlatformDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.ExternalPlatform{},
		&models.APIKey{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newExternalPlatformService(t *testing.T) (*ExternalPlatformService, *gorm.DB) {
	t.Helper()
	db := setupExternalPlatformDB(t)
	svc := NewExternalPlatformService(
		repository.NewExternalPlatformRepository(db),
		repository.NewAPIKeyRepository(db),
	)
	svc.SetEnvironment(environment.Test)
	return svc, db
}

func TestExternalPlatformService_Create_ReturnsPlainKeyOnce(t *testing.T) {
	svc, db := newExternalPlatformService(t)

	platform, plainKey, err := svc.Create(ExternalPlatformInput{
		Name:    "Test Merchant",
		Website: "https://test.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	if platform.ID == 0 {
		t.Error("expected platform ID to be set")
	}
	if !strings.HasPrefix(plainKey, "sk_test_") {
		t.Errorf("expected sk_test_ prefix on API key, got %s", plainKey)
	}

	// Verify the key was stored hashed, not plaintext.
	var stored models.APIKey
	db.Where("external_platform_id = ?", platform.ID).First(&stored)
	if stored.Key == plainKey {
		t.Error("API key was stored in plaintext — must be SHA-256 hashed")
	}
	if stored.Key != HashAPIKey(plainKey) {
		t.Error("stored hash does not match SHA-256 of plaintext")
	}
	if stored.Environment != "test" || stored.Prefix != plainKey[:12] {
		t.Errorf("stored environment/prefix = %q/%q, want test/%q", stored.Environment, stored.Prefix, plainKey[:12])
	}
}

func TestExternalPlatformService_Create_RequiresName(t *testing.T) {
	svc, _ := newExternalPlatformService(t)
	if _, _, err := svc.Create(ExternalPlatformInput{}); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestExternalPlatformService_RegenerateAPIKey_DeactivatesOld(t *testing.T) {
	svc, db := newExternalPlatformService(t)

	platform, oldKey, err := svc.Create(ExternalPlatformInput{Name: "Test"})
	if err != nil {
		t.Fatal(err)
	}

	newKey, err := svc.RegenerateAPIKey(platform.ID)
	if err != nil {
		t.Fatal(err)
	}
	if newKey == oldKey {
		t.Error("regenerated key must differ from old")
	}

	// Old key should be marked inactive.
	var oldRow models.APIKey
	db.Where("key = ?", HashAPIKey(oldKey)).First(&oldRow)
	if oldRow.Status != "inactive" {
		t.Errorf("expected old key status inactive, got %s", oldRow.Status)
	}

	// New key should be active.
	var newRow models.APIKey
	db.Where("key = ?", HashAPIKey(newKey)).First(&newRow)
	if newRow.Status != "active" {
		t.Errorf("expected new key status active, got %s", newRow.Status)
	}
}

func TestExternalPlatformService_GetByID_NotFound(t *testing.T) {
	svc, _ := newExternalPlatformService(t)
	if _, err := svc.GetByID(9999); err != ErrExternalPlatformNotFound {
		t.Errorf("expected ErrExternalPlatformNotFound, got %v", err)
	}
}

func TestExternalPlatformService_Update(t *testing.T) {
	svc, _ := newExternalPlatformService(t)

	platform, _, err := svc.Create(ExternalPlatformInput{Name: "Original"})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := svc.Update(platform.ID, ExternalPlatformInput{Name: "Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated" {
		t.Errorf("expected name Updated, got %s", updated.Name)
	}
}

func TestExternalPlatformService_Delete(t *testing.T) {
	svc, _ := newExternalPlatformService(t)

	platform, _, err := svc.Create(ExternalPlatformInput{Name: "ToDelete"})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(platform.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.GetByID(platform.ID); err == nil {
		t.Error("expected error after delete")
	}
}

func TestExternalPlatformService_RefusesToIssueKeysWithoutAnEnvironment(t *testing.T) {
	db := setupExternalPlatformDB(t)
	svc := NewExternalPlatformService(repository.NewExternalPlatformRepository(db), repository.NewAPIKeyRepository(db))
	if _, _, err := svc.Create(ExternalPlatformInput{Name: "x"}); !errors.Is(err, environment.ErrUnconfigured) {
		t.Fatalf("Create without an environment = %v, want ErrUnconfigured", err)
	}
}
