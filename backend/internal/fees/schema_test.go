package fees

import (
	"os"
	"testing"
)

func TestMigrationRepeatsSchemaVerbatim(t *testing.T) {
	raw, err := os.ReadFile("../database/migrations/2026100702_fees_rules.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != SchemaSQL() {
		t.Fatal("migration 2026100702_fees_rules.up.sql drifted from internal/fees/schema.sql; dev/test and production would diverge")
	}
}
