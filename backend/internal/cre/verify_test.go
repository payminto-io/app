package cre

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// memSubjects is an in-memory SubjectIndex for tests.
type memSubjects struct {
	rows map[Kind]map[[32]byte]Subject
}

func newMemSubjects() *memSubjects { return &memSubjects{rows: map[Kind]map[[32]byte]Subject{}} }

func (m *memSubjects) RememberSubjects(_ context.Context, subjects []Subject) error {
	for _, s := range subjects {
		if m.rows[s.Kind] == nil {
			m.rows[s.Kind] = map[[32]byte]Subject{}
		}
		m.rows[s.Kind][s.Key] = s
	}
	return nil
}

func (m *memSubjects) LookupSubject(_ context.Context, kind Kind, key [32]byte) (Subject, bool, error) {
	s, ok := m.rows[kind][key]
	return s, ok, nil
}

type fixture struct {
	key      *DevKey
	owner    [20]byte
	consumer [20]byte
	gateway  [32]byte
	wfIDs    map[Kind][32]byte
	subjects *memSubjects
	now      time.Time
	seen     map[string]bool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	key, err := NewDevKey()
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		key: key, gateway: GatewayID("https://pay.example.test"), subjects: newMemSubjects(),
		now: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC), seen: map[string]bool{},
		wfIDs: map[Kind][32]byte{KindSolvency: SubjectKey("wf-solvency"), KindDepositFinality: SubjectKey("wf-deposit"), KindConversionReference: SubjectKey("wf-conversion")},
	}
	copy(f.owner[:], common.HexToAddress("0x3333333333333333333333333333333333333333").Bytes())
	copy(f.consumer[:], common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	return f
}

func (f *fixture) verifier(provider string) *Verifier {
	v := &Verifier{
		Provider: provider, GatewayID: f.gateway, Consumer: f.consumer, Owner: f.owner, WorkflowIDs: f.wfIDs,
		Confirmations: 3, MaxReportAge: 24 * time.Hour, Subjects: f.subjects, Chain: "test-chain",
		Now:  func() time.Time { return f.now },
		Seen: func(_ context.Context, h []byte) (bool, error) { return f.seen[string(h)], nil },
	}
	if provider == ProviderMock {
		v.MockSigner = f.key.Address()
		v.Owner = f.owner
	}
	return v
}

func (f *fixture) metadata(kind Kind) []byte {
	m := Metadata{WorkflowID: f.wfIDs[kind], Owner: f.owner, ReportID: [2]byte{0, 1}}
	copy(m.WorkflowName[:], string(kind))
	return m.Encode()
}

func (f *fixture) deposit() PendingDeposit {
	return PendingDeposit{DepositID: "dep-1", Chain: "solana", Tx: "5sig", LogIndexOrSig: "5sig", Token: "USDC", ExpectedAmountMinor: big.NewInt(42_000_000), Destination: "DestAddr111"}
}

func (f *fixture) depositReport(t *testing.T, items []DepositItem) []byte {
	t.Helper()
	payload, err := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(-time.Minute), Items: items})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func (f *fixture) confirmedItem(t *testing.T) DepositItem {
	t.Helper()
	s := DepositSubject(f.deposit(), f.now.Add(-2*time.Minute))
	if err := f.subjects.RememberSubjects(context.Background(), []Subject{s}); err != nil {
		t.Fatal(err)
	}
	it, err := DepositItemFromSubject(s)
	if err != nil {
		t.Fatal(err)
	}
	it.Verdict = VerdictConfirmed
	it.SlotOrBlock = 1234
	return it
}

func (f *fixture) onChain(kind Kind, report []byte) RawAttestation {
	return RawAttestation{Kind: kind, Metadata: f.metadata(kind), Report: report, Evidence: Evidence{Emitter: f.consumer, TxHash: []byte{0xab}, BlockNumber: 100, HeadBlock: 110}}
}

func (f *fixture) signed(t *testing.T, kind Kind, report []byte) RawAttestation {
	t.Helper()
	meta := f.metadata(kind)
	sig, err := f.key.Sign(meta, report)
	if err != nil {
		t.Fatal(err)
	}
	return RawAttestation{Kind: kind, Metadata: meta, Report: report, Evidence: Evidence{Signature: sig}}
}

