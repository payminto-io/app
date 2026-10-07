package modules

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// EnvironmentModule is the process environment plus the guard money-moving modules share.
type EnvironmentModule struct {
	Environment environment.Environment
	Guard       *environment.ProcessGuard

	databaseName string
	databaseHost string
	testDatabase string
	allowName    string
	networkType  string
}

// WireEnvironment runs the boot gate against the configuration and returns the process guard.
// It needs no database so main can refuse before opening one.
func WireEnvironment(deps Deps) (*EnvironmentModule, error) {
	if deps.Config == nil {
		return nil, errors.New("environment: config is nil")
	}
	facts := deps.Config.BootFacts()
	if err := environment.CheckBoot(facts); err != nil {
		return nil, err
	}
	guard, err := environment.NewGuard(facts.Environment)
	if err != nil {
		return nil, err
	}
	return &EnvironmentModule{
		Environment:  facts.Environment,
		Guard:        guard,
		databaseName: facts.DatabaseName,
		databaseHost: facts.DatabaseHost,
		testDatabase: facts.TestDatabaseName,
		allowName:    facts.TestDatabaseAllowName,
		networkType:  strings.ToLower(strings.TrimSpace(facts.NetworkType)),
	}, nil
}

func (m *EnvironmentModule) policy(name string, stamp environment.Environment) environment.DatabasePolicy {
	return environment.DatabasePolicy{Name: name, TestName: m.testDatabase, AllowName: m.allowName, Host: m.databaseHost, Stamp: stamp}
}

// VerifyDatabase runs right after connecting and before any schema work: the name Postgres reports
// and the environment stamp are the authority, not the configured string.
func (m *EnvironmentModule) VerifyDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("environment: database is nil")
	}
	name, err := m.reportedDatabaseName(ctx, db)
	if err != nil {
		return err
	}
	stamp, err := m.readStamp(ctx, db)
	if err != nil {
		return err
	}
	return environment.CheckDatabase(m.Environment, m.policy(name, stamp))
}

// VerifySchema runs after schema preparation: the environment columns must exist and no key row may
// carry a visible prefix from the other environment.
func (m *EnvironmentModule) VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("environment: database is nil")
	}
	for _, t := range environmentColumns {
		if !db.Migrator().HasTable(t.table) {
			continue
		}
		if !db.Migrator().HasColumn(t.table, "environment") {
			return fmt.Errorf("%w: %s.environment is missing; apply migration %s", environment.ErrBoot, t.table, t.migration)
		}
	}
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return fmt.Errorf("%w: gateway_environment is missing; apply migration 2026100705_environment_isolation", environment.ErrBoot)
	}
	if !db.Migrator().HasTable("api_keys") {
		return nil
	}
	var mismatched int64
	err := db.WithContext(ctx).Table("api_keys").
		Where("prefix <> '' AND ((environment = ? AND substr(prefix, 1, 8) NOT IN (?, ?)) OR (environment = ? AND substr(prefix, 1, 8) NOT IN (?, ?)))",
			environment.Test, environment.Test.KeyPrefix(environment.SecretKey), environment.Test.KeyPrefix(environment.PublishableKey),
			environment.Live, environment.Live.KeyPrefix(environment.SecretKey), environment.Live.KeyPrefix(environment.PublishableKey)).
		Count(&mismatched).Error
	if err != nil {
		return fmt.Errorf("environment: inspect api_keys: %w", err)
	}
	if mismatched != 0 {
		return fmt.Errorf("%w: %d api_keys row(s) carry a prefix from the other environment", environment.ErrBoot, mismatched)
	}
	return nil
}

// environmentColumns are the tables whose rows carry an environment, with the migration that adds the column.
var environmentColumns = []struct{ table, migration string }{
	{"api_keys", "2026100705_environment_isolation"},
	{"ledger_accounts", "2026100705_environment_isolation"},
	{"ledger_journals", "2026100705_environment_isolation"},
	{"payment_links", "2026100706_links_payment_links"},
	{"payment_link_payments", "2026100706_links_payment_links"},
}

