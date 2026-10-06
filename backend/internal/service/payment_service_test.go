package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestValidatePaymentInput_Valid(t *testing.T) {
	input := CreatePaymentInput{
		AmountInUSD:   decimal.NewFromFloat(100.0),
		CustomerEmail: strPtr("user@example.com"),
	}
	if err := input.Validate(); err != nil {
		t.Errorf("expected valid input, got error: %v", err)
	}
}

func TestValidatePaymentInput_ZeroAmount(t *testing.T) {
	input := CreatePaymentInput{
		AmountInUSD: decimal.Zero,
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestValidatePaymentInput_NegativeAmount(t *testing.T) {
	input := CreatePaymentInput{
		AmountInUSD: decimal.NewFromFloat(-10),
	}
	if err := input.Validate(); err == nil {
		t.Error("expected error for negative amount")
	}
}

func TestGenerateReferenceID_Unique(t *testing.T) {
	id1 := GenerateReferenceID()
	id2 := GenerateReferenceID()
	if id1 == id2 {
		t.Error("expected unique reference IDs")
	}
}

func strPtr(s string) *string { return &s }
