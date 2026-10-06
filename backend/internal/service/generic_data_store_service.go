package service

import (
	"context"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// GenericDataStoreService manages the namespaced KV store used for idempotency
// keys, short-lived locks, cached rates, and any state that does not fit a
// domain table.
type GenericDataStoreService struct {
	repo repository.GenericDataStoreRepository
}

// NewGenericDataStoreService constructs a GenericDataStoreService.
func NewGenericDataStoreService(repo repository.GenericDataStoreRepository) *GenericDataStoreService {
	return &GenericDataStoreService{repo: repo}
}

// Get returns the value for namespace+key. Returns an error when the key does
// not exist or has expired.
func (s *GenericDataStoreService) Get(_ context.Context, namespace, key string) (*models.GenericDataStore, error) {
	entry, err := s.repo.Get(namespace, key)
	if err != nil {
		return nil, fmt.Errorf("generic data store get %s/%s: %w", namespace, key, err)
	}
	return entry, nil
}

// Set upserts namespace+key with the given value. ttl is the lifetime in
// seconds; nil means no expiry.
func (s *GenericDataStoreService) Set(_ context.Context, namespace, key, value string, ttlSeconds *int64) error {
	if err := s.repo.Set(namespace, key, value, ttlSeconds); err != nil {
		return fmt.Errorf("generic data store set %s/%s: %w", namespace, key, err)
	}
	return nil
}

// Delete removes the entry for namespace+key. Returns an error on DB failure;
// a missing key is not an error (rowsAffected is ignored here — callers that
// need the count, such as NonceService, call the repository directly).
func (s *GenericDataStoreService) Delete(_ context.Context, namespace, key string) error {
	_, err := s.repo.Delete(namespace, key)
	if err != nil {
		return fmt.Errorf("generic data store delete %s/%s: %w", namespace, key, err)
	}
	return nil
}

// ListByNamespace returns all non-expired entries in a namespace.
func (s *GenericDataStoreService) ListByNamespace(_ context.Context, namespace string) ([]models.GenericDataStore, error) {
	return s.repo.ListByNamespace(namespace)
}

// PurgeExpired deletes all TTL-expired entries. Should be called periodically.
func (s *GenericDataStoreService) PurgeExpired(_ context.Context) error {
	return s.repo.DeleteExpired()
}
