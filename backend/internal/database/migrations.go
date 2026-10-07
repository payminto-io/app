package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const migrationAdvisoryLockID int64 = 0x5061796d696e746f // "Payminto"

//go:embed migrations/*.up.sql
var migrationSQL embed.FS

type migration struct {
	version  int64
	name     string
	path     string
	sql      string
	checksum string
}

// MigrationResult describes one migration applied by this invocation. An empty
// result means the database was already at the current schema version.
type MigrationResult struct {
	Version  int64
	Name     string
	Checksum string
}

var migrationManifest = []struct {
	version int64
	name    string
	path    string
}{
	{2026082701, "payment_lifecycle_open", "migrations/2026082701_payment_lifecycle_open.up.sql"},
	{2026100701, "ledger_double_entry", "migrations/2026100701_ledger_double_entry.up.sql"},
}

// ApplyMigrations is the production schema-evolution seam. It accepts only a
// structurally validated current-model schema, serializes all runners with a
// PostgreSQL advisory lock, verifies immutable migration history, and records
// a dirty marker before executing each transactional expand migration.
//
// Empty, legacy, mixed, incomplete, dirty, checksum-mismatched, or unknown
// migration histories fail closed. This function intentionally does not seed
// catalog data and cannot bootstrap or cut over the legacy PayRam schema.
func ApplyMigrations(ctx context.Context, db *gorm.DB) ([]MigrationResult, error) {
	if db == nil {
		return nil, fmt.Errorf("migrations: database is nil")
	}
	if err := validateCurrentSchema(db); err != nil {
		return nil, fmt.Errorf("migrations: current schema validation failed: %w", err)
	}

	manifest, err := loadMigrationManifest()
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("migrations: underlying database: %w", err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("migrations: reserve connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationAdvisoryLockID); err != nil {
		return nil, fmt.Errorf("migrations: acquire advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationAdvisoryLockID)
	}()

	if err := ensureMigrationMetadata(ctx, conn); err != nil {
		return nil, err
	}
	applied, err := readAndVerifyMigrationHistory(ctx, conn, manifest)
	if err != nil {
		return nil, err
	}

	results := make([]MigrationResult, 0, len(manifest)-len(applied))
	for _, item := range manifest {
		if applied[item.version] {
			continue
		}
		if err := applyMigration(ctx, conn, item); err != nil {
			return results, err
		}
		results = append(results, MigrationResult{Version: item.version, Name: item.name, Checksum: item.checksum})
	}
	return results, nil
}

func loadMigrationManifest() ([]migration, error) {
	manifest := make([]migration, 0, len(migrationManifest))
	seen := make(map[int64]struct{}, len(migrationManifest))
	var previous int64
	for _, item := range migrationManifest {
		if item.version <= previous {
			return nil, fmt.Errorf("migrations: manifest versions must be strictly increasing at %d", item.version)
		}
		if _, exists := seen[item.version]; exists {
			return nil, fmt.Errorf("migrations: duplicate manifest version %d", item.version)
		}
		raw, err := migrationSQL.ReadFile(item.path)
		if err != nil {
			return nil, fmt.Errorf("migrations: read %s: %w", item.path, err)
		}
		sum := sha256.Sum256(raw)
		manifest = append(manifest, migration{
			version:  item.version,
			name:     item.name,
			path:     item.path,
			sql:      string(raw),
			checksum: hex.EncodeToString(sum[:]),
		})
		seen[item.version] = struct{}{}
		previous = item.version
	}
	return manifest, nil
}

func ensureMigrationMetadata(ctx context.Context, conn *sql.Conn) error {
	var hasMigrations, hasRuns bool
	if err := conn.QueryRowContext(ctx, `
SELECT to_regclass('schema_migrations') IS NOT NULL,
       to_regclass('migration_runs') IS NOT NULL`).Scan(&hasMigrations, &hasRuns); err != nil {
		return fmt.Errorf("migrations: inspect metadata: %w", err)
	}
	if hasMigrations != hasRuns {
		return fmt.Errorf("migrations: partial migration metadata detected; schema_migrations and migration_runs must both exist or both be absent")
	}
	if hasMigrations {
		return nil
	}
	var managedObjects int64
	if err := conn.QueryRowContext(ctx, `
SELECT
    (SELECT count(*)
       FROM pg_class AS object
       JOIN pg_namespace AS namespace ON namespace.oid = object.relnamespace
      WHERE namespace.nspname = current_schema()
        AND object.relname LIKE 'payment_lifecycle_%')
  + (SELECT count(*)
       FROM pg_proc AS object
       JOIN pg_namespace AS namespace ON namespace.oid = object.pronamespace
      WHERE namespace.nspname = current_schema()
        AND object.proname LIKE 'payment_lifecycle_%')`).Scan(&managedObjects); err != nil {
		return fmt.Errorf("migrations: inspect managed payment lifecycle objects: %w", err)
	}
	if managedObjects != 0 {
		return fmt.Errorf("migrations: found %d managed payment lifecycle object(s) without migration metadata; refusing to adopt or overwrite them", managedObjects)
	}

	const metadata = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint PRIMARY KEY,
    name text NOT NULL,
    checksum char(64) NOT NULL,
    dirty boolean NOT NULL DEFAULT true,
    applied_at timestamptz
);
CREATE TABLE IF NOT EXISTS migration_runs (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    version bigint NOT NULL,
    name text NOT NULL,
    checksum char(64) NOT NULL,
    state text NOT NULL CHECK (state IN ('running', 'applied', 'failed')),
    started_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    finished_at timestamptz,
    duration_ms bigint,
    error text
);
CREATE INDEX IF NOT EXISTS migration_runs_version_started_idx
    ON migration_runs (version, started_at DESC);`
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: begin metadata transaction: %w", err)
	}
	if _, err := tx.ExecContext(ctx, metadata); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migrations: create or validate metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrations: commit metadata: %w", err)
	}
	return nil
}

func readAndVerifyMigrationHistory(ctx context.Context, conn *sql.Conn, manifest []migration) (map[int64]bool, error) {
	known := make(map[int64]migration, len(manifest))
	for _, item := range manifest {
		known[item.version] = item
	}

	rows, err := conn.QueryContext(ctx, `SELECT version, name, checksum, dirty FROM schema_migrations ORDER BY version`)
	if err != nil {
		return nil, fmt.Errorf("migrations: read history: %w", err)
	}
	defer rows.Close()
	applied := make(map[int64]bool, len(manifest))
	prefixIndex := 0
	for rows.Next() {
		var version int64
		var name, checksum string
		var dirty bool
		if err := rows.Scan(&version, &name, &checksum, &dirty); err != nil {
			return nil, fmt.Errorf("migrations: scan history: %w", err)
		}
		expected, ok := known[version]
		if !ok {
			return nil, fmt.Errorf("migrations: database contains unknown migration version %d", version)
		}
		if prefixIndex >= len(manifest) || manifest[prefixIndex].version != version {
			return nil, fmt.Errorf("migrations: applied history is not a contiguous manifest prefix; version %d appears after a gap", version)
		}
		if dirty {
			return nil, fmt.Errorf("migrations: version %d is dirty; inspect and resolve before continuing", version)
		}
		if name != expected.name {
			return nil, fmt.Errorf("migrations: version %d name mismatch: database=%q manifest=%q", version, name, expected.name)
		}
		if checksum != expected.checksum {
			return nil, fmt.Errorf("migrations: version %d checksum mismatch", version)
		}
		applied[version] = true
		prefixIndex++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrations: iterate history: %w", err)
	}

	var running int64
	if err := conn.QueryRowContext(ctx, `SELECT count(*) FROM migration_runs WHERE state = 'running'`).Scan(&running); err != nil {
		return nil, fmt.Errorf("migrations: inspect runs: %w", err)
	}
	if running != 0 {
		return nil, fmt.Errorf("migrations: %d unfinished migration run(s) require operator review", running)
	}
	return applied, nil
}

func applyMigration(ctx context.Context, conn *sql.Conn, item migration) error {
	started := time.Now()
	markerTx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: begin dirty marker for %d: %w", item.version, err)
	}
	if _, err := markerTx.ExecContext(ctx, `
INSERT INTO schema_migrations (version, name, checksum, dirty)
VALUES ($1, $2, $3, true)`, item.version, item.name, item.checksum); err != nil {
		_ = markerTx.Rollback()
		return fmt.Errorf("migrations: mark version %d dirty: %w", item.version, err)
	}
	var runID int64
	if err := markerTx.QueryRowContext(ctx, `
INSERT INTO migration_runs (version, name, checksum, state)
VALUES ($1, $2, $3, 'running') RETURNING id`, item.version, item.name, item.checksum).Scan(&runID); err != nil {
		_ = markerTx.Rollback()
		return fmt.Errorf("migrations: record run for %d: %w", item.version, err)
	}
	if err := markerTx.Commit(); err != nil {
		return fmt.Errorf("migrations: commit dirty marker for %d: %w", item.version, err)
	}

	migrationTx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		markMigrationFailed(ctx, conn, runID, started, err)
		return fmt.Errorf("migrations: begin version %d: %w", item.version, err)
	}
	if _, err := migrationTx.ExecContext(ctx, item.sql); err != nil {
		_ = migrationTx.Rollback()
		markMigrationFailed(ctx, conn, runID, started, err)
		return fmt.Errorf("migrations: apply version %d (%s): %w", item.version, item.name, err)
	}
	if err := migrationTx.Commit(); err != nil {
		markMigrationFailed(ctx, conn, runID, started, err)
		return fmt.Errorf("migrations: commit version %d (%s): %w", item.version, item.name, err)
	}

	finished := time.Now()
	finishTx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrations: version %d applied but completion recording failed: %w", item.version, err)
	}
	if _, err := finishTx.ExecContext(ctx, `
UPDATE schema_migrations SET dirty = false, applied_at = $2 WHERE version = $1`, item.version, finished); err != nil {
		_ = finishTx.Rollback()
		return fmt.Errorf("migrations: version %d applied but completion recording failed: %w", item.version, err)
	}
	if _, err := finishTx.ExecContext(ctx, `
UPDATE migration_runs
SET state = 'applied', finished_at = $2, duration_ms = $3
WHERE id = $1 AND state = 'running'`, runID, finished, finished.Sub(started).Milliseconds()); err != nil {
		_ = finishTx.Rollback()
		return fmt.Errorf("migrations: version %d applied but run recording failed: %w", item.version, err)
	}
	if err := finishTx.Commit(); err != nil {
		return fmt.Errorf("migrations: version %d applied but completion commit failed: %w", item.version, err)
	}
	return nil
}

func markMigrationFailed(ctx context.Context, conn *sql.Conn, runID int64, started time.Time, migrationErr error) {
	_, _ = conn.ExecContext(ctx, `
UPDATE migration_runs
SET state = 'failed', finished_at = clock_timestamp(), duration_ms = $2, error = left($3, 4000)
WHERE id = $1 AND state = 'running'`, runID, time.Since(started).Milliseconds(), migrationErr.Error())
}
