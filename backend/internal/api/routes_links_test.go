package api

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/shopspring/decimal"
)

// linksFees prices card in USD at 2.9% and refuses UPI surcharges, like the default policy.
type linksFees struct{}

func (linksFees) Resolve(_ context.Context, q fees.Query) (fees.Rule, error) {
	if q.Currency != "USD" || (q.Method != fees.MethodCard && q.Method != fees.MethodUPI) {
		return fees.Rule{}, fees.ErrNoRule
	}
	return fees.Rule{ID: 4, Version: 2, Scope: fees.Scope{Method: q.Method, Currency: "USD"}, MinorUnits: 2,
		Percent: decimal.RequireFromString("2.9"), FeeBearer: fees.BearerMerchant}, nil
}

func (f linksFees) Preview(ctx context.Context, req fees.PreviewRequest) (fees.Breakdown, error) {
	r, err := f.Resolve(ctx, req.Query)
	if err != nil {
		return fees.Breakdown{}, err
	}
	if req.FeeBearer != nil {
		r.FeeBearer = *req.FeeBearer
	}
	if r.FeeBearer == fees.BearerCustomer && r.Method == fees.MethodUPI {
		return fees.Breakdown{}, fees.ErrSurchargeForbidden
	}
	return fees.Compute(r, req.Amount), nil
}

type linksCreator struct{ n int }

func (linksCreator) Connectors(context.Context, links.Environment, string, links.MethodSpec) ([]string, error) {
	return []string{"mock"}, nil
}

func (c *linksCreator) CreatePayment(_ context.Context, req links.PaymentRequest) (links.CreatedPayment, error) {
	c.n++
	return links.CreatedPayment{Reference: "pr_" + req.LinkPaymentID, CheckoutURL: "https://checkout.test/pay/pr_" + req.LinkPaymentID}, nil
}

func (c *linksCreator) FencePayment(context.Context, links.PaymentRequest) (links.CreatedPayment, bool, error) {
	return links.CreatedPayment{}, false, nil
}

func (c *linksCreator) CancelPayment(context.Context, string) error { return nil }

func (c *linksCreator) OpenPayments(_ context.Context, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func linksRouter(t *testing.T) (*gin.Engine, *links.MemStore, *linksCreator) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := links.NewMemStore()
	store.SetMerchantName(3, "Acme Coffee")
	creator := &linksCreator{}
	svc := links.NewService(store, linksFees{}, creator, links.WithCheckoutBaseURL("https://checkout.test"))
	r := gin.New()
	auth := func(c *gin.Context) {
		platform := uint(3)
		if c.GetHeader("X-Test-Platform") == "4" {
			platform = 4
		}
		c.Set("memberID", uint(7))
		c.Set("externalPlatformID", platform)
	}
	RegisterLinksRoutes(r.Group("/api/v2"), &modules.LinksModule{Port: svc}, LinksAuth{Merchant: auth})
	return r, store, creator
}

const linkBody = `{
	"title": "Coffee beans", "amount": "25.00", "currency": "usd", "methods": [{"method": "card"}],
	"success_message": "Thanks!", "reference_id": "order-9", "metadata": {"crm": "1", "plan": "gold"},
	"customer_field_policy": {"email": {"mode": "required"}},
	"questions": [{"key": "size", "label": "Size", "type": "select", "options": ["S", "M"], "required": true, "per_order": true}]
}`

