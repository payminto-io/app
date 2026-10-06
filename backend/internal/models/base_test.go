package models

import (
	"testing"
	"time"
)

func TestPaymintoModel_HasFields(t *testing.T) {
	m := PaymintoModel{
		ID:        1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if m.ID != 1 {
		t.Errorf("expected ID 1, got %d", m.ID)
	}
	if m.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestBaseModel_HasFields(t *testing.T) {
	m := BaseModel{
		ID:        42,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if m.ID != 42 {
		t.Errorf("expected ID 42, got %d", m.ID)
	}
	if m.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}
