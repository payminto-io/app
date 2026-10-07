package connectors

import (
	"fmt"
	"slices"
	"sort"
	"sync"
)

// Registry holds providers by code. Selection is configuration, never an import change (MODULES.md).
type Registry struct {
	mu     sync.RWMutex
	byCode map[Code]Connector
}

func NewRegistry() *Registry {
	return &Registry{byCode: map[Code]Connector{}}
}

// Register adds a provider; a second registration of the same code is a wiring bug and fails.
func (r *Registry) Register(c Connector) error {
	if c == nil || c.Code() == "" {
		return fmt.Errorf("%w: nil connector or empty code", ErrInvalidRequest)
	}
	caps := c.Capabilities()
	if len(caps.RawStatuses) == 0 {
		return fmt.Errorf("%w: connector %q declares no raw statuses", ErrInvalidRequest, c.Code())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.byCode[c.Code()]; dup {
		return fmt.Errorf("%w: connector %q registered twice", ErrInvalidRequest, c.Code())
	}
	r.byCode[c.Code()] = c
	return nil
}

func (r *Registry) Get(code Code) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.byCode[code]
	return c, ok
}

func (r *Registry) Codes() []Code {
	r.mu.RLock()
	defer r.mu.RUnlock()
	codes := make([]Code, 0, len(r.byCode))
	for code := range r.byCode {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

// Supports reports whether caps accept the method.
func (c Capabilities) Supports(m Method) bool {
	return slices.Contains(c.Methods, m)
}
