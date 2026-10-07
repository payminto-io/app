package mock

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/conformance"
)

type reserves []cre.Reserve

func (r reserves) Reserves(context.Context) ([]cre.Reserve, error) { return r, nil }

func TestConformance(t *testing.T) {
	gw := cre.GatewayID("https://pay.example.test")
	p, err := New(gw, WithReserves(reserves{{Asset: "USDC.SOLANA", Amount: big.NewInt(1_500_000), Decimals: 6}}))
	if err != nil {
		t.Fatal(err)
	}
	conformance.Run(t, conformance.Harness{Attester: p, Input: conformance.Inputs(gw)})
}

func TestMockReportsVerifyAndAreScriptable(t *testing.T) {
	gw := cre.GatewayID("https://pay.example.test")
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	p, err := New(gw, WithReserves(reserves{{Asset: "USDC.SOLANA", Amount: big.NewInt(1_500_000)}, {Asset: "USDC.SOLANA", Amount: big.NewInt(1)}}), WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	inputs := conformance.Inputs(gw)
	ctx := context.Background()

	v := &cre.Verifier{
		Provider: cre.ProviderMock, GatewayID: gw, MockSigner: p.Owner(),
		Bindings: map[cre.Kind]cre.Binding{cre.KindSolvency: p.Binding(cre.KindSolvency), cre.KindDepositFinality: p.Binding(cre.KindDepositFinality), cre.KindConversionReference: p.Binding(cre.KindConversionReference)},
		Subjects: cre.NewMemoryStore(), Now: func() time.Time { return now },
	}
	// The mock writes Keystone names: the verifier's binding uses the same derivation as the contract.
	if p.Binding(cre.KindSolvency).Name != cre.KeystoneName("solvency") {
		t.Fatal("mock workflow name is not the Keystone derivation")
	}
	_ = v.Subjects.RememberSubjects(ctx, []cre.Subject{
		cre.DepositSubject(cre.PendingDeposit{DepositID: "dep-conformance", Chain: "solana", Tx: "sig1", Token: "USDC", ExpectedAmountMinor: big.NewInt(5_000_000), Destination: "Dest111"}, now),
	})

	if _, err := p.Trigger(ctx, cre.KindSolvency, inputs(cre.KindSolvency)); err != nil {
		t.Fatal(err)
	}
	raws, _, _ := p.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	report, _ := cre.DecodeReport(raws[0].Report)
	items := report.Items.([]cre.SolvencyItem)
	if items[0].Reserves.Int64() != 1_500_001 || items[0].Liabilities.Int64() != 1_000_000 || items[0].Decimals != 6 {
		t.Fatalf("solvency item = %+v", items[0])
	}

	p.ScriptVerdict("dep-conformance", cre.VerdictMismatch)
	if _, err := p.Trigger(ctx, cre.KindDepositFinality, inputs(cre.KindDepositFinality)); err != nil {
		t.Fatal(err)
	}
	raws, _, _ = p.Poll(ctx, cre.KindDepositFinality, cre.Cursor{})
	rows, err := v.Verify(ctx, raws[0])
	if err != nil {
		t.Fatalf("mock report did not verify: %v", err)
	}
	if rows[0].Status != cre.StatusMismatch || rows[0].SubjectID != "dep-conformance" {
		t.Fatalf("scripted mismatch row = %+v", rows[0])
	}

	p.ScriptFailure(cre.KindConversionReference, errors.New("scripted outage"))
	if _, err := p.Trigger(ctx, cre.KindConversionReference, inputs(cre.KindConversionReference)); err == nil || err.Error() != "scripted outage" {
		t.Fatalf("scripted failure not honoured: %v", err)
	}
	p.ScriptFailure(cre.KindConversionReference, nil)

	p.ScriptDelay(time.Minute)
	if _, err := p.Trigger(ctx, cre.KindConversionReference, inputs(cre.KindConversionReference)); err != nil {
		t.Fatal(err)
	}
	if raws, _, _ := p.Poll(ctx, cre.KindConversionReference, cre.Cursor{}); len(raws) != 0 {
		t.Fatal("delayed report visible before its delay")
	}
	now = now.Add(2 * time.Minute)
	if raws, _, _ := p.Poll(ctx, cre.KindConversionReference, cre.Cursor{}); len(raws) != 1 {
		t.Fatal("delayed report missing after its delay")
	}
	if _, err := p.Trigger(ctx, cre.KindSolvency, []byte("not json")); !errors.Is(err, cre.ErrInvalidReport) {
		t.Fatalf("bad input: %v", err)
	}
}

// Without a reserve source the mock refuses a solvency run instead of attesting reserves it never observed.
func TestMockRefusesSolvencyWithoutReserves(t *testing.T) {
	gw := cre.GatewayID("https://pay.example.test")
	p, err := New(gw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Trigger(context.Background(), cre.KindSolvency, conformance.Inputs(gw)(cre.KindSolvency)); !errors.Is(err, ErrNoReserveSource) {
		t.Fatalf("err = %v", err)
	}
	partial, _ := New(gw, WithReserves(reserves{{Asset: "SOL", Amount: big.NewInt(1)}}))
	if _, err := partial.Trigger(context.Background(), cre.KindSolvency, conformance.Inputs(gw)(cre.KindSolvency)); !errors.Is(err, ErrNoReserveSource) {
		t.Fatalf("an asset with no observed reserve must be left out, not zeroed: %v", err)
	}
}
