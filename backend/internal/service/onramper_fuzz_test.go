package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

// FuzzOnramperSignatureVerify asserts that the verifier never accepts a
// signature for a payload it did not actually sign. This is a security
// invariant — a false positive here would let an attacker replay an
// Onramper webhook with arbitrary state transitions.
func FuzzOnramperSignatureVerify(f *testing.F) {
	secret := "shared-secret"
	svc := &OnramperPaymentsService{webhookSecret: secret}

	// Seed with a valid (payload, signature) pair.
	goodPayload := []byte(`{"sessionId":"ompr_1","status":"completed"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(goodPayload)
	goodSig := hex.EncodeToString(mac.Sum(nil))
	f.Add(goodPayload, goodSig)

	// Seed with tampered cases.
	f.Add(goodPayload, "")
	f.Add(goodPayload, "deadbeef")
	f.Add([]byte("{}"), goodSig)

	f.Fuzz(func(t *testing.T, payload []byte, signature string) {
		// Re-compute the expected signature for the payload.
		m := hmac.New(sha256.New, []byte(secret))
		m.Write(payload)
		expected := hex.EncodeToString(m.Sum(nil))

		got := svc.verifySignature(payload, signature)
		want := signature == expected
		if got != want {
			t.Fatalf("verifier mismatch: payload=%q sig=%q got=%v want=%v",
				payload, signature, got, want)
		}
	})
}
