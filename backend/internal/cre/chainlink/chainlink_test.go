package chainlink

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
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

// fakeChain is a LogReader fed by tests; it plays the consumer contract's event stream.
type fakeChain struct {
	mu   sync.Mutex
	head uint64
	logs []Log
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
			if i >= len(l.Topics) || l.Topics[i] != want[0] {
				match = false
			}
		}
		if match {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f *fakeChain) emit(t *testing.T, consumer common.Address, gatewayID [32]byte, kind cre.Kind, metadata, report []byte) {
	t.Helper()
	data, err := EncodeReportAttestedData(metadata, report)
	if err != nil {
		t.Fatal(err)
	}
	var kindTopic common.Hash
	kindTopic[31] = kind.Code()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.head++
	f.logs = append(f.logs, Log{
		Address: consumer, TxHash: crypto.Keccak256Hash(report), BlockNumber: f.head, Index: 0,
		Topics: []common.Hash{ReportAttestedTopic, common.BytesToHash(gatewayID[:]), kindTopic}, Data: data,
	})
	f.head += 5
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
	h := &harness{chain: &fakeChain{head: 100}, consumer: common.HexToAddress("0x1111111111111111111111111111111111111111"), gateway: cre.GatewayID("https://pay.example.test")}
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
	h.provider = New(Config{GatewayURL: h.server.URL, WorkflowIDs: h.wfIDs, KeyRef: "keyring://cre-trigger", Consumer: h.consumer, GatewayID: h.gateway, ChunkBlocks: 7}, h.signer, h.chain)
	return h
}

func (h *harness) settle(t *testing.T, kind cre.Kind) {
	t.Helper()
	meta := cre.Metadata{WorkflowID: h.wfIDs[kind], Owner: h.owner, ReportID: [2]byte{0, 1}}
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
	h.chain.emit(t, h.consumer, h.gateway, kind, meta.Encode(), payload)
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
	h.chain.emit(t, common.HexToAddress("0x9999999999999999999999999999999999999999"), h.gateway, cre.KindSolvency, meta.Encode(), payload)
	h.chain.emit(t, h.consumer, cre.GatewayID("https://other.example"), cre.KindSolvency, meta.Encode(), payload)
	h.settle(t, cre.KindDepositFinality)
	raws, next, err := h.provider.Poll(ctx, cre.KindSolvency, cursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(raws) != 1 || raws[0].Kind != cre.KindSolvency || raws[0].Evidence.Emitter != [20]byte(h.consumer) || raws[0].Evidence.HeadBlock != h.chain.head {
		t.Fatalf("raws = %+v", raws)
	}
	if next.Block != h.chain.head+1 {
		t.Fatalf("cursor = %+v, head %d", next, h.chain.head)
	}
	if again, _, _ := h.provider.Poll(ctx, cre.KindSolvency, next); len(again) != 0 {
		t.Fatal("replayed past the cursor")
	}
}
