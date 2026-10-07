package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/modules"
)

func publishedLink(t *testing.T, r *gin.Engine, body string) (id, code string) {
	t.Helper()
	w, out := do(r, http.MethodPost, "/api/v2/links", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("create %d %v", w.Code, out)
	}
	id = out["id"].(string)
	w, out = do(r, http.MethodPost, "/api/v2/links/"+id+"/publish", "")
	if w.Code != http.StatusOK {
		t.Fatalf("publish %d %v", w.Code, out)
	}
	return id, out["short_code"].(string)
}

func TestLinksCannotBeReachedFromAnotherPlatform(t *testing.T) {
	r, _, _ := linksRouter(t)
	id, _ := publishedLink(t, r, linkBody)
	other := []string{"X-Test-Platform", "4"}
	for _, rt := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/v2/links/" + id, ""},
		{http.MethodPatch, "/api/v2/links/" + id, `{"title": "mine now"}`},
		{http.MethodDelete, "/api/v2/links/" + id, ""},
		{http.MethodPost, "/api/v2/links/" + id + "/publish", ""},
		{http.MethodPost, "/api/v2/links/" + id + "/pause", ""},
		{http.MethodPost, "/api/v2/links/" + id + "/archive", ""},
		{http.MethodPost, "/api/v2/links/" + id + "/duplicate", ""},
	} {
		if w, out := do(r, rt.method, rt.path, rt.body, other...); w.Code != http.StatusNotFound || out["code"] != "link_not_found" {
			t.Errorf("%s %s from another platform: %d %v", rt.method, rt.path, w.Code, out)
		}
	}
	if w, out := do(r, http.MethodGet, "/api/v2/links", "", other...); out["total"] != float64(0) {
		t.Errorf("other platform listed %d %v", w.Code, out)
	}
	if w, out := do(r, http.MethodGet, "/api/v2/links/"+id, ""); w.Code != http.StatusOK || out["status"] != "active" || out["title"] != "Coffee beans" {
		t.Fatalf("owner's link changed: %d %v", w.Code, out)
	}
}

func TestLinksPatchIsGuardedByRevision(t *testing.T) {
	r, _, _ := linksRouter(t)
	_, out := do(r, http.MethodPost, "/api/v2/links", `{"title": "x"}`)
	id := out["id"].(string)
	if w, out := do(r, http.MethodPatch, "/api/v2/links/"+id, `{"title": "y"}`, "If-Match", "1"); w.Code != http.StatusOK || out["revision"] != float64(2) {
		t.Fatalf("patch at the current revision %d %v", w.Code, out)
	}
	if w, out := do(r, http.MethodPatch, "/api/v2/links/"+id, `{"title": "z"}`, "If-Match", `"1"`); w.Code != http.StatusConflict || out["code"] != "link_conflict" {
		t.Fatalf("patch at a stale revision %d %v", w.Code, out)
	}
	if w, _ := do(r, http.MethodPatch, "/api/v2/links/"+id, `{"title": "z"}`, "If-Match", "abc"); w.Code != http.StatusBadRequest {
		t.Fatalf("bad If-Match %d", w.Code)
	}
}

func TestLinksDetailCarriesTheFeePreview(t *testing.T) {
	r, _, _ := linksRouter(t)
	w, out := do(r, http.MethodPost, "/api/v2/links", linkBody)
	preview := out["fee_preview"].([]any)
	card := preview[0].(map[string]any)
	if w.Code != http.StatusCreated || card["rule_id"] != float64(4) || card["rule_version"] != float64(2) || card["fee"] != "0.73" || card["merchant_net"] != "24.27" || card["connector"] != "mock" {
		t.Fatalf("fee preview %d %v", w.Code, preview)
	}
	_, list := do(r, http.MethodGet, "/api/v2/links", "")
	if list["links"].([]any)[0].(map[string]any)["fee_preview"] != nil {
		t.Fatal("list rows priced every method")
	}
}

func TestLinksQRIsAnSVGOfTheShortLink(t *testing.T) {
	r, _, _ := linksRouter(t)
	_, code := publishedLink(t, r, linkBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v2/public/links/"+code+"/qr.svg", nil))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "image/svg+xml" || !strings.HasPrefix(w.Body.String(), "<svg") {
		t.Fatalf("qr %d %s %.60s", w.Code, w.Header().Get("Content-Type"), w.Body)
	}
	if w, _ := do(r, http.MethodGet, "/api/v2/public/links/AAAAAAAAAAAA/qr.svg", ""); w.Code != http.StatusNotFound {
		t.Fatalf("qr of an unknown link %d", w.Code)
	}
}

