package modules

import (
	"context"
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

func TestAdoptLive_Guards(t *testing.T) {
	ctx := context.Background()
	db := sqliteDB(t)
	if err := db.AutoMigrate(&environment.StampRow{}, &models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	test := &EnvironmentModule{Environment: environment.Test, databaseName: "payminto", databaseHost: "localhost", testDatabase: "payminto_test"}
	live := &EnvironmentModule{Environment: environment.Live, databaseName: "payminto", databaseHost: "db.internal", testDatabase: "payminto_test"}
	if _, err := test.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) {
		t.Fatalf("test process adopted: %v", err)
	}
	if _, err := live.AdoptLive(ctx, db, "payminto"); !environment.IsBootRefusal(err) || !strings.Contains(err.Error(), "not stamped") {
		t.Fatalf("unstamped database adopted: %v", err)
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
