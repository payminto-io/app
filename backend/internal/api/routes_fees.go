package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/shopspring/decimal"
)

// FeesAuth carries the guards fee routes need; modules cannot hold the auth services (import cycle).
type FeesAuth struct {
	// Merchant authenticates the preview caller (session or API key).
	Merchant gin.HandlerFunc
	// Session authenticates rule management; dashboard sessions only, never API keys.
	Session gin.HandlerFunc
	// Admin is the permission check for rule management.
	Admin gin.HandlerFunc
}

// RegisterFeesRoutes mounts fee routes on rg; admin routes are mounted only with both Session and Admin guards.
func RegisterFeesRoutes(rg *gin.RouterGroup, m *modules.FeesModule, auth FeesAuth) {
	h := &feesHandler{port: m.Port}
	if auth.Merchant != nil {
		rg.POST("/fees/preview", auth.Merchant, h.preview)
	}
	if auth.Session == nil || auth.Admin == nil {
		return
	}
	admin := rg.Group("/admin/fee-rules", auth.Session, auth.Admin, feesOperatorOnly(m))
	admin.GET("", h.list)
	admin.POST("", h.create)
	admin.GET("/:id", h.get)
	admin.POST("/:id/versions", h.newVersion)
}

// feesOperatorOnly keeps platform-wide fee rules to the operator platform (internal/fees/README.md "Access").
func feesOperatorOnly(m *modules.FeesModule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.AdminEnabled {
			feesAbort(c, http.StatusForbidden, "fee_admin_not_configured", "set FEES_OPERATOR_PLATFORM_ID to manage fee rules in this environment", nil)
			return
		}
		if m.OperatorPlatformID != 0 {
			pid, _ := c.Get("externalPlatformID")
			if id, ok := pid.(uint); !ok || id != m.OperatorPlatformID {
				feesAbort(c, http.StatusForbidden, "forbidden", "fee rules are managed by the operator platform", nil)
				return
			}
		}
		c.Next()
	}
}

type feesHandler struct {
	port fees.Port
}

type pricingBody struct {
	Percent       *decimal.Decimal `json:"percent"`
	Flat          *decimal.Decimal `json:"flat"`
	Slabs         []fees.Slab      `json:"slabs"`
	MinFee        *decimal.Decimal `json:"min_fee"`
	MaxFee        *decimal.Decimal `json:"max_fee"`
	Taxable       bool             `json:"taxable"`
	TaxPercent    *decimal.Decimal `json:"tax_percent"`
	FeeBearer     string           `json:"fee_bearer"`
	EffectiveFrom *time.Time       `json:"effective_from"`
	EffectiveTo   *time.Time       `json:"effective_to"`
}

type ruleBody struct {
	Method    string  `json:"method"`
	Connector *string `json:"connector"`
	CardType  *string `json:"card_type"`
	Region    *string `json:"region"`
	Currency  string  `json:"currency"`
	pricingBody
}

type previewBody struct {
	Amount    *decimal.Decimal `json:"amount"`
	Currency  string           `json:"currency"`
	Method    string           `json:"method"`
	Connector string           `json:"connector"`
	CardType  string           `json:"card_type"`
	Region    string           `json:"region"`
	FeeBearer *string          `json:"fee_bearer"`
}

type ruleResponse struct {
	ID            uint             `json:"id"`
	LineageID     string           `json:"lineage_id"`
	Version       int              `json:"version"`
	Method        fees.Method      `json:"method"`
	Connector     *string          `json:"connector"`
	CardType      *fees.CardType   `json:"card_type"`
	Region        *string          `json:"region"`
	Currency      string           `json:"currency"`
	Percent       decimal.Decimal  `json:"percent"`
	Flat          decimal.Decimal  `json:"flat"`
	Slabs         []fees.Slab      `json:"slabs"`
	MinFee        *decimal.Decimal `json:"min_fee"`
	MaxFee        *decimal.Decimal `json:"max_fee"`
	Taxable       bool             `json:"taxable"`
	TaxPercent    decimal.Decimal  `json:"tax_percent"`
	FeeBearer     fees.FeeBearer   `json:"fee_bearer"`
	EffectiveFrom time.Time        `json:"effective_from"`
	EffectiveTo   *time.Time       `json:"effective_to"`
	CreatedBy     string           `json:"created_by"`
	CreatedAt     time.Time        `json:"created_at"`
}

