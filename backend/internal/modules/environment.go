package modules

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"gorm.io/gorm"
)

// EnvironmentModule is the process environment plus the guard money-moving modules share.
type EnvironmentModule struct {
	Environment environment.Environment
	Guard       *environment.ProcessGuard

	databaseName string
	databaseHost string
	testDatabase string
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
	}, nil
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
	return environment.CheckDatabase(m.Environment, name, m.testDatabase, m.databaseHost, stamp)
}

// VerifySchema runs after schema preparation: the environment columns must exist and no key row may
// carry a visible prefix from the other environment.
func (m *EnvironmentModule) VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("environment: database is nil")
	}
	for _, table := range []string{"api_keys", "ledger_accounts", "ledger_journals"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if !db.Migrator().HasColumn(table, "environment") {
			return fmt.Errorf("%w: %s.environment is missing; apply migration 2026100705_environment_isolation", environment.ErrBoot, table)
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

// Stamp records the process environment on a new database; a stamp that disagrees refuses.
func (m *EnvironmentModule) Stamp(ctx context.Context, db *gorm.DB) error {
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return fmt.Errorf("%w: gateway_environment is missing; apply migration 2026100705_environment_isolation", environment.ErrBoot)
	}
	stamp, err := m.readStamp(ctx, db)
	if err != nil {
		return err
	}
	if stamp == m.Environment {
		return nil
	}
	if stamp != "" {
		return fmt.Errorf("%w: database is stamped %s, this process is %s", environment.ErrBoot, stamp, m.Environment)
	}
	row := environment.StampRow{ID: environment.StampID, Environment: m.Environment, StampedAt: time.Now().UTC()}
	if err := db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("environment: stamp database: %w", err)
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
	if err := environment.CheckDatabase(environment.Live, name, m.testDatabase, m.databaseHost, ""); err != nil {
		return result, err
	}
	if !db.Migrator().HasTable(&environment.StampRow{}) {
		return result, fmt.Errorf("%w: gateway_environment is missing; run migrations first", environment.ErrBoot)
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var stamp environment.StampRow
		if err := tx.First(&stamp, environment.StampID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w: database is not stamped; a database that never served test money needs no adoption, boot it live", environment.ErrBoot)
			}
			return fmt.Errorf("environment: read stamp: %w", err)
		}
		if stamp.Environment != environment.Test || stamp.AdoptedFrom != nil {
			return fmt.Errorf("%w: database is stamped %s (adopted from %v); adoption happens once", environment.ErrBoot, stamp.Environment, stamp.AdoptedFrom)
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
