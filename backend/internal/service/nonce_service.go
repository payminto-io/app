package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

const (
	nonceNamespace  = "nonce"
	nonceTTLSeconds = int64(600) // 10 minutes
)

// ErrNonceInvalid is returned when a nonce does not exist, has expired, or
// has already been consumed.
var ErrNonceInvalid = errors.New("nonce invalid or already consumed")

// NonceService generates and verifies single-use nonces for anti-replay
// protection in auth flows and signed requests. Backed by GenericDataStore.
type NonceService struct {
	gds *GenericDataStoreService
}

// NewNonceService constructs a NonceService.
func NewNonceService(gds *GenericDataStoreService) *NonceService {
	return &NonceService{gds: gds}
}

// Generate creates a cryptographically random 32-byte nonce, stores it in the
// GenericDataStore under the "nonce" namespace with a 10-minute TTL, and
// returns the hex-encoded nonce string.
func (s *NonceService) Generate(ctx context.Context, purpose string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("nonce generate: %w", err)
	}
	nonce := hex.EncodeToString(b)
	key := fmt.Sprintf("%s:%s", purpose, nonce)
	ttl := nonceTTLSeconds
	if err := s.gds.Set(ctx, nonceNamespace, key, "1", &ttl); err != nil {
		return "", fmt.Errorf("nonce store: %w", err)
	}
	return nonce, nil
}

// Consume atomically consumes a nonce by issuing a single DELETE against the
// database. If the row was already gone (expired or consumed by a concurrent
// caller) rowsAffected will be 0, and ErrNonceInvalid is returned. This
// eliminates the Get → compare → Delete TOCTOU race.
func (s *NonceService) Consume(ctx context.Context, nonce, purpose string) error {
	key := fmt.Sprintf("%s:%s", purpose, nonce)
	rowsAffected, err := s.gds.repo.Delete(nonceNamespace, key)
	if err != nil {
		return fmt.Errorf("consume nonce: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNonceInvalid
	}
	return nil
}
