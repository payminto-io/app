package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/shopspring/decimal"
)

// previewFees prices card and UPI at 2.9% in USD, bank at a flat 50 USD, and crypto in USDC at 1%.
type previewFees struct{}

func (previewFees) Resolve(_ context.Context, q fees.Query) (fees.Rule, error) {
	r := fees.Rule{ID: 9, Version: 3, Scope: fees.Scope{Method: q.Method, Currency: q.Currency}, MinorUnits: 2, FeeBearer: fees.BearerMerchant}
	switch {
	case q.Currency == "USD" && (q.Method == fees.MethodCard || q.Method == fees.MethodUPI):
		r.Percent = decimal.RequireFromString("2.9")
	case q.Currency == "USD" && q.Method == fees.MethodBank:
		r.Flat = decimal.RequireFromString("50")
	case q.Currency == "USDC" && q.Method == fees.MethodCrypto:
		r.MinorUnits, r.Percent = 6, decimal.RequireFromString("1")
	default:
		return fees.Rule{}, fees.ErrNoRule
	}
	return r, nil
}

func (f previewFees) Preview(ctx context.Context, req fees.PreviewRequest) (fees.Breakdown, error) {
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
	b := fees.Compute(r, req.Amount)
	if b.Fee.Add(b.Tax).GreaterThan(req.Amount) {
		return fees.Breakdown{}, fees.ErrFeeExceedsAmount
	}
	return b, nil
}

// catalogCreator offers card, UPI and bank in USD, USDC on SOL in USD, and card in EUR (which has no fee rule).
type catalogCreator struct{ linksCreator }

func (catalogCreator) Offerings(context.Context, links.Environment) ([]links.Offering, error) {
	return []links.Offering{
		{Currency: "usd", Method: links.MethodSpec{Method: fees.MethodCard}},
		{Currency: "USD", Method: links.MethodSpec{Method: fees.MethodUPI}},
		{Currency: "USD", Method: links.MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"}},
		{Currency: "EUR", Method: links.MethodSpec{Method: fees.MethodCard}},
	}, nil
}

func previewRouter(t *testing.T, creator links.PaymentCreator) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := links.NewMemStore()
	store.SetMerchantName(3, "Acme Coffee")
	svc := links.NewService(store, previewFees{}, creator, links.WithCheckoutBaseURL("https://checkout.test"))
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
	return r
}