func TestLinksMerchantAndPublicFlow(t *testing.T) {
	r, _, creator := linksRouter(t)

	w, out := do(r, http.MethodPost, "/api/v2/links", linkBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	id := out["id"].(string)
	if out["status"] != "draft" || out["currency"] != "USD" || out["short_code"] != nil || out["total"] != "25" ||
		out["customer_field_policy"].(map[string]any)["phone"].(map[string]any)["mode"] != "hidden" {
		t.Fatalf("create body %v", out)
	}

	w, out = do(r, http.MethodPatch, "/api/v2/links/"+id, `{"title": "Fresh beans", "metadata": {"crm": "2"}}`)
	if w.Code != http.StatusOK || out["title"] != "Fresh beans" || len(out["metadata"].(map[string]any)) != 1 || out["amount"] != "25" {
		t.Fatalf("patch %d %v", w.Code, out)
	}

	w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/publish", "")
	code, _ := out["short_code"].(string)
	if w.Code != http.StatusOK || out["status"] != "active" || !links.ValidShortCode(code) || out["url"] != "https://checkout.test/l/"+code {
		t.Fatalf("publish %d %v", w.Code, out)
	}

	w, out = do(r, http.MethodPatch, "/api/v2/links/"+id, `{"amount": "30.00"}`)
	if w.Code != http.StatusConflict || out["code"] != "link_published_immutable" || out["field"] != "amount" {
		t.Fatalf("live amount change %d %v", w.Code, out)
	}

	w, out = do(r, http.MethodGet, "/api/v2/public/links/"+code, "")
	if w.Code != http.StatusOK || out["available"] != true || out["merchant_name"] != "Acme Coffee" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("render %d %v", w.Code, out)
	}
	if _, leaked := out["metadata"]; leaked {
		t.Fatal("render model leaked metadata")
	}

	pay := `{"method": {"method": "card"}, "customer": {"email": "ada@example.test"}, "answers": {"size": "M"}}`
	w, out = do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay)
	if w.Code != http.StatusBadRequest || out["code"] != "idempotency_key_required" {
		t.Fatalf("pay without key %d %v", w.Code, out)
	}
	w, out = do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay, "Idempotency-Key", "abc")
	if w.Code != http.StatusCreated || out["amount"] != "25" || out["fee"] != "0.73" || out["customer_total"] != "25" ||
		out["replayed"] != false || out["method"].(map[string]any)["chain"] != nil {
		t.Fatalf("pay %d %v", w.Code, out)
	}
	ref := out["payment_reference"]
	w, out = do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay, "Idempotency-Key", "abc")
	if w.Code != http.StatusOK || out["replayed"] != true || out["payment_reference"] != ref || creator.n != 1 {
		t.Fatalf("replay %d %v", w.Code, out)
	}
	w, out = do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay, "Idempotency-Key", "def")
	if w.Code != http.StatusConflict || out["code"] != "link_use_limit_reached" {
		t.Fatalf("second payment on a single-use link %d %v", w.Code, out)
	}

	if w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/pause", ""); w.Code != http.StatusOK || out["status"] != "paused" {
		t.Fatalf("pause %d %v", w.Code, out)
	}
	if w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/pause", ""); w.Code != http.StatusConflict || out["code"] != "invalid_transition" {
		t.Fatalf("pause twice %d %v", w.Code, out)
	}
	w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/duplicate", "")
	if w.Code != http.StatusCreated || out["status"] != "draft" || out["id"] == id {
		t.Fatalf("duplicate %d %v", w.Code, out)
	}
	dupID := out["id"].(string)
	if w, _ = do(r, http.MethodPost, "/api/v2/links/"+id+"/archive", ""); w.Code != http.StatusOK {
		t.Fatalf("archive %d", w.Code)
	}
	if w, out = do(r, http.MethodGet, "/api/v2/public/links/"+code, ""); w.Code != http.StatusGone || out["code"] != "link_archived" {
		t.Fatalf("archived render %d %v", w.Code, out)
	}
	if w, out = do(r, http.MethodDelete, "/api/v2/links/"+id, ""); w.Code != http.StatusConflict || out["code"] != "link_not_deletable" {
		t.Fatalf("delete archived %d %v", w.Code, out)
	}
	if w, _ = do(r, http.MethodDelete, "/api/v2/links/"+dupID, ""); w.Code != http.StatusNoContent {
		t.Fatalf("delete draft %d", w.Code)
	}
	w, out = do(r, http.MethodGet, "/api/v2/links?status=archived", "")
	if w.Code != http.StatusOK || out["total"] != float64(1) {
		t.Fatalf("list %d %v", w.Code, out)
	}
}

