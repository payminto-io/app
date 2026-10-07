package solana

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ParseKeypair accepts the three forms a Solana secret key travels in: a base58 64-byte secret, the
// JSON byte array the Solana CLI writes, or a path to such a file.
func ParseKeypair(s string) (Ed25519Signer, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Ed25519Signer{}, errors.New("solana: empty keypair")
	}
	if strings.HasPrefix(s, "[") {
		return keypairFromJSON([]byte(s))
	}
	if raw := decodeBase58(s); len(raw) == ed25519.PrivateKeySize {
		return NewEd25519Signer(raw)
	}
	if raw := decodeBase58(s); len(raw) == ed25519.SeedSize {
		return NewEd25519Signer(ed25519.NewKeyFromSeed(raw))
	}
	data, err := os.ReadFile(s)
	if err != nil {
		return Ed25519Signer{}, fmt.Errorf("solana: keypair is not base58, JSON or a readable file: %w", err)
	}
	return keypairFromJSON(data)
}

func keypairFromJSON(data []byte) (Ed25519Signer, error) {
	var bytes []byte
	if err := json.Unmarshal(data, &bytes); err != nil {
		return Ed25519Signer{}, fmt.Errorf("solana: keypair JSON: %w", err)
	}
	if len(bytes) != ed25519.PrivateKeySize {
		return Ed25519Signer{}, fmt.Errorf("solana: keypair JSON has %d bytes, want %d", len(bytes), ed25519.PrivateKeySize)
	}
	return NewEd25519Signer(bytes)
}
