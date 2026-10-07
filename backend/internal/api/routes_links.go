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
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/shopspring/decimal"
)

// LinksAuth carries the guards link routes need; nil PublicRead or PublicPay mounts that route unthrottled.
type LinksAuth struct {
	// Merchant authenticates link management (dashboard session or API key).
	Merchant   gin.HandlerFunc
	PublicRead gin.HandlerFunc
	PublicPay  gin.HandlerFunc
}

// RegisterLinksRoutes mounts merchant routes under /links and the public checkout routes under /public/links.
func RegisterLinksRoutes(rg *gin.RouterGroup, m *modules.LinksModule, auth LinksAuth) {
	h := &linksHandler{port: m.Port}
	if auth.Merchant != nil {
		g := rg.Group("/links", auth.Merchant)
		g.POST("", h.create)
		g.GET("", h.list)
		g.GET("/:id", h.get)
		g.PATCH("/:id", h.update)
		g.DELETE("/:id", h.remove)
		g.POST("/:id/publish", h.publish)
		g.POST("/:id/pause", h.pause)
		g.POST("/:id/archive", h.archive)
		g.POST("/:id/duplicate", h.duplicate)
	}
	pub := rg.Group("/public/links")
	pub.GET("/:short_code", withGuard(auth.PublicRead, h.render)...)
	pub.GET("/:short_code/qr.svg", withGuard(auth.PublicRead, h.qr)...)
	pub.POST("/:short_code/pay", withGuard(auth.PublicPay, h.pay)...)
}

func withGuard(guard, handler gin.HandlerFunc) []gin.HandlerFunc {
	if guard == nil {
		return []gin.HandlerFunc{handler}
	}
	return []gin.HandlerFunc{guard, handler}
}

type linksHandler struct {
	port links.Port
}

// linksMaxBody bounds request bodies; a full form with 100 line items and 20 questions stays well under it.
const linksMaxBody = 64 << 10

