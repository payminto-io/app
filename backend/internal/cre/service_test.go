package cre

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"
)

// scriptedAttester returns the raws a test prepared and records cursors it was asked for.
type scriptedAttester struct {
	raws   []RawAttestation
	next   Cursor
	err    error
	polled []Cursor
}

func (s *scriptedAttester) Name() string { return ProviderChainlink }
func (s *scriptedAttester) Trigger(context.Context, Kind, []byte) (string, error) {
	return "", ErrUnsupported
}
func (s *scriptedAttester) Health(context.Context) Health { return Health{Status: HealthOK} }
func (s *scriptedAttester) Poll(_ context.Context, _ Kind, c Cursor) ([]RawAttestation, Cursor, error) {
	s.polled = append(s.polled, c)
	if s.err != nil {
		return nil, c, s.err
	}
	return s.raws, s.next, nil
}

func newPollService(t *testing.T, f *fixture, att *scriptedAttester) (*Service, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	_ = store.RememberSubjects(context.Background(), []Subject{DepositSubject(f.deposit(), f.now)})
	v := f.verifier(ProviderChainlink)
	v.Subjects, v.Seen = nil, nil
	svc := NewService(Config{Provider: ProviderChainlink, Chain: "test-chain", GatewayID: f.gateway, Bindings: f.bindings, SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute, PollInterval: time.Minute}, att, store, v, WithClock(func() time.Time { return f.now }))
	return svc, store
}

func depositScope() CursorScope { return CursorScope{Chain: "test-chain", Kind: KindDepositFinality} }

func (f *fixture) depositRaw(t *testing.T, block uint64, final bool, slot uint64) RawAttestation {
	t.Helper()
	it, _ := DepositItemFromSubject(DepositSubject(f.deposit(), f.now))
	it.Verdict, it.SlotOrBlock = VerdictConfirmed, slot
	raw := f.onChain(KindDepositFinality, f.depositReport(t, []DepositItem{it}))
	raw.Evidence.BlockNumber, raw.Evidence.HeadBlock, raw.Evidence.Final = block, 110, final
	raw.Evidence.TxHash = []byte{byte(block), byte(slot)}
	return raw
}

// C1: a report that is not yet final stops the cursor at its block; the next poll records it.
func TestPollCursorWaitsForFinality(t *testing.T) {
	f := newFixture(t)
	_ = f.verifier(ProviderChainlink)
	ctx := context.Background()
	att := &scriptedAttester{next: Cursor{Block: 111}}
	svc, store := newPollService(t, f, att)
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	att.raws = []RawAttestation{f.depositRaw(t, 100, true, 1), f.depositRaw(t, 105, false, 2)}

	n, err := svc.Poll(ctx, KindDepositFinality)
	if n != 1 || !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("first pass: recorded %d err %v", n, err)
	}
	cur, _ := store.GetCursor(ctx, depositScope())
	if cur.Block != 105 {
		t.Fatalf("cursor advanced past an unconfirmed block: %+v", cur)
	}
	att.raws[1].Evidence.Final = true
	n, err = svc.Poll(ctx, KindDepositFinality)
	if err != nil || n != 1 {
		t.Fatalf("second pass: recorded %d err %v", n, err)
	}
	cur, _ = store.GetCursor(ctx, depositScope())
	rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10)
	if cur.Block != 111 || len(rows) != 2 {
		t.Fatalf("after second pass: cursor %+v rows %d", cur, len(rows))
	}
}

