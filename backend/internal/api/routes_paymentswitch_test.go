package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type switchTestServer struct {
	t      *testing.T
	engine *gin.Engine
	mock   *mock.Connector
}

func newSwitchTestServer(t *testing.T) *switchTestServer {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := ledger.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := paymentswitch.Migrate(db); err != nil {
		t.Fatal(err)
	}
	m, err := modules.WirePaymentSwitch(modules.Deps{
		DB:           db,
		Config:       &config.Config{Switch: config.SwitchConfig{Connectors: []string{"mock", "chaindeposit"}, MockWebhookSecret: "s3cret"}},
		Ledger:       ledger.New(db),
		Environment:  config.EnvironmentTest,
		ChainDeposit: chaindeposit.NewMemoryBackend(),
	})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := m.Connectors.Get(mock.Code)
	engine := gin.New()
	auth := func(c *gin.Context) {
		if c.GetHeader("X-API-Key") != "good" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}
		c.Set("memberID", uint(7))
		c.Set("externalPlatformID", uint(3))
		c.Next()
	}
	RegisterPaymentSwitchRoutes(engine.Group("/api/v2"), m, auth)
	return &switchTestServer{t: t, engine: engine, mock: conn.(*mock.Connector)}
}

func signed(h http.Header) map[string]string {
	return map[string]string{mock.SignatureHeader: h.Get(mock.SignatureHeader), mock.TimestampHeader: h.Get(mock.TimestampHeader)}
}

