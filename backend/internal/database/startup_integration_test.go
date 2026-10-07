//go:build integration

package database

import (
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
)

func TestPrepareSchema_PostgresValidatesCurrentAndRejectsMixedSchema(t *testing.T) {
	owner, app, cleanup := migratedWithAppRole(t)
	defer cleanup()

	if err := PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate); err != nil {
		t.Fatalf("validate current PostgreSQL schema: %v", err)
	}

	if err := owner.Exec(`ALTER TABLE payment_requests ADD COLUMN status varchar(20)`).Error; err != nil {
		t.Fatalf("add legacy discriminator: %v", err)
	}
	err := PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "mixed") {
		t.Fatalf("PrepareSchema() error = %v, want mixed-schema rejection", err)
	}
}
