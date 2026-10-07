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
		Item: cre.DepositItem{DepositID: cre.SubjectKey("dep-1"), Amount: big.NewInt(42), Verdict: cre.VerdictConfirmed},
	}
	if err := store.SaveAttestations(ctx, []cre.Attestation{row}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveAttestations(ctx, []cre.Attestation{row}); err != nil {
		t.Fatalf("second save of the same subject must be a no-op: %v", err)
	}
	seen, err := store.Seen(ctx, row.PayloadHash)
	if err != nil || !seen {
		t.Fatalf("seen = %v err %v", seen, err)
	}
	back, ok, err := store.GetAttestation(ctx, row.ID)
	if err != nil || !ok {
		t.Fatalf("get: ok %v err %v", ok, err)
	}
	item, _ := back.Item.(map[string]any)
	if back.Status != cre.StatusAttested || back.BlockNumber != 7 || back.WorkflowID != row.WorkflowID || item["amount_minor"] != "42" || !back.ObservedAt.Equal(row.ObservedAt) {
		t.Fatalf("round trip = %+v", back)
	}
	latest, ok, _ := store.LatestAttestation(ctx, cre.KindDepositFinality)
	if !ok || latest.ID != row.ID {
		t.Fatalf("latest = %+v", latest)
	}
	list, _ := store.ListAttestations(ctx, "", 10)
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
	if err := store.SetCursor(ctx, cre.KindSolvency, cre.Cursor{Block: 5, Seq: 9}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetCursor(ctx, cre.KindSolvency, cre.Cursor{Block: 6, Seq: 10}); err != nil {
		t.Fatal(err)
	}
	cur, _ := store.GetCursor(ctx, cre.KindSolvency)
	if cur != (cre.Cursor{Block: 6, Seq: 10}) {
		t.Fatalf("cursor = %+v", cur)
	}
	if cur, _ := store.GetCursor(ctx, cre.KindConversionReference); cur != (cre.Cursor{}) {
		t.Fatalf("fresh cursor = %+v", cur)
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
	m, err := mock.New(gw)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &cre.Verifier{Provider: cre.ProviderMock, GatewayID: gw, Owner: m.Owner(), MockSigner: m.Owner(), MaxReportAge: time.Hour, Chain: "test-chain",
		WorkflowIDs: map[cre.Kind][32]byte{cre.KindSolvency: m.WorkflowID(cre.KindSolvency), cre.KindDepositFinality: m.WorkflowID(cre.KindDepositFinality), cre.KindConversionReference: m.WorkflowID(cre.KindConversionReference)}}
	svc := cre.NewService(cre.Config{Provider: cre.ProviderMock, Chain: "test-chain", GatewayID: gw, SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute, PollInterval: time.Minute, MaxReportAge: time.Hour, WorkflowIDs: verifier.WorkflowIDs}, m, store, verifier, cre.WithLiabilities(ledgerSource{l}))

	cp, err := svc.Liabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(cp.Assets) != 2 || cp.Assets[1].Asset != "USDC.SOLANA" || cp.Assets[1].Liabilities.String() != "10750000" || cp.MaxJournalID != 4 {
		t.Fatalf("checkpoint omitted or mis-scaled an asset: %+v", cp)
	}
	again, _ := svc.Liabilities(ctx)
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
	rows, _ := store.ListAttestations(ctx, cre.KindSolvency, 10)
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
	if rep.Workflows[0].State != "fresh" || rep.Workflows[0].LastRun == nil {
		t.Fatalf("status = %+v", rep.Workflows[0])
	}
}
