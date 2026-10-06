package api

import (
	"testing"
)

func TestNewRouter_NotNil(t *testing.T) {
	cfg := RouterConfig{
		Host: "http://localhost:8080",
	}
	// Router creation with nil DB/services should not panic during setup
	// (it only panics when requests hit the DB)
	r := NewRouter(cfg)
	if r == nil {
		t.Error("expected non-nil router")
	}
}
