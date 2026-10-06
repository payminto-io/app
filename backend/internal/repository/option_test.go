package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite in-memory: %v", err)
	}
	return db
}

func TestWithLimit_ZeroIsNoop(t *testing.T) {
	db := testDB(t)
	got := WithLimit(0)(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithLimit_Positive(t *testing.T) {
	db := testDB(t)
	got := WithLimit(10)(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithOffset_NegativeIsNoop(t *testing.T) {
	db := testDB(t)
	got := WithOffset(-5)(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithOrder_Ascending(t *testing.T) {
	db := testDB(t)
	got := WithAscendingOrder("created_at")(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithOrder_Descending(t *testing.T) {
	db := testDB(t)
	got := WithDescendingOrder("id")(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithPreload_NoArgs(t *testing.T) {
	db := testDB(t)
	got := WithPreload("Member")(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestWithPreload_WithArgs(t *testing.T) {
	db := testDB(t)
	got := WithPreload("Member", "state = ?", "active")(db)
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestApply_ChainedOptions(t *testing.T) {
	db := testDB(t)
	got := Apply(db, WithLimit(25), WithOffset(50), WithDescendingOrder("created_at"))
	if got == nil {
		t.Fatal("expected non-nil db")
	}
}
