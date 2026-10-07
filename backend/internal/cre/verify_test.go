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
	bindings map[Kind]Binding
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
	f.bindings = map[Kind]Binding{}
	for kind, id := range f.wfIDs {
		f.bindings[kind] = Binding{ID: id, Owner: f.owner, Name: KeystoneName(string(kind))}
	}
	return f
}

func (f *fixture) verifier(provider string) *Verifier {
	v := &Verifier{
		Provider: provider, GatewayID: f.gateway, Consumer: f.consumer, Bindings: f.bindings,
		Subjects: f.subjects, Chain: "test-chain",
		Now:  func() time.Time { return f.now },
		Seen: func(_ context.Context, h []byte) (bool, error) { return f.seen[string(h)], nil },
	}
	if provider == ProviderMock {
		v.MockSigner = f.key.Address()
	}
	return v
}

func (f *fixture) metadata(kind Kind) []byte {
	m := Metadata{WorkflowID: f.wfIDs[kind], Owner: f.owner, WorkflowName: KeystoneName(string(kind)), ReportID: [2]byte{0, 1}}
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
	return RawAttestation{Kind: kind, Metadata: f.metadata(kind), Report: report, Evidence: Evidence{Emitter: f.consumer, TxHash: []byte{0xab}, BlockNumber: 100, HeadBlock: 110, Final: true, ReportHash: [32]byte(PayloadHash(report))}}
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
	raw = f.onChain(KindDepositFinality, report)
	m, _ = DecodeMetadata(raw.Metadata)
	m.WorkflowName = KeystoneName("renamed-workflow")
	raw.Metadata = m.Encode()
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrWrongName) {
		t.Fatalf("wrong name: err = %v", err)
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
	raw.Evidence.Final = false
	err := func() error { _, err := v.Verify(context.Background(), raw); return err }()
	if !errors.Is(err, ErrUnconfirmed) || IsRejection(err) || !IsRetryable(err) {
		t.Fatalf("unconfirmed must be retryable, never a rejection: %v", err)
	}
	raw = f.onChain(KindDepositFinality, report)
	raw.Evidence.TxHash = nil
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("no tx: err = %v", err)
	}
	// A rebuilt report that does not hash to what the contract logged is not the report the DON signed.
	raw = f.onChain(KindDepositFinality, report)
	raw.Evidence.ReportHash[0] ^= 1
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrForged) {
		t.Fatalf("report hash mismatch: err = %v", err)
	}
}

func TestVerify_OldReportsRecordAndFutureOnesDoNot(t *testing.T) {
	f := newFixture(t)
	item := f.confirmedItem(t)
	// Age limits apply to freshness display, not to recording: a poller outage must not lose reports.
	old, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(-30 * 24 * time.Hour), Items: []DepositItem{item}})
	if rows, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, old)); err != nil || rows[0].Status != StatusAttested {
		t.Fatalf("old report refused: %v", err)
	}
	future, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(time.Hour), Items: []DepositItem{item}})
	if _, err := f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindDepositFinality, future)); !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("future: err = %v", err)
	}
}

// No ordering between distinct reports: two batches observed in the same second are both recorded (the contract rule).
func TestVerify_NoOrderingBetweenDistinctReports(t *testing.T) {
	f := newFixture(t)
	item := f.confirmedItem(t)
	v := f.verifier(ProviderChainlink)
	a, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(-time.Minute), Items: []DepositItem{item}})
	other := item
	other.SlotOrBlock = 99
	b, _ := EncodeReport(Report{Kind: KindDepositFinality, GatewayID: f.gateway, ObservedAt: f.now.Add(-time.Minute), Items: []DepositItem{other}})
	if _, err := v.Verify(context.Background(), f.onChain(KindDepositFinality, a)); err != nil {
		t.Fatal(err)
	}
	f.seen[string(PayloadHash(a))] = true
	if _, err := v.Verify(context.Background(), f.onChain(KindDepositFinality, b)); err != nil {
		t.Fatalf("second distinct report in the same second refused: %v", err)
	}
	if _, err := v.Verify(context.Background(), f.onChain(KindDepositFinality, a)); !errors.Is(err, ErrReplayed) {
		t.Fatalf("same bytes must be a replay: %v", err)
	}
}

