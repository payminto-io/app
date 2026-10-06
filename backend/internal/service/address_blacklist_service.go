package service

import (
	"context"
	"strings"
	"sync"
)

// AddressBlacklistService holds a cached set of blocked (chain, address)
// pairs loaded from the configurations table. Callers check against it
// before accepting deposits or broadcasting withdrawals.
//
// Addresses are normalised to lowercase for case-insensitive matching.
// Chain codes are case-insensitive as well.
type AddressBlacklistService struct {
	configSvc *ConfigurationService

	mu      sync.RWMutex
	entries map[string]struct{} // key = "chain|address"
}

// NewAddressBlacklistService wires the service.
func NewAddressBlacklistService(configSvc *ConfigurationService) *AddressBlacklistService {
	return &AddressBlacklistService{
		configSvc: configSvc,
		entries:   make(map[string]struct{}),
	}
}

// Load reads the comma-separated blacklist from the `address_blacklist`
// configuration key and rebuilds the in-memory set. Each entry is
// expected in the form "<chain>:<address>", e.g. "ETH:0xabc...".
// Unparseable entries are skipped.
func (s *AddressBlacklistService) Load() error {
	if s.configSvc == nil {
		return nil
	}
	cfg, err := s.configSvc.Get(context.Background(), "address_blacklist")
	var raw string
	if err == nil && cfg != nil {
		raw = cfg.Value
	}
	if err != nil || raw == "" {
		s.mu.Lock()
		s.entries = make(map[string]struct{})
		s.mu.Unlock()
		return nil
	}

	next := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		chain, addr, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		next[key(chain, addr)] = struct{}{}
	}

	s.mu.Lock()
	s.entries = next
	s.mu.Unlock()
	return nil
}

// IsBlacklisted reports whether the given (chain, address) pair is blocked.
// Matching is case-insensitive.
func (s *AddressBlacklistService) IsBlacklisted(chain, address string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.entries[key(chain, address)]
	return ok
}

func key(chain, address string) string {
	return strings.ToLower(chain) + "|" + strings.ToLower(address)
}
