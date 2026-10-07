package cre

import (
	"os"
	"testing"
)

func TestMigrationRepeatsSchemaVerbatim(t *testing.T) {
	raw, err := os.ReadFile("../database/migrations/2026100707_cre_attestations.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != SchemaSQL() {
		t.Fatal("migration 2026100707_cre_attestations.up.sql drifted from internal/cre/schema.sql")
	}
}
