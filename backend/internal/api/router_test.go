package api

import (
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/modules"
	"testing"
)

func TestNewRouter_NotNil(t *testing.T) {
	cfg := RouterConfig{
		Host:        "http://localhost:8080",
		Environment: &modules.EnvironmentModule{Environment: environment.Test},
	}
	// Router creation with nil DB/services should not panic during setup
	// (it only panics when requests hit the DB)
	r := NewRouter(cfg)
	if r == nil {
		t.Error("expected non-nil router")
	}
}

func TestNewRouter_RefusesToBuildWithoutAnEnvironment(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewRouter built a router with no environment")
		}
	}()
	NewRouter(RouterConfig{Host: "http://localhost:8080"})
}
