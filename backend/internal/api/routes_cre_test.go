package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/modules"
)

func routeTable(r *gin.Engine) []string {
	var out []string
	for _, ri := range r.Routes() {
		out = append(out, ri.Method+" "+ri.Path+" "+ri.Handler)
	}
	sort.Strings(out)
	return out
}

func creTestConfig(enabled bool, provider string, publicVerify bool) *config.Config {
	return &config.Config{
		Server:  config.ServerConfig{Environment: config.EnvironmentDevelopment, Port: 8090},
		Gateway: config.GatewayConfig{Environment: "test"},
		CRE: config.CREConfig{
			Enabled: enabled, Provider: provider, Chain: "ethereum-testnet-sepolia-base-1", SolvencyInterval: time.Hour, FinalityBatchInterval: time.Minute,
			PollInterval: 30 * time.Second, PublicVerifyEnabled: publicVerify,
			ReadTokenSolvency: "tok-solvency", ReadTokenDepositFinality: "tok-deposit", ReadTokenConversionReference: "tok-conversion",
			WorkflowNameSolvency: "solvency", WorkflowNameDepositFinality: "deposit-finality", WorkflowNameConversionReference: "conversion-reference",
		},
	}
}

type testDeposits struct{ rows []cre.PendingDeposit }

func (d testDeposits) PendingDeposits(context.Context, int) ([]cre.PendingDeposit, error) {
	return d.rows, nil
}
func (d testDeposits) Deposit(context.Context, string) (cre.PendingDeposit, bool, error) {
	return cre.PendingDeposit{}, false, nil
}

type testReserves struct{}

func (testReserves) Reserves(context.Context) ([]cre.Reserve, error) {
	return []cre.Reserve{{Asset: "USDC.SOLANA", Amount: big.NewInt(120), Decimals: 6}}, nil
}

type testLiabilities struct{}

func (testLiabilities) LiabilityTotals(context.Context) ([]cre.LedgerTotal, uint64, error) {
	return []cre.LedgerTotal{{Asset: "USDC.SOLANA", Total: "100"}}, 1, nil
}