// dataTables are the tables whose rows prove a database already holds merchant or money data; any
// row in any of them means a process must not decide what the database is. One list, tested.
var dataTables = []string{
	"members", "external_platforms", "api_keys",
	"payment_requests", "deposits", "deposit_addresses", "withdrawals", "sweeps",
	"wallets", "address_pools", "secrets_vaults",
	"ledger_accounts", "ledger_journals", "fee_rules",
	"payment_links", "payment_link_payments",
}

// DataTables lists the tables Stamp refuses to overlook.
func DataTables() []string { return append([]string(nil), dataTables...) }

// modeKey is the configurations row the blockchain network mode is stamped under (config.EnforceModeMatch).
const modeKey = "mode"

// Stamp records the process environment on an empty, unstamped database. An unstamped database that
// already holds data is never stamped by a process; only the adoption commands decide what it is.
func (m *EnvironmentModule) Stamp(ctx context.Context, db *gorm.DB) error {
	return m.Finalize(ctx, db, "")
}

// Finalize is the last boot step and the first write: in one transaction it checks the network-mode
// row against networkType (writing it when absent and networkType is set) and stamps an empty,
// unstamped database. Nothing is written when any check refuses.
func (m *EnvironmentModule) Finalize(ctx context.Context, db *gorm.DB, networkType string) error {
	if db == nil {
		return errors.New("environment: database is nil")
	}
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return fmt.Errorf("%w: gateway_environment is missing; apply migration 2026100705_environment_isolation", environment.ErrBoot)
	}
	networkType = strings.ToLower(strings.TrimSpace(networkType))
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stamp, err := m.readStamp(ctx, tx)
		if err != nil {
			return err
		}
		if stamp != "" && stamp != m.Environment {
			return fmt.Errorf("%w: database is stamped %s, this process is %s", environment.ErrBoot, stamp, m.Environment)
		}
		mode, err := readMode(ctx, tx)
		if err != nil {
			return err
		}
		if networkType != "" && mode != "" && mode != networkType {
			return fmt.Errorf("%w: network mode mismatch: database is stamped as %q but BLOCKCHAIN_NETWORK_TYPE=%q; redeploy with a fresh database to switch modes", environment.ErrBoot, mode, networkType)
		}
		if stamp == "" {
			name, err := m.reportedDatabaseName(ctx, tx)
			if err != nil {
				return err
			}
			populated, err := holdsData(ctx, tx)
			if err != nil {
				return err
			}
			if populated != "" {
				return fmt.Errorf("%w: database %q holds %s rows but carries no environment stamp; a process never decides what existing data is. Adopt it explicitly: go run ./cmd/migrate adopt-live --confirm-adopt-live=%s (or adopt-test --confirm-adopt-test=%s)", environment.ErrBoot, name, populated, name, name)
			}
			row := environment.StampRow{ID: environment.StampID, Environment: m.Environment, StampedAt: time.Now().UTC()}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("environment: stamp database: %w", err)
			}
		}
		if networkType != "" && mode == "" && tx.Migrator().HasTable(&models.Configuration{}) {
			if err := tx.Create(&models.Configuration{Key: modeKey, Value: networkType, Category: "system"}).Error; err != nil {
				return fmt.Errorf("environment: stamp network mode: %w", err)
			}
		}
		return nil
	})
}

// holdsData names the first data table with rows, or "" when every one is empty or absent.
func holdsData(ctx context.Context, db *gorm.DB) (string, error) {
	for _, table := range dataTables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		var n int64
		if err := db.WithContext(ctx).Table(table).Count(&n).Error; err != nil {
			return "", fmt.Errorf("environment: count %s: %w", table, err)
		}
		if n > 0 {
			return table, nil
		}
	}
	return "", nil
}

// readMode returns the stamped network mode, or "" when the configurations table or row is absent.
func readMode(ctx context.Context, db *gorm.DB) (string, error) {
	if !db.Migrator().HasTable(&models.Configuration{}) {
		return "", nil
	}
	var row models.Configuration
	err := db.WithContext(ctx).Where("key = ?", modeKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("environment: read network mode: %w", err)
	}
	return strings.ToLower(strings.TrimSpace(row.Value)), nil
}

// adoptionManualStep is what an operator runs when the migration could not hand the relabel function to the ledger owner.
const adoptionManualStep = "as a role that may: ALTER FUNCTION ledger_adopt_environment(text) OWNER TO ledger_owner; GRANT EXECUTE ON FUNCTION ledger_adopt_environment(text) TO <migrator>; (docs/OPERATIONS.md, Environments)"

