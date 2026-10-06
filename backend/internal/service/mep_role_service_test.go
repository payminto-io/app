package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMEPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Permission{},
		&models.Role{},
		&models.Member{},
		&models.ExternalPlatform{},
		&models.MemberExternalPlatformRole{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedMEPFixtures(t *testing.T, db *gorm.DB) (memberID, platformID, roleID uint) {
	t.Helper()
	// Create a permission.
	perm := &models.Permission{Name: "payments.read", DisplayName: "Read Payments", Description: "view"}
	db.Create(perm)

	// Create a role with that permission.
	role := &models.Role{Name: "viewer", DisplayName: "Viewer", Permissions: []models.Permission{*perm}}
	db.Create(role)

	// Create a member.
	email := "test@example.com"
	member := &models.Member{Name: "Test", Email: &email, State: "active", MemberType: "merchant"}
	db.Create(member)

	// Create an external platform.
	platform := &models.ExternalPlatform{Name: "TestPlatform"}
	db.Create(platform)

	return member.ID, platform.ID, role.ID
}

func TestMEPRoleService_AssignAndHasPermission(t *testing.T) {
	db := newMEPTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := NewMemberExternalPlatformRoleService(repo)

	memberID, platformID, roleID := seedMEPFixtures(t, db)
	ctx := t.Context()

	if err := svc.Assign(ctx, memberID, platformID, roleID); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	if !svc.HasPermission(ctx, memberID, platformID, "payments.read") {
		t.Error("expected HasPermission to return true for payments.read")
	}
}

func TestMEPRoleService_HasPermission_Negative(t *testing.T) {
	db := newMEPTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := NewMemberExternalPlatformRoleService(repo)

	memberID, platformID, roleID := seedMEPFixtures(t, db)
	ctx := t.Context()

	_ = svc.Assign(ctx, memberID, platformID, roleID)

	// viewer doesn't have system.admin
	if svc.HasPermission(ctx, memberID, platformID, "system.admin") {
		t.Error("viewer should not have system.admin")
	}
}

func TestMEPRoleService_HasPermission_UnknownMember(t *testing.T) {
	db := newMEPTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := NewMemberExternalPlatformRoleService(repo)

	ctx := t.Context()
	if svc.HasPermission(ctx, 9999, 9999, "payments.read") {
		t.Error("unknown member should not have any permission")
	}
}

func TestMEPRoleService_Revoke(t *testing.T) {
	db := newMEPTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := NewMemberExternalPlatformRoleService(repo)

	memberID, platformID, roleID := seedMEPFixtures(t, db)
	ctx := t.Context()

	_ = svc.Assign(ctx, memberID, platformID, roleID)
	if err := svc.Revoke(ctx, memberID, platformID, roleID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	// After revoke, permission should be gone.
	if svc.HasPermission(ctx, memberID, platformID, "payments.read") {
		t.Error("permission should be revoked")
	}
}

func TestMEPRoleService_CacheInvalidatedOnRevoke(t *testing.T) {
	db := newMEPTestDB(t)
	repo := repository.NewMemberExternalPlatformRoleRepository(db)
	svc := NewMemberExternalPlatformRoleService(repo)

	memberID, platformID, roleID := seedMEPFixtures(t, db)
	ctx := t.Context()

	_ = svc.Assign(ctx, memberID, platformID, roleID)
	// Prime cache.
	_ = svc.HasPermission(ctx, memberID, platformID, "payments.read")

	// Revoke should invalidate cache.
	_ = svc.Revoke(ctx, memberID, platformID, roleID)
	if svc.HasPermission(ctx, memberID, platformID, "payments.read") {
		t.Error("cache should be invalidated after revoke")
	}
}