type previewResponse struct {
	RuleID        uint            `json:"rule_id"`
	Version       int             `json:"version"`
	Currency      string          `json:"currency"`
	FeeBearer     fees.FeeBearer  `json:"fee_bearer"`
	Amount        decimal.Decimal `json:"amount"`
	Fee           decimal.Decimal `json:"fee"`
	Tax           decimal.Decimal `json:"tax"`
	CustomerTotal decimal.Decimal `json:"customer_total"`
	MerchantNet   decimal.Decimal `json:"merchant_net"`
}

func toRuleResponse(r fees.Rule) ruleResponse {
	return ruleResponse{
		ID: r.ID, LineageID: r.LineageID, Version: r.Version,
		Method: r.Method, Connector: r.Connector, CardType: r.CardType, Region: r.Region, Currency: r.Currency,
		Percent: r.Percent, Flat: r.Flat, Slabs: r.Slabs, MinFee: r.MinFee, MaxFee: r.MaxFee,
		Taxable: r.Taxable, TaxPercent: r.TaxPercent, FeeBearer: r.FeeBearer,
		EffectiveFrom: r.EffectiveFrom, EffectiveTo: r.EffectiveTo, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt,
	}
}

func orZero(v *decimal.Decimal) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return *v
}

func (p pricingBody) pricing() fees.Pricing {
	return fees.Pricing{
		Percent: orZero(p.Percent), Flat: orZero(p.Flat), Slabs: p.Slabs,
		MinFee: p.MinFee, MaxFee: p.MaxFee, Taxable: p.Taxable, TaxPercent: orZero(p.TaxPercent),
		FeeBearer:     fees.FeeBearer(strings.ToLower(strings.TrimSpace(p.FeeBearer))),
		EffectiveFrom: p.EffectiveFrom, EffectiveTo: p.EffectiveTo,
	}
}

func feesAbort(c *gin.Context, status int, code, message string, extra gin.H) {
	body := gin.H{"error": message, "code": code}
	for k, v := range extra {
		body[k] = v
	}
	c.AbortWithStatusJSON(status, body)
}

func feesError(c *gin.Context, err error) {
	var ve *fees.ValidationError
	var amb *fees.AmbiguousRuleError
	var overlap *fees.OverlapError
	switch {
	case errors.As(err, &ve):
		feesAbort(c, http.StatusBadRequest, "invalid_request", ve.Error(), gin.H{"field": ve.Field})
	case errors.As(err, &amb):
		feesAbort(c, http.StatusConflict, "ambiguous_fee_rule", amb.Error(), gin.H{"rule_ids": amb.RuleIDs})
	case errors.As(err, &overlap):
		feesAbort(c, http.StatusConflict, "overlapping_fee_rule", overlap.Error(), gin.H{"rule_ids": overlap.RuleIDs})
	case errors.Is(err, fees.ErrFeeExceedsAmount):
		feesAbort(c, http.StatusUnprocessableEntity, "fee_exceeds_amount", err.Error(), nil)
	case errors.Is(err, fees.ErrSurchargeForbidden):
		feesAbort(c, http.StatusUnprocessableEntity, "surcharge_forbidden", err.Error(), nil)
	case errors.Is(err, fees.ErrNoRule):
		feesAbort(c, http.StatusNotFound, "no_fee_rule", err.Error(), nil)
	case errors.Is(err, fees.ErrNotFound):
		feesAbort(c, http.StatusNotFound, "not_found", err.Error(), nil)
	case errors.Is(err, fees.ErrStaleVersion):
		feesAbort(c, http.StatusConflict, "stale_version", err.Error(), nil)
	default:
		feesAbort(c, http.StatusInternalServerError, "internal", "internal error", nil)
	}
}

// feesMaxBody bounds request bodies; fee bodies are a few hundred bytes.
const feesMaxBody = 16 << 10

