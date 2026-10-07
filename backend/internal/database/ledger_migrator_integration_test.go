//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// schemaOwnedBy builds the whole application schema as role, which owns the database the way a
// managed-Postgres admin user does but is not a cluster superuser, and returns a connection as it.
func schemaOwnedBy(t *testing.T, super *gorm.DB, role, attrs string) *gorm.DB {
	t.Helper()
	for _, stmt := range []string{
		`CREATE ROLE ` + role + ` LOGIN PASSWORD 'pw' ` + attrs,
		`ALTER DATABASE payminto_test OWNER TO ` + role,
	} {
		if err := super.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	db := ConnectTestDBAs(t, super, role, "pw")
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate as %s: %v", role, err)
	}
	if err := MigrateExpandSchema(db); err != nil {
		t.Fatalf("MigrateExpandSchema as %s: %v", role, err)
	}
	return db
}

func ledgerTableOwners(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var owners []string
	if err := db.Raw(`SELECT DISTINCT tableowner FROM pg_tables WHERE tablename IN ('ledger_accounts', 'ledger_journals', 'ledger_lines')`).Scan(&owners).Error; err != nil {
		t.Fatal(err)
	}
	return owners
}

func postOne(t *testing.T, db *gorm.DB) {
	t.Helper()
	_, err := ledger.New(db).Post(context.Background(), ledger.Journal{Kind: ledger.KindPayment, IdempotencyKey: "after-migration", Lines: []ledger.Line{
		{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}, Amount: decimal.NewFromInt(1)},
		{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m", Asset: "USDC", Kind: ledger.KindLiability}, Amount: decimal.NewFromInt(-1)},
	}})
	if err != nil {
		t.Fatalf("migrator must keep SELECT, INSERT after the migration: %v", err)
	}
}

func TestLedgerMigration_CreateRoleMigratorTransfersOwnership(t *testing.T) {
	super, cleanup := NewEmptyTestDB(t)
	defer cleanup()
	migrator := schemaOwnedBy(t, super, "migrator_createrole", "CREATEROLE")

	if _, err := ApplyMigrations(context.Background(), migrator); err != nil {
		t.Fatalf("ApplyMigrations as a CREATEROLE non-superuser: %v", err)
	}
	if owners := ledgerTableOwners(t, migrator); len(owners) != 1 || owners[0] != "ledger_owner" {
		t.Fatalf("ledger table owners = %v, want ledger_owner", owners)
	}
	postOne(t, migrator)
	var member bool
	if err := super.Raw(`SELECT pg_has_role('migrator_createrole', 'ledger_owner', 'SET')`).Scan(&member).Error; err != nil {
		t.Fatal(err)
	}
	if !member {
		t.Fatal("the migrator must be able to SET ROLE ledger_owner for future ledger migrations")
	}
}

func TestLedgerMigration_PlainMigratorTakesTheNoticePath(t *testing.T) {
	super, cleanup := NewEmptyTestDB(t)
	defer cleanup()
	migrator := schemaOwnedBy(t, super, "migrator_plain", "")

	if _, err := ApplyMigrations(context.Background(), migrator); err != nil {
		t.Fatalf("ApplyMigrations as a role without CREATEROLE must succeed on the notice path: %v", err)
	}
	if owners := ledgerTableOwners(t, migrator); len(owners) != 1 || owners[0] != "migrator_plain" {
		t.Fatalf("ledger table owners = %v, want migrator_plain (no transfer possible)", owners)
	}
	postOne(t, migrator)
}
