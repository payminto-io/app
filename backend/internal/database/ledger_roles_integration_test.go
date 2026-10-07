//go:build integration

package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	testAppRole     = "payminto_app_test"
	testAppPassword = "app_test_password"
)

// migratedWithAppRole applies the checksummed migrations as the privileged test user, creates an
// application role with ordinary DML on every table, narrows it on the ledger, and connects as it.
func migratedWithAppRole(t *testing.T) (owner *gorm.DB, app *gorm.DB, cleanup func()) {
	t.Helper()
	owner, cleanup = NewTestDB(t)
	if _, err := ApplyMigrations(context.Background(), owner); err != nil {
		cleanup()
		t.Fatalf("ApplyMigrations: %v", err)
	}
	for _, stmt := range []string{
		`CREATE ROLE ` + testAppRole + ` LOGIN PASSWORD '` + testAppPassword + `'`,
		`GRANT USAGE ON SCHEMA public TO ` + testAppRole,
		`GRANT ALL ON ALL TABLES IN SCHEMA public TO ` + testAppRole,
		`GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO ` + testAppRole,
	} {
		if err := owner.Exec(stmt).Error; err != nil {
			cleanup()
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if err := ledger.GrantAppRole(owner, testAppRole); err != nil {
		cleanup()
		t.Fatalf("GrantAppRole: %v", err)
	}
	app = ConnectTestDBAs(t, owner, testAppRole, testAppPassword)
	return owner, app, cleanup
}

func TestLedgerMigrationMovesOwnershipAwayFromTheMigrator(t *testing.T) {
	owner, _, cleanup := migratedWithAppRole(t)
	defer cleanup()
	var owners []string
	if err := owner.Raw(`SELECT DISTINCT tableowner FROM pg_tables WHERE tablename IN ('ledger_accounts', 'ledger_journals', 'ledger_lines')`).Scan(&owners).Error; err != nil {
		t.Fatal(err)
	}
	if len(owners) != 1 || owners[0] != "ledger_owner" {
		t.Fatalf("ledger table owners = %v, want only ledger_owner", owners)
	}
	var fnOwners []string
	if err := owner.Raw(`SELECT DISTINCT pg_get_userbyid(proowner) FROM pg_proc WHERE proname LIKE 'ledger_%'`).Scan(&fnOwners).Error; err != nil {
		t.Fatal(err)
	}
	if len(fnOwners) != 1 || fnOwners[0] != "ledger_owner" {
		t.Fatalf("ledger function owners = %v, want only ledger_owner", fnOwners)
	}
}

func TestLedgerAppRoleCanPostButCannotMutateOrWeakenTheLedger(t *testing.T) {
	_, app, cleanup := migratedWithAppRole(t)
	defer cleanup()
	s := ledger.New(app)
	ctx := context.Background()
	hot := ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}
	id, err := s.Post(ctx, ledger.Journal{
		Kind:           ledger.KindPayment,
		IdempotencyKey: "app:1",
		Lines: []ledger.Line{
			{Account: hot, Amount: decimal.NewFromInt(5)},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m", Asset: "USDC", Kind: ledger.KindLiability}, Amount: decimal.NewFromInt(-5)},
		},
	})
	if err != nil {
		t.Fatalf("app role must be able to post: %v", err)
	}
	hotID, _ := s.AccountID(ctx, hot)
	if bal, err := s.Balance(ctx, hotID); err != nil || !bal.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("app role must be able to read balances: %s, %v", bal, err)
	}

	denied := map[string]string{
		`UPDATE ledger_lines SET amount = amount + 1 WHERE journal_id = ?`:  "permission denied",
		`DELETE FROM ledger_lines WHERE journal_id = ?`:                     "permission denied",
		`TRUNCATE ledger_lines`:                                             "permission denied",
		`ALTER TABLE ledger_lines DISABLE TRIGGER ledger_lines_append_only`: "must be owner",
		// Refused either as non-owner of the function or, on PG15+, for lacking CREATE on the schema.
		`CREATE OR REPLACE FUNCTION ledger_check_journal_balance() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NULL; END; $$`: "permission denied|must be owner",
		`DROP TRIGGER ledger_lines_balance ON ledger_lines`:                                                                           "must be owner",
	}
	for stmt, want := range denied {
		var err error
		if strings.Contains(stmt, "?") {
			err = app.Exec(stmt, id).Error
		} else {
			err = app.Exec(stmt).Error
		}
		if err == nil || !containsAny(err.Error(), strings.Split(want, "|")) {
			t.Errorf("%s: got %v, want %q", stmt, err, want)
		}
	}
}

func TestPrepareSchema_LiveAcceptsTheAppRoleAndRefusesTheOwner(t *testing.T) {
	owner, app, cleanup := migratedWithAppRole(t)
	defer cleanup()
	for _, env := range []string{config.EnvironmentStaging, config.EnvironmentProduction} {
		if err := PrepareSchema(app, env, config.SchemaModeValidate); err != nil {
			t.Fatalf("PrepareSchema(app role, %s) = %v, want nil", env, err)
		}
		err := PrepareSchema(owner, env, config.SchemaModeValidate)
		if err == nil || !strings.Contains(err.Error(), "ledger") {
			t.Fatalf("PrepareSchema(owner, %s) = %v, want refusal: the connection can rewrite the ledger", env, err)
		}
	}
	// Development may run as the owner; the guarantee is for live only.
	if err := PrepareSchema(owner, config.EnvironmentDevelopment, config.SchemaModeValidate); err != nil {
		t.Fatalf("PrepareSchema(owner, development) = %v, want nil", err)
	}
}