func TestLinksPreviewCustomerAmountHasNoTotals(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	w, out := do(r, http.MethodPost, "/api/v2/links/preview", `{"title": "Tip jar", "amount_mode": "customer", "amount_min": "10", "amount_max": "100",
		"currency": "USD", "fee_bearer": "customer", "methods": [{"method": "card"}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("preview %d %s", w.Code, w.Body)
	}
	m := out["model"].(map[string]any)
	if m["amount"] != nil || m["merchant_name"] != "Acme Coffee" || m["short_code"] != "" {
		t.Fatalf("model %v", m)
	}
	method := m["methods"].([]any)[0].(map[string]any)
	if method["fee"] != nil || method["tax"] != nil || method["customer_total"] != nil {
		t.Fatalf("customer-entered amount carried totals: %v", method)
	}
}

func TestLinksPreviewDropsRefusedMethodsWithReasons(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	w, out := do(r, http.MethodPost, "/api/v2/links/preview", `{"title": "Beans", "amount": "25.00", "currency": "USD", "fee_bearer": "customer",
		"methods": [{"method": "card"}, {"method": "crypto", "chain": "SOL", "asset": "USDC"}, {"method": "bank"}, {"method": "upi"}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("preview %d %s", w.Code, w.Body)
	}
	m := out["model"].(map[string]any)
	methods := m["methods"].([]any)
	if len(methods) != 1 {
		t.Fatalf("methods %v", methods)
	}
	card := methods[0].(map[string]any)
	if card["method"] != "card" || card["fee"] != "0.73" || card["customer_total"] != "25.73" {
		t.Fatalf("card %v", card)
	}
	want := map[string]string{"crypto": "surcharge_needs_quote", "bank": "fee_exceeds_amount", "upi": "surcharge_forbidden"}
	dropped := out["dropped_methods"].([]any)
	if len(dropped) != len(want) {
		t.Fatalf("dropped %v", dropped)
	}
	for _, d := range dropped {
		d := d.(map[string]any)
		if want[d["method"].(string)] != d["code"] || d["message"] == "" {
			t.Fatalf("dropped entry %v", d)
		}
	}
}

func TestLinksPreviewLineItemsCarryServerTotals(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	w, out := do(r, http.MethodPost, "/api/v2/links/preview", `{"title": "Kit", "amount_mode": "line_items", "currency": "USD",
		"line_items": [{"name": "Mug", "quantity": 2, "unit_price": "5", "tax_rate": "10"}], "methods": [{"method": "card"}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("preview %d %s", w.Code, w.Body)
	}
	m := out["model"].(map[string]any)
	li := m["line_items"].([]any)[0].(map[string]any)
	if li["subtotal"] != "10" || li["tax"] != "1" || li["total"] != "11" || m["subtotal"] != "10" || m["tax_total"] != "1" || m["amount"] != "11" {
		t.Fatalf("line items %v %v", m, li)
	}
}

func TestLinksPreviewRefusesBadShapeWithFields(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	w, out := do(r, http.MethodPost, "/api/v2/links/preview", `{"currency": "ZZZ", "accent_color": "red"}`)
	if w.Code != http.StatusUnprocessableEntity || len(out["errors"].([]any)) != 2 {
		t.Fatalf("preview %d %v", w.Code, out)
	}
}

func TestLinksPreviewEqualsPublicRenderForSameLink(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	body := `{"title": "Beans", "description": "Whole", "amount": "25.00", "currency": "USD", "fee_bearer": "customer",
		"methods": [{"method": "card"}, {"method": "bank"}], "success_message": "Thanks", "customer_field_policy": {"name": {"mode": "hidden", "prefill": "Ada"}, "email": {"mode": "required"}, "phone": {"mode": "hidden"}}}`
	w, out := do(r, http.MethodPost, "/api/v2/links", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	id := out["id"].(string)
	if out["merchant_name"] != "Acme Coffee" {
		t.Fatalf("merchant_name %v", out["merchant_name"])
	}
	// Bank is refused at publish (fee above amount), so publish card only, then preview the same form over the live link.
	body = `{"title": "Beans", "description": "Whole", "amount": "25.00", "currency": "USD", "fee_bearer": "customer",
		"methods": [{"method": "card"}], "success_message": "Thanks", "customer_field_policy": {"name": {"mode": "hidden", "prefill": "Ada"}, "email": {"mode": "required"}, "phone": {"mode": "hidden"}}}`
	if w, _ = do(r, http.MethodPatch, "/api/v2/links/"+id, body); w.Code != http.StatusOK {
		t.Fatalf("patch %d %s", w.Code, w.Body)
	}
	w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/publish", "")
	if w.Code != http.StatusOK {
		t.Fatalf("publish %d %s", w.Code, w.Body)
	}
	code := out["short_code"].(string)
	pub := httpJSON(t, r, http.MethodGet, "/api/v2/public/links/"+code, "")
	prev := httpJSON(t, r, http.MethodPost, "/api/v2/links/preview?link_id="+id, body)
	model, _ := json.Marshal(prev["model"])
	public, _ := json.Marshal(pub)
	if string(model) != string(public) {
		t.Fatalf("preview differs from public render\npreview %s\npublic  %s", model, public)
	}
}

func TestLinksOptionsListsOnlyPublishableOfferings(t *testing.T) {
	r := previewRouter(t, &catalogCreator{})
	w, out := do(r, http.MethodGet, "/api/v2/links/options", "")
	if w.Code != http.StatusOK {
		t.Fatalf("options %d %s", w.Code, w.Body)
	}
	got, _ := json.Marshal(out)
	want := `{"currencies":["USD"],"environment":"test","methods":[` +
		`{"asset":null,"chain":null,"currencies":["USD"],"method":"card"},` +
		`{"asset":null,"chain":null,"currencies":["USD"],"method":"upi"},` +
		`{"asset":"USDC","chain":"SOL","currencies":["USD"],"method":"crypto"}]}`
	if string(got) != want {
		t.Fatalf("options\n got %s\nwant %s", got, want)
	}

	// A creator that cannot list what it takes offers nothing rather than guessing.
	w, out = do(previewRouter(t, &linksCreator{}), http.MethodGet, "/api/v2/links/options", "")
	if w.Code != http.StatusOK || len(out["methods"].([]any)) != 0 || len(out["currencies"].([]any)) != 0 {
		t.Fatalf("options without a catalog %d %v", w.Code, out)
	}
}

func TestLinksPreviewRefusesAnotherMerchantsLink(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	w, out := do(r, http.MethodPost, "/api/v2/links", `{"title": "Theirs", "amount": "5", "currency": "USD"}`, "X-Test-Platform", "4")
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	w, out = do(r, http.MethodPost, "/api/v2/links/preview?link_id="+out["id"].(string), `{"title": "Mine", "currency": "USD"}`)
	if w.Code != http.StatusNotFound || out["code"] != "link_not_found" {
		t.Fatalf("cross-merchant preview %d %v", w.Code, out)
	}
}

func TestLinksPreviewWritesNothing(t *testing.T) {
	r := previewRouter(t, &linksCreator{})
	body := `{"title": "Beans", "amount": "25.00", "currency": "USD", "methods": [{"method": "card"}]}`
	httpJSON(t, r, http.MethodPost, "/api/v2/links/preview", body)
	if list := httpJSON(t, r, http.MethodGet, "/api/v2/links", ""); list["total"] != float64(0) {
		t.Fatalf("preview created a link: %v", list)
	}

	w, created := do(r, http.MethodPost, "/api/v2/links", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	id := created["id"].(string)
	httpJSON(t, r, http.MethodPost, "/api/v2/links/preview?link_id="+id, `{"title": "Other", "amount": "99.00", "currency": "USD"}`)
	after := httpJSON(t, r, http.MethodGet, "/api/v2/links/"+id, "")
	if after["title"] != "Beans" || after["amount"] != created["amount"] || after["revision"] != created["revision"] || after["updated_at"] != created["updated_at"] {
		t.Fatalf("preview changed the stored link: before %v after %v", created, after)
	}
	if list := httpJSON(t, r, http.MethodGet, "/api/v2/links", ""); list["total"] != float64(1) {
		t.Fatalf("links after preview: %v", list)
	}
}

func httpJSON(t *testing.T, r *gin.Engine, method, path, body string) map[string]any {
	t.Helper()
	w, out := do(r, method, path, body)
	if w.Code != http.StatusOK {
		t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body)
	}
	return out
}
