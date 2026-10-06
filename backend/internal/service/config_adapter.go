package service

import (
	"errors"

	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// ConfigRepoAdapter wraps a ConfigurationRepository to implement
// config.ConfigurationReader. Lets cmd/server call EnforceModeMatch without
// making the config package import the repository package.
type ConfigRepoAdapter struct {
	repo repository.ConfigurationRepository
}

// NewConfigRepoAdapter constructs a ConfigRepoAdapter wrapping the given
// ConfigurationRepository.
func NewConfigRepoAdapter(repo repository.ConfigurationRepository) *ConfigRepoAdapter {
	return &ConfigRepoAdapter{repo: repo}
}

// Get returns the configuration value for the given key.
func (a *ConfigRepoAdapter) Get(key string) (string, error) {
	c, err := a.repo.Get(key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		return "", err
	}
	return c.Value, nil
}

// Set stores or updates the configuration value for the given key.
func (a *ConfigRepoAdapter) Set(key, value string) error {
	return a.repo.Set(key, value)
}
