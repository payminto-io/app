package repository

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newConfigTestDB(t *testing.T) *gorm.DB {
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

func TestConfigRepo_SetAndGet(t *testing.T) {
	repo := NewConfigurationRepository(newConfigTestDB(t))
	if err := repo.Set("mode", "testnet"); err != nil {
		t.Fatal(err)
	}
	c, err := repo.Get("mode")
	if err != nil {
		t.Fatal(err)
	}
	if c.Value != "testnet" {
		t.Errorf("expected testnet, got %s", c.Value)
	}
}

func TestConfigRepo_SetUpdatesExisting(t *testing.T) {
	repo := NewConfigurationRepository(newConfigTestDB(t))
	_ = repo.Set("mode", "testnet")
	_ = repo.Set("mode", "mainnet")
	c, _ := repo.Get("mode")
	if c.Value != "mainnet" {
		t.Errorf("expected mainnet after upsert, got %s", c.Value)
	}
}

func TestConfigRepo_Delete(t *testing.T) {
	repo := NewConfigurationRepository(newConfigTestDB(t))
	_ = repo.Set("k", "v")
	if err := repo.Delete("k"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get("k"); err == nil {
		t.Error("expected error after delete")
	}
}

func TestConfigRepo_List(t *testing.T) {
	repo := NewConfigurationRepository(newConfigTestDB(t))
	_ = repo.Set("a", "1")
	_ = repo.Set("b", "2")
	_ = repo.Set("c", "3")
	list, err := repo.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3, got %d", len(list))
	}
}

func TestConfigRepo_SetWithDescription(t *testing.T) {
	repo := NewConfigurationRepository(newConfigTestDB(t))
	if err := repo.SetWithDescription("mode", "testnet", "Network mode stamp", "system"); err != nil {
		t.Fatal(err)
	}
	c, _ := repo.Get("mode")
	if c.Description == nil || *c.Description != "Network mode stamp" {
		t.Error("expected description to be set")
	}
	if c.Category != "system" {
		t.Errorf("expected category 'system', got %q", c.Category)
	}
}
