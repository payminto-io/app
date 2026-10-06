package repository_test

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and auto-migrates the provided
// model types. Each test gets its own independent database.
func openTestDB(t *testing.T, dst ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&mode=memory"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// Use a unique DB per test by using _test prefix and the test name in file path
	// For simplicity we use a fresh connection each test by resetting the shared cache
	if err := db.AutoMigrate(dst...); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

// openFreshDB creates a truly isolated in-memory database (no sharing).
func openFreshDB(t *testing.T, dst ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	if err := db.AutoMigrate(dst...); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}
	return db
}

// ---------------------------------------------------------------------------
// Member tests
// ---------------------------------------------------------------------------

func TestMemberRepo_CreateAndGet(t *testing.T) {
	db := openFreshDB(t, &models.Member{}, &models.Role{})
	repo := repository.NewMemberRepository(db)

	email := "alice@example.com"
	m := &models.Member{
		Name:  "Alice",
		Email: &email,
	}
	if err := repo.Create(m); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if m.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Alice" {
		t.Errorf("expected Name=Alice, got %q", got.Name)
	}
}

func TestMemberRepo_GetByEmail(t *testing.T) {
	db := openFreshDB(t, &models.Member{}, &models.Role{})
	repo := repository.NewMemberRepository(db)

	email := "bob@example.com"
	m := &models.Member{Name: "Bob", Email: &email}
	if err := repo.Create(m); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByEmail("bob@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.Name != "Bob" {
		t.Errorf("expected Name=Bob, got %q", got.Name)
	}
}

func TestMemberRepo_GetByEmail_NotFound(t *testing.T) {
	db := openFreshDB(t, &models.Member{}, &models.Role{})
	repo := repository.NewMemberRepository(db)

	_, err := repo.GetByEmail("nobody@example.com")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestMemberRepo_List_And_Count(t *testing.T) {
	db := openFreshDB(t, &models.Member{}, &models.Role{})
	repo := repository.NewMemberRepository(db)

	for _, name := range []string{"C1", "C2", "C3"} {
		n := name + "@e.com"
		if err := repo.Create(&models.Member{Name: name, Email: &n}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	members, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members, got %d", len(members))
	}

	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}

	// Test pagination via options.
	page, err := repo.List(repository.WithLimit(2), repository.WithOffset(1))
	if err != nil {
		t.Fatalf("List with opts: %v", err)
	}
	if len(page) != 2 {
		t.Errorf("expected 2 members with limit=2 offset=1, got %d", len(page))
	}
}

// ---------------------------------------------------------------------------
// APIKey tests
// ---------------------------------------------------------------------------

func TestAPIKeyRepo_CreateAndGetByKey(t *testing.T) {
	db := openFreshDB(t, &models.ExternalPlatform{}, &models.Member{}, &models.Role{}, &models.APIKey{})
	platformRepo := repository.NewExternalPlatformRepository(db)
	keyRepo := repository.NewAPIKeyRepository(db)

	// Create a platform first (required FK).
	platform := &models.ExternalPlatform{Name: "TestPlatform", SuccessEndpoint: "https://example.com/ok"}
	if err := platformRepo.Create(platform); err != nil {
		t.Fatalf("Create platform: %v", err)
	}

	raw := "super-secret-key"
	hashed := repository.Sha256Sum(raw)

	k := &models.APIKey{
		Key:                hashed,
		ExternalPlatformID: platform.ID,
		Status:             "active",
	}
	if err := keyRepo.Create(k); err != nil {
		t.Fatalf("Create API key: %v", err)
	}

	got, err := keyRepo.GetByKey(hashed)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if got.ID != k.ID {
		t.Errorf("expected ID=%d, got %d", k.ID, got.ID)
	}
	if got.Status != "active" {
		t.Errorf("expected status=active, got %q", got.Status)
	}
}

func TestAPIKeyRepo_DeactivateByMemberID(t *testing.T) {
	db := openFreshDB(t, &models.ExternalPlatform{}, &models.Member{}, &models.Role{}, &models.APIKey{})
	platformRepo := repository.NewExternalPlatformRepository(db)
	memberRepo := repository.NewMemberRepository(db)
	keyRepo := repository.NewAPIKeyRepository(db)

	platform := &models.ExternalPlatform{Name: "P2", SuccessEndpoint: "https://example.com/ok"}
	if err := platformRepo.Create(platform); err != nil {
		t.Fatalf("Create platform: %v", err)
	}

	email := "dave@example.com"
	member := &models.Member{Name: "Dave", Email: &email}
	if err := memberRepo.Create(member); err != nil {
		t.Fatalf("Create member: %v", err)
	}

	// Create two API keys for the member.
	for i, rawKey := range []string{"key-a", "key-b"} {
		_ = i
		k := &models.APIKey{
			Key:                repository.Sha256Sum(rawKey),
			ExternalPlatformID: platform.ID,
			MemberID:           &member.ID,
			Status:             "active",
		}
		if err := keyRepo.Create(k); err != nil {
			t.Fatalf("Create key: %v", err)
		}
	}

	if err := keyRepo.DeactivateByMemberID(member.ID); err != nil {
		t.Fatalf("DeactivateByMemberID: %v", err)
	}

	keys, err := keyRepo.ListByMemberID(member.ID)
	if err != nil {
		t.Fatalf("ListByMemberID: %v", err)
	}
	for _, k := range keys {
		if k.Status != "inactive" {
			t.Errorf("expected status=inactive, got %q for key ID=%d", k.Status, k.ID)
		}
	}
}

// ---------------------------------------------------------------------------
// ExternalPlatform tests
// ---------------------------------------------------------------------------

func TestExternalPlatformRepo_CreateAndList(t *testing.T) {
	db := openFreshDB(t, &models.ExternalPlatform{})
	repo := repository.NewExternalPlatformRepository(db)

	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		p := &models.ExternalPlatform{Name: name, SuccessEndpoint: "https://example.com"}
		if err := repo.Create(p); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	platforms, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(platforms) != 3 {
		t.Errorf("expected 3 platforms, got %d", len(platforms))
	}

	count, err := repo.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}

	got, err := repo.GetByName("Beta")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Name != "Beta" {
		t.Errorf("expected Name=Beta, got %q", got.Name)
	}

	// GetByID not found.
	_, err = repo.GetByID(99999)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Role tests
// ---------------------------------------------------------------------------

func TestRoleRepo_CreateAndGetByName(t *testing.T) {
	db := openFreshDB(t, &models.Role{}, &models.Permission{})
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// Create some permissions.
	p1 := &models.Permission{Name: "payments:read", DisplayName: "Read Payments", Description: "Can read payments"}
	p2 := &models.Permission{Name: "payments:write", DisplayName: "Write Payments", Description: "Can write payments"}
	if err := permRepo.Create(p1); err != nil {
		t.Fatalf("Create perm1: %v", err)
	}
	if err := permRepo.Create(p2); err != nil {
		t.Fatalf("Create perm2: %v", err)
	}

	role := &models.Role{
		Name:        "merchant",
		DisplayName: "Merchant",
		Permissions: []models.Permission{*p1, *p2},
	}
	if err := roleRepo.Create(role); err != nil {
		t.Fatalf("Create role: %v", err)
	}

	got, err := roleRepo.GetByName("merchant")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.DisplayName != "Merchant" {
		t.Errorf("expected DisplayName=Merchant, got %q", got.DisplayName)
	}
	if len(got.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(got.Permissions))
	}
}