type linkResponse struct {
	ID          string            `json:"id"`
	Status      links.Status      `json:"status"`
	Environment links.Environment `json:"environment"`
	ShortCode   *string           `json:"short_code"`
	URL         *string           `json:"url"`
	Total       *decimal.Decimal  `json:"total"`
	UsesCount   int               `json:"uses_count"`
	Revision    int               `json:"revision"`
	PublishedAt *time.Time        `json:"published_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	// FeePreview is set on single-link responses; lists leave it null to avoid pricing every row.
	FeePreview []links.MethodPreview `json:"fee_preview"`
	links.Input
}

// toDetail is toResponse plus the per-method fee preview the form shows before publishing.
func (h *linksHandler) toDetail(c *gin.Context, l links.Link) linkResponse {
	r := h.toResponse(l)
	r.FeePreview = h.port.FeePreview(c.Request.Context(), l)
	return r
}

func (h *linksHandler) toResponse(l links.Link) linkResponse {
	r := linkResponse{
		ID: l.ID, Status: l.Status, Environment: l.Environment, Total: l.Total, UsesCount: l.UsesCount,
		Revision: l.Revision, PublishedAt: l.PublishedAt, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt, Input: l.Input,
	}
	if l.ShortCode != "" {
		code, url := l.ShortCode, h.port.URL(l.ShortCode)
		r.ShortCode, r.URL = &code, &url
	}
	return r
}

type payMethod struct {
	Method fees.Method `json:"method"`
	Chain  *string     `json:"chain"`
	Asset  *string     `json:"asset"`
}

type payResponse struct {
	PaymentReference   string           `json:"payment_reference"`
	Amount             decimal.Decimal  `json:"amount"`
	Currency           string           `json:"currency"`
	Fee                *decimal.Decimal `json:"fee"`
	Tax                *decimal.Decimal `json:"tax"`
	CustomerTotal      decimal.Decimal  `json:"customer_total"`
	FeeBearer          fees.FeeBearer   `json:"fee_bearer"`
	Method             payMethod        `json:"method"`
	CheckoutURL        *string          `json:"checkout_url"`
	DepositAddress     *string          `json:"deposit_address"`
	ExpiresAt          *time.Time       `json:"expires_at"`
	SuccessRedirectURL *string          `json:"success_redirect_url"`
	Replayed           bool             `json:"replayed"`
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// linksStatus maps a typed code to HTTP; everything unlisted is a 422 business refusal.
func linksStatus(code links.Code) int {
	switch code {
	case links.CodeNotFound:
		return http.StatusNotFound
	case links.CodeArchived, links.CodeExpired:
		return http.StatusGone
	case links.CodeInvalidTransition, links.CodePublishedImmutable, links.CodeNotEditable, links.CodeNotDeletable,
		links.CodeConflict, links.CodePaused, links.CodeUseLimitReached, links.CodeIdempotencyKeyReused, links.CodePaymentInProgress,
		links.CodeUseLimitBelowUses, links.CodeEnvironmentMismatch:
		return http.StatusConflict
	case links.CodeOpenPaymentsLimit:
		return http.StatusTooManyRequests
	case links.CodeInvalidRequest, links.CodeIdempotencyKeyRequired:
		return http.StatusBadRequest
	case links.CodePaymentCreationFailed:
		return http.StatusBadGateway
	case links.CodeShortCodeExhausted:
		return http.StatusServiceUnavailable
	}
	return http.StatusUnprocessableEntity
}

type linksErrItem struct {
	Code    links.Code `json:"code"`
	Field   string     `json:"field,omitempty"`
	Message string     `json:"message"`
}

func linksError(c *gin.Context, err error) {
	var e *links.Error
	if !errors.As(err, &e) {
		_ = c.Error(err)
		feesAbort(c, http.StatusInternalServerError, "internal", "internal error", nil)
		return
	}
	extra := gin.H{}
	if e.RetryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(int((e.RetryAfter+time.Second-1)/time.Second)))
	}
	if e.Field != "" {
		extra["field"] = e.Field
	}
	if len(e.Errors) > 0 {
		items := make([]linksErrItem, len(e.Errors))
		for i, x := range e.Errors {
			items[i] = linksErrItem{Code: x.Code, Field: x.Field, Message: x.Message}
		}
		extra["errors"] = items
	}
	feesAbort(c, linksStatus(e.Code), string(e.Code), e.Message, extra)
}

// readBody returns the raw body, refusing more than linksMaxBody.
func readBody(c *gin.Context) ([]byte, bool) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, linksMaxBody+1))
	if err != nil {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "could not read body", nil)
		return nil, false
	}
	if len(raw) > linksMaxBody {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "body larger than 64 KiB", nil)
		return nil, false
	}
	return raw, true
}

// decodeOnto strictly decodes raw over dst, so absent keys keep dst's values and unknown keys are refused.
func decodeOnto(c *gin.Context, raw []byte, dst any) bool {
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

func linksActor(c *gin.Context) (links.Actor, bool) {
	m, _ := c.Get("memberID")
	p, _ := c.Get("externalPlatformID")
	mid, ok1 := m.(uint)
	pid, ok2 := p.(uint)
	if !ok1 || !ok2 || mid == 0 || pid == 0 {
		feesAbort(c, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return links.Actor{}, false
	}
	return links.Actor{MemberID: mid, PlatformID: pid}, true
}

func (h *linksHandler) create(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	raw, ok := readBody(c)
	if !ok {
		return
	}
	in := links.DefaultInput()
	if !decodeOnto(c, raw, &in) {
		return
	}
	l, err := h.port.Create(c.Request.Context(), actor, in)
	if err != nil {
		linksError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toDetail(c, l))
}

func (h *linksHandler) list(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	ls, total, err := h.port.List(c.Request.Context(), actor.PlatformID, links.ListFilter{Status: links.Status(c.Query("status")), Limit: limit, Offset: offset})
	if err != nil {
		linksError(c, err)
		return
	}
	out := make([]linkResponse, len(ls))
	for i, l := range ls {
		out[i] = h.toResponse(l)
	}
	c.JSON(http.StatusOK, gin.H{"links": out, "total": total})
}

func (h *linksHandler) get(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	l, err := h.port.Get(c.Request.Context(), actor.PlatformID, c.Param("id"))
	if err != nil {
		linksError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toDetail(c, l))
}

// update is a top-level merge: each key sent replaces that field whole; keys not sent keep their value.
func (h *linksHandler) update(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	raw, ok := readBody(c)
	if !ok {
		return
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(raw, &patch); err != nil {
		feesAbort(c, http.StatusBadRequest, "invalid_json", "body must be a JSON object", nil)
		return
	}
	ctx := c.Request.Context()
	cur, err := h.port.Get(ctx, actor.PlatformID, c.Param("id"))
	if err != nil {
		linksError(c, err)
		return
	}
	// The merge base's revision guards the save; If-Match pins an older one the client read.
	revision := cur.Revision
	if im := strings.Trim(strings.TrimSpace(c.GetHeader("If-Match")), `"`); im != "" {
		if revision, err = strconv.Atoi(im); err != nil {
			feesAbort(c, http.StatusBadRequest, "invalid_request", "If-Match must be the link's revision number", gin.H{"field": "If-Match"})
			return
		}
	}
	baseRaw, err := json.Marshal(cur.Input)
	if err != nil {
		linksError(c, err)
		return
	}
	var merged map[string]json.RawMessage
	if err := json.Unmarshal(baseRaw, &merged); err != nil {
		linksError(c, err)
		return
	}
	for k, v := range patch {
		merged[k] = v
	}
	mergedRaw, _ := json.Marshal(merged)
	var in links.Input
	if !decodeOnto(c, mergedRaw, &in) {
		return
	}
	l, err := h.port.Update(ctx, actor.PlatformID, cur.ID, revision, in)
	if err != nil {
		linksError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toDetail(c, l))
}

