package ledger

import (
	_ "embed"
	"fmt"

	"gorm.io/gorm"
)

//go:embed constraints.sql
var constraintsSQL string

// InstallConstraints adds the append-only triggers, the deferred balance check and the
// FK/CHECK constraints after AutoMigrate. Postgres only; SQLite unit tests rely on Validate.
func InstallConstraints(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(constraintsSQL).Error; err != nil {
		return fmt.Errorf("ledger: install constraints: %w", err)
	}
	return nil
}

// Migrate creates the ledger tables and constraints for development and test databases.
// Production uses the checksummed migration 2026100701_ledger_double_entry instead.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(Models()...); err != nil {
		return fmt.Errorf("ledger: automigrate: %w", err)
	}
	return InstallConstraints(db)
}
