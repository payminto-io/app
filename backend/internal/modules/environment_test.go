package modules

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func liveConfig() *config.Config {
	return &config.Config{
		Server:     config.ServerConfig{Environment: config.EnvironmentProduction},
		Database:   config.DatabaseConfig{Host: "db.internal", Database: "gateway", TestDatabase: "gateway_test", SSLMode: "verify-full"},
		Blockchain: config.BlockchainConfig{NetworkType: "mainnet"},
		Security:   config.SecurityConfig{JWTSecret: "a-strong-jwt-secret-value-with-32-plus-chars"},
		Gateway:    config.GatewayConfig{Environment: "live"},
	}
}

func TestWireEnvironment_LiveRefusesDevKeystoreAndMock(t *testing.T) {
	cfg := liveConfig()
	if _, err := WireEnvironment(Deps{Config: cfg}); err != nil {
		t.Fatalf("live happy path refused: %v", err)
	}
	cfg.Security.DevKeystore = true
	if _, err := WireEnvironment(Deps{Config: cfg}); !environment.IsBootRefusal(err) {
		t.Fatalf("dev keystore in live = %v, want boot refusal", err)
	}
	cfg = liveConfig()
	cfg.Modules.Providers = map[string]string{"custody": "mock"}
	if _, err := WireEnvironment(Deps{Config: cfg}); !environment.IsBootRefusal(err) {
		t.Fatalf("mock provider in live = %v, want boot refusal", err)
	}
	// Whatever a slot resolves to is what the guard judges, so a wired slot with no config cannot run the mock in live.
	m, err := WireEnvironment(Deps{Config: liveConfig()})
	if err != nil {
		t.Fatal(err)
	}
	for _, slot := range environment.KnownSlots {
		if err := m.Guard.RequireProvider(slot, environment.ResolveProvider(m.Environment, cfg.Modules.Providers[slot])); !errors.Is(err, environment.ErrProvider) {
			t.Fatalf("live %s without config = %v, want ErrProvider", slot, err)
		}
	}
}

func TestWireEnvironment_TestDefaultsAndGuard(t *testing.T) {
	cfg := &config.Config{Database: config.DatabaseConfig{Host: "localhost", Database: "payminto"}, Blockchain: config.BlockchainConfig{NetworkType: "testnet"}, Gateway: config.GatewayConfig{Environment: "test"}}
	m, err := WireEnvironment(Deps{Config: cfg})
	if err != nil {
		t.Fatal(err)
	}
	if m.Environment != environment.Test || m.Guard.Current() != environment.Test {
		t.Fatalf("module = %+v", m)
	}
}

func sqliteDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestVerifySchema(t *testing.T) {
	db := sqliteDB(t)
	m := &EnvironmentModule{Environment: environment.Live}
	ctx := context.Background()
	if err := db.Exec(`CREATE TABLE api_keys (id integer primary key, key text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := m.VerifySchema(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("missing column = %v, want boot refusal", err)
	}
	if err := db.Exec(`DROP TABLE api_keys`).Error; err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"payment_links", "payment_link_payments"} {
		if err := db.Exec(`CREATE TABLE ` + table + ` (id integer primary key)`).Error; err != nil {
			t.Fatal(err)
		}
		if err := m.VerifySchema(ctx, db); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), table+".environment") || !strings.Contains(err.Error(), "2026100706") {
			t.Fatalf("%s without environment = %v, want boot refusal naming the links migration", table, err)
		}
		if err := db.Exec(`DROP TABLE ` + table).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	if err := m.VerifySchema(ctx, db); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "gateway_environment") {
		t.Fatalf("missing stamp table = %v, want boot refusal", err)
	}
	if err := db.AutoMigrate(&environment.StampRow{}); err != nil {
		t.Fatal(err)
	}
	rows := []models.APIKey{
		{Key: "a", ExternalPlatformID: 1, Environment: environment.Live, Prefix: "sk_live_abcd"},
		{Key: "b", ExternalPlatformID: 1},
		{Key: "adopted", ExternalPlatformID: 1, Environment: environment.Live},
		{Key: "pk", ExternalPlatformID: 1, Environment: environment.Test, Prefix: "pk_test_abcd"},
	}
	for i := range rows {
		if err := db.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := m.VerifySchema(ctx, db); err != nil {
		t.Fatalf("consistent rows refused: %v", err)
	}
	for _, bad := range []models.APIKey{
		{Key: "c", ExternalPlatformID: 1, Environment: environment.Live, Prefix: "sk_test_abcd"},
		{Key: "d", ExternalPlatformID: 1, Environment: environment.Test, Prefix: "sktest_ab"}, // contains "test" but is not a prefix
	} {
		if err := db.Create(&bad).Error; err != nil {
			t.Fatal(err)
		}
		if err := m.VerifySchema(ctx, db); !environment.IsBootRefusal(err) {
			t.Fatalf("prefix %q = %v, want boot refusal", bad.Prefix, err)
		}
		db.Unscoped().Delete(&bad)
	}
}

func TestVerifyDatabaseAndStamp(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "gateway", databaseHost: "db.internal", testDatabase: "payminto_test"}
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "gateway", databaseHost: "localhost", testDatabase: "payminto_test"}
	if err := live.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("new database refused: %v", err)
	}
	if err := live.Stamp(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("stamp without the table = %v, want boot refusal", err)
	}
	if err := db.AutoMigrate(&environment.StampRow{}); err != nil {
		t.Fatal(err)
	}
	if err := live.Stamp(ctx, db); err != nil {
		t.Fatalf("first stamp: %v", err)
	}
	if err := live.Stamp(ctx, db); err != nil {
		t.Fatalf("stamp is not idempotent: %v", err)
	}
	if err := live.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("live on live-stamped database refused: %v", err)
	}
	if err := test.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "stamped live") {
		t.Fatalf("test process on a live-stamped loopback database = %v, want refusal", err)
	}
	if err := test.Stamp(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("test stamp over live = %v, want refusal", err)
	}
	var n int64
	db.Model(&environment.StampRow{}).Count(&n)
	if n != 1 {
		t.Fatalf("stamp rows = %d, want 1", n)
	}
}

func TestStamp_RefusesAnUnstampedDatabaseThatHoldsData(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	if err := db.Create(&models.APIKey{Key: "legacy", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	err := live.Stamp(ctx, db)
	if !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "adopt-live --confirm-adopt-live=payminto") || !strings.Contains(err.Error(), "adopt-test --confirm-adopt-test=payminto") {
		t.Fatalf("Stamp over data = %v, want refusal naming both adoption commands", err)
	}
	var n int64
	db.Model(&environment.StampRow{}).Count(&n)
	if n != 0 {
		t.Fatal("a process stamped a database that holds data")
	}
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "localhost", testDatabase: "payminto_test"}
	if err := test.Stamp(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("test process stamped a populated database: %v", err)
	}
	db.Unscoped().Where("1 = 1").Delete(&models.APIKey{})
	if err := live.Stamp(ctx, db); err != nil {
		t.Fatalf("empty database refused: %v", err)
	}
}

func TestAdoptLive_AcceptsAnUnstampedPopulatedDatabase(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "legacy", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	result, err := live.AdoptLive(ctx, db, "payminto")
	if err != nil || result.APIKeys != 1 {
		t.Fatalf("adopt unstamped = %+v, %v", result, err)
	}
	var stamp environment.StampRow
	if err := db.First(&stamp, environment.StampID).Error; err != nil || stamp.Environment != environment.Live || stamp.AdoptedFrom == nil {
		t.Fatalf("stamp after adoption = %+v, %v", stamp, err)
	}
	if err := live.Stamp(ctx, db); err != nil {
		t.Fatalf("live boot after adoption refused: %v", err)
	}
}

func TestAdoptTest(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "legacy", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	if _, err := live.AdoptTest(ctx, db, "payminto"); !environment.IsBootRefusal(err) {
		t.Fatalf("live process ran adopt-test: %v", err)
	}
	remote := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test"}
	if _, err := remote.AdoptTest(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "GATEWAY_TEST_DATABASE_NAME") {
		t.Fatalf("remote non-_test database adopted as test without the allow name: %v", err)
	}
	allowed := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", allowName: "payminto"}
	if _, err := allowed.AdoptTest(ctx, db, "other"); !environment.IsBootRefusal(err) {
		t.Fatalf("wrong confirmation accepted: %v", err)
	}
	name, err := allowed.AdoptTest(ctx, db, "payminto")
	if err != nil || name != "payminto" {
		t.Fatalf("adopt-test = %q, %v", name, err)
	}
	if err := allowed.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("test process refused its adopted database: %v", err)
	}
	if err := live.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("live process accepted a test-adopted database: %v", err)
	}
	if _, err := allowed.AdoptTest(ctx, db, "payminto"); !environment.IsBootRefusal(err) {
		t.Fatalf("second adopt-test accepted: %v", err)
	}
}

func TestAdoptLive_Guards(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "localhost", testDatabase: "payminto_test"}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	if _, err := test.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) {
		t.Fatalf("test process adopted: %v", err)
	}
	if err := test.Stamp(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := live.AdoptLive(ctx, db, "other"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "confirm-adopt-live") {
		t.Fatalf("wrong confirmation accepted: %v", err)
	}
	if _, err := live.AdoptLive(ctx, db, ""); !environment.IsBootRefusal(err) {
		t.Fatalf("empty confirmation accepted: %v", err)
	}
	for _, row := range []models.APIKey{
		{Key: "legacy", ExternalPlatformID: 1},
		{Key: "test", ExternalPlatformID: 1, Environment: environment.Test, Prefix: "sk_test_abcd"},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	result, err := live.AdoptLive(ctx, db, " Payminto ")
	if err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if result.APIKeys != 1 {
		t.Fatalf("relabelled keys = %d, want only the legacy one", result.APIKeys)
	}
	var keys []models.APIKey
	db.Order("key").Find(&keys)
	if keys[0].Environment != environment.Live || keys[1].Environment != environment.Test {
		t.Fatalf("keys after adoption = %+v", keys)
	}
	var stamp environment.StampRow
	db.First(&stamp, environment.StampID)
	if stamp.Environment != environment.Live || stamp.AdoptedFrom == nil || *stamp.AdoptedFrom != environment.Test || stamp.AdoptedAt == nil {
		t.Fatalf("stamp after adoption = %+v", stamp)
	}
	if _, err := live.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "once") {
		t.Fatalf("second adoption accepted: %v", err)
	}
	if err := live.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("live refused its adopted database: %v", err)
	}
	if err := test.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("test process accepted the adopted database: %v", err)
	}
}

func TestStamp_EveryDataTableCountsAsData(t *testing.T) {
	ctx := context.Background()
	for _, table := range DataTables() {
		t.Run(table, func(t *testing.T) {
			db := sqliteDB(t)
			if err := db.AutoMigrate(&environment.StampRow{}); err != nil {
				t.Fatal(err)
			}
			if err := db.Exec(`CREATE TABLE ` + table + ` (id integer primary key)`).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Exec(`INSERT INTO ` + table + ` (id) VALUES (1)`).Error; err != nil {
				t.Fatal(err)
			}
			live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
			err := live.Stamp(ctx, db)
			if !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "holds "+table+" rows") {
				t.Fatalf("Stamp over a %s row = %v, want refusal naming the table", table, err)
			}
		})
	}
	expected := []string{"members", "external_platforms", "api_keys", "payment_requests", "deposits", "deposit_addresses", "withdrawals", "sweeps", "wallets", "address_pools", "secrets_vaults", "ledger_accounts", "ledger_journals", "fee_rules", "payment_links", "payment_link_payments"}
	if got := DataTables(); strings.Join(got, ",") != strings.Join(expected, ",") {
		t.Fatalf("DataTables() = %v", got)
	}
}

func TestFinalize_WritesStampAndModeTogetherOnlyAfterEveryCheck(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.Configuration{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	// Populated and unstamped: refused, and no mode row may be left behind.
	if err := db.Create(&models.APIKey{Key: "legacy", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := live.Finalize(ctx, db, "mainnet"); !environment.IsBootRefusal(err) {
		t.Fatalf("Finalize over data = %v", err)
	}
	var modes, stamps int64
	db.Model(&models.Configuration{}).Count(&modes)
	db.Model(&environment.StampRow{}).Count(&stamps)
	if modes != 0 || stamps != 0 {
		t.Fatalf("a refused boot wrote mode rows %d, stamps %d", modes, stamps)
	}
	db.Unscoped().Where("1 = 1").Delete(&models.APIKey{})
	// Mode row disagrees: refused, nothing written.
	if err := db.Create(&models.Configuration{Key: "mode", Value: "testnet"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := live.Finalize(ctx, db, "mainnet"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "network mode mismatch") {
		t.Fatalf("Finalize with a testnet row = %v", err)
	}
	db.Model(&environment.StampRow{}).Count(&stamps)
	if stamps != 0 {
		t.Fatal("stamp written despite the mode mismatch")
	}
	db.Unscoped().Where("key = ?", "mode").Delete(&models.Configuration{})
	// Empty and acceptable: stamp and mode row land together.
	if err := live.Finalize(ctx, db, "mainnet"); err != nil {
		t.Fatalf("Finalize on an empty database: %v", err)
	}
	var mode models.Configuration
	if err := db.Where("key = ?", "mode").First(&mode).Error; err != nil || mode.Value != "mainnet" {
		t.Fatalf("mode row after Finalize = %+v, %v", mode, err)
	}
	if err := live.Finalize(ctx, db, "mainnet"); err != nil {
		t.Fatalf("Finalize is not idempotent: %v", err)
	}
}

func TestAdoption_ChecksTheNetworkMode(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.Configuration{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "legacy", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Configuration{Key: "mode", Value: "testnet"}).Error; err != nil {
		t.Fatal(err)
	}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test", networkType: "mainnet"}
	if _, err := live.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "network mode") {
		t.Fatalf("adopt-live on a testnet database = %v, want refusal", err)
	}
	db.Model(&models.Configuration{}).Where("key = ?", "mode").Update("value", "mainnet")
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "localhost", testDatabase: "payminto_test", networkType: "testnet"}
	if _, err := test.AdoptTest(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "mainnet") {
		t.Fatalf("adopt-test on a mainnet database = %v, want refusal", err)
	}
	if _, err := live.AdoptLive(ctx, db, "payminto"); err != nil {
		t.Fatalf("adopt-live on a mainnet database: %v", err)
	}
}
