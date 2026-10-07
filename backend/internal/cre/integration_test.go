//go:build integration

package cre_test

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"gorm.io/gorm"

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

func (s ledgerSource) LiabilityTotals(ctx context.Context) (cre.LedgerSnapshot, error) {
	snap, err := s.l.LiabilityTotals(ctx)
	if err != nil {
		return cre.LedgerSnapshot{}, err
	}
	out := cre.LedgerSnapshot{Head: snap.Head, TakenAt: snap.TakenAt, Totals: make([]cre.LedgerTotal, 0, len(snap.Totals))}
	for _, t := range snap.Totals {
		out.Totals = append(out.Totals, cre.LedgerTotal{Asset: t.Asset, Total: t.Total.String()})
	}
	return out, nil
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

	snap, err := l.LiabilityTotals(ctx)
	if err != nil {
		t.Fatal(err)
	}
	totals, head := snap.Totals, snap.Head
	if snap.TakenAt.IsZero() {
		t.Fatal("snapshot carries no database time")
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
	snap2, _ := l.LiabilityTotals(ctx)
	totals2, head2 := snap2.Totals, snap2.Head
	if head2 != 5 || !totals2[2].Total.Equal(d("11.75")) {
		t.Fatalf("totals after a post = %+v head %d", totals2, head2)
	}
}

// auditRecipe runs docs/cre/OPERATIONS.md "Recomputing a checkpoint hash" verbatim: both SQL blocks from the
// document, then the scaling and canonical JSON it describes. It returns the per-asset figures and the hash.
func auditRecipe(t *testing.T, db *gorm.DB, hash [32]byte) ([]cre.AssetTotal, uint64, [32]byte) {
	t.Helper()
	doc, err := os.ReadFile("../../../docs/cre/OPERATIONS.md")
	if err != nil {
		t.Fatal(err)
	}
	section := string(doc)
	start := strings.Index(section, "### Recomputing a checkpoint hash")
	if start < 0 {
		t.Fatal("OPERATIONS.md has no audit recipe section")
	}
	section = section[start:]
	if end := strings.Index(section[4:], "\n## "); end >= 0 {
		section = section[:end+4]
	}
	var blocks []string
	for rest := section; ; {
		i := strings.Index(rest, "```sql")
		if i < 0 {
			break
		}
		rest = rest[i+len("```sql"):]
		j := strings.Index(rest, "```")
		blocks = append(blocks, rest[:j])
		rest = rest[j+3:]
	}
	if len(blocks) != 2 {
		t.Fatalf("audit recipe has %d SQL blocks, want 2", len(blocks))
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var headText, takenAt sql.NullString
	if err := sqlDB.QueryRowContext(ctx, blocks[0], hex.EncodeToString(hash[:])).Scan(&headText, &takenAt); err != nil {
		t.Fatalf("recipe step 1: %v", err)
	}
	if !headText.Valid || !takenAt.Valid {
		t.Fatalf("recipe step 1: checkpoint facts lack max_journal_id or taken_at (%v, %v)", headText, takenAt)
	}
	head, err := strconv.ParseUint(headText.String, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := sqlDB.QueryContext(ctx, blocks[1], "test", head)
	if err != nil {
		t.Fatalf("recipe step 2: %v", err)
	}
	defer rows.Close()
	decimals, _ := cre.ParseDecimals("")
	var assets []cre.AssetTotal
	var parts []string
	for rows.Next() {
		var asset, liabilities string
		if err := rows.Scan(&asset, &liabilities); err != nil {
			t.Fatal(err)
		}
		minor, dec, ok := decimals.Minor(asset, liabilities)
		if !ok || minor.Sign() < 0 {
			continue
		}
		assets = append(assets, cre.AssetTotal{Asset: asset, Liabilities: minor, Decimals: dec})
		parts = append(parts, fmt.Sprintf(`{"asset":%q,"liabilities_minor":"%s","decimals":%d}`, asset, minor, dec))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	canon := fmt.Sprintf(`{"max_journal_id":%d,"taken_at":%q,"assets":[%s]}`, head, takenAt.String, strings.Join(parts, ","))
	var out [32]byte
	copy(out[:], crypto.Keccak256([]byte(canon)))
	return assets, head, out
}

// R2: the head is a commit watermark. A posting transaction that took a lower journal id and commits after a
// later one must be inside the checkpoint, so the documented recipe reproduces the hash exactly.
func TestIntegration_CheckpointHashReproducesFromTheAuditRecipe(t *testing.T) {
	db, cleanup := database.NewTestDB(t)
	defer cleanup()
	ctx := context.Background()
	l := ledger.New(db)
	journal := func(key, member, asset, amount string) ledger.Journal {
		return ledger.Journal{
			Kind: ledger.KindPayment, Reference: ledger.Reference{Type: "payment", ID: key}, IdempotencyKey: key,
			Lines: []ledger.Line{
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "hot", Asset: asset, Kind: ledger.KindAsset}, Amount: d(amount)},
				{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: member, Asset: asset, Kind: ledger.KindLiability}, Amount: d(amount).Neg()},
			},
		}
	}
	// Accounts exist before the race so no poster waits on another's uncommitted account insert.
	for i, asset := range []string{"USDC.SOLANA", "SOL"} {
		for _, m := range []string{"m1", "m2", "m3"} {
			if _, err := l.Post(ctx, journal(fmt.Sprintf("seed-%d-%s", i, m), m, asset, "1")); err != nil {
				t.Fatal(err)
			}
		}
	}
	svc := cre.NewService(cre.Config{Provider: cre.ProviderMock, Chain: "test-chain", GatewayID: cre.GatewayID("https://pay.example.test"), SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute, PollInterval: time.Minute},
		nil, cre.NewPostgresStore(db), &cre.Verifier{Provider: cre.ProviderMock}, cre.WithLiabilities(ledgerSource{l}))
	check := func(cp cre.Checkpoint) {
		t.Helper()
		assets, head, hash := auditRecipe(t, db, cp.Hash)
		if head != cp.MaxJournalID || len(assets) != len(cp.Assets) {
			t.Fatalf("recipe head %d assets %+v, checkpoint head %d assets %+v", head, assets, cp.MaxJournalID, cp.Assets)
		}
		for i := range assets {
			if assets[i].Asset != cp.Assets[i].Asset || assets[i].Liabilities.Cmp(cp.Assets[i].Liabilities) != 0 {
				t.Fatalf("asset %d: recipe %s %s, checkpoint %s %s (head %d)", i, assets[i].Asset, assets[i].Liabilities, cp.Assets[i].Asset, cp.Assets[i].Liabilities, head)
			}
		}
		if hash != cp.Hash {
			t.Fatalf("recipe hash %x, checkpoint hash %x", hash, cp.Hash)
		}
	}

	// Deterministic case: A takes a journal id and stays open; B takes a higher id and commits; the checkpoint
	// starts; A commits afterwards. A visibility watermark would hash B without A.
	aPosted, releaseA, aDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		aDone <- l.Transaction(ctx, func(tx *gorm.DB) error {
			if _, err := l.PostIn(ctx, tx, journal("held-a", "m1", "USDC.SOLANA", "5")); err != nil {
				return err
			}
			close(aPosted)
			<-releaseA
			return nil
		})
	}()
	<-aPosted
	if _, err := l.Post(ctx, journal("later-b", "m2", "USDC.SOLANA", "7")); err != nil {
		t.Fatal(err)
	}
	cpDone := make(chan cre.Checkpoint, 1)
	cpErr := make(chan error, 1)
	go func() {
		cp, err := svc.PublishCheckpoint(ctx)
		cpErr <- err
		cpDone <- cp
	}()
	time.Sleep(300 * time.Millisecond)
	close(releaseA)
	if err := <-aDone; err != nil {
		t.Fatal(err)
	}
	if err := <-cpErr; err != nil {
		t.Fatal(err)
	}
	check(<-cpDone)

	// Stress: posters run while checkpoints are taken; every checkpoint must reproduce.
	stop := make(chan struct{})
	var wg sync.WaitGroup
	var posted atomic.Int64
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			members := []string{"m1", "m2", "m3"}
			for i := 0; ; i++ {
				select {
				case <-stop:
					return
				default:
				}
				asset := []string{"USDC.SOLANA", "SOL"}[(w+i)%2]
				err := l.Transaction(ctx, func(tx *gorm.DB) error {
					if _, err := l.PostIn(ctx, tx, journal(fmt.Sprintf("w%d-%d", w, i), members[i%3], asset, "0.5")); err != nil {
						return err
					}
					time.Sleep(time.Duration(i%4) * time.Millisecond)
					return nil
				})
				if err != nil {
					t.Error(err)
					return
				}
				posted.Add(1)
			}
		}(w)
	}
	var cps []cre.Checkpoint
	for i := 0; i < 8; i++ {
		time.Sleep(25 * time.Millisecond)
		cp, err := svc.PublishCheckpoint(ctx)
		if err != nil {
			t.Fatal(err)
		}
		cps = append(cps, cp)
	}
	close(stop)
	wg.Wait()
	if posted.Load() < 20 {
		t.Fatalf("only %d concurrent postings; the race was not exercised", posted.Load())
	}
	for _, cp := range cps {
		check(cp)
	}
}