// verifyAdoptFunction refuses early when the relabel function is missing or not owned by the ledger owner.
func verifyAdoptFunction(ctx context.Context, db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	var row struct {
		Installed  bool
		OwnerMatch bool
	}
	err := db.WithContext(ctx).Raw(`
SELECT count(p.oid) > 0 AS installed,
       COALESCE(bool_and(p.proowner = c.relowner), false) AS owner_match
  FROM pg_class c
  JOIN pg_namespace n ON n.oid = c.relnamespace
  LEFT JOIN pg_proc p ON p.pronamespace = n.oid AND p.proname = 'ledger_adopt_environment'
 WHERE n.nspname = current_schema() AND c.relname = 'ledger_accounts'`).Scan(&row).Error
	if err != nil {
		return fmt.Errorf("environment: inspect ledger_adopt_environment: %w", err)
	}
	if !row.Installed {
		return fmt.Errorf("%w: ledger_adopt_environment is not installed (the migration could not hand it to the ledger owner); install it %s", environment.ErrBoot, adoptionManualStep)
	}
	if !row.OwnerMatch {
		return fmt.Errorf("%w: ledger_adopt_environment is not owned by the owner of the ledger tables; fix it %s", environment.ErrBoot, adoptionManualStep)
	}
	return nil
}

// AdoptResult counts what AdoptLive relabelled.
type AdoptResult struct {
	Database       string
	LedgerAccounts int64
	LedgerJournals int64
	APIKeys        int64
}

// AdoptLive relabels a database that has only ever served test money as live, once. It runs only
// from a live-configured process, only when the stamp says test and was never adopted before, and
// only when confirm names the database Postgres reports. Ledger rows move through
// ledger_adopt_environment (owned by ledger_owner); legacy keys (no visible prefix) become live;
// prefixed test keys stay test keys. Everything happens in one transaction.
func (m *EnvironmentModule) AdoptLive(ctx context.Context, db *gorm.DB, confirm string) (AdoptResult, error) {
	var result AdoptResult
	if db == nil {
		return result, errors.New("environment: database is nil")
	}
	if m.Environment != environment.Live {
		return result, fmt.Errorf("%w: adopt-live runs from a live-configured process, this one is %s", environment.ErrBoot, m.Environment)
	}
	name, err := m.reportedDatabaseName(ctx, db)
	if err != nil {
		return result, err
	}
	result.Database = name
	if environment.NormalizeDatabaseName(confirm) == "" || environment.NormalizeDatabaseName(confirm) != environment.NormalizeDatabaseName(name) {
		return result, fmt.Errorf("%w: --confirm-adopt-live must name the connected database %q", environment.ErrBoot, name)
	}
	if err := environment.CheckDatabase(environment.Live, m.policy(name, "")); err != nil {
		return result, err
	}
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return result, fmt.Errorf("%w: gateway_environment is missing; run migrations first", environment.ErrBoot)
	}
	if db.Migrator().HasTable("ledger_accounts") {
		if err := verifyAdoptFunction(ctx, db); err != nil {
			return result, err
		}
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stamp environment.StampRow
		stamped := true
		if err := tx.First(&stamp, environment.StampID).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("environment: read stamp: %w", err)
			}
			stamped = false
		}
		if stamped && (stamp.Environment != environment.Test || stamp.AdoptedFrom != nil) {
			return fmt.Errorf("%w: database is stamped %s (adopted from %v); adoption happens once", environment.ErrBoot, stamp.Environment, stamp.AdoptedFrom)
		}
		mode, err := readMode(ctx, tx)
		if err != nil {
			return err
		}
		if mode == "" {
			mode = m.networkType
		}
		if mode != "mainnet" {
			return fmt.Errorf("%w: live money is mainnet; this database's network mode is %q, so its rows are testnet money and cannot be adopted as live", environment.ErrBoot, mode)
		}
		if tx.Migrator().HasTable("ledger_accounts") {
			var counts struct {
				Accounts int64
				Journals int64
			}
			if err := tx.Raw(`SELECT accounts, journals FROM ledger_adopt_environment(?)`, string(environment.Live)).Scan(&counts).Error; err != nil {
				return fmt.Errorf("environment: relabel ledger: %w", err)
			}
			result.LedgerAccounts, result.LedgerJournals = counts.Accounts, counts.Journals
		}
		if tx.Migrator().HasTable("api_keys") {
			res := tx.Table("api_keys").Where("environment = ? AND prefix = ''", environment.Test).Update("environment", environment.Live)
			if res.Error != nil {
				return fmt.Errorf("environment: relabel api keys: %w", res.Error)
			}
			result.APIKeys = res.RowsAffected
		}
		now := time.Now().UTC()
		from := environment.Test
		if !stamped {
			row := environment.StampRow{ID: environment.StampID, Environment: environment.Live, StampedAt: now, AdoptedFrom: &from, AdoptedAt: &now}
			if err := tx.Create(&row).Error; err != nil {
				return fmt.Errorf("environment: stamp adopted database: %w", err)
			}
			return nil
		}
		res := tx.Model(&environment.StampRow{}).Where("id = ? AND environment = ?", environment.StampID, environment.Test).
			Updates(map[string]any{"environment": environment.Live, "adopted_from": &from, "adopted_at": &now})
		if res.Error != nil {
			return fmt.Errorf("environment: restamp: %w", res.Error)
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("%w: stamp changed under us; nothing adopted", environment.ErrBoot)
		}
		return nil
	})
	if err != nil {
		return AdoptResult{Database: name}, err
	}
	return result, nil
}

