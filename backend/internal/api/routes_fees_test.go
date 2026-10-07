package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeFees struct {
	err         error
	gotPreview  fees.PreviewRequest
	gotInput    fees.RuleInput
	gotPricing  fees.Pricing
	gotActor    string
	gotRuleID   uint
	previewResp fees.Breakdown
}

func (f *fakeFees) Resolve(context.Context, fees.Query) (fees.Rule, error) { return fees.Rule{}, f.err }
func (f *fakeFees) Preview(_ context.Context, req fees.PreviewRequest) (fees.Breakdown, error) {
	f.gotPreview = req
	return f.previewResp, f.err
}
func (f *fakeFees) CreateRule(_ context.Context, in fees.RuleInput, actor string) (fees.Rule, error) {
	f.gotInput, f.gotActor = in, actor
	return fees.Rule{ID: 1, Version: 1, Scope: in.Scope}, f.err
}
func (f *fakeFees) NewVersion(_ context.Context, id uint, p fees.Pricing, actor string) (fees.Rule, error) {
	f.gotRuleID, f.gotPricing, f.gotActor = id, p, actor
	return fees.Rule{ID: 2, Version: 2}, f.err
}
func (f *fakeFees) GetRule(_ context.Context, id uint) (fees.Rule, error) {
	f.gotRuleID = id
	return fees.Rule{ID: id}, f.err
}
func (f *fakeFees) ListRules(context.Context, fees.RuleFilter) ([]fees.Rule, error) {
	return []fees.Rule{{ID: 1}}, f.err
}
func (f *fakeFees) Snapshot(context.Context, *gorm.DB, fees.AttemptRef, fees.Query) (fees.Snapshot, error) {
	return fees.Snapshot{}, f.err
}
func (f *fakeFees) PostFee(context.Context, *gorm.DB, fees.AttemptRef, decimal.Decimal) (fees.Breakdown, error) {
	return fees.Breakdown{}, f.err
}

func pass(c *gin.Context) { c.Next() }

func feesRouter(port fees.Port, m modules.FeesModule, platformID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := func(c *gin.Context) {
		c.Set("memberID", uint(5))
		c.Set("externalPlatformID", platformID)
	}
	m.Port = port
	RegisterFeesRoutes(r.Group("/api/v1"), &m, FeesAuth{Merchant: auth, Session: auth, Admin: pass})
	return r
}

