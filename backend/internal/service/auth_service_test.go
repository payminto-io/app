package service

import "testing"

func TestGenerateAPIKey_Length(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}
	if len(key) < 32 {
		t.Errorf("expected key length >= 32, got %d", len(key))
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	key := "test-api-key-123"
	h1 := HashAPIKey(key)
	h2 := HashAPIKey(key)
	if h1 != h2 {
		t.Errorf("same key should produce same hash")
	}
}

func TestHashAPIKey_DifferentKeys(t *testing.T) {
	h1 := HashAPIKey("key-1")
	h2 := HashAPIKey("key-2")
	if h1 == h2 {
		t.Errorf("different keys should produce different hashes")
	}
}