// AdoptTest stamps an unstamped database that already holds data as test. Nothing is relabelled:
// every pre-ticket row already reads test. It runs only from a test-configured process, only when the
// database carries no stamp, and only when confirm names the database Postgres reports.
func (m *EnvironmentModule) AdoptTest(ctx context.Context, db *gorm.DB, confirm string) (string, error) {
	if db == nil {
		return "", errors.New("environment: database is nil")
	}
	if m.Environment != environment.Test {
		return "", fmt.Errorf("%w: adopt-test runs from a test-configured process, this one is %s", environment.ErrBoot, m.Environment)
	}
	name, err := m.reportedDatabaseName(ctx, db)
	if err != nil {
		return "", err
	}
	if environment.NormalizeDatabaseName(confirm) == "" || environment.NormalizeDatabaseName(confirm) != environment.NormalizeDatabaseName(name) {
		return name, fmt.Errorf("%w: --confirm-adopt-test must name the connected database %q", environment.ErrBoot, name)
	}
	if err := environment.CheckDatabase(environment.Test, m.policy(name, "")); err != nil {
		return name, err
	}
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return name, fmt.Errorf("%w: gateway_environment is missing; run migrations first", environment.ErrBoot)
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stamp environment.StampRow
		if err := tx.First(&stamp, environment.StampID).Error; err == nil {
			return fmt.Errorf("%w: database is already stamped %s; adopt-test is for an unstamped database", environment.ErrBoot, stamp.Environment)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("environment: read stamp: %w", err)
		}
		mode, err := readMode(ctx, tx)
		if err != nil {
			return err
		}
		if mode == "mainnet" {
			return fmt.Errorf("%w: this database's network mode is mainnet; its rows are live money and cannot be stamped test", environment.ErrBoot)
		}
		row := environment.StampRow{ID: environment.StampID, Environment: environment.Test, StampedAt: time.Now().UTC()}
		if err := tx.Create(&row).Error; err != nil {
			return fmt.Errorf("environment: stamp database: %w", err)
		}
		return nil
	})
	return name, err
}

func (m *EnvironmentModule) reportedDatabaseName(ctx context.Context, db *gorm.DB) (string, error) {
	if db.Dialector.Name() != "postgres" {
		return m.databaseName, nil
	}
	var name string
	if err := db.WithContext(ctx).Raw(`SELECT current_database()`).Scan(&name).Error; err != nil {
		return "", fmt.Errorf("environment: read current_database(): %w", err)
	}
	return name, nil
}

// readStamp returns "" when the table or row does not exist yet.
func (m *EnvironmentModule) readStamp(ctx context.Context, db *gorm.DB) (environment.Environment, error) {
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return "", nil
	}
	var row environment.StampRow
	err := db.WithContext(ctx).First(&row, environment.StampID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("environment: read stamp: %w", err)
	}
	return row.Environment, nil
}