func do(r *gin.Engine, method, path, body string, headers ...string) (*httptest.ResponseRecorder, map[string]any) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	r.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestFeesPreviewReturnsSnakeCaseStrings(t *testing.T) {
	fake := &fakeFees{previewResp: fees.Breakdown{
		RuleID: 3, Version: 2, Currency: "USD", FeeBearer: fees.BearerCustomer,
		Amount: decimal.RequireFromString("100"), Fee: decimal.RequireFromString("2.5"), Tax: decimal.RequireFromString("0.45"),
		CustomerTotal: decimal.RequireFromString("102.95"), MerchantNet: decimal.RequireFromString("100"),
	}}
	r := feesRouter(fake, modules.FeesModule{AdminEnabled: true}, 1)
	w, out := do(r, http.MethodPost, "/api/v1/fees/preview",
		`{"amount": 100, "currency": "USD", "method": "card", "connector": "stripe", "card_type": "credit", "region": "IN", "fee_bearer": "Customer"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body)
	}
	for k, want := range map[string]any{"rule_id": float64(3), "version": float64(2), "fee": "2.5", "tax": "0.45", "customer_total": "102.95", "merchant_net": "100"} {
		if out[k] != want {
			t.Errorf("%s = %#v, want %#v", k, out[k], want)
		}
	}
	q := fake.gotPreview
	if q.Connector != "stripe" || q.CardType != fees.CardCredit || q.Region != "IN" || q.FeeBearer == nil || *q.FeeBearer != fees.BearerCustomer || !q.Amount.Equal(decimal.NewFromInt(100)) {
		t.Errorf("request passed to port = %+v", q)
	}
}

func TestFeesPreviewValidation(t *testing.T) {
	r := feesRouter(&fakeFees{}, modules.FeesModule{AdminEnabled: true}, 1)
	cases := []struct {
		name, body, code, field string
	}{
		{"malformed", `{"amount":`, "invalid_json", ""},
		{"unknown field", `{"amount":"1","currency":"USD","method":"card","amount_cents":1}`, "invalid_json", ""},
		{"amount missing", `{"currency":"USD","method":"card"}`, "invalid_request", "amount"},
		{"amount not a number", `{"amount":"ten","currency":"USD","method":"card"}`, "invalid_json", ""},
		{"trailing data", `{"amount":"1","currency":"USD","method":"card"} {}`, "invalid_json", ""},
		{"body too large", `{"amount":"` + strings.Repeat("1", 20000) + `","currency":"USD","method":"card"}`, "invalid_json", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, out := do(r, http.MethodPost, "/api/v1/fees/preview", tc.body)
			if w.Code != http.StatusBadRequest || out["code"] != tc.code || (tc.field != "" && out["field"] != tc.field) {
				t.Fatalf("status %d body %.300s", w.Code, w.Body)
			}
		})
	}
}

func TestFeesErrorMapping(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{&fees.ValidationError{Field: "currency", Reason: "unknown"}, http.StatusBadRequest, "invalid_request"},
		{fmt.Errorf("%w: upi", fees.ErrSurchargeForbidden), http.StatusUnprocessableEntity, "surcharge_forbidden"},
		{fees.ErrFeeExceedsAmount, http.StatusUnprocessableEntity, "fee_exceeds_amount"},
		{&fees.AmbiguousRuleError{Specificity: fees.SpecConnector, RuleIDs: []uint{4, 9}}, http.StatusConflict, "ambiguous_fee_rule"},
		{&fees.OverlapError{RuleIDs: []uint{4}}, http.StatusConflict, "overlapping_fee_rule"},
		{fees.ErrNoRule, http.StatusNotFound, "no_fee_rule"},
		{fees.ErrNotFound, http.StatusNotFound, "not_found"},
		{fees.ErrStaleVersion, http.StatusConflict, "stale_version"},
		{fmt.Errorf("db down"), http.StatusInternalServerError, "internal"},
	}
	for _, tc := range cases {
		r := feesRouter(&fakeFees{err: tc.err}, modules.FeesModule{AdminEnabled: true}, 1)
		w, out := do(r, http.MethodPost, "/api/v1/fees/preview", `{"amount":"1","currency":"USD","method":"card"}`)
		if w.Code != tc.status || out["code"] != tc.code {
			t.Errorf("%v: status %d body %s", tc.err, w.Code, w.Body)
		}
		if tc.code == "ambiguous_fee_rule" && fmt.Sprint(out["rule_ids"]) != "[4 9]" {
			t.Errorf("rule_ids = %v", out["rule_ids"])
		}
		if tc.code == "overlapping_fee_rule" && fmt.Sprint(out["rule_ids"]) != "[4]" {
			t.Errorf("rule_ids = %v", out["rule_ids"])
		}
		if tc.code == "internal" && strings.Contains(w.Body.String(), "db down") {
			t.Error("internal error detail leaked to the client")
		}
	}
}

func TestFeesAdminCreateAndVersion(t *testing.T) {
	fake := &fakeFees{}
	r := feesRouter(fake, modules.FeesModule{AdminEnabled: true}, 1)
	w, out := do(r, http.MethodPost, "/api/v1/admin/fee-rules",
		`{"method":"card","connector":"stripe","card_type":"debit","currency":"USD","percent":"2.9","flat":0.3,"min_fee":"0.5","taxable":true,"tax_percent":"18","fee_bearer":"merchant","slabs":[{"up_to":"100","percent":"1","flat":"0"},{"up_to":null,"percent":"0.5","flat":"0"}]}`)
	if w.Code != http.StatusCreated || out["id"] != float64(1) {
		t.Fatalf("create: status %d body %s", w.Code, w.Body)
	}
	in := fake.gotInput
	if fake.gotActor != "member:5" || in.Method != fees.MethodCard || *in.Connector != "stripe" || *in.CardType != fees.CardDebit ||
		!in.Percent.Equal(decimal.RequireFromString("2.9")) || !in.Flat.Equal(decimal.RequireFromString("0.3")) ||
		!in.MinFee.Equal(decimal.RequireFromString("0.5")) || in.MaxFee != nil || len(in.Slabs) != 2 || in.Slabs[1].UpTo != nil {
		t.Fatalf("input passed to port = %+v actor %s", in, fake.gotActor)
	}
	w, _ = do(r, http.MethodPost, "/api/v1/admin/fee-rules/7/versions", `{"percent":"3","fee_bearer":"merchant","effective_from":"2030-01-01T00:00:00Z"}`)
	if w.Code != http.StatusCreated || fake.gotRuleID != 7 || !fake.gotPricing.Percent.Equal(decimal.NewFromInt(3)) || fake.gotPricing.EffectiveFrom == nil {
		t.Fatalf("version: status %d body %s pricing %+v", w.Code, w.Body, fake.gotPricing)
	}
	w, out = do(r, http.MethodPost, "/api/v1/admin/fee-rules/7/versions", `{"method":"upi","percent":"3","fee_bearer":"merchant"}`)
	if w.Code != http.StatusBadRequest || out["code"] != "invalid_json" {
		t.Fatalf("scope change in a version body: status %d body %s", w.Code, w.Body)
	}
	for _, path := range []string{"/api/v1/admin/fee-rules/abc/versions", "/api/v1/admin/fee-rules/0/versions"} {
		w, out = do(r, http.MethodPost, path, `{"fee_bearer":"merchant"}`)
		if w.Code != http.StatusBadRequest || out["field"] != "id" {
			t.Fatalf("%s: status %d body %s", path, w.Code, w.Body)
		}
	}
	w, out = do(r, http.MethodGet, "/api/v1/admin/fee-rules", "")
	if w.Code != http.StatusOK || len(out["fee_rules"].([]any)) != 1 {
		t.Fatalf("list: status %d body %s", w.Code, w.Body)
	}
}

func TestFeesAdminOperatorGate(t *testing.T) {
	cases := []struct {
		name     string
		module   modules.FeesModule
		platform uint
		status   int
	}{
		{"not configured in live", modules.FeesModule{AdminEnabled: false}, 1, http.StatusForbidden},
		{"other platform", modules.FeesModule{AdminEnabled: true, OperatorPlatformID: 2}, 1, http.StatusForbidden},
		{"operator platform", modules.FeesModule{AdminEnabled: true, OperatorPlatformID: 2}, 2, http.StatusOK},
		{"open in development", modules.FeesModule{AdminEnabled: true}, 1, http.StatusOK},
	}
	for _, tc := range cases {
		r := feesRouter(&fakeFees{}, tc.module, tc.platform)
		if w, _ := do(r, http.MethodGet, "/api/v1/admin/fee-rules", ""); w.Code != tc.status {
			t.Errorf("%s: status %d body %s", tc.name, w.Code, w.Body)
		}
		if w, _ := do(r, http.MethodPost, "/api/v1/fees/preview", `{"amount":"1","currency":"USD","method":"card"}`); w.Code != http.StatusOK {
			t.Errorf("%s: preview status %d", tc.name, w.Code)
		}
	}
}

func TestFeesAdminRoutesNeedSessionAndPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterFeesRoutes(r.Group("/api/v1"), &modules.FeesModule{Port: &fakeFees{}, AdminEnabled: true}, FeesAuth{Merchant: pass})
	if w, _ := do(r, http.MethodGet, "/api/v1/admin/fee-rules", ""); w.Code != http.StatusNotFound {
		t.Fatalf("admin routes mounted without session and permission guards: status %d", w.Code)
	}
}

func TestFeesCreateNeedsAnAuthenticatedMember(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterFeesRoutes(r.Group("/api/v1"), &modules.FeesModule{Port: &fakeFees{}, AdminEnabled: true}, FeesAuth{Merchant: pass, Session: pass, Admin: pass})
	if w, _ := do(r, http.MethodPost, "/api/v1/admin/fee-rules", `{"method":"card","currency":"USD","fee_bearer":"merchant"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", w.Code)
	}
}

// rbacFixture builds the real auth and RBAC services over SQLite with an admin and a plain merchant.
type rbacFixture struct {
	auth                 *service.AuthService
	mep                  *service.MemberExternalPlatformRoleService
	admin, merchant      models.Member
	operator, otherAdmin uint
}

func newRBACFixture(t *testing.T) rbacFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Permission{}, &models.Role{}, &models.Member{}, &models.ExternalPlatform{}, &models.MemberExternalPlatformRole{}); err != nil {
		t.Fatal(err)
	}
	mep := service.NewMemberExternalPlatformRoleService(repository.NewMemberExternalPlatformRoleRepository(db))
	sysAdmin := models.Permission{Name: "system.admin", DisplayName: "System Admin"}
	read := models.Permission{Name: "payments.read", DisplayName: "Read"}
	db.Create(&sysAdmin)
	db.Create(&read)
	ownerRole := models.Role{Name: "owner", DisplayName: "Owner", Permissions: []models.Permission{sysAdmin, read}}
	memberRole := models.Role{Name: "member", DisplayName: "Member", Permissions: []models.Permission{read}}
	db.Create(&ownerRole)
	db.Create(&memberRole)
	f := rbacFixture{mep: mep, auth: service.NewAuthService(nil, nil, strings.Repeat("s", 40))}
	f.admin = models.Member{Name: "Admin", MemberType: "merchant", State: "active"}
	f.merchant = models.Member{Name: "Merchant", MemberType: "merchant", State: "active"}
	db.Create(&f.admin)
	db.Create(&f.merchant)
	op, other := models.ExternalPlatform{Name: "Operator"}, models.ExternalPlatform{Name: "Other"}
	db.Create(&op)
	db.Create(&other)
	f.operator, f.otherAdmin = op.ID, other.ID
	ctx := context.Background()
	for _, a := range []struct{ member, platform, role uint }{
		{f.admin.ID, op.ID, ownerRole.ID}, {f.admin.ID, other.ID, ownerRole.ID}, {f.merchant.ID, op.ID, memberRole.ID},
	} {
		if err := mep.Assign(ctx, a.member, a.platform, a.role); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f rbacFixture) bearer(t *testing.T, m models.Member, platform uint) string {
	t.Helper()
	tok, err := f.auth.GenerateJWT(&m, platform)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func TestNewRouterEnforcesSessionPermissionAndOperatorOnFeeRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newRBACFixture(t)
	r := NewRouter(RouterConfig{
		AuthSvc:    f.auth,
		MEPRoleSvc: f.mep,
		Fees:       &modules.FeesModule{Port: &fakeFees{}, AdminEnabled: true, OperatorPlatformID: f.operator},
	})
	cases := []struct {
		name   string
		header []string
		status int
	}{
		{"no credentials", nil, http.StatusUnauthorized},
		{"api key is not a session", []string{"X-API-Key", "pk_anything"}, http.StatusUnauthorized},
		{"merchant without system.admin", []string{"Authorization", f.bearer(t, f.merchant, f.operator)}, http.StatusForbidden},
		{"system.admin on a non-operator platform", []string{"Authorization", f.bearer(t, f.admin, f.otherAdmin)}, http.StatusForbidden},
		{"system.admin on the operator platform", []string{"Authorization", f.bearer(t, f.admin, f.operator)}, http.StatusOK},
	}
	for _, tc := range cases {
		w, out := do(r, http.MethodGet, "/api/v1/admin/fee-rules", "", tc.header...)
		if w.Code != tc.status {
			t.Errorf("%s: status %d body %s", tc.name, w.Code, w.Body)
		}
		if tc.name == "system.admin on a non-operator platform" && out["code"] != "forbidden" {
			t.Errorf("%s: expected the operator gate, got %v", tc.name, out)
		}
	}
	for _, rt := range []struct{ method, path string }{
		{http.MethodPost, "/api/v1/fees/preview"},
		{http.MethodPost, "/api/v1/admin/fee-rules"},
		{http.MethodGet, "/api/v1/admin/fee-rules/1"},
		{http.MethodPost, "/api/v1/admin/fee-rules/1/versions"},
	} {
		if w, _ := do(r, rt.method, rt.path, `{}`); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without credentials: status %d, want 401", rt.method, rt.path, w.Code)
		}
	}
	// A merchant session can preview.
	w, _ := do(r, http.MethodPost, "/api/v1/fees/preview", `{"amount":"1","currency":"USD","method":"card"}`, "Authorization", f.bearer(t, f.merchant, f.operator))
	if w.Code != http.StatusOK {
		t.Errorf("merchant preview: status %d body %s", w.Code, w.Body)
	}
}