func TestLinksValidationErrorsAreTypedAndComplete(t *testing.T) {
	r, _, _ := linksRouter(t)
	w, out := do(r, http.MethodPost, "/api/v2/links", `{"amount": "0", "currency": "ZZZ", "accent_color": "blue"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d %s", w.Code, w.Body)
	}
	items := out["errors"].([]any)
	var codes []string
	for _, it := range items {
		codes = append(codes, it.(map[string]any)["code"].(string))
	}
	if out["code"] != codes[0] || !slices.Contains(codes, "currency_unsupported") || !slices.Contains(codes, "accent_color_invalid") {
		t.Fatalf("errors %v", out)
	}

	_, out = do(r, http.MethodPost, "/api/v2/links", `{"title": "x", "currency": "USD", "amount": "5"}`)
	id := out["id"].(string)
	w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/publish", "")
	if w.Code != http.StatusUnprocessableEntity || out["code"] != "methods_required" || out["field"] != "methods" {
		t.Fatalf("publish incomplete %d %v", w.Code, out)
	}
}

func TestLinksRefuseMalformedBodies(t *testing.T) {
	r, _, _ := linksRouter(t)
	cases := []struct{ method, path, body string }{
		{http.MethodPost, "/api/v2/links", `{"title": "x", "surprise": true}`},
		{http.MethodPost, "/api/v2/links", `{"title": "x"} {"title": "y"}`},
		{http.MethodPost, "/api/v2/links", `[1]`},
		{http.MethodPost, "/api/v2/links", `{"title": "` + strings.Repeat("x", 70<<10) + `"}`},
	}
	for _, tc := range cases {
		if w, out := do(r, tc.method, tc.path, tc.body); w.Code != http.StatusBadRequest || out["code"] != "invalid_json" {
			t.Errorf("%.40s: %d %v", tc.body, w.Code, out)
		}
	}
	for _, body := range []string{`{"amount": 1e999999999}`, `{"amount_mode": "customer", "amount_max": "1e-999999999"}`,
		`{"amount_mode": "line_items", "line_items": [{"name": "a", "quantity": 1, "unit_price": 1e999999999}]}`} {
		if w, out := do(r, http.MethodPost, "/api/v2/links", body); w.Code != http.StatusUnprocessableEntity {
			t.Errorf("absurd decimal %s: %d %v", body, w.Code, out)
		}
	}
	_, out := do(r, http.MethodPost, "/api/v2/links", `{"title": "x"}`)
	if w, out := do(r, http.MethodPatch, "/api/v2/links/"+out["id"].(string), `{"total": "1"}`); w.Code != http.StatusBadRequest {
		t.Errorf("patch of a read-only field %d %v", w.Code, out)
	}
	if w, out := do(r, http.MethodGet, "/api/v2/links/nope", ""); w.Code != http.StatusNotFound || out["code"] != "link_not_found" {
		t.Errorf("unknown id %d %v", w.Code, out)
	}
	if w, out := do(r, http.MethodGet, "/api/v2/public/links/bad!code", ""); w.Code != http.StatusNotFound {
		t.Errorf("bad short code %d %v", w.Code, out)
	}
}

func TestLinksRenderModelJSONContract(t *testing.T) {
	r, _, _ := linksRouter(t)
	_, out := do(r, http.MethodPost, "/api/v2/links", linkBody)
	_, out = do(r, http.MethodPost, "/api/v2/links/"+out["id"].(string)+"/publish", "")
	w, _ := do(r, http.MethodGet, "/api/v2/public/links/"+out["short_code"].(string), "")
	var m map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	want := []string{
		"amount", "amount_max", "amount_min", "amount_mode", "available", "billing_required", "branding",
		"chain_tolerance_bps", "currency", "customer_fields", "description", "expires_at", "failure_message",
		"failure_retry", "fee_bearer", "line_items", "merchant_name", "methods", "questions", "quote_expiry_seconds",
		"receipt_email", "shipping_required", "short_code", "subtotal", "success_message", "success_mode", "tax_total",
		"title", "unavailable_reason", "url",
	}
	if !slices.Equal(keys, want) {
		t.Fatalf("render keys\n got %v\nwant %v", keys, want)
	}
}

func TestLinksStatusMapping(t *testing.T) {
	cases := map[links.Code]int{
		links.CodeNotFound: 404, links.CodeArchived: 410, links.CodeExpired: 410, links.CodePaused: 409,
		links.CodeUseLimitReached: 409, links.CodeInvalidTransition: 409, links.CodePublishedImmutable: 409,
		links.CodeIdempotencyKeyReused: 409, links.CodePaymentInProgress: 409, links.CodeIdempotencyKeyRequired: 400,
		links.CodePaymentCreationFailed: 502, links.CodeShortCodeExhausted: 503, links.CodeMethodNoFeeRule: 422,
		links.CodeSurchargeForbidden: 422, links.CodeAnswerRequired: 422,
	}
	for code, want := range cases {
		if got := linksStatus(code); got != want {
			t.Errorf("%s: %d, want %d", code, got, want)
		}
	}
}

func TestNewRouterGuardsMerchantLinkRoutesButNotTheCheckout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newRBACFixture(t)
	store := links.NewMemStore()
	svc := links.NewService(store, linksFees{}, &linksCreator{})
	r := NewRouter(RouterConfig{AuthSvc: f.auth, MEPRoleSvc: f.mep, Links: &modules.LinksModule{Port: svc},
		Environment: &modules.EnvironmentModule{Environment: environment.Test}})
	for _, rt := range []struct{ method, path string }{
		{http.MethodPost, "/api/v2/links"}, {http.MethodGet, "/api/v2/links"}, {http.MethodGet, "/api/v2/links/x"},
		{http.MethodPatch, "/api/v2/links/x"}, {http.MethodDelete, "/api/v2/links/x"},
		{http.MethodPost, "/api/v2/links/x/publish"}, {http.MethodPost, "/api/v2/links/x/pause"},
		{http.MethodPost, "/api/v2/links/x/archive"}, {http.MethodPost, "/api/v2/links/x/duplicate"},
	} {
		if w, _ := do(r, rt.method, rt.path, `{}`); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without credentials: %d", rt.method, rt.path, w.Code)
		}
	}
	w, out := do(r, http.MethodPost, "/api/v2/links", `{"title": "x"}`, "Authorization", f.bearer(t, f.merchant, f.operator))
	if w.Code != http.StatusCreated {
		t.Fatalf("merchant session create %d %v", w.Code, out)
	}
	if w, _ := do(r, http.MethodGet, "/api/v2/public/links/AAAAAAAAAAAA", ""); w.Code != http.StatusNotFound {
		t.Fatalf("public render needs no credentials, got %d", w.Code)
	}
}
