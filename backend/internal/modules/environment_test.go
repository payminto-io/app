package modules

import (
	"context"
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

func TestVerifyDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	m := &EnvironmentModule{Environment: environment.Live}
	ctx := context.Background()
	if err := db.Exec(`CREATE TABLE api_keys (id integer primary key, key text)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := m.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("missing column = %v, want boot refusal", err)
	}
	if err := db.Exec(`DROP TABLE api_keys`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.APIKey{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "a", ExternalPlatformID: 1, Environment: environment.Live, Prefix: "sk_live_abcd"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.APIKey{Key: "b", ExternalPlatformID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := m.VerifyDatabase(ctx, db); err != nil {
		t.Fatalf("consistent rows refused: %v", err)
	}
	if err := db.Create(&models.APIKey{Key: "c", ExternalPlatformID: 1, Environment: environment.Live, Prefix: "sk_test_abcd"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := m.VerifyDatabase(ctx, db); !environment.IsBootRefusal(err) {
		t.Fatalf("prefix mismatch = %v, want boot refusal", err)
	}
}