// C1: a provider or RPC error leaves the cursor untouched; a definitive refusal is kept as a failed row and passed.
func TestPollCursorOnErrorsAndRefusals(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	att := &scriptedAttester{err: errors.New("Post \"https://rpc.example/v2/8f3a1c9d2e7b4a6f5c8d9e0f1a2b3c4d\": timeout")}
	svc, store := newPollService(t, f, att)
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	if _, err := svc.Poll(ctx, KindDepositFinality); !errors.Is(err, ErrNotFinal) || IsRejection(err) {
		t.Fatalf("rpc error must be retryable: %v", err)
	}
	if cur, _ := store.GetCursor(ctx, depositScope()); cur.Block != 90 {
		t.Fatalf("cursor moved on an RPC error: %+v", cur)
	}

	att.err, att.next = nil, Cursor{Block: 111}
	forged := f.depositRaw(t, 100, true, 3)
	forged.Evidence.ReportHash[0] ^= 1
	att.raws = []RawAttestation{forged}
	if n, err := svc.Poll(ctx, KindDepositFinality); err != nil || n != 0 {
		t.Fatalf("forged pass: %d %v", n, err)
	}
	cur, _ := store.GetCursor(ctx, depositScope())
	rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10)
	if cur.Block != 111 || len(rows) != 1 || rows[0].Status != StatusFailed || rows[0].SubjectType != "report" {
		t.Fatalf("refusal not recorded: cursor %+v rows %+v", cur, rows)
	}
}

// M2: a concurrent duplicate that inserts nothing is a replay, with no ids and no event.
func TestSubmitReportsOnlyInsertedRows(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc, _ := newPollService(t, f, &scriptedAttester{})
	raw := f.depositRaw(t, 100, true, 4)
	rows, err := svc.Submit(ctx, raw)
	if err != nil || len(rows) != 1 {
		t.Fatalf("first submit: %v", err)
	}
	svc.verifier.Seen = func(context.Context, []byte) (bool, error) { return false, nil } // the race: Seen says no
	if _, err := svc.Submit(ctx, raw); !errors.Is(err, ErrReplayed) {
		t.Fatalf("second submit = %v, want ErrReplayed", err)
	}
}

// I5: freshness comes from the latest verified row; a mismatch row never reads as fresh.
func TestStatusFreshnessUsesVerifiedRowsOnly(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc, _ := newPollService(t, f, &scriptedAttester{})
	it, _ := DepositItemFromSubject(DepositSubject(f.deposit(), f.now))
	it.Verdict, it.Amount = VerdictConfirmed, big.NewInt(1)
	bad := f.onChain(KindDepositFinality, f.depositReport(t, []DepositItem{it}))
	if rows, err := svc.Submit(ctx, bad); err != nil || rows[0].Status != StatusMismatch {
		t.Fatalf("mismatch submit: %+v %v", rows, err)
	}
	rep, _ := svc.Status(ctx)
	w := rep.Workflows[1]
	if w.Kind != KindDepositFinality || w.State != "never" || w.LastAttestation == nil || w.LastVerified != nil {
		t.Fatalf("status after a mismatch = %+v", w)
	}
	if _, err := svc.Submit(ctx, f.depositRaw(t, 100, true, 5)); err != nil {
		t.Fatal(err)
	}
	rep, _ = svc.Status(ctx)
	if rep.Workflows[1].State != "fresh" || rep.Workflows[1].LastVerified == nil {
		t.Fatalf("status after a verified row = %+v", rep.Workflows[1])
	}
}

// I4: a public liabilities read never publishes a checkpoint; an authenticated one may.
func TestLiabilitiesPublicReadNeverWrites(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	svc, store := newPollService(t, f, &scriptedAttester{})
	WithLiabilities(fakeLiabilities{totals: []LedgerTotal{{"USDC.SOLANA", "1"}}, max: 1})(svc)
	if _, err := svc.Liabilities(ctx, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("public read with nothing published = %v", err)
	}
	if _, ok, _ := store.LatestSubject(ctx, KindSolvency); ok {
		t.Fatal("public read published a checkpoint")
	}
	cp, err := svc.Liabilities(ctx, true)
	if err != nil || len(cp.Assets) != 1 {
		t.Fatalf("authenticated read: %+v %v", cp, err)
	}
	pub, err := svc.Liabilities(ctx, false)
	if err != nil || pub.Hash != cp.Hash {
		t.Fatalf("public read after publish: %v", err)
	}
	if _, has := CheckpointJSON(pub, false)["max_journal_id"]; has {
		t.Fatal("public body carries the journal counter")
	}
	if _, has := CheckpointJSON(pub, true)["max_journal_id"]; !has {
		t.Fatal("authenticated body lacks the journal counter")
	}
}

