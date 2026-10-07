//go:build integration

package database

import (
	"context"
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm"
)

func TestApplyMigrationsRejectsEmptySchemaWithoutMutation(t *testing.T) {
	db, cleanup := NewEmptyTestDB(t)
	defer cleanup()

	_, err := ApplyMigrations(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("ApplyMigrations() error = %v, want current-schema validation failure", err)
	}
	if db.Migrator().HasTable("schema_migrations") {
		t.Fatal("ApplyMigrations() created migration metadata in an empty schema")
	}
}

func TestApplyMigrationsRejectsUntrackedManagedObjectsWithoutCreatingMetadata(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()
	if err := db.Exec(`CREATE TABLE payment_lifecycle_invoices (invoice_id text primary key)`).Error; err != nil {
		t.Fatalf("create untracked managed object: %v", err)
	}

	_, err := ApplyMigrations(context.Background(), db)
	if err == nil || !strings.Contains(err.Error(), "managed payment lifecycle") {
		t.Fatalf("ApplyMigrations() error = %v, want managed-object preflight failure", err)
	}
	if db.Migrator().HasTable("schema_migrations") || db.Migrator().HasTable("migration_runs") {
		t.Fatal("ApplyMigrations() mutated metadata after managed-object preflight failure")
	}
}

func TestApplyMigrationsRejectsLegacyOrMixedSchemaWithoutMetadata(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T) (*gorm.DB, func())
		want  string
	}{
		{
			name: "legacy",
			setup: func(t *testing.T) (*gorm.DB, func()) {
				db, cleanup := NewEmptyTestDB(t)
				if err := db.Exec(`CREATE TABLE payment_requests (id bigint primary key, status text, expire_at timestamptz)`).Error; err != nil {
					cleanup()
					t.Fatalf("create legacy marker: %v", err)
				}
				return db, cleanup
			},
			want: "legacy PayRam",
		},
		{
			name: "mixed",
			setup: func(t *testing.T) (*gorm.DB, func()) {
				db, cleanup := NewTestDB(t)
				if err := db.Exec(`ALTER TABLE payment_requests ADD COLUMN status text`).Error; err != nil {
					cleanup()
					t.Fatalf("create mixed marker: %v", err)
				}
				return db, cleanup
			},
			want: "mixed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := tc.setup(t)
			defer cleanup()
			_, err := ApplyMigrations(context.Background(), db)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ApplyMigrations() error = %v, want %q", err, tc.want)
			}
			if db.Migrator().HasTable("schema_migrations") {
				t.Fatal("ApplyMigrations() created metadata for a rejected schema")
			}
		})
	}
}

func TestApplyMigrationsAppliesPaymentLifecycleFoundationOnce(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	first, err := ApplyMigrations(context.Background(), db)
	if err != nil {
		t.Fatalf("first ApplyMigrations() error = %v", err)
	}
	if len(first) != 7 || first[0].Version != 2026082701 || first[1].Version != 2026100701 || first[2].Version != 2026100702 || first[3].Version != 2026100703 || first[4].Version != 2026100705 || first[5].Version != 2026100710 || first[6].Version != 2026100711 {
		t.Fatalf("first result = %#v, want migrations 2026082701, 2026100701, 2026100702, 2026100703, 2026100705, 2026100710 and 2026100711", first)
	}

	second, err := ApplyMigrations(context.Background(), db)
	if err != nil {
		t.Fatalf("second ApplyMigrations() error = %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second result = %#v, want no applied migrations", second)
	}

	for _, table := range []string{
		"payment_lifecycle_invoices",
		"payment_lifecycle_quotes",
		"payment_lifecycle_payment_methods",
		"payment_lifecycle_idempotency_receipts",
		"payment_lifecycle_deposit_addresses",
		"payment_lifecycle_address_assignments",
		"payment_lifecycle_history",
		"payment_lifecycle_outbox_events",
		"payment_links",
		"payment_link_line_items",
		"payment_link_questions",
		"payment_link_payments",
		"payment_link_answers",
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("required expand table %q is missing", table)
		}
	}

	var applied, runs int64
	if err := db.Table("schema_migrations").Where("version = ? AND dirty = false", 2026082701).Count(&applied).Error; err != nil {
		t.Fatalf("count schema migration: %v", err)
	}
	if err := db.Table("migration_runs").Where("version = ? AND state = ?", 2026082701, "applied").Count(&runs).Error; err != nil {
		t.Fatalf("count migration run: %v", err)
	}
	if applied != 1 || runs != 1 {
		t.Fatalf("applied migrations = %d, applied runs = %d; want 1, 1", applied, runs)
	}
}

func TestApplyMigrationsSerializesConcurrentRunners(t *testing.T) {
	db, cleanup := NewTestDB(t)
	defer cleanup()

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := ApplyMigrations(context.Background(), db)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent ApplyMigrations() error = %v", err)
		}
	}

	var applied, runs int64
	if err := db.Table("schema_migrations").Where("version = ?", 2026082701).Count(&applied).Error; err != nil {
		t.Fatalf("count schema migration: %v", err)
	}
	if err := db.Table("migration_runs").Where("version = ? AND state = ?", 2026082701, "applied").Count(&runs).Error; err != nil {
		t.Fatalf("count migration runs: %v", err)
	}
	if applied != 1 || runs != 1 {
		t.Fatalf("applied migrations = %d, applied runs = %d; want 1, 1", applied, runs)
	}
}

func TestApplyMigrationsRejectsDirtyOrChangedMigrationHistory(t *testing.T) {
	tests := []struct {
		name      string
		mutate    string
		wantError string
	}{
		{name: "dirty", mutate: `UPDATE schema_migrations SET dirty = true WHERE version = 2026082701`, wantError: "dirty"},
		{name: "checksum", mutate: `UPDATE schema_migrations SET checksum = repeat('0', 64) WHERE version = 2026082701`, wantError: "checksum"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			defer cleanup()
			if _, err := ApplyMigrations(context.Background(), db); err != nil {
				t.Fatalf("seed migration history: %v", err)
			}
			if err := db.Exec(tc.mutate).Error; err != nil {
				t.Fatalf("mutate migration history: %v", err)
			}

			_, err := ApplyMigrations(context.Background(), db)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.wantError) {
				t.Fatalf("ApplyMigrations() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}

func TestApplyMigrationsRejectsUnknownOrUnfinishedHistory(t *testing.T) {
	tests := []struct {
		name      string
		mutate    string
		wantError string
	}{
		{
			name: "unknown version",
			mutate: `INSERT INTO schema_migrations (version, name, checksum, dirty, applied_at)
                 VALUES (9999999999, 'unknown', repeat('f', 64), false, clock_timestamp())`,
			wantError: "unknown migration version",
		},
		{
			name: "unfinished run",
			mutate: `INSERT INTO migration_runs (version, name, checksum, state)
                 VALUES (2026082701, 'payment_lifecycle_open', repeat('f', 64), 'running')`,
			wantError: "unfinished migration",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := NewTestDB(t)
			defer cleanup()
			if _, err := ApplyMigrations(context.Background(), db); err != nil {
				t.Fatalf("seed migration history: %v", err)
			}
			if err := db.Exec(tc.mutate).Error; err != nil {
				t.Fatalf("mutate migration history: %v", err)
			}
			_, err := ApplyMigrations(context.Background(), db)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("ApplyMigrations() error = %v, want %q", err, tc.wantError)
			}
		})
	}
}
