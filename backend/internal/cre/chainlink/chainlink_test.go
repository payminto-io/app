package chainlink

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/cre/conformance"
)

// memorySigner plays the signer service in tests; the provider only ever sees the key reference.
type memorySigner struct {
	keys map[string]*ecdsa.PrivateKey
}

func newMemorySigner(t *testing.T, refs ...string) *memorySigner {
	t.Helper()
	s := &memorySigner{keys: map[string]*ecdsa.PrivateKey{}}
	for _, ref := range refs {
		k, err := crypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		s.keys[ref] = k
	}
	return s
}

func (s *memorySigner) Address(_ context.Context, ref string) (common.Address, error) {
	k, ok := s.keys[ref]
	if !ok {
		return common.Address{}, ErrSignerUnavailable
	}
	return crypto.PubkeyToAddress(k.PublicKey), nil
}

func (s *memorySigner) SignMessage(_ context.Context, ref string, msg []byte) ([]byte, error) {
	k, ok := s.keys[ref]
	if !ok {
		return nil, ErrSignerUnavailable
	}
	sig, err := crypto.Sign(EIP191Hash(msg), k)
	if err != nil {
		return nil, err
	}
	sig[64] += 27
	return sig, nil
}

// fakeChain is a LogReader fed by tests; it plays the consumer contract's event stream and the forwarder calldata.
type fakeChain struct {
	mu     sync.Mutex
	head   uint64
	logs   []Log
	inputs map[common.Hash][]byte
	// txErr makes TransactionInput fail (an RPC outage mid-poll).
	txErr error
}

func (f *fakeChain) TransactionInput(_ context.Context, tx common.Hash) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.txErr != nil {
		return nil, f.txErr
	}
	in, ok := f.inputs[tx]
	if !ok {
		return nil, errors.New("not found")
	}
	return in, nil
}

func (f *fakeChain) FinalizedHead(context.Context) (uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.head, nil
}

func (f *fakeChain) FilterLogs(_ context.Context, address common.Address, from, to uint64, topics [][]common.Hash) ([]Log, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []Log
	for _, l := range f.logs {
		if l.Address != address || l.BlockNumber < from || l.BlockNumber > to {
			continue
		}
		match := true
		for i, want := range topics {
			if len(want) == 0 {
				continue
			}
			if i >= len(l.Topics) {
				match = false
				continue
			}
			any := false
			for _, w := range want {
				if l.Topics[i] == w {
					any = true
				}
			}
			if !any {
				match = false
			}
		}
		if match {
			out = append(out, l)
		}
	}
	return out, nil
}

// emit records one forwarder delivery: the logs the contract emits and the calldata the forwarder received.
func (f *fakeChain) emit(t *testing.T, consumer common.Address, meta cre.Metadata, report []byte, ignoredItems ...int) common.Hash {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.head++
	tx := crypto.Keccak256Hash(report, []byte{byte(f.head)})
	logs, err := LogsForReport(consumer, meta, report, tx, f.head, 0, ignoredItems...)
	if err != nil {
		t.Fatal(err)
	}
	f.logs = append(f.logs, logs...)
	input, err := EncodeForwarderCall(consumer, meta.Encode(), report, make([]byte, 96), nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.inputs == nil {
		f.inputs = map[common.Hash][]byte{}
	}
	f.inputs[tx] = input
	f.head += 5
	return tx
}

type harness struct {
	provider *Provider
	chain    *fakeChain
	server   *httptest.Server
	signer   *memorySigner
	consumer common.Address
	gateway  [32]byte
	owner    [20]byte
	wfIDs    map[cre.Kind][32]byte
	seen     []string
	mu       sync.Mutex
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{chain: &fakeChain{head: 100, inputs: map[common.Hash][]byte{}}, consumer: common.HexToAddress("0x1111111111111111111111111111111111111111"), gateway: cre.GatewayID("https://pay.example.test")}
	copy(h.owner[:], common.HexToAddress("0x3333333333333333333333333333333333333333").Bytes())
	h.wfIDs = map[cre.Kind][32]byte{cre.KindSolvency: cre.SubjectKey("wf-s"), cre.KindDepositFinality: cre.SubjectKey("wf-d"), cre.KindConversionReference: cre.SubjectKey("wf-c")}
	h.signer = newMemorySigner(t, "keyring://cre-trigger")
	h.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, err := VerifyTriggerJWT(token, body, time.Now()); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		var req struct {
			Method string `json:"method"`
			Params struct {
				Workflow struct {
					WorkflowID string `json:"workflowID"`
				} `json:"workflow"`
			} `json:"params"`
		}
		if json.Unmarshal(body, &req) != nil || req.Method != "workflows.execute" || len(req.Params.Workflow.WorkflowID) != 64 {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		h.mu.Lock()
		h.seen = append(h.seen, req.Params.Workflow.WorkflowID)
		h.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ACCEPTED", "workflow_execution_id": "exec-" + req.Params.Workflow.WorkflowID[:8]})
	}))
	t.Cleanup(h.server.Close)
	h.provider = New(Config{GatewayURL: h.server.URL, WorkflowIDs: h.wfIDs, KeyRef: "keyring://cre-trigger", Consumer: h.consumer, GatewayID: h.gateway, ChunkBlocks: 7, StartBlock: 1}, h.signer, h.chain)
	return h
}

