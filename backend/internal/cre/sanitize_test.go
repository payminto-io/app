package cre

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeStripsURLsAndTokens(t *testing.T) {
	err := errors.New(`Post "https://mainnet.example.io/v2/8f3a1c9d2e7b4a6f5c8d9e0f1a2b3c4d": dial tcp: connection refused; bearer sk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345`)
	got := SanitizeError(err)
	for _, secret := range []string{"8f3a1c9d2e7b4a6f5c8d9e0f1a2b3c4d", "mainnet.example.io", "sk_live_ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"} {
		if strings.Contains(got, secret) {
			t.Fatalf("%q leaked into %q", secret, got)
		}
	}
	if !strings.Contains(got, "connection refused") {
		t.Fatalf("classification lost: %q", got)
	}
	if SanitizeError(nil) != "" {
		t.Fatal("nil")
	}
}
