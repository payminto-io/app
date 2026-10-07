package paymentswitch

import (
	_ "embed"
	"fmt"

	"gorm.io/gorm"
)

//go:embed constraints.sql
var constraintsSQL string

// InstallConstraints adds the CHECK and FK constraints after AutoMigrate. Postgres only; SQLite unit tests
// rely on the state machine in code.
func InstallConstraints(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	if err := db.Exec(constraintsSQL).Error; err != nil {
		return fmt.Errorf("paymentswitch: install constraints: %w", err)
	}
	return nil
}