// N1: a refusal caused by this gateway's own configuration is retryable: nothing is stored, the cursor stays,
// and once the binding is corrected the same on-chain report is recorded exactly once.
func TestPollConfigurationMismatchIsRetryable(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	att := &scriptedAttester{next: Cursor{Block: 111}}
	svc, store := newPollService(t, f, att)
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	raw := f.depositRaw(t, 100, true, 7)
	att.raws = []RawAttestation{raw}

	good := f.bindings[KindDepositFinality]
	typo := good
	typo.Name = KeystoneName("deposit-finaltiy")
	svc.verifier.Bindings[KindDepositFinality] = typo
	n, err := svc.Poll(ctx, KindDepositFinality)
	if n != 0 || !errors.Is(err, ErrWrongName) || IsRejection(err) == false || !IsConfigurationMismatch(err) {
		t.Fatalf("mistyped binding: recorded %d err %v", n, err)
	}
	if cur, _ := store.GetCursor(ctx, depositScope()); cur.Block != 100 {
		t.Fatalf("cursor passed a configuration refusal: %+v", cur)
	}
	if rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10); len(rows) != 0 {
		t.Fatalf("a configuration refusal stored rows: %+v", rows)
	}
	// Clock ahead of the DON beyond tolerance is the same class.
	svc.verifier.Bindings[KindDepositFinality] = good
	late := f.now
	svc.verifier.Now = func() time.Time { return late.Add(-time.Hour) }
	if _, err := svc.Poll(ctx, KindDepositFinality); !errors.Is(err, ErrClockAhead) || !IsConfigurationMismatch(err) {
		t.Fatalf("clock: %v", err)
	}
	svc.verifier.Now = func() time.Time { return late }

	// Fix the binding, rewind, re-poll: accepted once; a second re-poll is a replay of a stored attestation.
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	n, err = svc.Poll(ctx, KindDepositFinality)
	if err != nil || n != 1 {
		t.Fatalf("after the fix: recorded %d err %v", n, err)
	}
	cur, _ := store.GetCursor(ctx, depositScope())
	rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10)
	if cur.Block != 111 || len(rows) != 1 || rows[0].Status != StatusAttested {
		t.Fatalf("after the fix: cursor %+v rows %+v", cur, rows)
	}
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	if n, err := svc.Poll(ctx, KindDepositFinality); err != nil || n != 0 {
		t.Fatalf("re-poll of a stored report: %d %v", n, err)
	}
	if rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10); len(rows) != 1 {
		t.Fatal("re-poll duplicated the row")
	}
}

// A stored refusal row never counts for replay, and refusals of undecodable calldata are keyed per log.
func TestRefusalRowsNeverBlockRecording(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	att := &scriptedAttester{next: Cursor{Block: 111}}
	svc, store := newPollService(t, f, att)
	_ = store.SetCursor(ctx, depositScope(), Cursor{Block: 90})
	a := f.depositRaw(t, 100, true, 8)
	a.Report, a.Evidence.ReportHash = nil, [32]byte{}
	b := f.depositRaw(t, 101, true, 9)
	b.Report, b.Evidence.ReportHash, b.Evidence.LogIndex = nil, [32]byte{}, 3
	att.raws = []RawAttestation{a, b}
	if n, err := svc.Poll(ctx, KindDepositFinality); err != nil || n != 0 {
		t.Fatalf("undecodable: %d %v", n, err)
	}
	rows, _ := store.ListAttestations(ctx, ProviderChainlink, KindDepositFinality, 10)
	if len(rows) != 2 || rows[0].SubjectType != "report" || string(rows[0].PayloadHash) == string(rows[1].PayloadHash) {
		t.Fatalf("refusal rows = %+v", rows)
	}
	seen, _ := store.Seen(ctx, ProviderChainlink, PayloadHash(nil))
	if seen {
		t.Fatal("a refusal row counted as seen")
	}
}