func TestVerify_ValidChainlinkDeposit(t *testing.T) {
	f := newFixture(t)
	raw := f.onChain(KindDepositFinality, f.depositReport(t, []DepositItem{f.confirmedItem(t)}))
	rows, err := f.verifier(ProviderChainlink).Verify(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Status != StatusAttested || rows[0].SubjectID != "dep-1" || rows[0].Provider != ProviderChainlink || rows[0].Chain != "test-chain" {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].WorkflowID != f.wfIDs[KindDepositFinality] || rows[0].WorkflowOwner != f.owner || rows[0].BlockNumber != 100 {
		t.Fatalf("row provenance = %+v", rows[0])
	}
}

func TestVerify_ValidMockSignature(t *testing.T) {
	f := newFixture(t)
	raw := f.signed(t, KindDepositFinality, f.depositReport(t, []DepositItem{f.confirmedItem(t)}))
	rows, err := f.verifier(ProviderMock).Verify(context.Background(), raw)
	if err != nil || len(rows) != 1 || rows[0].Status != StatusAttested || rows[0].Provider != ProviderMock {
		t.Fatalf("rows = %+v err %v", rows, err)
	}
}

func TestVerify_ForgedSignature(t *testing.T) {
	f := newFixture(t)
	other, _ := NewDevKey()
	report := f.depositReport(t, []DepositItem{f.confirmedItem(t)})
	meta := f.metadata(KindDepositFinality)
	sig, _ := other.Sign(meta, report)
	raw := RawAttestation{Kind: KindDepositFinality, Metadata: meta, Report: report, Evidence: Evidence{Signature: sig}}
	if _, err := f.verifier(ProviderMock).Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("err = %v, want ErrForged", err)
	}
	// Tampering with the report after signing is also a forgery.
	raw = f.signed(t, KindDepositFinality, report)
	raw.Report[len(raw.Report)-1] ^= 1
	if _, err := f.verifier(ProviderMock).Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("tampered report: err = %v, want ErrForged", err)
	}
	// Chainlink never accepts a signature in place of a log the gateway read itself.
	raw = f.signed(t, KindDepositFinality, f.depositReport(t, []DepositItem{f.confirmedItem(t)}))
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("signature accepted by chainlink verifier: %v", err)
	}
}

func TestVerify_Replayed(t *testing.T) {
	f := newFixture(t)
	report := f.depositReport(t, []DepositItem{f.confirmedItem(t)})
	f.seen[string(PayloadHash(report))] = true
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, report)); !errors.Is(err, ErrReplayed) {
		t.Fatalf("err = %v, want ErrReplayed", err)
	}
}

func TestVerify_WrongWorkflowOwnerGateway(t *testing.T) {
	f := newFixture(t)
	report := f.depositReport(t, []DepositItem{f.confirmedItem(t)})
	v := f.verifier(ProviderChainlink)

	raw := f.onChain(KindDepositFinality, report)
	raw.Metadata = f.metadata(KindSolvency)
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrWrongWorkflow) {
		t.Fatalf("other workflow id: err = %v", err)
	}
	raw = f.onChain(KindDepositFinality, report)
	m, _ := DecodeMetadata(raw.Metadata)
	m.WorkflowID = SubjectKey("updated-workflow")
	raw.Metadata = m.Encode()
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrWrongWorkflow) {
		t.Fatalf("unknown workflow id: err = %v", err)
	}
	raw = f.onChain(KindDepositFinality, report)
	m, _ = DecodeMetadata(raw.Metadata)
	m.Owner = [20]byte{9}
	raw.Metadata = m.Encode()
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrWrongOwner) {
		t.Fatalf("wrong owner: err = %v", err)
	}
	foreign, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: GatewayID("https://other.example"), ObservedAt: f.now, Items: []DepositItem{f.confirmedItem(t)}})
	if _, err := v.Verify(context.Background(), f.onChain(KindDepositFinality, foreign)); !errors.Is(err, ErrWrongGateway) {
		t.Fatalf("wrong gateway: err = %v", err)
	}
}

func TestVerify_WrongEmitterAndUnconfirmed(t *testing.T) {
	f := newFixture(t)
	report := f.depositReport(t, []DepositItem{f.confirmedItem(t)})
	v := f.verifier(ProviderChainlink)
	raw := f.onChain(KindDepositFinality, report)
	raw.Evidence.Emitter = [20]byte{7}
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrWrongEmitter) {
		t.Fatalf("wrong emitter: err = %v", err)
	}
	raw = f.onChain(KindDepositFinality, report)
	raw.Evidence.HeadBlock = raw.Evidence.BlockNumber + 2
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("unconfirmed: err = %v", err)
	}
	raw = f.onChain(KindDepositFinality, report)
	raw.Evidence.TxHash = nil
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("no tx: err = %v", err)
	}
}