func creModule(t *testing.T, cfg *config.Config) *modules.CREModule {
	t.Helper()
	m, err := modules.WireCRE(modules.Deps{Config: cfg}, modules.CREOptions{
		Store: cre.NewMemoryStore(), Liabilities: testLiabilities{}, Reserves: testReserves{},
		Deposits: testDeposits{rows: []cre.PendingDeposit{{DepositID: "dep-1", Chain: "solana", Tx: "sig1", LogIndexOrSig: "sig1", Token: "USDC", ExpectedAmountMinor: big.NewInt(42), Destination: "Dest1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// The owner's non-negotiable: with CRE off, the route table is byte-for-byte what it is without the module.
func TestRouterUnchangedWhenCREIsOff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := &modules.EnvironmentModule{Environment: environment.Test}
	without := routeTable(NewRouter(RouterConfig{Host: "http://localhost", Environment: env}))
	off := creModule(t, creTestConfig(false, "mock", true))
	withOff := routeTable(NewRouter(RouterConfig{Host: "http://localhost", Environment: env, CRE: off}))
	if strings.Join(without, "\n") != strings.Join(withOff, "\n") {
		t.Fatalf("route table changed with CRE off:\n%s\n---\n%s", strings.Join(without, "\n"), strings.Join(withOff, "\n"))
	}
	if off.Service.Worker() != nil {
		t.Fatal("a disabled module produced a worker")
	}
	on := creModule(t, creTestConfig(true, "mock", true))
	withOn := routeTable(NewRouter(RouterConfig{Host: "http://localhost", Environment: env, CRE: on}))
	if len(withOn) <= len(without) {
		t.Fatal("an enabled module mounted no routes")
	}
	for _, want := range []string{"GET /api/v1/cre/liabilities", "GET /api/v1/cre/pending-deposits", "GET /api/v1/cre/conversions", "POST /api/v1/cre/reports/:kind", "GET /api/v1/public/attestations/:id"} {
		found := false
		for _, have := range withOn {
			if strings.HasPrefix(have, want+" ") {
				found = true
			}
		}
		if !found {
			t.Errorf("route %s missing", want)
		}
	}
	for _, have := range withOn {
		if strings.Contains(have, "/cre/status") {
			t.Error("dashboard routes mounted without an auth service")
		}
	}
}

func creRouter(t *testing.T, m *modules.CREModule) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	session := func(c *gin.Context) { c.Set("memberID", uint(5)); c.Set("externalPlatformID", uint(1)) }
	RegisterCRERoutes(r.Group("/api/v1"), m, CREAuth{Session: session, Owner: pass, RateLimit: func(c *gin.Context) { c.Header("X-Test-Limited", "1"); c.Next() }})
	return r
}

func TestCRERoutes_WorkflowCredentialsAreScopedPerWorkflow(t *testing.T) {
	m := creModule(t, creTestConfig(true, "mock", false))
	r := creRouter(t, m)
	cases := []struct {
		path, token string
		want        int
	}{
		{"/api/v1/cre/liabilities", "tok-solvency", 200},
		{"/api/v1/cre/liabilities", "tok-deposit", 401},
		{"/api/v1/cre/liabilities", "", 401},
		{"/api/v1/cre/pending-deposits", "tok-deposit", 200},
		{"/api/v1/cre/pending-deposits", "tok-solvency", 401},
		{"/api/v1/cre/conversions", "tok-conversion", 200},
		{"/api/v1/cre/conversions?since=yesterday", "tok-conversion", 400},
	}
	for _, tc := range cases {
		w, _ := do(r, http.MethodGet, tc.path, "", "Authorization", "Bearer "+tc.token)
		if w.Code != tc.want {
			t.Errorf("%s with %q = %d, want %d: %s", tc.path, tc.token, w.Code, tc.want, w.Body)
		}
	}
	w, body := do(r, http.MethodGet, "/api/v1/cre/liabilities", "", "Authorization", "Bearer tok-solvency")
	assets, _ := body["assets"].([]any)
	if w.Code != 200 || len(assets) != 1 || body["checkpoint_hash"] == nil || body["max_journal_id"] == nil {
		t.Fatalf("liabilities = %v", body)
	}
	if w.Header().Get("X-Test-Limited") != "1" {
		t.Fatal("rate limiter not applied to a cre route")
	}
	_, body = do(r, http.MethodGet, "/api/v1/cre/pending-deposits", "", "Authorization", "Bearer tok-deposit")
	deposits, _ := body["deposits"].([]any)
	if len(deposits) != 1 || deposits[0].(map[string]any)["expected_amount_minor"] != "42" {
		t.Fatalf("pending deposits = %v", body)
	}
	// Public verify on: the liabilities read is open but serves only what was published, and never publishes.
	open := creRouter(t, creModule(t, creTestConfig(true, "mock", true)))
	if w, _ := do(open, http.MethodGet, "/api/v1/cre/liabilities", ""); w.Code != 503 {
		t.Fatalf("public liabilities before any publish = %d, want 503", w.Code)
	}
	if w, _ := do(open, http.MethodGet, "/api/v1/cre/liabilities", "", "Authorization", "Bearer tok-solvency"); w.Code != 200 {
		t.Fatalf("authenticated publish = %d", w.Code)
	}
	w, pub := do(open, http.MethodGet, "/api/v1/cre/liabilities", "")
	if w.Code != 200 || pub["checkpoint_hash"] == nil || pub["max_journal_id"] != nil {
		t.Fatalf("public liabilities after publish = %d %v", w.Code, pub)
	}
	if w, _ := do(open, http.MethodGet, "/api/v1/public/attestations/x", ""); w.Header().Get("X-Test-Limited") != "1" {
		t.Fatal("rate limiter not applied to the public route")
	}
	if w, _ := do(open, http.MethodGet, "/api/v1/cre/pending-deposits", ""); w.Code != 401 {
		t.Fatalf("pending deposits without credential = %d", w.Code)
	}
}

func signedReport(t *testing.T, m *modules.CREModule, kind cre.Kind, items any) string {
	t.Helper()
	ctx := context.Background()
	input := []byte("{}")
	switch kind {
	case cre.KindDepositFinality:
		rows, _ := m.Service.PendingDeposits(ctx, 12)
		input, _ = json.Marshal(map[string]any{"deposits": cre.DepositsJSON(rows)})
	case cre.KindSolvency:
		cp, _ := m.Service.Liabilities(ctx, true)
		input, _ = json.Marshal(cre.CheckpointJSON(cp, true))
	}
	if _, err := m.Mock.Trigger(ctx, kind, input); err != nil {
		t.Fatal(err)
	}
	raws, _, _ := m.Mock.Poll(ctx, kind, cre.Cursor{})
	raw := raws[len(raws)-1]
	body, _ := json.Marshal(map[string]any{
		"metadata": "0x" + common.Bytes2Hex(raw.Metadata), "report": "0x" + common.Bytes2Hex(raw.Report),
		"signature": "0x" + common.Bytes2Hex(raw.Evidence.Signature), "execution_id": raw.ExecutionID, "simulated": true,
	})
	return string(body)
}

func TestCRERoutes_SubmitVerifiesReplaysAndForgeries(t *testing.T) {
	m := creModule(t, creTestConfig(true, "mock", true))
	r := creRouter(t, m)
	body := signedReport(t, m, cre.KindDepositFinality, nil)

	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", body, "Authorization", "Bearer tok-solvency"); w.Code != 401 {
		t.Fatalf("other workflow's credential accepted: %d", w.Code)
	}
	// The credential is checked from the path before the body is read: a bad kind or no token never parses.
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/bogus", "{"); w.Code != 400 {
		t.Fatalf("bogus kind = %d", w.Code)
	}
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", strings.Repeat("x", 100<<10)); w.Code != 401 {
		t.Fatalf("unauthenticated oversized body = %d, want 401 before parsing", w.Code)
	}
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", strings.Repeat("x", 100<<10), "Authorization", "Bearer tok-deposit"); w.Code != 400 {
		t.Fatalf("oversized body = %d", w.Code)
	}
	w, resp := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", body, "Authorization", "Bearer tok-deposit")
	if w.Code != 202 || resp["recorded"] != float64(1) || resp["attested"] != float64(1) {
		t.Fatalf("submit = %d %v", w.Code, resp)
	}
	ids := resp["attestation_ids"].([]any)
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", body, "Authorization", "Bearer tok-deposit"); w.Code != 409 {
		t.Fatalf("replay = %d", w.Code)
	}
	forged := strings.Replace(body, `"signature":"0x`, `"signature":"0x00`, 1)
	forged = forged[:len(forged)-4] + forged[len(forged)-2:]
	if w, resp := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", forged, "Authorization", "Bearer tok-deposit"); w.Code != 422 && w.Code != 400 {
		t.Fatalf("forged = %d %v", w.Code, resp)
	}
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/reports/deposit_finality", `{"extra":1}`, "Authorization", "Bearer tok-deposit"); w.Code != 400 {
		t.Fatalf("unknown field = %d", w.Code)
	}

	w, got := do(r, http.MethodGet, "/api/v1/cre/attestations/"+ids[0].(string), "")
	if w.Code != 200 || got["status"] != "attested" || got["provider"] != "mock" || got["simulated"] != true || got["subject_id"] != "dep-1" {
		t.Fatalf("get = %d %v", w.Code, got)
	}
	w, pub := do(r, http.MethodGet, "/api/v1/public/attestations/"+ids[0].(string), "")
	if w.Code != 200 || pub["independently_signed"] != false || pub["simulated"] != true || pub["gateway_id"] == nil {
		t.Fatalf("public = %d %v", w.Code, pub)
	}
	if w, _ := do(r, http.MethodGet, "/api/v1/public/attestations/nope", ""); w.Code != 404 {
		t.Fatalf("unknown public id = %d", w.Code)
	}
	w, list := do(r, http.MethodGet, "/api/v1/cre/attestations?kind=deposit-finality", "")
	if w.Code != 200 || len(list["attestations"].([]any)) != 1 {
		t.Fatalf("list = %d %v", w.Code, list)
	}
}