func TestVerify_UnknownAndMismatchedSubjectsAreNeverAttested(t *testing.T) {
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
	for i, want := range []struct {
		status Status
		err    error
	}{{StatusMismatch, ErrSubjectMismatch}, {StatusMismatch, ErrSubjectMismatch}, {StatusFailed, ErrUnknownSubject}} {
		r := rows[i+1]
		if r.Status != want.status || !containsErr(r.Reason, want.err) {
			t.Errorf("row %d = %s %q, want %s with %v", i+1, r.Status, r.Reason, want.status, want.err)
		}
	}
	if rows[3].SubjectID[:2] != "0x" {
		t.Errorf("unknown subject id = %q, want the raw key", rows[3].SubjectID)
	}
}

func TestVerify_SolvencyAndConversionFactsMustMatchWhatWasServed(t *testing.T) {
	f := newFixture(t)
	cp := Checkpoint{ID: "cp-1", TakenAt: f.now.Add(-time.Minute), MaxJournalID: 9, Assets: []AssetTotal{{Asset: "USDC.SOLANA", Liabilities: big.NewInt(100), Decimals: 6}, {Asset: "SOL", Liabilities: big.NewInt(7), Decimals: 9}}}
	cp.Hash = CheckpointHash(cp)
	conv := Conversion{ConversionID: "cv-1", ExecutedAt: f.now, Base: "USDC", Quote: "USD", ExecutedRate: "0.9999", AmountMinor: big.NewInt(5)}
	_ = f.subjects.RememberSubjects(context.Background(), []Subject{CheckpointSubject(cp), ConversionSubject(conv, f.now)})

	sol, _ := EncodeReport(Report{Kind: KindSolvency, GatewayID: f.gateway, ObservedAt: f.now, Items: []SolvencyItem{
		{CheckpointHash: cp.Hash, Asset: LabelKey("USDC.SOLANA"), Liabilities: big.NewInt(100), Reserves: big.NewInt(150), Decimals: 6},
		{CheckpointHash: cp.Hash, Asset: LabelKey("SOL"), Liabilities: big.NewInt(7), Reserves: big.NewInt(1), Decimals: 9},
		{CheckpointHash: cp.Hash, Asset: LabelKey("USDC.SOLANA"), Liabilities: big.NewInt(101), Reserves: big.NewInt(150), Decimals: 6},
		{CheckpointHash: cp.Hash, Asset: LabelKey("USDC.SOLANA"), Liabilities: big.NewInt(100), Reserves: big.NewInt(150), Decimals: 2},
		{CheckpointHash: cp.Hash, Asset: LabelKey("BTC"), Liabilities: big.NewInt(1), Reserves: big.NewInt(1), Decimals: 8},
		{CheckpointHash: SubjectKey("not-published"), Asset: LabelKey("SOL"), Liabilities: big.NewInt(7), Reserves: big.NewInt(1), Decimals: 9},
	}})
	raw := f.onChain(KindSolvency, sol)
	raw.Ignored = [][32]byte{LabelKey("SOL")}
	rows, err := f.verifier(ProviderChainlink).Verify(context.Background(), raw)
	if err != nil || len(rows) != 6 {
		t.Fatalf("rows = %+v err %v", rows, err)
	}
	want := []Status{StatusAttested, StatusIgnored, StatusMismatch, StatusMismatch, StatusMismatch, StatusFailed}
	for i, r := range rows {
		if r.Status != want[i] {
			t.Errorf("solvency row %d = %s (%s), want %s", i, r.Status, r.Reason, want[i])
		}
	}
	if rows[0].SubjectID != "cp-1" || rows[2].Reason == "" || rows[4].Reason == "" {
		t.Fatalf("rows = %+v", rows)
	}

	cv, _ := EncodeReport(Report{Kind: KindConversionReference, GatewayID: f.gateway, ObservedAt: f.now, Items: []ConversionItem{
		{ConversionID: SubjectKey("cv-1"), Pair: LabelKey("USDC/USD"), ReferenceRate: big.NewInt(1), ReferenceDecimals: 8, DeviationBps: big.NewInt(2), RoundID: big.NewInt(1)},
		{ConversionID: SubjectKey("cv-1"), Pair: LabelKey("EUR/USD"), ReferenceRate: big.NewInt(1), ReferenceDecimals: 8, DeviationBps: big.NewInt(2), RoundID: big.NewInt(1)},
	}})
	rows, err = f.verifier(ProviderChainlink).Verify(context.Background(), f.onChain(KindConversionReference, cv))
	if err != nil || len(rows) != 2 || rows[0].Status != StatusAttested || rows[0].SubjectID != "cv-1" || rows[1].Status != StatusMismatch {
		t.Fatalf("conversion rows = %+v err %v", rows, err)
	}
}

