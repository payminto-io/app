package ledger

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

var (
	tables    = []string{"ledger_accounts", "ledger_journals", "ledger_lines"}
	sequences = []string{"ledger_accounts_id_seq", "ledger_journals_id_seq", "ledger_lines_id_seq"}
	triggers  = []string{
		"ledger_accounts_append_only", "ledger_journals_append_only", "ledger_lines_append_only",
		"ledger_accounts_no_truncate", "ledger_journals_no_truncate", "ledger_lines_no_truncate",
		"ledger_journals_stamp_txid", "ledger_lines_same_transaction",
		"ledger_lines_balance", "ledger_journals_has_lines",
	}
	functions = []string{
		"ledger_reject_mutation", "ledger_check_journal_balance", "ledger_check_journal_has_lines",
		"ledger_stamp_journal_txid", "ledger_check_line_same_transaction",
	}
)

// MigrationVersion is the checksummed migration that creates the ledger; validate mode requires it recorded.
const MigrationVersion int64 = 2026100701

// ValidateSchema fails when any ledger table is missing or, on Postgres, when a trigger on the
// ledger tables in the current schema is missing or disabled, or a trigger function in that schema
// is missing or lacks the pinned search_path the shipped definition carries.
func ValidateSchema(db *gorm.DB) error {
	migrator := db.Migrator()
	for _, table := range tables {
		if !migrator.HasTable(table) {
			return fmt.Errorf("ledger: required table %q is missing; run the schema migrations before starting", table)
		}
	}
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	var missing []string
	for _, trigger := range triggers {
		var enabled int64
		err := db.Raw(`
SELECT count(*) FROM pg_trigger t
JOIN pg_class c ON c.oid = t.tgrelid
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE t.tgname = ? AND NOT t.tgisinternal AND t.tgenabled = 'O'
  AND n.nspname = current_schema() AND c.relname IN ('ledger_accounts', 'ledger_journals', 'ledger_lines')`, trigger).Scan(&enabled).Error
		if err != nil {
			return fmt.Errorf("ledger: inspect trigger %s: %w", trigger, err)
		}
		if enabled == 0 {
			missing = append(missing, "trigger "+trigger+" (missing or disabled)")
		}
	}
	for _, fn := range functions {
		var pinned int64
		err := db.Raw(`
SELECT count(*) FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE p.proname = ? AND n.nspname = current_schema()
  AND p.prorettype = 'trigger'::regtype
  AND 'search_path=pg_catalog, pg_temp' = ANY(p.proconfig)`, fn).Scan(&pinned).Error
		if err != nil {
			return fmt.Errorf("ledger: inspect function %s: %w", fn, err)
		}
		if pinned == 0 {
			missing = append(missing, "function "+fn+" (missing or without the pinned search_path)")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ledger: guarantees missing from the database: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateMigrationRecorded fails unless schema_migrations holds the ledger migration, applied clean.
func ValidateMigrationRecorded(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	var n int64
	err := db.Raw(`SELECT count(*) FROM schema_migrations WHERE version = ? AND dirty = false`, MigrationVersion).Scan(&n).Error
	if err != nil || n == 0 {
		return fmt.Errorf("ledger: migration %d is not recorded as applied (%v); run cmd/migrate", MigrationVersion, err)
	}
	return nil
}

type privilegeRow struct {
	Superuser      bool
	OwnerMember    bool
	CanMutate      bool
	CanPost        bool
	Tables         int64
	FunctionOwner  bool
	Functions      int64
	CreateDatabase bool
	CreateSchemas  string
}

// ValidatePrivileges fails when the connected role could rewrite the ledger: it must not be a
// superuser, must not own (or be a member of the owner of) the ledger tables or their trigger
// functions, must hold no UPDATE, DELETE, TRUNCATE or TRIGGER privilege on the tables, must be
// able to SELECT and INSERT, and must not be able to CREATE objects in the database or any schema
// (so nothing it creates can ever sit on a search path). TEMP is not refused: the trigger functions
// pin pg_temp last and qualify every reference, so a temporary object cannot shadow anything.
func ValidatePrivileges(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return errors.New("ledger: privilege validation needs Postgres")
	}
	var row privilegeRow
	err := db.Raw(`
SELECT
    (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) AS superuser,
    (SELECT COALESCE(bool_or(pg_has_role(current_user, p.proowner, 'MEMBER')), false)
       FROM pg_proc p JOIN pg_namespace pn ON pn.oid = p.pronamespace
      WHERE pn.nspname = current_schema() AND p.proname IN ('ledger_reject_mutation', 'ledger_check_journal_balance',
            'ledger_check_journal_has_lines', 'ledger_stamp_journal_txid', 'ledger_check_line_same_transaction')) AS function_owner,
    (SELECT count(*) FROM pg_proc p JOIN pg_namespace pn ON pn.oid = p.pronamespace
      WHERE pn.nspname = current_schema() AND p.proname IN ('ledger_reject_mutation', 'ledger_check_journal_balance',
            'ledger_check_journal_has_lines', 'ledger_stamp_journal_txid', 'ledger_check_line_same_transaction')) AS functions,
    has_database_privilege(current_user, current_database(), 'CREATE') AS create_database,
    (SELECT COALESCE(string_agg(nspname, ', ' ORDER BY nspname), '') FROM pg_namespace
      WHERE nspname NOT LIKE 'pg\_%' AND nspname <> 'information_schema'
        AND has_schema_privilege(current_user, oid, 'CREATE')) AS create_schemas,
    COALESCE(bool_or(pg_has_role(current_user, c.relowner, 'MEMBER')), false) AS owner_member,
    COALESCE(bool_or(
        has_table_privilege(current_user, c.oid, 'UPDATE')
        OR has_table_privilege(current_user, c.oid, 'DELETE')
        OR has_table_privilege(current_user, c.oid, 'TRUNCATE')
        OR has_table_privilege(current_user, c.oid, 'TRIGGER')), false) AS can_mutate,
    COALESCE(bool_and(
        has_table_privilege(current_user, c.oid, 'SELECT')
        AND has_table_privilege(current_user, c.oid, 'INSERT')), false) AS can_post,
    count(*) AS tables
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = current_schema() AND c.relname IN ('ledger_accounts', 'ledger_journals', 'ledger_lines')`).Scan(&row).Error
	if err != nil {
		return fmt.Errorf("ledger: inspect privileges: %w", err)
	}
	switch {
	case row.Tables != 3:
		return fmt.Errorf("ledger: expected 3 ledger tables in %s, found %d", "current schema", row.Tables)
	case row.Superuser:
		return errors.New("ledger: the application role is a superuser and can rewrite the ledger; connect as a role with SELECT, INSERT only (docs/OPERATIONS.md, Ledger roles)")
	case row.OwnerMember:
		return errors.New("ledger: the application role owns the ledger tables and can disable their triggers; run migrations as a privileged role and the server as a separate one (docs/OPERATIONS.md, Ledger roles)")
	case row.CanMutate:
		return errors.New("ledger: the application role holds UPDATE, DELETE, TRUNCATE or TRIGGER on the ledger tables; revoke them (docs/OPERATIONS.md, Ledger roles)")
	case !row.CanPost:
		return errors.New("ledger: the application role lacks SELECT or INSERT on the ledger tables (docs/OPERATIONS.md, Ledger roles)")
	case row.Functions != 5:
		return fmt.Errorf("ledger: expected 5 trigger functions in the current schema, found %d", row.Functions)
	case row.FunctionOwner:
		return errors.New("ledger: the application role owns a ledger trigger function and could replace it; transfer the functions to ledger_owner (docs/OPERATIONS.md, Ledger roles)")
	case row.CreateDatabase:
		return errors.New("ledger: the application role holds CREATE on the database and could add schemas; revoke it (docs/OPERATIONS.md, Ledger roles)")
	case row.CreateSchemas != "":
		return fmt.Errorf("ledger: the application role holds CREATE on schema(s) %s and could create shadowing objects; revoke it (docs/OPERATIONS.md, Ledger roles)", row.CreateSchemas)
	}
	return nil
}

var roleNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// GrantAppRole narrows role to SELECT and INSERT on the ledger (plus sequence use); idempotent.
// Run by cmd/migrate as the privileged role, after the migration moved ownership to ledger_owner.
func GrantAppRole(db *gorm.DB, role string) error {
	if !roleNamePattern.MatchString(role) {
		return fmt.Errorf("ledger: invalid role name %q", role)
	}
	quoted := `"` + role + `"`
	stmts := []string{
		"REVOKE ALL ON " + strings.Join(tables, ", ") + " FROM " + quoted,
		"GRANT SELECT, INSERT ON " + strings.Join(tables, ", ") + " TO " + quoted,
		"GRANT USAGE, SELECT ON SEQUENCE " + strings.Join(sequences, ", ") + " TO " + quoted,
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, stmt := range stmts {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("ledger: %s: %w", stmt, err)
			}
		}
		return nil
	})
}