func (h *linksHandler) remove(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	if err := h.port.Delete(c.Request.Context(), actor.PlatformID, c.Param("id")); err != nil {
		linksError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *linksHandler) transition(c *gin.Context, do func(ctx *gin.Context, platformID uint, id string) (links.Link, error)) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	l, err := do(c, actor.PlatformID, c.Param("id"))
	if err != nil {
		linksError(c, err)
		return
	}
	c.JSON(http.StatusOK, h.toDetail(c, l))
}

func (h *linksHandler) publish(c *gin.Context) {
	h.transition(c, func(ctx *gin.Context, pid uint, id string) (links.Link, error) {
		return h.port.Publish(ctx.Request.Context(), pid, id)
	})
}

func (h *linksHandler) pause(c *gin.Context) {
	h.transition(c, func(ctx *gin.Context, pid uint, id string) (links.Link, error) {
		return h.port.Pause(ctx.Request.Context(), pid, id)
	})
}

func (h *linksHandler) archive(c *gin.Context) {
	h.transition(c, func(ctx *gin.Context, pid uint, id string) (links.Link, error) {
		return h.port.Archive(ctx.Request.Context(), pid, id)
	})
}

func (h *linksHandler) duplicate(c *gin.Context) {
	actor, ok := linksActor(c)
	if !ok {
		return
	}
	l, err := h.port.Duplicate(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		linksError(c, err)
		return
	}
	c.JSON(http.StatusCreated, h.toDetail(c, l))
}

func (h *linksHandler) render(c *gin.Context) {
	m, err := h.port.Render(c.Request.Context(), c.Param("short_code"))
	if err != nil {
		linksError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, m)
}

// qr serves the short link as an SVG QR code; only published links have one.
func (h *linksHandler) qr(c *gin.Context) {
	m, err := h.port.Render(c.Request.Context(), c.Param("short_code"))
	if err != nil {
		linksError(c, err)
		return
	}
	svg, err := links.QRSVG(m.URL)
	if err != nil {
		linksError(c, err)
		return
	}
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
	c.Data(http.StatusOK, "image/svg+xml", []byte(svg))
}

func (h *linksHandler) pay(c *gin.Context) {
	raw, ok := readBody(c)
	if !ok {
		return
	}
	var req links.PayRequest
	if !decodeOnto(c, raw, &req) {
		return
	}
	req.IdempotencyKey = c.GetHeader("Idempotency-Key")
	req.ClientIP = c.ClientIP()
	res, err := h.port.Pay(c.Request.Context(), c.Param("short_code"), req)
	if err != nil {
		linksError(c, err)
		return
	}
	status := http.StatusCreated
	if res.Replayed {
		status = http.StatusOK
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, payResponse{
		PaymentReference: res.PaymentReference, Amount: res.Amount, Currency: res.Currency, Fee: res.Fee, Tax: res.Tax,
		CustomerTotal: res.CustomerTotal, FeeBearer: res.FeeBearer,
		Method:      payMethod{Method: res.Method.Method, Chain: optStr(res.Method.Chain), Asset: optStr(res.Method.Asset)},
		CheckoutURL: optStr(res.CheckoutURL), DepositAddress: optStr(res.DepositAddress), ExpiresAt: res.ExpiresAt,
		SuccessRedirectURL: optStr(res.SuccessRedirectURL), Replayed: res.Replayed,
	})
}
