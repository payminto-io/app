package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRoleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRoleService_SeedDefaults(t *testing.T) {
	db := newRoleTestDB(t)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	permSvc := NewPermissionService(permRepo)
	roleSvc := NewRoleService(roleRepo, permRepo)

	ctx := t.Context()
	// Permissions must exist before roles can reference them.
	if err := permSvc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	roles, err := roleSvc.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != len(standardRoles) {
		t.Errorf("expected %d roles, got %d", len(standardRoles), len(roles))
	}
}

func TestRoleService_SeedDefaults_Idempotent(t *testing.T) {
	db := newRoleTestDB(t)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	permSvc := NewPermissionService(permRepo)
	roleSvc := NewRoleService(roleRepo, permRepo)

	ctx := t.Context()
	_ = permSvc.SeedDefaults(ctx)
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := roleSvc.SeedDefaults(ctx); err != nil {
		t.Fatalf("second seed should be idempotent: %v", err)
	}
	roles, _ := roleSvc.ListAll(ctx)
	if len(roles) != len(standardRoles) {
		t.Errorf("expected %d roles after double-seed, got %d", len(standardRoles), len(roles))
	}
}

func TestRoleService_CreateRole(t *testing.T) {
	db := newRoleTestDB(t)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	permSvc := NewPermissionService(permRepo)
	roleSvc := NewRoleService(roleRepo, permRepo)

	ctx := t.Context()
	_ = permSvc.SeedDefaults(ctx)
	perms, _ := permRepo.List()

	role, err := roleSvc.CreateRole(ctx, CreateRoleInput{
		Name:          "custom",
		DisplayName:   "Custom Role",
		Description:   "Test role",
		PermissionIDs: []uint{perms[0].ID},
	})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if role.Name != "custom" {
		t.Errorf("expected name 'custom', got %s", role.Name)
	}
}

func TestRoleService_UpdatePermissions(t *testing.T) {
	db := newRoleTestDB(t)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	permSvc := NewPermissionService(permRepo)
	roleSvc := NewRoleService(roleRepo, permRepo)

	ctx := t.Context()
	_ = permSvc.SeedDefaults(ctx)
	perms, _ := permRepo.List()

	role, _ := roleSvc.CreateRole(ctx, CreateRoleInput{
		Name:        "temp",
		DisplayName: "Temp",
	})

	if err := roleSvc.UpdatePermissions(ctx, role.ID, []uint{perms[0].ID, perms[1].ID}); err != nil {
		t.Fatalf("UpdatePermissions: %v", err)
	}

	updated, _ := roleSvc.GetByID(ctx, role.ID)
	if len(updated.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(updated.Permissions))
	}
}

func TestRoleService_GetByName(t *testing.T) {
	db := newRoleTestDB(t)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	permSvc := NewPermissionService(permRepo)
	roleSvc := NewRoleService(roleRepo, permRepo)

	ctx := t.Context()
	_ = permSvc.SeedDefaults(ctx)
	_ = roleSvc.SeedDefaults(ctx)

	role, err := roleSvc.GetByName(ctx, "owner")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if role.Name != "owner" {
		t.Errorf("expected owner, got %s", role.Name)
	}
	// Owner should have all permissions.
	if len(role.Permissions) == 0 {
		t.Error("owner should have permissions")
	}
}
