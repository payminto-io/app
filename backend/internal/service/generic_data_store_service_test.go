package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newGDSTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.GenericDataStore{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func newGDSSvc(t *testing.T) *GenericDataStoreService {
	t.Helper()
	db := newGDSTestDB(t)
	return NewGenericDataStoreService(repository.NewGenericDataStoreRepository(db))
}

func TestGenericDataStoreService_SetAndGet(t *testing.T) {
	svc := newGDSSvc(t)
	ctx := t.Context()

	if err := svc.Set(ctx, "ns", "k1", "hello", nil); err != nil {
		t.Fatalf("Set: %v", err)
	}
	entry, err := svc.Get(ctx, "ns", "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if entry.Value != "hello" {
		t.Errorf("expected hello, got %s", entry.Value)
	}
}

func TestGenericDataStoreService_Delete(t *testing.T) {
	svc := newGDSSvc(t)
	ctx := t.Context()

	_ = svc.Set(ctx, "ns", "k2", "v", nil)
	if err := svc.Delete(ctx, "ns", "k2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(ctx, "ns", "k2"); err == nil {
		t.Error("expected error after delete")
	}
}

func TestGenericDataStoreService_ListByNamespace(t *testing.T) {
	svc := newGDSSvc(t)
	ctx := t.Context()

	_ = svc.Set(ctx, "nslist", "k1", "v1", nil)
	_ = svc.Set(ctx, "nslist", "k2", "v2", nil)
	_ = svc.Set(ctx, "other", "k3", "v3", nil)

	entries, err := svc.ListByNamespace(ctx, "nslist")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2, got %d", len(entries))
	}
}

func TestGenericDataStoreService_TTLExpired(t *testing.T) {
	svc := newGDSSvc(t)
	ctx := t.Context()

	// TTL of -1 second = already expired.
	ttl := int64(-1)
	_ = svc.Set(ctx, "ns", "expired", "v", &ttl)

	// Get should return not found for expired entry.
	if _, err := svc.Get(ctx, "ns", "expired"); err == nil {
		t.Error("expected error for expired key")
	}
}

func TestGenericDataStoreService_Upsert(t *testing.T) {
	svc := newGDSSvc(t)
	ctx := t.Context()

	_ = svc.Set(ctx, "ns", "k", "v1", nil)
	_ = svc.Set(ctx, "ns", "k", "v2", nil)

	entry, err := svc.Get(ctx, "ns", "k")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Value != "v2" {
		t.Errorf("expected v2 after upsert, got %s", entry.Value)
	}
}
