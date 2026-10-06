package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMemberTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.Role{},
		&models.Permission{},
		&models.MemberExternalPlatformRole{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newMemberSvc(t *testing.T, db *gorm.DB) *MemberService {
	t.Helper()
	memberRepo := repository.NewMemberRepository(db)
	mepRepo := repository.NewMemberExternalPlatformRoleRepository(db)
	mepSvc := NewMemberExternalPlatformRoleService(mepRepo)
	return NewMemberService(memberRepo, mepSvc)
}

func TestMemberService_CreateMember(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	m, err := svc.CreateMember(ctx, CreateMemberInput{
		Name:       "Alice",
		Email:      "alice@example.com",
		Password:   "secret123",
		MemberType: "merchant",
	})
	if err != nil {
		t.Fatalf("CreateMember: %v", err)
	}
	if m.Name != "Alice" {
		t.Errorf("expected Alice, got %s", m.Name)
	}
	if m.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestMemberService_DuplicateEmailRejected(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	_, err := svc.CreateMember(ctx, CreateMemberInput{
		Name:  "Alice",
		Email: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.CreateMember(ctx, CreateMemberInput{
		Name:  "Alice2",
		Email: "alice@example.com",
	})
	if err == nil {
		t.Error("expected error for duplicate email, got nil")
	}
}

func TestMemberService_GetByID(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	m, _ := svc.CreateMember(ctx, CreateMemberInput{Name: "Bob", Email: "bob@example.com"})

	got, err := svc.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != m.ID {
		t.Error("ID mismatch")
	}
}

func TestMemberService_GetByEmail(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	_, _ = svc.CreateMember(ctx, CreateMemberInput{Name: "Carol", Email: "carol@example.com"})

	got, err := svc.GetByEmail(ctx, "carol@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.Name != "Carol" {
		t.Errorf("expected Carol, got %s", got.Name)
	}
}

func TestMemberService_ActivateDeactivate(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	m, _ := svc.CreateMember(ctx, CreateMemberInput{Name: "Dave", Email: "dave@example.com"})

	if err := svc.Deactivate(ctx, m.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	got, _ := svc.GetByID(ctx, m.ID)
	if got.State != "inactive" {
		t.Errorf("expected inactive, got %s", got.State)
	}

	if err := svc.Activate(ctx, m.ID); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	got, _ = svc.GetByID(ctx, m.ID)
	if got.State != "active" {
		t.Errorf("expected active, got %s", got.State)
	}
}

func TestMemberService_ChangePassword(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	m, _ := svc.CreateMember(ctx, CreateMemberInput{
		Name:     "Eve",
		Email:    "eve@example.com",
		Password: "oldpass",
	})

	if err := svc.ChangePassword(ctx, m.ID, "oldpass", "newpass"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	// Wrong current password should fail.
	if err := svc.ChangePassword(ctx, m.ID, "oldpass", "anotherpass"); err == nil {
		t.Error("expected error for wrong current password")
	}
}

func TestMemberService_Update(t *testing.T) {
	db := newMemberTestDB(t)
	svc := newMemberSvc(t, db)

	ctx := t.Context()
	m, _ := svc.CreateMember(ctx, CreateMemberInput{Name: "Frank", Email: "frank@example.com"})

	updated, err := svc.Update(ctx, m.ID, UpdateMemberInput{Name: "Franklin"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Franklin" {
		t.Errorf("expected Franklin, got %s", updated.Name)
	}
}