// A report padded past the contract's exact length is malformed under every provider (the contract's _decodeHeader).
func TestVerify_PaddedReportIsMalformed(t *testing.T) {
	f := newFixture(t)
	report := f.depositReport(t, []DepositItem{f.confirmedItem(t)})
	padded := append(append([]byte{}, report...), make([]byte, 32)...)
	raw := f.signed(t, KindDepositFinality, padded)
	if _, err := f.verifier(ProviderMock).Verify(context.Background(), raw); !errors.Is(err, ErrInvalidReport) {
		t.Fatalf("padded report accepted: %v", err)
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

// The CRE simulator signs with a fixed identity; such records are simulated (never production) and refused in live.
func TestVerify_SimulatorIdentityIsSimulatedAndRefusedInLive(t *testing.T) {
	f := newFixture(t)
	item := f.confirmedItem(t)
	report := f.depositReport(t, []DepositItem{item})
	f.bindings[KindDepositFinality] = Binding{ID: SimulatorWorkflowID, Owner: SimulatorOwner, Name: KeystoneName("deposit_finality")}
	raw := f.onChain(KindDepositFinality, report)
	m := Metadata{WorkflowID: SimulatorWorkflowID, Owner: SimulatorOwner, WorkflowName: KeystoneName("deposit_finality"), ReportID: [2]byte{0, 1}}
	raw.Metadata = m.Encode()
	v := f.verifier(ProviderChainlink)
	rows, err := v.Verify(context.Background(), raw)
	if err != nil || len(rows) != 1 || !rows[0].Simulated || rows[0].Status != StatusAttested {
		t.Fatalf("simulator rows = %+v err %v", rows, err)
	}
	v.Live = true
	if _, err := v.Verify(context.Background(), raw); !errors.Is(err, ErrSimulated) || !IsRejection(err) {
		t.Fatalf("live: err = %v, want ErrSimulated", err)
	}
	// A simulation forwarder marks every record simulated, whatever identity signed it.
	f2 := newFixture(t)
	v2 := f2.verifier(ProviderChainlink)
	v2.SimulatedForwarder = true
	rows, err = v2.Verify(context.Background(), f2.onChain(KindDepositFinality, f2.depositReport(t, []DepositItem{f2.confirmedItem(t)})))
	if err != nil || !rows[0].Simulated {
		t.Fatalf("simulation forwarder rows = %+v err %v", rows, err)
	}
	v2.Live = true
	if _, err := v2.Verify(context.Background(), f2.onChain(KindDepositFinality, f2.depositReport(t, []DepositItem{f2.confirmedItem(t)}))); !errors.Is(err, ErrSimulated) {
		t.Fatalf("live simulation forwarder: %v", err)
	}
}
