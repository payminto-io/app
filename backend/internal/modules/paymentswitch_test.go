package modules

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func deps(t *testing.T, env string, conns ...string) Deps {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return Deps{
		DB:           db,
		Config:       &config.Config{Switch: config.SwitchConfig{Connectors: conns, MockWebhookSecret: "s"}},
		Ledger:       ledger.New(db),
		Environment:  env,
		ChainDeposit: chaindeposit.NewMemoryBackend(),
	}
}

func TestWirePaymentSwitch_RegistersConfiguredConnectors(t *testing.T) {
	m, err := WirePaymentSwitch(deps(t, config.EnvironmentDevelopment, "mock", " ChainDeposit ", "mock"))
	if err != nil {
		t.Fatalf("wire: %v", err)
	}
	if got := m.Connectors.Codes(); len(got) != 2 || got[0] != "chaindeposit" || got[1] != "mock" {
		t.Fatalf("codes = %v", got)
	}
	if len(m.Enabled) != 2 || m.Enabled[0] != "mock" {
		t.Fatalf("enabled order = %v, want mock first", m.Enabled)
	}
}

func TestWirePaymentSwitch_RefusesMockInDeploymentAndUnknownCodes(t *testing.T) {
	if _, err := WirePaymentSwitch(deps(t, config.EnvironmentProduction, "mock")); !errors.Is(err, ErrMockInDeployment) {
		t.Fatalf("production mock err = %v", err)
	}
	if _, err := WirePaymentSwitch(deps(t, config.EnvironmentStaging, "chaindeposit", "mock")); !errors.Is(err, ErrMockInDeployment) {
		t.Fatalf("staging mock err = %v", err)
	}
	if _, err := WirePaymentSwitch(deps(t, config.EnvironmentDevelopment, "stripe")); !errors.Is(err, connectors.ErrUnknownConnector) {
		t.Fatalf("unknown connector err = %v", err)
	}
	d := deps(t, config.EnvironmentDevelopment, "chaindeposit")
	d.ChainDeposit = nil
	if _, err := WirePaymentSwitch(d); err == nil {
		t.Fatal("chaindeposit without a backend must not wire")
	}
	if m, err := WirePaymentSwitch(deps(t, config.EnvironmentProduction, "chaindeposit")); err != nil || len(m.Enabled) != 1 {
		t.Fatalf("production with only chaindeposit = %+v, %v", m, err)
	}
}
