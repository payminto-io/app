package service

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newConfigServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Configuration{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestConfigurationService_SetAndGet(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	if err := svc.Set(ctx, "sweep.threshold", "100", "min sweep amount"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	cfg, err := svc.Get(ctx, "sweep.threshold")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cfg.Value != "100" {
		t.Errorf("expected 100, got %s", cfg.Value)
	}
}

func TestConfigurationService_GetString(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	_ = svc.Set(ctx, "mode", "testnet", "")

	v := svc.GetString(ctx, "mode", "mainnet")
	if v != "testnet" {
		t.Errorf("expected testnet, got %s", v)
	}

	// Missing key returns default.
	v2 := svc.GetString(ctx, "nonexistent", "fallback")
	if v2 != "fallback" {
		t.Errorf("expected fallback, got %s", v2)
	}
}

func TestConfigurationService_GetInt(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	_ = svc.Set(ctx, "max.retries", "5", "")

	v := svc.GetInt(ctx, "max.retries", 3)
	if v != 5 {
		t.Errorf("expected 5, got %d", v)
	}

	v2 := svc.GetInt(ctx, "missing", 7)
	if v2 != 7 {
		t.Errorf("expected 7, got %d", v2)
	}
}

func TestConfigurationService_GetBool(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	_ = svc.Set(ctx, "feature.enabled", "true", "")

	v := svc.GetBool(ctx, "feature.enabled", false)
	if !v {
		t.Error("expected true")
	}

	_ = svc.Set(ctx, "feature.disabled", "false", "")
	v2 := svc.GetBool(ctx, "feature.disabled", true)
	if v2 {
		t.Error("expected false")
	}
}

func TestConfigurationService_CacheInvalidatedOnSet(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	_ = svc.Set(ctx, "key", "v1", "")

	// Prime cache.
	v1 := svc.GetString(ctx, "key", "")
	if v1 != "v1" {
		t.Errorf("expected v1, got %s", v1)
	}

	// Update — cache should be invalidated.
	_ = svc.Set(ctx, "key", "v2", "")
	v2 := svc.GetString(ctx, "key", "")
	if v2 != "v2" {
		t.Errorf("expected v2 after update, got %s", v2)
	}
}

func TestConfigurationService_TTLExpiry(t *testing.T) {
	db := newConfigServiceDB(t)
	repo := repository.NewConfigurationRepository(db)
	svc := NewConfigurationService(repo)

	ctx := t.Context()
	_ = svc.Set(ctx, "ttlkey", "value", "")

	// Manually expire the cache entry.
	svc.mu.Lock()
	if e, ok := svc.cache["ttlkey"]; ok {
		e.expiresAt = time.Now().Add(-1 * time.Second)
	}
	svc.mu.Unlock()

	// Should reload from DB.
	v := svc.GetString(ctx, "ttlkey", "fallback")
	if v != "value" {
		t.Errorf("expected value after TTL reload, got %s", v)
	}
}

// countingConfigRepo is a minimal ConfigurationRepository that counts Get calls.
type countingConfigRepo struct {
	inner    repository.ConfigurationRepository
	getCalls int
}

func (r *countingConfigRepo) Get(key string) (*models.Configuration, error) {
	r.getCalls++
	return r.inner.Get(key)
}

func (r *countingConfigRepo) Set(key, value string) error {
	return r.inner.Set(key, value)
}

func (r *countingConfigRepo) SetWithDescription(key, value, description, category string) error {
	return r.inner.SetWithDescription(key, value, description, category)
}

func (r *countingConfigRepo) List(opts ...repository.QueryOption) ([]models.Configuration, error) {
	return r.inner.List(opts...)
}

func (r *countingConfigRepo) Delete(key string) error {
	return r.inner.Delete(key)
}

func TestConfigurationService_Get_NegativeCacheHit(t *testing.T) {
	db := newConfigServiceDB(t)
	inner := repository.NewConfigurationRepository(db)
	counting := &countingConfigRepo{inner: inner}
	svc := NewConfigurationService(counting)

	ctx := t.Context()

	// First call — key is absent, goes to DB.
	v1 := svc.GetString(ctx, "absent.key", "default")
	if v1 != "default" {
		t.Errorf("expected default, got %s", v1)
	}
	if counting.getCalls != 1 {
		t.Errorf("expected 1 DB call after first miss, got %d", counting.getCalls)
	}

	// Second call — should be served from the negative cache without hitting DB.
	v2 := svc.GetString(ctx, "absent.key", "default")
	if v2 != "default" {
		t.Errorf("expected default on second call, got %s", v2)
	}
	if counting.getCalls != 1 {
		t.Errorf("second call hit DB (getCalls=%d); expected negative cache hit", counting.getCalls)
	}

	// After cache expiry the DB is consulted again.
	svc.mu.Lock()
	if e, ok := svc.cache["absent.key"]; ok {
		e.expiresAt = time.Now().Add(-1 * time.Second)
	}
	svc.mu.Unlock()

	_ = svc.GetString(ctx, "absent.key", "default")
	if counting.getCalls != 2 {
		t.Errorf("expected 2 DB calls after cache expiry, got %d", counting.getCalls)
	}
}
