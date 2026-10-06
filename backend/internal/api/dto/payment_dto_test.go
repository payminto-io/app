package dto

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

func TestToPaymentResponse_Basic(t *testing.T) {
	p := &models.PaymentRequest{
		ReferenceID: "test-ref-123",
		AmountInUSD: decimal.NewFromFloat(100.50),
		State:       "OPEN",
	}
	p.CreatedAt = time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	resp := ToPaymentResponse(p)

	if resp.ReferenceID != "test-ref-123" {
		t.Errorf("expected test-ref-123, got %s", resp.ReferenceID)
	}
	if resp.State != "OPEN" {
		t.Errorf("expected OPEN, got %s", resp.State)
	}
	if resp.CreatedAt != "2026-04-07T12:00:00Z" {
		t.Errorf("expected 2026-04-07T12:00:00Z, got %s", resp.CreatedAt)
	}
	if !resp.AmountInUSD.Equal(decimal.NewFromFloat(100.50)) {
		t.Errorf("expected 100.50, got %s", resp.AmountInUSD.String())
	}
}

func TestToPaymentResponse_WithExpiry(t *testing.T) {
	expiry := time.Date(2026, 4, 7, 12, 30, 0, 0, time.UTC)
	email := "test@example.com"
	p := &models.PaymentRequest{
		ReferenceID:   "test-ref-456",
		AmountInUSD:   decimal.NewFromFloat(50),
		State:         "FILLED",
		CustomerEmail: &email,
		ExpiresAt:     &expiry,
	}
	p.CreatedAt = time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)

	resp := ToPaymentResponse(p)

	if resp.ExpiresAt == nil {
		t.Fatal("expected ExpiresAt to be set")
	}
	if *resp.ExpiresAt != "2026-04-07T12:30:00Z" {
		t.Errorf("expected 2026-04-07T12:30:00Z, got %s", *resp.ExpiresAt)
	}
	if resp.CustomerEmail == nil || *resp.CustomerEmail != "test@example.com" {
		t.Error("expected customer email to be set")
	}
}

func TestToPaymentResponse_NilOptionals(t *testing.T) {
	p := &models.PaymentRequest{
		ReferenceID: "test-ref-789",
		AmountInUSD: decimal.NewFromFloat(10),
		State:       "CANCELLED",
	}
	p.CreatedAt = time.Now()

	resp := ToPaymentResponse(p)

	if resp.ExpiresAt != nil {
		t.Error("expected ExpiresAt to be nil")
	}
	if resp.CustomerEmail != nil {
		t.Error("expected CustomerEmail to be nil")
	}
}