func (h *harness) settle(t *testing.T, kind cre.Kind) {
	t.Helper()
	meta := cre.Metadata{WorkflowID: h.wfIDs[kind], Owner: h.owner, WorkflowName: cre.KeystoneName(string(kind)), ReportID: [2]byte{0, 1}}
	var report cre.Report
	switch kind {
	case cre.KindSolvency:
		report = cre.Report{Kind: kind, GatewayID: h.gateway, ObservedAt: time.Now(), Items: []cre.SolvencyItem{{CheckpointHash: cre.SubjectKey("cp"), Asset: cre.LabelKey("USDC"), Liabilities: big.NewInt(1), Reserves: big.NewInt(2), Decimals: 6}}}
	case cre.KindDepositFinality:
		report = cre.Report{Kind: kind, GatewayID: h.gateway, ObservedAt: time.Now(), Items: []cre.DepositItem{{DepositID: cre.SubjectKey("d"), Amount: big.NewInt(1), Verdict: 1}}}
	default:
		report = cre.Report{Kind: kind, GatewayID: h.gateway, ObservedAt: time.Now(), Items: []cre.ConversionItem{{ConversionID: cre.SubjectKey("c"), ReferenceRate: big.NewInt(1), DeviationBps: big.NewInt(0), RoundID: big.NewInt(1)}}}
	}
	payload, err := cre.EncodeReport(report)
	if err != nil {
		t.Fatal(err)
	}
	h.chain.emit(t, h.consumer, meta, payload)
}

func TestConformance(t *testing.T) {
	h := newHarness(t)
	conformance.Run(t, conformance.Harness{Attester: h.provider, Input: conformance.Inputs(h.gateway), Settle: func(kind cre.Kind, _ string) { h.settle(t, kind) }})
}

