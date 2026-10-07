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
