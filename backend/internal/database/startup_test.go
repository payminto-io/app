package database

import (
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPrepareSchema_ValidateRejectsMissingCurrentSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	err = PrepareSchema(db, "DEVELOPMENT", config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "account_addresses") {
		t.Fatalf("PrepareSchema() error = %v, want missing current schema error", err)
	}
}

func TestPrepareSchema_AutoMigrateIsExplicitlyAvailableInDevelopment(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	if err := PrepareSchema(db, "DEVELOPMENT", config.SchemaModeAutoMigrate); err != nil {
		t.Fatalf("PrepareSchema() error = %v", err)
	}
	if !db.Migrator().HasColumn("payment_requests", "state") {
		t.Fatal("PrepareSchema() did not create the current payment_requests schema")
	}
}

func TestPrepareSchema_AutoMigrateIsExplicitlyAvailableInTest(t *testing.T) {
	db := openStartupTestDB(t)
	if err := PrepareSchema(db, config.EnvironmentTest, config.SchemaModeAutoMigrate); err != nil {
		t.Fatalf("PrepareSchema() error = %v", err)
	}
}

func TestPrepareSchema_ProductionCannotAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	err = PrepareSchema(db, "PRODUCTION", config.SchemaModeAutoMigrate)
	if err == nil || !strings.Contains(err.Error(), "development and test") {
		t.Fatalf("PrepareSchema() error = %v, want production rejection", err)
	}
	if db.Migrator().HasTable("payment_requests") {
		t.Fatal("PrepareSchema() changed an empty production database")
	}
}

func TestPrepareSchema_StagingCannotAutoMigrate(t *testing.T) {
	db := openStartupTestDB(t)
	err := PrepareSchema(db, config.EnvironmentStaging, config.SchemaModeAutoMigrate)
	if err == nil || !strings.Contains(err.Error(), "development and test") {
		t.Fatalf("PrepareSchema() error = %v, want staging rejection", err)
	}
	if db.Migrator().HasTable("payment_requests") {
		t.Fatal("PrepareSchema() changed an empty staging database")
	}
}

func TestPrepareSchema_RejectsUnknownEnvironment(t *testing.T) {
	db := openStartupTestDB(t)
	err := PrepareSchema(db, "prodution", config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "environment") {
		t.Fatalf("PrepareSchema() error = %v, want unknown environment rejection", err)
	}
}

func TestPrepareSchema_ValidateChecksTheCompleteTableManifest(t *testing.T) {
	db := openStartupTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if err := db.Migrator().DropTable("analytics_filters"); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	err := PrepareSchema(db, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "analytics_filters") {
		t.Fatalf("PrepareSchema() error = %v, want missing manifest table rejection", err)
	}
}

func TestPrepareSchema_ValidateRejectsLegacySchema(t *testing.T) {
	db := openStartupTestDB(t)
	if err := db.Exec(`CREATE TABLE payment_requests (id integer primary key, status text, expire_at datetime)`).Error; err != nil {
		t.Fatalf("create legacy table: %v", err)
	}

	err := PrepareSchema(db, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "legacy PayRam") {
		t.Fatalf("PrepareSchema() error = %v, want legacy fingerprint rejection", err)
	}
}

func TestPrepareSchema_ValidateRejectsMixedLegacyAndCurrentSchema(t *testing.T) {
	db := openStartupTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	if err := db.Exec(`ALTER TABLE payment_requests ADD COLUMN status text`).Error; err != nil {
		t.Fatalf("add legacy marker: %v", err)
	}

	err := PrepareSchema(db, config.EnvironmentProduction, config.SchemaModeValidate)
	if err == nil || !strings.Contains(err.Error(), "mixed") {
		t.Fatalf("PrepareSchema() error = %v, want mixed-schema rejection", err)
	}
}

func TestCurrentSchemaManifestCoversEveryAutoMigrateTable(t *testing.T) {
	db := openStartupTestDB(t)
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	tables, err := db.Migrator().GetTables()
	if err != nil {
		t.Fatalf("GetTables() error = %v", err)
	}
	for _, table := range tables {
		if table == "sqlite_sequence" {
			continue
		}
		if !currentSchemaContainsTable(table) {
			t.Errorf("AutoMigrate table %q is absent from current schema manifest", table)
		}
	}
}

func openStartupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	return db
}
