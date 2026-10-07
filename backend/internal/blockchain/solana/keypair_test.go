package solana

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil/base58"
)

// I3: a mistyped secret must fail with a fixed message that never contains the input.
func TestParseKeypair_NeverEchoesTheInput(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	good := base58.Encode(priv)
	typo := good[:len(good)-1] + "typo"
	_, err := ParseKeypair(typo)
	if err == nil {
		t.Fatal("typo accepted")
	}
	if strings.Contains(err.Error(), typo) || strings.Contains(err.Error(), good[:12]) {
		t.Fatalf("SECRET ECHOED in error: %v", err)
	}
	if _, err := ParseKeypair("[1,2,3]"); err == nil || strings.Contains(err.Error(), "[1,2,3]") {
		t.Fatalf("short JSON array: %v", err)
	}
}

func TestParseKeypair_AcceptsBase58JSONAndExplicitFile(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ParseKeypair(base58.Encode(priv))
	if err != nil || string(s.Key) != string(priv) {
		t.Fatalf("base58 64: %v", err)
	}
	s, err = ParseKeypair(base58.Encode(priv.Seed()))
	if err != nil || string(s.Key) != string(priv) {
		t.Fatalf("base58 seed: %v", err)
	}
	ints := make([]int, len(priv))
	for i, b := range priv {
		ints[i] = int(b)
	}
	raw, _ := json.Marshal(ints)
	if s, err = ParseKeypair(string(raw)); err != nil || string(s.Key) != string(priv) {
		t.Fatalf("json: %v", err)
	}
	path := filepath.Join(t.TempDir(), "id.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if s, err = ParseKeypair("file:" + path); err != nil || string(s.Key) != string(priv) {
		t.Fatalf("file: %v", err)
	}
	// A bare path is not read: the value is treated as a key and refused without echo.
	if _, err := ParseKeypair(path); err == nil || strings.Contains(err.Error(), path) {
		t.Fatalf("bare path: %v", err)
	}
	if _, err := ParseKeypair("file:" + filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing file accepted")
	}
}
