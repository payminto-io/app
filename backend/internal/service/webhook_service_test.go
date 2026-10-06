package service

import "testing"

func TestGenerateWebhookSignature_NotEmpty(t *testing.T) {
	sig := GenerateWebhookSignature(`{"event":"payment.confirmed"}`, "secret123")
	if sig == "" {
		t.Error("expected non-empty signature")
	}
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test123"
	payload := `{"event":"payment.confirmed","reference_id":"abc"}`
	sig := GenerateWebhookSignature(payload, secret)
	if !VerifyWebhookSignature(payload, sig, secret) {
		t.Error("signature verification failed")
	}
}

func TestVerifyWebhookSignature_WrongSecret(t *testing.T) {
	payload := `{"event":"payment.confirmed"}`
	sig := GenerateWebhookSignature(payload, "correct_secret")
	if VerifyWebhookSignature(payload, sig, "wrong_secret") {
		t.Error("should not verify with wrong secret")
	}
}

func TestGenerateWebhookSignature_Deterministic(t *testing.T) {
	s1 := GenerateWebhookSignature("data", "key")
	s2 := GenerateWebhookSignature("data", "key")
	if s1 != s2 {
		t.Error("same input should produce same signature")
	}
}

func TestGenerateWebhookSignature_DifferentPayloads(t *testing.T) {
	s1 := GenerateWebhookSignature("payload1", "key")
	s2 := GenerateWebhookSignature("payload2", "key")
	if s1 == s2 {
		t.Error("different payloads should produce different signatures")
	}
}