func (s *switchTestServer) do(method, path string, body any, headers map[string]string) (int, map[string]any) {
	s.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if raw, ok := body.([]byte); ok {
			buf.Write(raw)
		} else {
			_ = json.NewEncoder(&buf).Encode(body)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "good")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func payment(out map[string]any) map[string]any {
	p, _ := out["payment"].(map[string]any)
	return p
}

func TestSwitchRoutes_CreateConfirmGetRefund(t *testing.T) {
	s := newSwitchTestServer(t)
	code, out := s.do(http.MethodPost, "/api/v2/payments", map[string]any{
		"amount": "100", "asset": "USD", "capture_method": "manual",
		"payment_method": map[string]any{"type": "card", "token": "success"},
		"metadata":       map[string]string{"order": "42"},
	}, map[string]string{"Idempotency-Key": "order-42"})
	if code != http.StatusCreated {
		t.Fatalf("create = %d %v", code, out)
	}
	p := payment(out)
	id, _ := p["id"].(string)
	if id == "" || p["status"] != "requires_confirmation" || p["amount"] != "100" || p["idempotency_key"] != "order-42" || p["amount_captured"] != "0" {
		t.Fatalf("payment = %v", p)
	}
	if _, hasCamel := p["captureMethod"]; hasCamel || p["capture_method"] != "manual" {
		t.Fatalf("JSON must be snake_case: %v", p)
	}

	code, out = s.do(http.MethodPost, "/api/v2/payments", map[string]any{
		"amount": "100", "asset": "USD", "capture_method": "manual",
		"payment_method": map[string]any{"type": "card", "token": "success"},
		"metadata":       map[string]string{"order": "42"},
	}, map[string]string{"Idempotency-Key": "order-42"})
	if code != http.StatusCreated || payment(out)["id"] != id {
		t.Fatalf("replay = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments", map[string]any{"amount": "101", "asset": "USD"}, map[string]string{"Idempotency-Key": "order-42"})
	if code != http.StatusConflict || out["error"].(map[string]any)["code"] != "idempotency_conflict" {
		t.Fatalf("conflict = %d %v", code, out)
	}

	code, out = s.do(http.MethodPost, "/api/v2/payments/"+id+"/confirm", nil, nil)
	if code != http.StatusOK || payment(out)["status"] != "requires_capture" {
		t.Fatalf("confirm = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments/"+id+"/confirm", nil, nil)
	if code != http.StatusConflict || out["error"].(map[string]any)["code"] != "invalid_state" {
		t.Fatalf("second confirm = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments/"+id+"/capture", map[string]any{"amount": "100"}, nil)
	if code != http.StatusOK || payment(out)["status"] != "succeeded" {
		t.Fatalf("capture = %d %v", code, out)
	}
	code, out = s.do(http.MethodGet, "/api/v2/payments/"+id, nil, nil)
	if code != http.StatusOK {
		t.Fatalf("get = %d %v", code, out)
	}
	attempts, _ := payment(out)["attempts"].([]any)
	if len(attempts) != 1 || attempts[0].(map[string]any)["status"] != "charged" || attempts[0].(map[string]any)["connector_status"] != "captured" {
		t.Fatalf("attempts = %v", attempts)
	}
	if _, has := attempts[0].(map[string]any)["amount_received"]; has {
		t.Fatalf("a card attempt must not claim a received amount: %v", attempts[0])
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments/"+id+"/refunds", map[string]any{"amount": "25", "reason": "goodwill"}, map[string]string{"Idempotency-Key": "ref-1"})
	if code != http.StatusCreated {
		t.Fatalf("refund = %d %v", code, out)
	}
	r := out["refund"].(map[string]any)
	if r["status"] != "succeeded" || r["amount"] != "25" || r["payment_id"] != id {
		t.Fatalf("refund body = %v", r)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments/"+id+"/refunds", map[string]any{"amount": "100"}, nil)
	if code != http.StatusUnprocessableEntity || out["error"].(map[string]any)["code"] != "amount_exceeds" {
		t.Fatalf("over-refund = %d %v", code, out)
	}
	code, out = s.do(http.MethodGet, "/api/v2/payments/"+id, nil, nil)
	if payment(out)["amount_refunded"] != "25" || len(payment(out)["refunds"].([]any)) != 1 {
		t.Fatalf("get after refund = %d %v", code, out)
	}
}

func TestSwitchRoutes_WebhookAndErrors(t *testing.T) {
	s := newSwitchTestServer(t)
	code, out := s.do(http.MethodPost, "/api/v2/payments", map[string]any{
		"amount": "50", "asset": "USD", "confirm": true,
		"payment_method": map[string]any{"type": "card", "token": "requires_action"},
	}, nil)
	if code != http.StatusCreated || payment(out)["status"] != "requires_action" {
		t.Fatalf("create = %d %v", code, out)
	}
	id := payment(out)["id"].(string)
	next := payment(out)["next_action"].(map[string]any)
	if next["type"] != "redirect" || next["redirect_url"] == "" {
		t.Fatalf("next_action = %v", next)
	}
	code, out = s.do(http.MethodGet, "/api/v2/payments/"+id, nil, nil)
	txID := payment(out)["attempts"].([]any)[0].(map[string]any)["connector_transaction_id"].(string)

	headers, body := s.mock.SignWebhook(mock.Event{EventID: "evt_1", TransactionID: txID, Status: "captured"})
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/mock", body, signed(headers))
	if code != http.StatusOK || out["payment_id"] != id || out["ignored"] != false {
		t.Fatalf("webhook = %d %v", code, out)
	}
	// A replay is 200 with the ignored marker so the provider stops retrying (I3).
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/mock", body, signed(headers))
	if code != http.StatusOK || out["ignored"] != true || out["reason"] != "replay" {
		t.Fatalf("replay = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/mock", body, map[string]string{mock.SignatureHeader: "bad", mock.TimestampHeader: headers.Get(mock.TimestampHeader)})
	if code != http.StatusUnauthorized || out["error"].(map[string]any)["code"] != "webhook_signature" {
		t.Fatalf("bad signature = %d %v", code, out)
	}
	staleH, staleB := s.mock.SignWebhookAt(mock.Event{EventID: "evt_old", TransactionID: txID, Status: "captured"}, time.Now().Add(-time.Hour))
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/mock", staleB, signed(staleH))
	if code != http.StatusUnauthorized || out["error"].(map[string]any)["code"] != "webhook_stale" {
		t.Fatalf("stale = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/stripe", body, nil)
	if code != http.StatusNotFound || out["error"].(map[string]any)["code"] != "unknown_connector" {
		t.Fatalf("unknown connector = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/webhooks/chaindeposit", body, nil)
	if code != http.StatusUnprocessableEntity || out["error"].(map[string]any)["code"] != "unsupported" {
		t.Fatalf("connector without webhooks = %d %v", code, out)
	}
	code, out = s.do(http.MethodGet, "/api/v2/payments/"+id, nil, nil)
	if payment(out)["status"] != "succeeded" {
		t.Fatalf("after webhook = %d %v", code, out)
	}

	code, out = s.do(http.MethodGet, "/api/v2/payments/pi_missing", nil, nil)
	if code != http.StatusNotFound || out["error"].(map[string]any)["code"] != "not_found" {
		t.Fatalf("missing = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments", []byte("not-json"), nil)
	if code != http.StatusBadRequest {
		t.Fatalf("bad json = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments", map[string]any{"amount": "1.5e", "asset": "USD"}, nil)
	if code != http.StatusBadRequest || out["error"].(map[string]any)["code"] != "invalid_request" {
		t.Fatalf("bad amount = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments", map[string]any{"amount": "10", "asset": "USD", "confirm": true, "payment_method": map[string]any{"type": "bank", "token": "success"}}, nil)
	if code != http.StatusCreated || payment(out)["status"] != "succeeded" {
		t.Fatalf("bank via mock = %d %v", code, out)
	}
	code, out = s.do(http.MethodPost, "/api/v2/payments", map[string]any{"amount": "10", "asset": "USD", "confirm": true, "payment_method": map[string]any{"type": "chain", "token": "usdc", "details": map[string]string{"chain": "ETH", "asset": "USDC"}}}, nil)
	if code != http.StatusCreated || payment(out)["status"] != "requires_action" || payment(out)["connector_code"] != "chaindeposit" {
		t.Fatalf("chain deposit = %d %v", code, out)
	}
	if na := payment(out)["next_action"].(map[string]any); na["type"] != "pay_to_address" || na["address"] == "" || na["asset"] != "USDC" {
		t.Fatalf("chain next_action = %v", na)
	}
}

// M4: a create whose confirm fails before any claim still returns the payment id.
func TestSwitchRoutes_CreateConfirmFailureCarriesThePaymentID(t *testing.T) {
	s := newSwitchTestServer(t)
	code, out := s.do(http.MethodPost, "/api/v2/payments", map[string]any{
		"amount": "10", "asset": "USD", "confirm": true,
		"payment_method": map[string]any{"type": "carrier_pigeon", "token": "x"},
	}, nil)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("code = %d %v", code, out)
	}
	e := out["error"].(map[string]any)
	if e["code"] != "no_connector" || e["payment_id"] == "" || e["payment_id"] == nil {
		t.Fatalf("error = %v", e)
	}
}

func TestSwitchRoutes_RequireAuth(t *testing.T) {
	s := newSwitchTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/payments", bytes.NewBufferString(`{"amount":"1","asset":"USD"}`))
	w := httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no key = %d", w.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v2/payments/pi_x", nil)
	w = httptest.NewRecorder()
	s.engine.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no key on get = %d", w.Code)
	}
}
