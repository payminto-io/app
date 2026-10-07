//go:build integration
// +build integration

package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewTestDB spins up a Postgres 16 testcontainer and returns a connected *gorm.DB
// plus a cleanup function. Tag your test files with `//go:build integration` to
// opt in. Run with: go test -tags=integration ./...
func NewTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	db, cleanup := NewEmptyTestDB(t)
	if err := AutoMigrate(db); err != nil {
		cleanup()
		t.Fatalf("migrate test db: %v", err)
	}
	if err := MigrateExpandSchema(db); err != nil {
		cleanup()
		t.Fatalf("migrate test db: %v", err)
	}
	return db, cleanup
}

// NewEmptyTestDB starts PostgreSQL without creating an application schema. It
// is used to prove that production migration paths fail closed on empty,
// legacy, and otherwise unclassified databases.
func NewEmptyTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()
	if dsn := os.Getenv("PAYMINTO_INTEGRATION_DATABASE_URL"); dsn != "" {
		return newIsolatedSchemaTestDB(t, dsn)
	}
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "payminto",
			"POSTGRES_PASSWORD": "payminto_test",
			"POSTGRES_DB":       "payminto_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	cfg := config.DatabaseConfig{
		Host:     host,
		Port:     port.Int(),
		Database: "payminto_test",
		Username: "payminto",
		Password: "payminto_test",
		SSLMode:  "disable",
	}

	db, err := Connect(cfg)
	if err != nil {
		_ = container.Terminate(ctx)
		t.Fatalf("connect to test db: %v", err)
	}

	cleanup := func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		if err := container.Terminate(context.Background()); err != nil {
			fmt.Printf("warn: failed to terminate test container: %v\n", err)
		}
	}
	return db, cleanup
}

// newIsolatedSchemaTestDB permits PostgreSQL contract tests where a container
// provider is unavailable. The explicit URL must identify a disposable test
// database on which the current user may create and drop isolated schemas.
func newIsolatedSchemaTestDB(t *testing.T, dsn string) (*gorm.DB, func()) {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" {
		t.Fatalf("PAYMINTO_INTEGRATION_DATABASE_URL must be a PostgreSQL URL: %v", err)
	}
	schema := "payminto_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create isolated integration schema: %v", err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{})
	if err != nil {
		_ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		t.Fatalf("connect isolated integration schema: %v", err)
	}
	cleanup := func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		_ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
		if sqlDB, dbErr := admin.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}
	return db, cleanup
}
