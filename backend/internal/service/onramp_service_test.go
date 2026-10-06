package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

func TestOnrampService_NoProvider(t *testing.T) {
	svc := NewOnrampService("", "")
	p := &models.PaymentRequest{AmountInUSD: decimal.NewFromFloat(100)}
	_, err := svc.CreateSession(p, "0xaddr", "USDC")
	if err == nil {
		t.Error("expected error when provider not configured")
	}
}

func TestOnrampService_CreateSession(t *testing.T) {
	svc := NewOnrampService("https://onramp.example.com", "test-key")
	p := &models.PaymentRequest{
		ReferenceID: "ref-123",
		AmountInUSD: decimal.NewFromFloat(50),
	}
	session, err := svc.CreateSession(p, "0xdeposit", "USDC")
	if err != nil {
		t.Fatal(err)
	}
	if session.SessionID != "ref-123" {
		t.Errorf("expected ref-123, got %s", session.SessionID)
	}
	if session.RedirectURL == "" {
		t.Error("expected non-empty redirect URL")
	}
}
