package modules

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type stubFees struct{}

func (stubFees) Snapshot(context.Context, *gorm.DB, paymentswitch.FeeRef, paymentswitch.FeeQuery) error {
	return nil
}
func (stubFees) PostFee(context.Context, *gorm.DB, paymentswitch.FeeRef, decimal.Decimal) error {
	return nil
}

// gatewayEnv is the process environment a server environment implies in these tests: staging/production are live.
func gatewayEnv(serverEnv string) environment.Environment {
	if serverEnv == config.EnvironmentProduction || serverEnv == config.EnvironmentStaging {
		return environment.Live
	}
	return environment.Test
}

func envModule(t *testing.T, serverEnv string) *EnvironmentModule {
	t.Helper()
	guard, err := environment.NewGuard(gatewayEnv(serverEnv))
	if err != nil {
		t.Fatal(err)
	}
	return &EnvironmentModule{Environment: gatewayEnv(serverEnv), Guard: guard}
}

func deps(t *testing.T, env string, conns ...string) Deps {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return Deps{
		DB:           db,
		Config:       &config.Config{Server: config.ServerConfig{Environment: env}, Gateway: config.GatewayConfig{Environment: string(gatewayEnv(env))}, Switch: config.SwitchConfig{Connectors: conns, MockWebhookSecret: "s"}},
		Environment:  envModule(t, env),
		Ledger:       ledger.New(db, ledger.WithEnvironment(gatewayEnv(env))),
		ChainDeposit: chaindeposit.NewMemoryBackend(),
		Fees:         stubFees{},
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
	if _, err := WirePaymentSwitch(deps(t, config.EnvironmentProduction, "mock")); !errors.Is(err, environment.ErrProvider) {
		t.Fatalf("live mock err = %v, want the environment guard's ErrProvider", err)
	}
	if _, err := WirePaymentSwitch(deps(t, config.EnvironmentStaging, "chaindeposit", "mock")); !errors.Is(err, environment.ErrProvider) {
		t.Fatalf("staging mock err = %v", err)
	}
	noEnv := deps(t, config.EnvironmentProduction, "chaindeposit")
	noEnv.Environment = nil
	if _, err := WirePaymentSwitch(noEnv); err == nil {
		t.Fatal("a deployment without the environment module must not wire")
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
	noFees := deps(t, config.EnvironmentProduction, "chaindeposit")
	noFees.Fees = nil
	if _, err := WirePaymentSwitch(noFees); err == nil {
		t.Fatal("production without the fees module must not wire")
	}
}

// An unset SWITCH_CONNECTORS follows the environment contract: the mock in test, nothing or CONNECTORS_PROVIDER in live.
func TestWirePaymentSwitch_UnsetConnectorsFollowTheEnvironmentContract(t *testing.T) {
	test := deps(t, config.EnvironmentDevelopment)
	test.Config.Switch.Connectors = nil
	m, err := WirePaymentSwitch(test)
	if err != nil || len(m.Enabled) != 2 || m.Enabled[0] != "mock" {
		t.Fatalf("test default = %+v, %v", m, err)
	}
	live := deps(t, config.EnvironmentProduction)
	live.Config.Switch.Connectors = nil
	if _, err := WirePaymentSwitch(live); !errors.Is(err, environment.ErrProvider) {
		t.Fatalf("live with no connector resolved must refuse to boot (environment contract), err = %v", err)
	}
	live.Config.Modules.Providers = map[string]string{ConnectorsSlot: "chaindeposit"}
	m, err = WirePaymentSwitch(live)
	if err != nil || len(m.Enabled) != 1 || m.Enabled[0] != "chaindeposit" {
		t.Fatalf("live CONNECTORS_PROVIDER = %+v, %v", m, err)
	}
	live.Config.Modules.Providers = map[string]string{ConnectorsSlot: "mock"}
	if _, err := WirePaymentSwitch(live); !errors.Is(err, environment.ErrProvider) {
		t.Fatalf("live CONNECTORS_PROVIDER=mock err = %v", err)
	}
	explicit := deps(t, config.EnvironmentDevelopment, "chaindeposit")
	if m, err := WirePaymentSwitch(explicit); err != nil || len(m.Enabled) != 1 {
		t.Fatalf("explicit list = %+v, %v", m, err)
	}
}

// F1: the switch posts through a ledger bound to the process guard; a bare test-environment ledger in a live
// process is refused at wiring, before any connector can charge.
func TestWirePaymentSwitch_RefusesALedgerBoundToAnotherEnvironment(t *testing.T) {
	live := deps(t, config.EnvironmentProduction, "chaindeposit")
	live.Ledger = ledger.New(live.DB)
	if _, err := WirePaymentSwitch(live); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("bare ledger in live err = %v, want ErrMismatch", err)
	}
	live.Ledger = ledger.New(live.DB, ledger.WithEnvironment(environment.Live))
	m, err := WirePaymentSwitch(live)
	if err != nil || m.Service == nil {
		t.Fatalf("guarded live ledger = %+v, %v", m, err)
	}
	test := deps(t, config.EnvironmentDevelopment, "mock")
	test.Ledger = ledger.New(test.DB, ledger.WithEnvironment(environment.Live))
	if _, err := WirePaymentSwitch(test); !errors.Is(err, environment.ErrMismatch) {
		t.Fatalf("live ledger in a test process err = %v", err)
	}
}

// The payment record anchoring fees carries the intent's expiry, so Payminto's expiry worker closes it.
func TestPaymintoPaymentRecords_OpenRowsExpireWithTheIntent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.PaymentRequest{}); err != nil {
		t.Fatal(err)
	}
	expires := time.Now().Add(-time.Minute).UTC()
	id, err := PaymintoPaymentRecords{DB: db}.Open(context.Background(), db, paymentswitch.PaymentRecord{IntentID: "pi_x", MerchantID: "7", PlatformID: "3", Money: paymentswitch.Money{Amount: decimal.NewFromInt(10), Asset: "USD"}, ExpiresAt: expires})
	if err != nil || id == 0 {
		t.Fatalf("open = %d, %v", id, err)
	}
	var row models.PaymentRequest
	db.First(&row, id)
	if row.ExpiresAt == nil || !row.ExpiresAt.Equal(expires) {
		t.Fatalf("expires_at = %v, want the intent's %v", row.ExpiresAt, expires)
	}
	n, err := repository.NewPaymentRepository(db).ExpireStale(time.Now())
	if err != nil || n != 1 {
		t.Fatalf("ExpireStale = %d, %v; the anchor row must be closed by Payminto's worker", n, err)
	}
	db.First(&row, id)
	if row.State != models.PaymentStateCancelled {
		t.Fatalf("state = %s", row.State)
	}
	if _, err := (PaymintoPaymentRecords{DB: db}).Open(context.Background(), db, paymentswitch.PaymentRecord{IntentID: "pi_y", MerchantID: "7", PlatformID: "3", Money: paymentswitch.Money{Amount: decimal.NewFromInt(10), Asset: "USD"}}); err == nil {
		t.Fatal("a record without an expiry must be refused")
	}
}