func TestVerify_StaleAndFuture(t *testing.T) {
	f := newFixture(t)
	item := f.confirmedItem(t)
	old, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(-25 * time.Hour), Items: []DepositItem{item}})
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, old)); !errors.Is(err, ErrStale) {
		t.Fatalf("stale: err = %v", err)
	}
	future, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(time.Hour), Items: []DepositItem{item}})
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, future)); !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("future: err = %v", err)
	}
}

func TestVerify_UnknownAndMismatchedSubjectsAreStoredFailed(t *testing.T) {
	f := newFixture(t)
	good := f.confirmedItem(t)
	short := good
	short.Amount = big.NewInt(41_000_000)
	notFound := good
	notFound.Verdict = VerdictNotFound
	unknown := good
	unknown.DepositID = SubjectKey("never-asked")
	rows, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, f.depositReport(t, []DepositItem{good, short, notFound, unknown})))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[0].Status != StatusAttested {
		t.Fatalf("rows = %+v", rows)
	}
	for i, want := range []error{ErrSubjectMismatch, ErrSubjectMismatch, ErrUnknownSubject} {
		r := rows[i+1]
		if r.Status != StatusFailed || !containsErr(r.Reason, want) {
			t.Errorf("row %d = %s %q, want failed with %v", i+1, r.Status, r.Reason, want)
		}
	}
	if rows[3].SubjectID[:2] != "0x" {
		t.Errorf("unknown subject id = %q, want the raw key", rows[3].SubjectID)
	}
}

func TestVerify_SolvencyAndConversionSubjects(t *testing.T) {
	f := newFixture(t)
	cp := Checkpoint{ID: "cp-1", TakenAt: f.now.Add(-time.Minute), MaxJournalID: 9, Assets: []AssetTotal{{Asset: "USDC.SOLANA", Liabilities: big.NewInt(100), Decimals: 6}}}
	cp.Hash = CheckpointHash(cp)
	conv := Conversion{ConversionID: "cv-1", ExecutedAt: f.now, Base: "USDC", Quote: "USD", ExecutedRate: "0.9999", AmountMinor: big.NewInt(5)}
	_ = f.subjects.RememberSubjects(context.Background(), []Subject{CheckpointSubject(cp), ConversionSubject(conv, f.now)})

	sol, _ := EncodeReport(Report{Kind: KindSolvency, GatewayID: f.gateway, ObservedAt: f.now, Items: []SolvencyItem{
		{CheckpointHash: cp.Hash, Asset: LabelKey("USDC.SOLANA"), Liabilities: big.NewInt(100), Reserves: big.NewInt(150), Decimals: 6},
		{CheckpointHash: SubjectKey("not-published"), Asset: LabelKey("SOL"), Liabilities: big.NewInt(1), Reserves: big.NewInt(1), Decimals: 9},
	}})
	rows, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindSolvency, sol))
	if err != nil || len(rows) != 2 || rows[0].Status != StatusAttested || rows[0].SubjectID != "cp-1" || rows[1].Status != StatusFailed {
		t.Fatalf("solvency rows = %+v err %v", rows, err)
	}
	cv, _ := EncodeReport(Report{Kind: KindConversionReference, GatewayID: f.gateway, ObservedAt: f.now, Items: []ConversionItem{
		{ConversionID: SubjectKey("cv-1"), Pair: LabelKey("USDC/USD"), ReferenceRate: big.NewInt(1), ReferenceDecimals: 8, DeviationBps: big.NewInt(2), RoundID: big.NewInt(1)},
	}})
	rows, err = f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindConversionReference, cv))
	if err != nil || len(rows) != 1 || rows[0].Status != StatusAttested || rows[0].SubjectID != "cv-1" {
		t.Fatalf("conversion rows = %+v err %v", rows, err)
	}
}

func TestVerify_NoneProviderRefusesEverything(t *testing.T) {
	f := newFixture(t)
	raw := f.onChain(KindDepositFinality, f.depositReport(t, []DepositItem{f.confirmedItem(t)}))
	if _, err := f.verifier(ProviderNone).Verify(context.Background(), raw); !errors.Is(err, ErrDisabled) {
		t.Fatalf("err = %v", err)
	}
}

func TestVerify_EmptyReportIsInvalid(t *testing.T) {
	f := newFixture(t)
	empty, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now, Items: []DepositItem{}})
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, empty)); !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("err = %v", err)
	}
}

func containsErr(reason string, target error) bool {
	return len(reason) >= len(target.Error()) && reason[:len(target.Error())] == target.Error()
}