func TestPrepareSchema_ValidateRequiresTheLedgerMigrationRecord(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	// Tables exist from MigrateExpandSchema, but nothing recorded migration 2026100701.
	err := PrepareSchema(db, config.EnvironmentDevelopment, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "2026100701") {
		t.Fatalf("PrepareSchema() = %v, want missing migration record", err)
	}
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// grantSchemaCreate gives role a schema it may create in, the setup a database owner or a "fix-up" session has.
func grantSchemaCreate(t *testing.T, owner *gorm.DB, role string) {
	t.Helper()
	for _, stmt := range []string{
		`CREATE SCHEMA IF NOT EXISTS evil`,
		`GRANT USAGE, CREATE ON SCHEMA evil TO ` + role,
	} {
		if err := owner.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestLedgerTriggersCannotBeShadowedThroughSearchPath(t *testing.T) {
	owner, app, cleanup := migratedWithAppRole(t)
	defer cleanup()
	grantSchemaCreate(t, owner, testAppRole)
	s := ledger.New(app)
	ctx := context.Background()
	hot := ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: "USDC", Kind: ledger.KindAsset}
	member := ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "m", Asset: "USDC", Kind: ledger.KindLiability}
	id, err := s.Post(ctx, ledger.Journal{Kind: ledger.KindPayment, IdempotencyKey: "seed", Lines: []ledger.Line{
		{Account: hot, Amount: decimal.NewFromInt(5)}, {Account: member, Amount: decimal.NewFromInt(-5)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	hotID, _ := s.AccountID(ctx, hot)

	// Bypass 1: a view named ledger_lines that reports every amount as zero, first on the search path.
	for _, stmt := range []string{
		`CREATE VIEW evil.ledger_lines AS SELECT id, journal_id, account_id, asset, 0::numeric AS amount, created_at FROM public.ledger_lines`,
		`CREATE VIEW evil.ledger_journals AS SELECT * FROM public.ledger_journals`,
	} {
		if err := app.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	err = app.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL search_path = evil, public`).Error; err != nil {
			return err
		}
		j := ledger.JournalRow{Kind: ledger.KindPayment, ReferenceType: "t", ReferenceID: "1", IdempotencyKey: "shadow:1", RequestHash: strings.Repeat("a", 64), PostedAt: time.Now(), Metadata: ledger.Metadata{}}
		if err := tx.Table("public.ledger_journals").Create(&j).Error; err != nil {
			return err
		}
		return tx.Table("public.ledger_lines").Create(&ledger.LineRow{JournalID: j.ID, AccountID: hotID, Asset: "USDC", Amount: decimal.NewFromInt(1000)}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "does not balance") {
		t.Fatalf("one-sided journal through a shadowing view = %v, want balance trigger error", err)
	}

	// Bypass 2: a txid_current() that returns the posting txid of a committed journal.
	var oldTxid int64
	if err := owner.Raw(`SELECT posting_txid FROM ledger_journals WHERE id = ?`, id).Scan(&oldTxid).Error; err != nil {
		t.Fatal(err)
	}
	if err := app.Exec(fmt.Sprintf(`CREATE FUNCTION evil.txid_current() RETURNS bigint LANGUAGE sql AS 'SELECT %d::bigint'`, oldTxid)).Error; err != nil {
		t.Fatal(err)
	}
	err = app.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET LOCAL search_path = evil, pg_catalog, public`).Error; err != nil {
			return err
		}
		return tx.Table("public.ledger_lines").Create(&[]ledger.LineRow{
			{JournalID: id, AccountID: hotID, Asset: "USDC", Amount: decimal.NewFromInt(1000)},
			{JournalID: id, AccountID: hotID, Asset: "USDC", Amount: decimal.NewFromInt(-1000)},
		}).Error
	})
	if err == nil || !strings.Contains(err.Error(), "sealed") {
		t.Fatalf("late lines with a forged txid_current = %v, want sealed-journal rejection", err)
	}
	if bal, err := s.Balance(ctx, hotID); err != nil || !bal.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("balance = %s, %v; want 5", bal, err)
	}

	// A role that can create objects anywhere is refused at boot even though the triggers now hold.
	err = PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "CREATE") {
		t.Fatalf("PrepareSchema(app role with CREATE on a schema) = %v, want refusal", err)
	}
}

func TestLedgerBootChecksEffectNotNames(t *testing.T) {
	owner, app, cleanup := migratedWithAppRole(t)
	defer cleanup()
	if err := PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	// A disabled trigger still has its name.
	if err := owner.Exec(`ALTER TABLE ledger_lines DISABLE TRIGGER ledger_lines_balance`).Error; err != nil {
		t.Fatal(err)
	}
	err := PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "ledger_lines_balance") {
		t.Fatalf("PrepareSchema with a disabled trigger = %v, want refusal naming it", err)
	}
	if err := owner.Exec(`ALTER TABLE ledger_lines ENABLE TRIGGER ledger_lines_balance`).Error; err != nil {
		t.Fatal(err)
	}
	// A function the app role owns can be replaced with a no-op.
	if err := owner.Exec(`ALTER FUNCTION ledger_check_journal_balance() OWNER TO ` + testAppRole).Error; err != nil {
		t.Fatal(err)
	}
	err = PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "function") {
		t.Fatalf("PrepareSchema with an app-owned trigger function = %v, want refusal", err)
	}
	if err := owner.Exec(`ALTER FUNCTION ledger_check_journal_balance() OWNER TO ledger_owner`).Error; err != nil {
		t.Fatal(err)
	}
	// A function without the pinned search_path is not the shipped definition.
	if err := owner.Exec(`ALTER FUNCTION ledger_check_journal_balance() RESET search_path`).Error; err != nil {
		t.Fatal(err)
	}
	err = PrepareSchema(app, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "search_path") {
		t.Fatalf("PrepareSchema with an unpinned trigger function = %v, want refusal", err)
	}
}