// decodeStrict rejects unknown fields so a version body cannot silently carry scope changes.
func decodeStrict(c *gin.Context, dst any) bool {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, feesMaxBody+1))
	if err != nil {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "could not read body", nil)
		return false
	}
	if len(raw) > feesMaxBody {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "body larger than 16 KiB", nil)
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		feesAbort(c, http.StatusBadRequest, "invalid_json", fmt.Sprintf("invalid JSON body: %v", err), nil)
		return false
	}
	if dec.More() {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "body must be one JSON object", nil)
		return false
	}
	return true
}

func feesActor(c *gin.Context) (string, bool) {
	v, _ := c.Get("memberID")
	id, ok := v.(uint)
	if !ok || id == 0 {
		feesAbort(c, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return "", false
	}
	return "member:" + strconv.FormatUint(uint64(id), 10), true
}

func ruleIDParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		feesAbort(c, http.StatusBadRequest, "invalid_request", "id must be a positive integer", gin.H{"field": "id"})
		return 0, false
	}
	return uint(id), true
}

func (h *feesHandler) preview(c *gin.Context) {
	var body previewBody
	if !decodeStrict(c, &body) {
		return
	}
	if body.Amount == nil {
		feesAbort(c, http.StatusBadRequest, "invalid_request", "amount is required", gin.H{"field": "amount"})
		return
	}
	req := fees.PreviewRequest{
		Query: fees.Query{
			Method: fees.Method(body.Method), Connector: body.Connector, CardType: fees.CardType(body.CardType),
			Region: body.Region, Currency: body.Currency,
		},
		Amount: *body.Amount,
	}
	if body.FeeBearer != nil {
		b := fees.FeeBearer(strings.ToLower(strings.TrimSpace(*body.FeeBearer)))
		req.FeeBearer = &b
	}
	b, err := h.port.Preview(c.Request.Context(), req)
	if err != nil {
		feesError(c, err)
		return
	}
	c.JSON(http.StatusOK, previewResponse{
		RuleID: b.RuleID, Version: b.Version, Currency: b.Currency, FeeBearer: b.FeeBearer,
		Amount: b.Amount, Fee: b.Fee, Tax: b.Tax, CustomerTotal: b.CustomerTotal, MerchantNet: b.MerchantNet,
	})
}

func (h *feesHandler) list(c *gin.Context) {
	rules, err := h.port.ListRules(c.Request.Context(), fees.RuleFilter{
		Method: fees.Method(c.Query("method")), Currency: c.Query("currency"), LineageID: c.Query("lineage_id"),
	})
	if err != nil {
		feesError(c, err)
		return
	}
	out := make([]ruleResponse, len(rules))
	for i, r := range rules {
		out[i] = toRuleResponse(r)
	}
	c.JSON(http.StatusOK, gin.H{"fee_rules": out})
}

func (h *feesHandler) get(c *gin.Context) {
	id, ok := ruleIDParam(c)
	if !ok {
		return
	}
	r, err := h.port.GetRule(c.Request.Context(), id)
	if err != nil {
		feesError(c, err)
		return
	}
	c.JSON(http.StatusOK, toRuleResponse(r))
}

func (h *feesHandler) create(c *gin.Context) {
	actor, ok := feesActor(c)
	if !ok {
		return
	}
	var body ruleBody
	if !decodeStrict(c, &body) {
		return
	}
	in := fees.RuleInput{
		Scope: fees.Scope{
			Method: fees.Method(body.Method), Connector: body.Connector, Region: body.Region, Currency: body.Currency,
		},
		Pricing: body.pricing(),
	}
	if body.CardType != nil {
		ct := fees.CardType(*body.CardType)
		in.CardType = &ct
	}
	r, err := h.port.CreateRule(c.Request.Context(), in, actor)
	if err != nil {
		feesError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRuleResponse(r))
}

func (h *feesHandler) newVersion(c *gin.Context) {
	actor, ok := feesActor(c)
	if !ok {
		return
	}
	id, ok := ruleIDParam(c)
	if !ok {
		return
	}
	var body pricingBody
	if !decodeStrict(c, &body) {
		return
	}
	r, err := h.port.NewVersion(c.Request.Context(), id, body.pricing(), actor)
	if err != nil {
		feesError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toRuleResponse(r))
}