func TestCRERoutes_StatusAndRun(t *testing.T) {
	m := creModule(t, creTestConfig(true, "mock", true))
	r := creRouter(t, m)
	w, st := do(r, http.MethodGet, "/api/v1/cre/status", "")
	if w.Code != 200 || st["enabled"] != true || st["provider"] != "mock" || st["environment"] != "test" {
		t.Fatalf("status = %d %v", w.Code, st)
	}
	workflows := st["workflows"].([]any)
	if len(workflows) != 3 || workflows[0].(map[string]any)["state"] != "never" || workflows[0].(map[string]any)["last_attestation"] != nil {
		t.Fatalf("workflows = %v", workflows)
	}
	if w, run := do(r, http.MethodPost, "/api/v1/cre/runs/solvency", ""); w.Code != 202 || run["status"] != "accepted" {
		t.Fatalf("run = %d %v", w.Code, run)
	}
	if w, _ := do(r, http.MethodPost, "/api/v1/cre/runs/bogus", ""); w.Code != 400 {
		t.Fatalf("bogus run = %d", w.Code)
	}
	m.Mock.ScriptFailure(cre.KindSolvency, fmt.Errorf("gateway rate limited"))
	if w, run := do(r, http.MethodPost, "/api/v1/cre/runs/solvency", ""); w.Code != 502 || run["status"] != "failed" {
		t.Fatalf("failed run = %d %v", w.Code, run)
	}
	m.Mock.ScriptFailure(cre.KindSolvency, nil)
	if _, err := m.Service.Poll(context.Background(), cre.KindSolvency); err != nil {
		t.Fatal(err)
	}
	_, st = do(r, http.MethodGet, "/api/v1/cre/status", "")
	sol := st["workflows"].([]any)[0].(map[string]any)
	if sol["state"] != "fresh" || sol["last_attestation"] == nil || sol["last_verified"] == nil || sol["last_run"].(map[string]any)["status"] != "failed" || sol["workflow_name"] != "58c66935b7" {
		t.Fatalf("solvency status = %v", sol)
	}
	h := st["health"].(map[string]any)
	if h["status"] != "ok" || st["trigger_signer_address"] == "" {
		t.Fatalf("health = %v", st)
	}
}
