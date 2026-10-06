package repository

import (
	"crypto/sha256"
	"encoding/hex"
)

// Sha256Sum returns the lowercase hex SHA-256 hash of s. Used for API key and
// refresh-token storage so we never persist plaintext secrets. Mirrors PayRam's
// utils_repo.go helper.
func Sha256Sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