func TestTriggerJWT(t *testing.T) {
	signer := newMemorySigner(t, "keyring://cre-trigger")
	now := time.Now()
	req, err := BuildTriggerRequest(context.Background(), signer, "keyring://cre-trigger", cre.SubjectKey("wf"), json.RawMessage(`{"b":1,"a":{"z":true,"y":[1,2]}}`), now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	addr, err := VerifyTriggerJWT(req.JWT, req.Body, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if want, _ := signer.Address(context.Background(), "keyring://cre-trigger"); addr != want {
		t.Fatalf("iss = %s, want %s", addr, want)
	}
	if !strings.Contains(string(req.Body), `"params":{"input":{"a":{"y":[1,2],"z":true},"b":1},"workflow"`) {
		t.Fatalf("body is not key-sorted: %s", req.Body)
	}
	if _, err := VerifyTriggerJWT(req.JWT, append(req.Body, ' '), now); err == nil {
		t.Fatal("digest of a different body verified")
	}
	if _, err := VerifyTriggerJWT(req.JWT, req.Body, now.Add(6*time.Minute)); err == nil {
		t.Fatal("expired token verified")
	}
	parts := strings.Split(req.JWT, ".")
	if _, err := VerifyTriggerJWT(parts[0]+"."+parts[1]+"."+parts[2][:len(parts[2])-2]+"AA", req.Body, now); err == nil {
		t.Fatal("tampered signature verified")
	}
	// The API process never holds a key: an unavailable signer yields no request at all.
	if _, err := BuildTriggerRequest(context.Background(), UnavailableSigner{}, "keyring://cre-trigger", cre.SubjectKey("wf"), nil, now, time.Minute); err == nil {
		t.Fatal("request built without a signer")
	}
}

func TestTriggerRefusals(t *testing.T) {
	h := newHarness(t)
	if _, err := h.provider.Trigger(context.Background(), cre.Kind("bogus"), []byte("{}")); err == nil {
		t.Fatal("unknown kind triggered")
	}
	unsigned := New(Config{GatewayURL: h.server.URL, WorkflowIDs: h.wfIDs, KeyRef: "keyring://missing", Consumer: h.consumer, GatewayID: h.gateway}, h.signer, h.chain)
	if _, err := unsigned.Trigger(context.Background(), cre.KindSolvency, []byte("{}")); err == nil {
		t.Fatal("trigger without a resolvable key reference succeeded")
	}
	if got := unsigned.Health(context.Background()); got.Status != cre.HealthDegraded {
		t.Fatalf("health with missing signer = %+v", got)
	}
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"status":"REJECTED"}`))
	}))
	defer refusing.Close()
	limited := New(Config{GatewayURL: refusing.URL, WorkflowIDs: h.wfIDs, KeyRef: "keyring://cre-trigger", Consumer: h.consumer, GatewayID: h.gateway}, h.signer, h.chain)
	if _, err := limited.Trigger(context.Background(), cre.KindSolvency, []byte("{}")); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("rejected trigger reported as success: %v", err)
	}
}

func TestPollReadsOnlyTheConsumerAndThisGateway(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	_, cursor, _ := h.provider.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	h.settle(t, cre.KindSolvency)
	// A log from another contract, and one for another gateway, must not come back.
	meta := cre.Metadata{WorkflowID: h.wfIDs[cre.KindSolvency], Owner: h.owner}
	payload, _ := cre.EncodeReport(cre.Report{Kind: cre.KindSolvency, GatewayID: h.gateway, ObservedAt: time.Now(), Items: []cre.SolvencyItem{{Liabilities: big.NewInt(1), Reserves: big.NewInt(1)}}})
	h.chain.emit(t, common.HexToAddress("0x9999999999999999999999999999999999999999"), meta, payload)
	foreign, _ := cre.EncodeReport(cre.Report{Kind: cre.KindSolvency, GatewayID: cre.GatewayID("https://other.example"), ObservedAt: time.Now(), Items: []cre.SolvencyItem{{Liabilities: big.NewInt(1), Reserves: big.NewInt(1)}}})
	h.chain.emit(t, h.consumer, meta, foreign)
	h.settle(t, cre.KindDepositFinality)
	raws, next, err := h.provider.Poll(ctx, cre.KindSolvency, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(raws) != 1 || raws[0].Kind != cre.KindSolvency || raws[0].Evidence.Emitter != [20]byte(h.consumer) || raws[0].Evidence.HeadBlock != h.chain.head {
		t.Fatalf("raws = %+v", raws)
	}
	// The report bytes come from the forwarder calldata and hash to the logged reportHash.
	if raws[0].Evidence.ReportHash != [32]byte(cre.PayloadHash(raws[0].Report)) || !raws[0].Evidence.Final {
		t.Fatal("calldata report does not hash to the contract's reportHash, or is not marked final")
	}
	rebuilt, err := cre.DecodeReport(raws[0].Report)
	if err != nil || rebuilt.Items.([]cre.SolvencyItem)[0].Reserves.Int64() != 2 {
		t.Fatalf("rebuilt = %+v err %v", rebuilt, err)
	}
	back, _ := cre.DecodeMetadata(raws[0].Metadata)
	if back.WorkflowID != h.wfIDs[cre.KindSolvency] || back.Owner != h.owner {
		t.Fatalf("metadata = %+v", back)
	}
	if next.Block != h.chain.head+1 {
		t.Fatalf("cursor = %+v, head %d", next, h.chain.head)
	}
	if again, _, _ := h.provider.Poll(ctx, cre.KindSolvency, next); len(again) != 0 {
		t.Fatal("replayed past the cursor")
	}
}

func solvencyReport(h *harness, observed time.Time, assets ...string) ([]byte, cre.Metadata) {
	items := make([]cre.SolvencyItem, 0, len(assets))
	for i, a := range assets {
		items = append(items, cre.SolvencyItem{CheckpointHash: cre.SubjectKey("cp"), Asset: cre.LabelKey(a), Liabilities: big.NewInt(int64(i + 1)), Reserves: big.NewInt(2), Decimals: 6})
	}
	report, _ := cre.EncodeReport(cre.Report{Kind: cre.KindSolvency, GatewayID: h.gateway, ObservedAt: observed, Items: items})
	return report, cre.Metadata{WorkflowID: h.wfIDs[cre.KindSolvency], Owner: h.owner, WorkflowName: cre.KeystoneName("solvency"), ReportID: [2]byte{0, 1}}
}

// Every per-item solvency event comes back in item order (SolvencyAttested stored, SolvencyIgnored not), including a
// duplicate asset whose first occurrence the contract stored.
func TestPollCarriesIgnoredSolvencyItems(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	_, cursor, _ := h.provider.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	report, meta := solvencyReport(h, time.Now(), "USDC", "SOL", "USDC")
	h.chain.emit(t, h.consumer, meta, report, 2)
	raws, _, err := h.provider.Poll(ctx, cre.KindSolvency, cursor)
	if err != nil || len(raws) != 1 {
		t.Fatalf("raws = %+v err %v", raws, err)
	}
	want := []cre.ItemOutcome{{Key: cre.LabelKey("USDC"), Stored: true}, {Key: cre.LabelKey("SOL"), Stored: true}, {Key: cre.LabelKey("USDC"), Stored: false}}
	if len(raws[0].Outcomes) != len(want) || string(raws[0].Report) != string(report) {
		t.Fatalf("outcomes = %+v", raws[0].Outcomes)
	}
	for i := range want {
		if raws[0].Outcomes[i] != want[i] {
			t.Fatalf("outcome %d = %+v, want %+v", i, raws[0].Outcomes[i], want[i])
		}
	}
	if raws[0].Evidence.ReportHash != [32]byte(cre.PayloadHash(report)) {
		t.Fatal("report hash")
	}
}

// A reorged log (Removed) is never used; two reports in one block come back in log order.
func TestPollSkipsRemovedLogsAndOrdersWithinABlock(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	_, cursor, _ := h.provider.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	first, meta := solvencyReport(h, time.Now().Add(-2*time.Second), "USDC")
	second, _ := solvencyReport(h, time.Now().Add(-time.Second), "SOL")
	removed, _ := solvencyReport(h, time.Now(), "BTC")
	h.chain.emit(t, h.consumer, meta, first)
	// Put the second report in the same block as the first, at later log indexes.
	h.chain.mu.Lock()
	block := h.chain.logs[len(h.chain.logs)-1].BlockNumber
	tx2 := crypto.Keccak256Hash(second)
	logs2, _ := LogsForReport(h.consumer, meta, second, tx2, block, 10)
	h.chain.logs = append(h.chain.logs, logs2...)
	h.chain.inputs[tx2], _ = EncodeForwarderCall(h.consumer, meta.Encode(), second, make([]byte, 96), nil)
	tx3 := crypto.Keccak256Hash(removed)
	logs3, _ := LogsForReport(h.consumer, meta, removed, tx3, block, 20)
	for i := range logs3 {
		logs3[i].Removed = true
	}
	h.chain.logs = append(h.chain.logs, logs3...)
	h.chain.inputs[tx3], _ = EncodeForwarderCall(h.consumer, meta.Encode(), removed, make([]byte, 96), nil)
	h.chain.mu.Unlock()
	raws, _, err := h.provider.Poll(ctx, cre.KindSolvency, cursor)
	if err != nil || len(raws) != 2 {
		t.Fatalf("raws = %d err %v", len(raws), err)
	}
	if string(raws[0].Report) != string(first) || string(raws[1].Report) != string(second) || raws[0].Evidence.LogIndex >= raws[1].Evidence.LogIndex {
		t.Fatal("order within the block")
	}
}

// Calldata that is not a forwarder delivery to our consumer yields no report hash, which the verifier refuses as forged;
// an RPC failure reading the transaction is an error so the cursor stays put.
func TestPollCalldataGuards(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	_, cursor, _ := h.provider.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	report, meta := solvencyReport(h, time.Now(), "USDC")
	tx := h.chain.emit(t, h.consumer, meta, report)
	h.chain.mu.Lock()
	h.chain.inputs[tx], _ = EncodeForwarderCall(common.HexToAddress("0x9999999999999999999999999999999999999999"), meta.Encode(), report, make([]byte, 96), nil)
	h.chain.mu.Unlock()
	raws, _, err := h.provider.Poll(ctx, cre.KindSolvency, cursor)
	if err != nil || len(raws) != 1 || raws[0].Evidence.ReportHash != ([32]byte{}) {
		t.Fatalf("other receiver: raws = %+v err %v", raws, err)
	}
	h.chain.mu.Lock()
	h.chain.txErr = errors.New("Post \"https://rpc.example/v2/8f3a1c9d2e7b4a6f5c8d9e0f1a2b3c4d\": timeout")
	h.chain.mu.Unlock()
	if _, next, err := h.provider.Poll(ctx, cre.KindSolvency, cursor); err == nil || next != cursor || strings.Contains(err.Error(), "8f3a1c9d") {
		t.Fatalf("tx read failure: next=%+v err=%v", next, err)
	}
}

// Forwarder calldata round trip: the receiver slice is rawReport[109:], exactly what the contract hashes.
func TestForwarderCalldataRoundTrip(t *testing.T) {
	h := newHarness(t)
	report, meta := solvencyReport(h, time.Now(), "USDC")
	input, err := EncodeForwarderCall(h.consumer, meta.Encode(), report, make([]byte, 96), [][]byte{make([]byte, 65)})
	if err != nil {
		t.Fatal(err)
	}
	receiver, metadata, got, err := DecodeForwarderCall(input)
	if err != nil || receiver != h.consumer || string(metadata) != string(meta.Encode()) || string(got) != string(report) {
		t.Fatalf("round trip: %v", err)
	}
	if _, _, _, err := DecodeForwarderCall(input[:40]); err == nil {
		t.Fatal("short calldata decoded")
	}
	if _, _, _, err := DecodeForwarderCall(append([]byte{1, 2, 3, 4}, input[4:]...)); err == nil {
		t.Fatal("other selector decoded")
	}
}

// A fresh cursor reads from the configured start block, so reports delivered before enabling are not skipped.
func TestPollFreshCursorStartsAtTheStartBlock(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	report, meta := solvencyReport(h, time.Now(), "USDC")
	h.chain.emit(t, h.consumer, meta, report) // delivered before the module polls for the first time
	raws, next, err := h.provider.Poll(ctx, cre.KindSolvency, cre.Cursor{})
	if err != nil || len(raws) != 1 || next.Block != h.chain.head+1 {
		t.Fatalf("fresh cursor: raws %d next %+v err %v", len(raws), next, err)
	}
	unconfigured := New(Config{WorkflowIDs: h.wfIDs, Consumer: h.consumer, GatewayID: h.gateway}, nil, h.chain)
	if _, _, err := unconfigured.Poll(ctx, cre.KindSolvency, cre.Cursor{}); err == nil {
		t.Fatal("fresh cursor without CRE_START_BLOCK polled")
	}
}
