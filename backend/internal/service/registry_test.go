package service

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRegistry(t *testing.T) *ServiceRegistry {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	cfg := &config.Config{
		Server:     config.ServerConfig{Port: 8080, Environment: "DEVELOPMENT"},
		Blockchain: config.BlockchainConfig{NetworkType: "testnet"},
		Security:   config.SecurityConfig{JWTSecret: "test-secret"},
	}
	reg, err := NewServiceRegistry(db, nil, cfg)
	if err != nil {
		t.Fatalf("NewServiceRegistry: %v", err)
	}
	return reg
}

func TestServiceRegistry_NilDB(t *testing.T) {
	cfg := &config.Config{}
	_, err := NewServiceRegistry(nil, nil, cfg)
	if err == nil {
		t.Error("expected error for nil db")
	}
}

func TestServiceRegistry_NilConfig(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_, err := NewServiceRegistry(db, nil, nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestServiceRegistry_ConstructsAllRepos(t *testing.T) {
	reg := newTestRegistry(t)

	repos := map[string]any{
		"MemberRepo":             reg.MemberRepo(),
		"APIKeyRepo":             reg.APIKeyRepo(),
		"ExternalPlatformRepo":   reg.ExternalPlatformRepo(),
		"RoleRepo":               reg.RoleRepo(),
		"PermissionRepo":         reg.PermissionRepo(),
		"PaymentRepo":            reg.PaymentRepo(),
		"DepositRepo":            reg.DepositRepo(),
		"DepositAddressRepo":     reg.DepositAddressRepo(),
		"WalletRepo":             reg.WalletRepo(),
		"AddressPoolRepo":        reg.AddressPoolRepo(),
		"BlockchainRepo":         reg.BlockchainRepo(),
		"BlockchainFamilyRepo":   reg.BlockchainFamilyRepo(),
		"CurrencyRepo":           reg.CurrencyRepo(),
		"BlockchainCurrencyRepo": reg.BlockchainCurrencyRepo(),
		"WebhookRepo":            reg.WebhookRepo(),
		"WebhookDeliveryLogRepo": reg.WebhookDeliveryLogRepo(),
		"RPCNodeRepo":            reg.RPCNodeRepo(),
		"WalletXpubRepo":         reg.WalletXpubRepo(),
		"WalletFunctionRepo":     reg.WalletFunctionRepo(),
		"WalletSCWRepo":          reg.WalletSCWRepo(),
		"AddressRepo":            reg.AddressRepo(),
	}
	for name, r := range repos {
		if r == nil {
			t.Errorf("%s should not be nil", name)
		}
	}
}

func TestServiceRegistry_ConstructsAllServices(t *testing.T) {
	reg := newTestRegistry(t)
	if reg.AuthService() == nil {
		t.Error("AuthService should not be nil")
	}
	if reg.PaymentService() == nil {
		t.Error("PaymentService should not be nil")
	}
	if reg.WebhookService() == nil {
		t.Error("WebhookService should not be nil")
	}
	if reg.OnrampService() == nil {
		t.Error("OnrampService should not be nil")
	}
	if reg.WalletService() == nil {
		t.Error("WalletService should not be nil")
	}
	if reg.AddressService() == nil {
		t.Error("AddressService should not be nil")
	}
	if reg.DepositService() == nil {
		t.Error("DepositService should not be nil")
	}
	if reg.AddressPoolService() == nil {
		t.Error("AddressPoolService should not be nil")
	}
	if reg.DepositAddressService() == nil {
		t.Error("DepositAddressService should not be nil")
	}
}

func TestServiceRegistry_NetworkType(t *testing.T) {
	reg := newTestRegistry(t)
	if reg.NetworkType() != "testnet" {
		t.Errorf("expected testnet, got %s", reg.NetworkType())
	}
}

func TestServiceRegistry_DBAccessor(t *testing.T) {
	reg := newTestRegistry(t)
	if reg.DB() == nil {
		t.Error("DB() should not be nil")
	}
}

func TestServiceRegistry_AdapterRegistry(t *testing.T) {
	reg := newTestRegistry(t)
	if reg.AdapterRegistry() == nil {
		t.Error("AdapterRegistry should not be nil")
	}
	if reg.AdapterRegistry().Len() != 0 {
		t.Error("AdapterRegistry should start empty")
	}
}