type ambiguousCreator struct{ linksCreator }

func (ambiguousCreator) CreatePayment(context.Context, links.PaymentRequest) (links.CreatedPayment, error) {
	return links.CreatedPayment{}, context.DeadlineExceeded
}

func TestLinksPayInProgressCarriesRetryAfterAndCapsAre429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := links.NewMemStore()
	svc := links.NewService(store, linksFees{}, &ambiguousCreator{}, links.WithLimits(links.ReserveLimits{MaxOpenPerClient: 1}))
	r := gin.New()
	auth := func(c *gin.Context) { c.Set("memberID", uint(7)); c.Set("externalPlatformID", uint(3)) }
	RegisterLinksRoutes(r.Group("/api/v2"), &modules.LinksModule{Port: svc}, LinksAuth{Merchant: auth})
	_, code := publishedLink(t, r, strings.Replace(linkBody, `"title"`, `"multi_use": true, "title"`, 1))
	pay := `{"method": {"method": "card"}, "customer": {"email": "ada@example.test"}, "answers": {"size": "M"}}`
	w, out := do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay, "Idempotency-Key", "a")
	if w.Code != http.StatusConflict || out["code"] != "payment_in_progress" || w.Header().Get("Retry-After") == "" {
		t.Fatalf("ambiguous creation %d %v %q", w.Code, out, w.Header().Get("Retry-After"))
	}
	w, out = do(r, http.MethodPost, "/api/v2/public/links/"+code+"/pay", pay, "Idempotency-Key", "b")
	if w.Code != http.StatusTooManyRequests || out["code"] != "open_payments_limit" {
		t.Fatalf("second open payment from one client %d %v", w.Code, out)
	}
}

func TestRouterIgnoresSpoofedForwardedForAndLimitsWithoutRedis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := links.NewService(links.NewMemStore(), linksFees{}, &linksCreator{})
	r := NewRouter(RouterConfig{Links: &modules.LinksModule{Port: svc}, Environment: &modules.EnvironmentModule{Environment: environment.Test}})
	limited := 0
	for i := range linksPublicPayPerMinute + 5 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v2/public/links/AAAAAAAAAAAA/pay", strings.NewReader(`{}`))
		req.RemoteAddr = "203.0.113.9:4000"
		req.Header.Set("X-Forwarded-For", "10.0.0."+string(rune('0'+i%10)))
		r.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 5 {
		t.Fatalf("%d of %d requests limited; X-Forwarded-For must not mint new clients and no Redis must not mean no limit", limited, linksPublicPayPerMinute+5)
	}
}

func TestRouterTrustsConfiguredProxyAndWarnsAboutOthers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := links.NewService(links.NewMemStore(), linksFees{}, &linksCreator{})
	var warned []string
	r := NewRouter(RouterConfig{
		Links: &modules.LinksModule{Port: svc}, Environment: &modules.EnvironmentModule{Environment: environment.Test},
		TrustedProxies: []string{"172.29.86.10"}, forwardingWarning: func(peer string) { warned = append(warned, peer) },
	})
	send := func(remote, xff string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v2/public/links/AAAAAAAAAAAA/pay", strings.NewReader(`{}`))
		req.RemoteAddr = remote
		req.Header.Set("X-Forwarded-For", xff)
		r.ServeHTTP(w, req)
		return w.Code
	}
	// Through the trusted proxy, each forwarded client has its own budget.
	for i := range linksPublicPayPerMinute + 5 {
		if code := send("172.29.86.10:5000", "203.0.113."+strconv.Itoa(i)); code == http.StatusTooManyRequests {
			t.Fatalf("distinct client %d behind the trusted proxy was limited", i)
		}
	}
	if len(warned) != 0 {
		t.Fatalf("warned about the trusted proxy: %v", warned)
	}
	send("198.51.100.20:5000", "203.0.113.1")
	send("198.51.100.21:5000", "203.0.113.1")
	if len(warned) != 1 || warned[0] != "198.51.100.20" {
		t.Fatalf("warnings %v, want one for the first untrusted forwarder", warned)
	}
}
