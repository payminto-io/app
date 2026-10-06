package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBaseRepository_WithDB(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	base := &BaseRepository{DB: db}

	tx := db.Begin()
	defer tx.Rollback()

	scoped := base.WithDB(tx)
	if scoped == nil {
		t.Fatal("expected non-nil scoped repo")
	}
	if scoped.DB == base.DB {
		t.Error("expected scoped repo to hold the tx, not the base db")
	}
}
