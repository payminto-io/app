package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Permission{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPermissionService_SeedDefaults(t *testing.T) {
	db := newPermissionTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)

	ctx := t.Context()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}

	perms, err := svc.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != len(defaultPermissions) {
		t.Errorf("expected %d permissions, got %d", len(defaultPermissions), len(perms))
	}
}

func TestPermissionService_SeedDefaults_Idempotent(t *testing.T) {
	db := newPermissionTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)

	ctx := t.Context()
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := svc.SeedDefaults(ctx); err != nil {
		t.Fatalf("second seed should be idempotent: %v", err)
	}

	perms, _ := svc.ListAll(ctx)
	if len(perms) != len(defaultPermissions) {
		t.Errorf("expected %d permissions after double-seed, got %d", len(defaultPermissions), len(perms))
	}
}

func TestPermissionService_GetByName(t *testing.T) {
	db := newPermissionTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)

	ctx := t.Context()
	_ = svc.SeedDefaults(ctx)

	p, err := svc.GetByName(ctx, "payments.read")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if p.Name != "payments.read" {
		t.Errorf("expected payments.read, got %s", p.Name)
	}
}

func TestPermissionService_GetByID(t *testing.T) {
	db := newPermissionTestDB(t)
	repo := repository.NewPermissionRepository(db)
	svc := NewPermissionService(repo)

	ctx := t.Context()
	_ = svc.SeedDefaults(ctx)
	perms, _ := svc.ListAll(ctx)

	p, err := svc.GetByID(ctx, perms[0].ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p.ID != perms[0].ID {
		t.Error("ID mismatch")
	}
}
