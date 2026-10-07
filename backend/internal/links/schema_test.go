package links

import (
	"os"
	"testing"
)

func TestMigrationRepeatsSchemaVerbatim(t *testing.T) {
	raw, err := os.ReadFile("../database/migrations/2026100710_links_payment_links.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != SchemaSQL() {
		t.Fatal("migration 2026100710_links_payment_links.up.sql drifted from internal/links/schema.sql; dev/test and production would diverge")
	}
}
