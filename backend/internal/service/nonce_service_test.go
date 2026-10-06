package service

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNonceSvc(t *testing.T) *NonceService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.GenericDataStore{}); err != nil {
		t.Fatal(err)
	}
	gdsRepo := repository.NewGenericDataStoreRepository(db)
	gdsSvc := NewGenericDataStoreService(gdsRepo)
	return NewNonceService(gdsSvc)
}

func TestNonceService_GenerateAndConsume(t *testing.T) {
	svc := newNonceSvc(t)
	ctx := t.Context()

	nonce, err := svc.Generate(ctx, "login")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(nonce) == 0 {
		t.Error("expected non-empty nonce")
	}

	if err := svc.Consume(ctx, nonce, "login"); err != nil {
		t.Fatalf("Consume: %v", err)
	}
}

func TestNonceService_DoubleConsumeFails(t *testing.T) {
	svc := newNonceSvc(t)
	ctx := t.Context()

	nonce, _ := svc.Generate(ctx, "signup")
	_ = svc.Consume(ctx, nonce, "signup")

	if err := svc.Consume(ctx, nonce, "signup"); err == nil {
		t.Error("expected error on second consume")
	}
}

func TestNonceService_WrongPurposeFails(t *testing.T) {
	svc := newNonceSvc(t)
	ctx := t.Context()

	nonce, _ := svc.Generate(ctx, "login")

	if err := svc.Consume(ctx, nonce, "signup"); err == nil {
		t.Error("expected error when consuming with wrong purpose")
	}
}

func TestNonceService_UnknownNonceFails(t *testing.T) {
	svc := newNonceSvc(t)
	ctx := t.Context()

	if err := svc.Consume(ctx, "bogus", "login"); err == nil {
		t.Error("expected error for unknown nonce")
	}
}

// TestNonceService_Consume_Concurrent verifies that exactly one of N concurrent
// callers wins when all attempt to consume the same nonce simultaneously.
func TestNonceService_Consume_Concurrent(t *testing.T) {
	// Use a shared DB to allow concurrent access (SQLite in-memory is
	// single-connection; the race detector still validates the logic).
	svc := newNonceSvc(t)
	ctx := t.Context()

	nonce, err := svc.Generate(ctx, "concurrent")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	const goroutines = 10
	var wg sync.WaitGroup
	var wins atomic.Int32

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if err := svc.Consume(ctx, nonce, "concurrent"); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := wins.Load(); got != 1 {
		t.Errorf("expected exactly 1 winner, got %d", got)
	}
}
