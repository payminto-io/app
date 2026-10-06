package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

const configCacheTTL = 60 * time.Second

// configCacheEntry is a single cached configuration value.
// found=false means the key was absent in the DB (a cached miss), so we avoid
// a DB round-trip on every call for the duration of the TTL.
type configCacheEntry struct {
	value     string
	found     bool // false = cached miss (key not in DB)
	expiresAt time.Time
}

// ConfigurationService provides runtime key-value configuration with an
// in-memory 60-second cache so hot paths don't hammer the database.
// Missing keys are also cached for the TTL duration (negative caching).
type ConfigurationService struct {
	repo repository.ConfigurationRepository

	mu    sync.RWMutex
	cache map[string]*configCacheEntry
}

// NewConfigurationService constructs a ConfigurationService.
func NewConfigurationService(repo repository.ConfigurationRepository) *ConfigurationService {
	return &ConfigurationService{
		repo:  repo,
		cache: make(map[string]*configCacheEntry),
	}
}

// Get returns the raw Configuration model for a key.
func (s *ConfigurationService) Get(_ context.Context, key string) (*models.Configuration, error) {
	return s.repo.Get(key)
}

// Set upserts a key-value pair (no description, no category).
func (s *ConfigurationService) Set(_ context.Context, key, value, description string) error {
	if err := s.repo.SetWithDescription(key, value, description, "runtime"); err != nil {
		return fmt.Errorf("configuration set %q: %w", key, err)
	}
	s.invalidate(key)
	return nil
}

// GetString returns the string value for the key, or defaultVal when the key
// is absent.
func (s *ConfigurationService) GetString(ctx context.Context, key, defaultVal string) string {
	v, found, err := s.cachedValue(ctx, key)
	if err != nil || !found {
		return defaultVal
	}
	return v
}

// GetInt returns the integer value for the key, or defaultVal when absent or
// unparseable.
func (s *ConfigurationService) GetInt(ctx context.Context, key string, defaultVal int) int {
	v, found, err := s.cachedValue(ctx, key)
	if err != nil || !found {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetBool returns the boolean value for the key ("true"/"1"/"yes"), or
// defaultVal when absent or unparseable.
func (s *ConfigurationService) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	v, found, err := s.cachedValue(ctx, key)
	if err != nil || !found {
		return defaultVal
	}
	switch v {
	case "true", "1", "yes":
		return true
	case "false", "0", "no":
		return false
	}
	return defaultVal
}

// ListAll returns all configuration entries.
func (s *ConfigurationService) ListAll(_ context.Context) ([]models.Configuration, error) {
	return s.repo.List()
}

// cachedValue returns the value string and whether the key exists, using a
// 60 s in-memory cache. Missing keys are also cached (negative cache) so
// repeated lookups for absent keys do not hit the database every call.
func (s *ConfigurationService) cachedValue(_ context.Context, key string) (value string, found bool, err error) {
	now := time.Now()

	// Fast path.
	s.mu.RLock()
	if entry, ok := s.cache[key]; ok && now.Before(entry.expiresAt) {
		v := entry.value
		f := entry.found
		s.mu.RUnlock()
		return v, f, nil
	}
	s.mu.RUnlock()

	// Slow path: load from DB.
	cfg, dbErr := s.repo.Get(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	if dbErr != nil {
		if errors.Is(dbErr, gorm.ErrRecordNotFound) {
			// Cache the miss so subsequent calls skip the DB for the TTL window.
			s.cache[key] = &configCacheEntry{
				value:     "",
				found:     false,
				expiresAt: now.Add(configCacheTTL),
			}
			return "", false, nil
		}
		return "", false, dbErr
	}

	s.cache[key] = &configCacheEntry{
		value:     cfg.Value,
		found:     true,
		expiresAt: now.Add(configCacheTTL),
	}
	return cfg.Value, true, nil
}

// invalidate removes a key from the in-memory cache after a write.
func (s *ConfigurationService) invalidate(key string) {
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}
