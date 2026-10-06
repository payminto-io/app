package blockchain

import (
	"fmt"
	"maps"
	"sync"
)

// AdapterRegistry is a thread-safe map of chain code → ChainAdapter. The
// ServiceRegistry constructs one at startup, registers one adapter per
// active blockchain row in the database, and hands it to any service that
// needs to pick an adapter by chain code (block processors, sweep service,
// withdrawal service).
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]ChainAdapter
}

// NewAdapterRegistry creates an empty registry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		adapters: make(map[string]ChainAdapter),
	}
}

// Register adds an adapter keyed by its Code(). Returns an error if an
// adapter for that code is already registered.
func (r *AdapterRegistry) Register(adapter ChainAdapter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	code := adapter.Code()
	if _, exists := r.adapters[code]; exists {
		return fmt.Errorf("adapter for chain %q already registered", code)
	}
	r.adapters[code] = adapter
	return nil
}

// Get returns the adapter for a chain code. Returns an error if no such
// adapter is registered.
func (r *AdapterRegistry) Get(code string) (ChainAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[code]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for chain %q", code)
	}
	return adapter, nil
}

// Has reports whether an adapter is registered for the given code.
func (r *AdapterRegistry) Has(code string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.adapters[code]
	return ok
}

// Codes returns the list of registered chain codes in unspecified order.
func (r *AdapterRegistry) Codes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.adapters))
	for code := range r.adapters {
		out = append(out, code)
	}
	return out
}

// All returns a copy of the adapter map for iteration.
func (r *AdapterRegistry) All() map[string]ChainAdapter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return maps.Clone(r.adapters)
}

// Len returns the number of registered adapters.
func (r *AdapterRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.adapters)
}

// CountByMode returns (mainnet, testnet) adapter counts.
func (r *AdapterRegistry) CountByMode() (mainnet, testnet int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.adapters {
		if a.IsMainnet() {
			mainnet++
		} else {
			testnet++
		}
	}
	return
}