func TestRoleRepo_UpdatePermissions(t *testing.T) {
	db := openFreshDB(t, &models.Role{}, &models.Permission{})
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	p1 := &models.Permission{Name: "perm:a", DisplayName: "Perm A", Description: "A"}
	p2 := &models.Permission{Name: "perm:b", DisplayName: "Perm B", Description: "B"}
	p3 := &models.Permission{Name: "perm:c", DisplayName: "Perm C", Description: "C"}
	for _, p := range []*models.Permission{p1, p2, p3} {
		if err := permRepo.Create(p); err != nil {
			t.Fatalf("Create perm: %v", err)
		}
	}

	role := &models.Role{Name: "admin", DisplayName: "Admin", Permissions: []models.Permission{*p1}}
	if err := roleRepo.Create(role); err != nil {
		t.Fatalf("Create role: %v", err)
	}

	// Replace with p2 and p3.
	if err := roleRepo.UpdatePermissions(role.ID, []uint{p2.ID, p3.ID}); err != nil {
		t.Fatalf("UpdatePermissions: %v", err)
	}

	got, err := roleRepo.GetByID(role.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Permissions) != 2 {
		t.Errorf("expected 2 permissions after update, got %d", len(got.Permissions))
	}
}

func TestRoleRepo_GetByNames(t *testing.T) {
	db := openFreshDB(t, &models.Role{}, &models.Permission{})
	roleRepo := repository.NewRoleRepository(db)

	for _, name := range []string{"viewer", "editor", "owner"} {
		r := &models.Role{Name: name, DisplayName: name}
		if err := roleRepo.Create(r); err != nil {
			t.Fatalf("Create role %s: %v", name, err)
		}
	}

	roles, err := roleRepo.GetByNames([]string{"viewer", "owner"})
	if err != nil {
		t.Fatalf("GetByNames: %v", err)
	}
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

// ---------------------------------------------------------------------------
// Permission tests
// ---------------------------------------------------------------------------

func TestPermissionRepo_GetByNames_Batch(t *testing.T) {
	db := openFreshDB(t, &models.Permission{})
	repo := repository.NewPermissionRepository(db)

	names := []string{"wallet:read", "wallet:write", "sweep:trigger"}
	for _, n := range names {
		p := &models.Permission{Name: n, DisplayName: n, Description: n}
		if err := repo.Create(p); err != nil {
			t.Fatalf("Create perm %s: %v", n, err)
		}
	}

	perms, err := repo.GetByNames([]string{"wallet:read", "sweep:trigger"})
	if err != nil {
		t.Fatalf("GetByNames: %v", err)
	}
	if len(perms) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(perms))
	}

	// List all.
	all, err := repo.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 permissions, got %d", len(all))
	}

	// GetByName not found.
	_, err = repo.GetByName("nonexistent:perm")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}
