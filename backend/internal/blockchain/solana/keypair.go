package solana

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// ErrKeypairFormat is returned for any unparsable secret; it never carries the input.
var ErrKeypairFormat = errors.New("solana: keypair must be a 64-byte or 32-byte base58 secret, a JSON byte array, or file:<path>")

// ParseKeypair accepts a base58 secret (64-byte key or 32-byte seed), the JSON byte array the
// Solana CLI writes, or "file:<path>" to such a file. A bare value is never read as a path and
// no error mentions the input, so a mistyped secret cannot reach a log.
func ParseKeypair(s string) (Ed25519Signer, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Ed25519Signer{}, ErrKeypairFormat
	}
	if path, ok := strings.CutPrefix(s, "file:"); ok {
		data, err := os.ReadFile(path)
		if err != nil {
			return Ed25519Signer{}, errors.New("solana: keypair file could not be read")
		}
		return keypairFromJSON(data)
	}
	if strings.HasPrefix(s, "[") {
		return keypairFromJSON([]byte(s))
	}
	raw := decodeBase58(s)
	switch len(raw) {
	case ed25519.PrivateKeySize:
		return NewEd25519Signer(raw)
	case ed25519.SeedSize:
		return NewEd25519Signer(ed25519.NewKeyFromSeed(raw))
	}
	return Ed25519Signer{}, ErrKeypairFormat
}

func keypairFromJSON(data []byte) (Ed25519Signer, error) {
	var bytes []byte
	if err := json.Unmarshal(data, &bytes); err != nil || len(bytes) != ed25519.PrivateKeySize {
		return Ed25519Signer{}, ErrKeypairFormat
	}
	return NewEd25519Signer(bytes)
}
