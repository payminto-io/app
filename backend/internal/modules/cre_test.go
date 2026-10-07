package modules

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/chainlink"
	"github.com/payminto/payminto/backend/internal/environment"
)

func creConfig(enabled bool, provider string) *config.Config {
	return &config.Config{
		Server:  config.ServerConfig{Environment: config.EnvironmentDevelopment, Port: 8090},
		Gateway: config.GatewayConfig{Environment: "test"},
		CRE: config.CREConfig{
			Enabled: enabled, Provider: provider, Chain: "ethereum-testnet-sepolia-base-1", SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute,
			PollInterval: 30 * time.Second, PublicVerifyEnabled: true, ReadTokenSolvency: "tok-s",
			WorkflowNameSolvency: "solvency", WorkflowNameDepositFinality: "deposit-finality", WorkflowNameConversionReference: "conversion-reference",
		},
	}
}

func TestWireCRE_NoneIsANoOp(t *testing.T) {
	for _, cfg := range []*config.Config{creConfig(false, "mock"), creConfig(true, "none")} {
		m, err := WireCRE(Deps{Config: cfg}, CREOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if m.Enabled() || m.Service.Worker() != nil || m.Provider != "none" || m.Mock != nil {
			t.Fatalf("none module = %+v", m)
		}
		if _, err := m.Service.Liabilities(context.Background(), true); !errors.Is(err, cre.ErrDisabled) {
			t.Fatalf("disabled read = %v", err)
		}
		if d, _ := m.Gate.Decide(context.Background(), cre.SettlementSubject{}); d != cre.GateProceed {
			t.Fatalf("gate = %s", d)
		}
		rep, _ := m.Service.Status(context.Background())
		if rep.Enabled || rep.Provider != "none" || rep.Environment != environment.Test {
			t.Fatalf("status = %+v", rep)
		}
	}
}

func TestWireCRE_MockInTestAndRefusedInLive(t *testing.T) {
	testGuard, _ := environment.NewGuard(environment.Test)
	m, err := WireCRE(Deps{Config: creConfig(true, "mock"), Environment: &EnvironmentModule{Environment: environment.Test, Guard: testGuard}}, CREOptions{Store: cre.NewMemoryStore()})
	if err != nil {
		t.Fatal(err)
	}
	if !m.Enabled() || m.Mock == nil || m.Service.Worker() == nil || m.Service.Worker().Name() != "cre-attester" {
		t.Fatalf("mock module = %+v", m)
	}
	if !m.Service.Authorize(cre.KindSolvency, "tok-s") || m.Service.Authorize(cre.KindSolvency, "nope") || m.Service.Authorize(cre.KindDepositFinality, "") {
		t.Fatal("credential checks")
	}
	liveGuard, _ := environment.NewGuard(environment.Live)
	_, err = WireCRE(Deps{Config: creConfig(true, "mock"), Environment: &EnvironmentModule{Environment: environment.Live, Guard: liveGuard}}, CREOptions{Store: cre.NewMemoryStore()})
	if !errors.Is(err, environment.ErrProvider) {
		t.Fatalf("mock in live = %v, want ErrProvider", err)
	}
	// Off in live is fine: the slot resolves to none, which is not the mock.
	if _, err := WireCRE(Deps{Config: creConfig(false, "mock"), Environment: &EnvironmentModule{Environment: environment.Live, Guard: liveGuard}}, CREOptions{}); err != nil {
		t.Fatalf("off in live refused: %v", err)
	}
}

func TestWireCRE_MockRoundTrip(t *testing.T) {
	m, err := WireCRE(Deps{Config: creConfig(true, "mock")}, CREOptions{Store: cre.NewMemoryStore(), Reserves: fakeReserves{}, Liabilities: fakeLiabilities{}})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	run, err := m.Service.Run(ctx, cre.KindSolvency)
	if err != nil || run.Status != cre.RunAccepted {
		t.Fatalf("run = %+v err %v", run, err)
	}
	n, err := m.Service.Poll(ctx, cre.KindSolvency)
	if err != nil || n != 1 {
		t.Fatalf("poll recorded %d, err %v", n, err)
	}
	rep, err := m.Service.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	w := rep.Workflows[0]
	if w.Kind != cre.KindSolvency || w.State != "fresh" || w.LastAttestation == nil || w.LastAttestation.Provider != "mock" || w.LastRun == nil {
		t.Fatalf("workflow status = %+v", w)
	}
	item, _ := w.LastAttestation.Item.(map[string]any)
	if w.LastAttestation.Status != cre.StatusAttested || item["liabilities_minor"] != "12500000" || item["asset"] != "USDC.SOLANA" {
		t.Fatalf("attestation = %+v item %+v", w.LastAttestation, item)
	}
	if rep.TriggerSignerAddress == "" || rep.WorkflowOwner != rep.TriggerSignerAddress || w.LastVerified == nil {
		t.Fatalf("mock signer not reported: %+v", rep)
	}
}

func TestWireCRE_ChainlinkNeedsAReaderWithoutRPC(t *testing.T) {
	cfg := creConfig(true, "chainlink")
	cfg.CRE.ChainRPCURL = "http://127.0.0.1:1"
	cfg.CRE.ConsumerAddress = "0x1111111111111111111111111111111111111111"
	cfg.CRE.WorkflowOwner = "0x3333333333333333333333333333333333333333"
	cfg.CRE.WorkflowIDSolvency = strings.Repeat("a", 64)
	cfg.CRE.WorkflowNameSolvency = "solvency"
	cfg.CRE.TriggerSigner = "keyring://cre-trigger"
	m, err := WireCRE(Deps{Config: cfg}, CREOptions{Store: cre.NewMemoryStore(), Reader: fakeReader{}})
	if err != nil {
		t.Fatal(err)
	}
	sc := m.Service.Config()
	b := sc.Bindings[cre.KindSolvency]
	if b.ID != [32]byte([]byte(strings.Repeat("\xaa", 32))) || sc.TriggerSigner != "keyring://cre-trigger" || b.Name != cre.KeystoneName("solvency") || common.BytesToAddress(b.Owner[:]) != common.HexToAddress(cfg.CRE.WorkflowOwner) {
		t.Fatalf("chainlink binding = %+v", b)
	}
	// Health degrades rather than failing: the signer service is not wired in this ticket.
	if h := m.Service.Config(); h.Provider != "chainlink" {
		t.Fatal("provider")
	}
}

// The API process holds a reference to the trigger key, never the key: no CREConfig field may carry key material.
func TestCREConfigHoldsNoKeyMaterial(t *testing.T) {
	typ := reflect.TypeOf(config.CREConfig{})
	for i := 0; i < typ.NumField(); i++ {
		name := strings.ToLower(typ.Field(i).Name)
		if strings.Contains(name, "privatekey") || strings.Contains(name, "secret") || strings.Contains(name, "mnemonic") {
			t.Errorf("CREConfig.%s looks like key material", typ.Field(i).Name)
		}
	}
	// config.Load refuses a raw key in CRE_TRIGGER_SIGNER (config.TestCRE_TriggerSignerIsAReferenceNeverAKey).
}

type fakeLiabilities struct{}

func (fakeLiabilities) LiabilityTotals(context.Context) ([]cre.LedgerTotal, uint64, error) {
	return []cre.LedgerTotal{{Asset: "USDC.SOLANA", Total: "12.5"}}, 3, nil
}

type fakeReserves struct{}

func (fakeReserves) Reserves(context.Context) ([]cre.Reserve, error) {
	return []cre.Reserve{{Asset: "USDC.SOLANA", Amount: big.NewInt(13_000_000), Decimals: 6}}, nil
}

type fakeReader struct{}

func (fakeReader) FinalizedHead(context.Context) (uint64, error) { return 1, nil }
func (fakeReader) TransactionInput(context.Context, common.Hash) ([]byte, error) {
	return nil, nil
}
func (fakeReader) FilterLogs(context.Context, common.Address, uint64, uint64, [][]common.Hash) ([]chainlink.Log, error) {
	return nil, nil
}
