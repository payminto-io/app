package modules

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/environment"
	"gorm.io/gorm"
)

// EnvironmentModule is the process environment plus the guard money-moving modules share.
type EnvironmentModule struct {
	Environment environment.Environment
	Guard       *environment.ProcessGuard
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
	return &EnvironmentModule{Environment: facts.Environment, Guard: guard}, nil
}

// VerifyDatabase refuses a schema that cannot hold environments and any key row whose
// visible prefix disagrees with its environment column.
func (m *EnvironmentModule) VerifyDatabase(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return errors.New("environment: database is nil")
	}
	for _, table := range []string{"api_keys", "ledger_accounts"} {
		if !db.Migrator().HasTable(table) {
			continue
		}
		if !db.Migrator().HasColumn(table, "environment") {
			return fmt.Errorf("%w: %s.environment is missing; apply migration 2026100705_environment_isolation", environment.ErrBoot, table)
		}
	}
	if !db.Migrator().HasTable("api_keys") {
		return nil
	}
	var mismatched int64
	err := db.WithContext(ctx).Table("api_keys").
		Where("prefix <> '' AND ((environment = ? AND prefix NOT LIKE ?) OR (environment = ? AND prefix NOT LIKE ?))",
			environment.Test, "%_test_%", environment.Live, "%_live_%").
		Count(&mismatched).Error
	if err != nil {
		return fmt.Errorf("environment: inspect api_keys: %w", err)
	}
	if mismatched != 0 {
		return fmt.Errorf("%w: %d api_keys row(s) carry a prefix from the other environment", environment.ErrBoot, mismatched)
	}
	return nil
}
