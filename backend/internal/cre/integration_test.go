//go:build integration

package cre_test

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/mock"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type fixedReserves struct{}

func (fixedReserves) Reserves(context.Context) ([]cre.Reserve, error) {
	return []cre.Reserve{{Asset: "USDC.SOLANA", Amount: big.NewInt(20_000_000), Decimals: 6}, {Asset: "SOL", Amount: big.NewInt(3_000_000_000), Decimals: 9}}, nil
}

type ledgerSource struct{ l *ledger.Service }

func (s ledgerSource) LiabilityTotals(ctx context.Context) ([]cre.LedgerTotal, uint64, error) {
	totals, head, err := s.l.LiabilityTotals(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]cre.LedgerTotal, 0, len(totals))
	for _, t := range totals {
		out = append(out, cre.LedgerTotal{Asset: t.Asset, Total: t.Total.String()})
	}
	return out, head, nil
}

func TestIntegration_SchemaConvergesAndStoreRoundTrips(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := database.ApplyMigrations(ctx, db); err != nil {
		t.Fatalf("ApplyMigrations after dev-style migrate: %v", err)
	}
	for _, table := range []string{"cre_attestations", "cre_subjects", "cre_runs", "cre_cursors"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("table %s missing", table)
		}
	}
	store := cre.NewPostgresStore(db)
	now := time.Now().UTC().Truncate(time.Microsecond)

	dep := cre.PendingDeposit{DepositID: "dep-1", Chain: "solana", Tx: "sig1", Token: "USDC", ExpectedAmountMinor: big.NewInt(42), Destination: "Dest1"}
	subject := cre.DepositSubject(dep, now)
	if err := store.RememberSubjects(ctx, []cre.Subject{subject}); err != nil {
		t.Fatal(err)
	}
	// A later ask must not rewrite the facts the verifier compares against.
	changed := cre.DepositSubject(cre.PendingDeposit{DepositID: "dep-1", Chain: "solana", Tx: "sig1", Token: "USDC", ExpectedAmountMinor: big.NewInt(99), Destination: "Dest1"}, now.Add(time.Minute))
	if err := store.RememberSubjects(ctx, []cre.Subject{changed}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.LookupSubject(ctx, cre.KindDepositFinality, cre.SubjectKey("dep-1"))
	if err != nil || !ok || got.ID != "dep-1" || got.Facts["expected_amount_minor"] != "42" || !got.AskedAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("subject = %+v ok %v err %v", got, ok, err)
	}
	if _, ok, _ := store.LookupSubject(ctx, cre.KindSolvency, cre.SubjectKey("dep-1")); ok {
		t.Fatal("subject visible under another kind")
	}

	row := cre.Attestation{
		ID: "7c1b4d0e-7a1f-4a4b-9e7a-1f2e3d4c5b6a", Kind: cre.KindDepositFinality, SubjectType: cre.SubjectDeposit, SubjectID: "dep-1",
		PayloadHash: cre.PayloadHash([]byte("payload")), Payload: []byte("payload"), Chain: "test-chain", TxHash: []byte{1, 2}, BlockNumber: 7,
		WorkflowID: cre.SubjectKey("wf"), ReportID: [2]byte{0, 1}, ObservedAt: now.Add(-time.Minute), RecordedAt: now, Status: cre.StatusAttested, Provider: cre.ProviderMock,
		OnChain: cre.OnChainEmitted, FactCheck: cre.StatusAttested,
		Item: cre.DepositItem{DepositID: cre.SubjectKey("dep-1"), Amount: big.NewInt(42), Verdict: cre.VerdictConfirmed},
	}
	inserted, err := store.SaveAttestations(ctx, []cre.Attestation{row})
	if err != nil || len(inserted) != 1 {
		t.Fatalf("first save inserted %d: %v", len(inserted), err)
	}
	again := row
	again.ID = "9d2e4f6a-1b3c-4d5e-8f7a-2b4c6d8e0f1a"
	if dup, err := store.SaveAttestations(ctx, []cre.Attestation{again}); err != nil || len(dup) != 0 {
		t.Fatalf("second save of the same item must insert nothing: %d %v", len(dup), err)
	}
	seen, err := store.Seen(ctx, cre.ProviderMock, row.PayloadHash)
	if err != nil || !seen {
		t.Fatalf("seen = %v err %v", seen, err)
	}
	if other, _ := store.Seen(ctx, cre.ProviderChainlink, row.PayloadHash); other {
		t.Fatal("a mock row must not count as seen for chainlink")
	}
	back, ok, err := store.GetAttestation(ctx, row.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok %v err %v", ok, err)
	}
	item, _ := back.Item.(map[string]any)
	if back.OnChain != cre.OnChainEmitted || back.FactCheck != cre.StatusAttested {
		t.Fatalf("contract outcome and fact check not persisted: %q %q", back.OnChain, back.FactCheck)
	}
	if back.Status != cre.StatusAttested || back.BlockNumber != 7 || back.WorkflowID != row.WorkflowID || item["amount_minor"] != "42" || !back.ObservedAt.Equal(row.ObservedAt) {
		t.Fatalf("round trip = %+v", back)
	}
	latest, ok, _ := store.LatestAttestation(ctx, cre.ProviderMock, cre.KindDepositFinality, "")
	if !ok || latest.ID != row.ID {
		t.Fatalf("latest = %+v", latest)
	}
	if _, ok, _ := store.LatestAttestation(ctx, cre.ProviderMock, cre.KindDepositFinality, cre.StatusFailed); ok {
		t.Fatal("status filter ignored")
	}
	list, _ := store.ListAttestations(ctx, cre.ProviderMock, "", 10)
	if len(list) != 1 {
		t.Fatalf("list = %d rows", len(list))
	}
	if err := store.RecordRun(ctx, cre.Run{Kind: cre.KindSolvency, Provider: "mock", ExecutionID: "e1", Status: cre.RunAccepted, StartedAt: now}); err != nil {
		t.Fatal(err)
	}
	run, ok, _ := store.LatestRun(ctx, cre.KindSolvency)
	if !ok || run.ExecutionID != "e1" {
		t.Fatalf("run = %+v", run)
	}
	scope := cre.CursorScope{Chain: "base-sepolia", Consumer: "0x1111", Kind: cre.KindSolvency}
	if err := store.SetCursor(ctx, scope, cre.Cursor{Block: 5, Seq: 9}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCursor(ctx, scope, cre.Cursor{Block: 6, Seq: 10}); err != nil {
		t.Fatal(err)
	}
	cur, _ := store.GetCursor(ctx, scope)
	if cur != (cre.Cursor{Block: 6, Seq: 10}) {
		t.Fatalf("cursor = %+v", cur)
	}
	// A height means nothing on another chain or consumer: those scopes start fresh.
	if cur, _ := store.GetCursor(ctx, cre.CursorScope{Chain: "base", Consumer: "0x1111", Kind: cre.KindSolvency}); cur != (cre.Cursor{}) {
		t.Fatalf("other chain cursor = %+v", cur)
	}
	if cur, _ := store.GetCursor(ctx, cre.CursorScope{Chain: "base-sepolia", Consumer: "0x2222", Kind: cre.KindSolvency}); cur != (cre.Cursor{}) {
		t.Fatalf("other consumer cursor = %+v", cur)
	}
	// Uniqueness is per provider, like Seen: the same bytes under another provider insert their own row.
	other := row
	other.ID, other.Provider = "3e5f7a9b-2c4d-4e6f-8a0b-1c3d5e7f9a1b", cre.ProviderChainlink
	if ins, err := store.SaveAttestations(ctx, []cre.Attestation{other}); err != nil || len(ins) != 1 {
		t.Fatalf("other provider insert: %d %v", len(ins), err)
	}
}

func TestIntegration_LedgerLiabilitiesFlowIntoACheckpointAndAMockAttestation(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()
	l := ledger.New(db)
	post := func(key, member, asset, amount string) {
		t.Helper()
		_, err := l.Post(ctx, ledger.Journal{
			Kind: ledger.KindPayment, Reference: ledger.Reference{Type: "payment", ID: key}, IdempotencyKey: key,
			Lines: []ledger.Line{
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: asset, Kind: ledger.KindAsset}, Amount: d(amount)},
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: member, Asset: asset, Kind: ledger.KindLiability}, Amount: d(amount).Neg()},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	post("p1", "m1", "USDC.SOLANA", "10.5")
	post("p2", "m2", "USDC.SOLANA", "0.25")
	post("p3", "m1", "SOL", "2")
	post("p4", "m1", "MYSTERY", "1")

	totals, head, err := l.LiabilityTotals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(totals) != 3 || totals[1].Asset != "SOL" || !totals[2].Total.Equal(d("10.75")) || head != 4 {
		t.Fatalf("totals = %+v head %d", totals, head)
	}

	gw := cre.GatewayID("https://pay.example.test")
	store := cre.NewPostgresStore(db)
	m, err := mock.New(gw, mock.WithReserves(fixedReserves{}))
	if err != nil {
		t.Fatal(err)
	}
	bindings := map[cre.Kind]cre.Binding{cre.KindSolvency: m.Binding(cre.KindSolvency), cre.KindDepositFinality: m.Binding(cre.KindDepositFinality), cre.KindConversionReference: m.Binding(cre.KindConversionReference)}
	verifier := &cre.Verifier{Provider: cre.ProviderMock, GatewayID: gw, MockSigner: m.Owner(), Chain: "test-chain", Bindings: bindings}
	svc := cre.NewService(cre.Config{Provider: cre.ProviderMock, Chain: "test-chain", GatewayID: gw, SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute, PollInterval: time.Minute, Bindings: bindings}, m, store, verifier, cre.WithLiabilities(ledgerSource{l}))

	cp, err := svc.Liabilities(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(cp.Assets) != 2 || cp.Assets[1].Asset != "USDC.SOLANA" || cp.Assets[1].Liabilities.String() != "10750000" || cp.MaxJournalID != 4 {
		t.Fatalf("checkpoint omitted or mis-scaled an asset: %+v", cp)
	}
	again, _ := svc.Liabilities(ctx, true)
	if again.Hash != cp.Hash {
		t.Fatal("a checkpoint younger than the interval must be reused")
	}
	if _, err := svc.Run(ctx, cre.KindSolvency); err != nil {
		t.Fatal(err)
	}
	n, err := svc.Poll(ctx, cre.KindSolvency)
	if err != nil || n != 1 {
		t.Fatalf("poll = %d err %v", n, err)
	}
	rows, _ := store.ListAttestations(ctx, cre.ProviderMock, cre.KindSolvency, 10)
	if len(rows) != 2 {
		t.Fatalf("want one row per asset, got %d", len(rows))
	}
	for _, r := range rows {
		if r.Status != cre.StatusAttested || r.Provider != cre.ProviderMock {
			t.Fatalf("row = %+v", r)
		}
	}
	// Polling the same mock records again is a replay the store refuses, not a second row.
	raws, _, _ := m.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	if _, err := svc.Submit(ctx, raws[0]); !errors.Is(err, cre.ErrReplayed) {
		t.Fatalf("replay = %v", err)
	}
	rep, _ := svc.Status(ctx)
	if rep.Workflows[0].State != "fresh" || rep.Workflows[0].LastRun == nil || rep.Workflows[0].LastVerified == nil {
		t.Fatalf("status = %+v", rep.Workflows[0])
	}
	// The hashed figures are reproducible: lines above the head are not part of the sum (one consistent snapshot).
	post("p5", "m1", "USDC.SOLANA", "1")
	totals2, head2, _ := l.LiabilityTotals(ctx)
	if head2 != 5 || !totals2[2].Total.Equal(d("11.75")) {
		t.Fatalf("totals after a post = %+v head %d", totals2, head2)
	}
}
