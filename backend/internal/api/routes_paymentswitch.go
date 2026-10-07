package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
)

// RegisterPaymentSwitchRoutes mounts the v2 payments API. Merchant routes sit behind auth (the same
// JWT-or-API-key chain as v1); the webhook route is public and verified by the connector.
func RegisterPaymentSwitchRoutes(rg *gin.RouterGroup, m *modules.PaymentSwitchModule, auth gin.HandlerFunc) {
	h := &paymentSwitchHandler{svc: m.Service}
	payments := rg.Group("/payments")
	payments.Use(auth)
	{
		payments.POST("", h.create)
		payments.GET("/:id", h.get)
		payments.POST("/:id/confirm", h.confirm)
		payments.POST("/:id/capture", h.capture)
		payments.POST("/:id/cancel", h.cancel)
		payments.POST("/:id/refunds", h.refund)
	}
	rg.POST("/webhooks/:connector", h.webhook)
}

type paymentSwitchHandler struct {
	svc *paymentswitch.Service
}

// Wire DTOs: snake_case, amounts as decimal strings, never floats.
type paymentMethodDTO struct {
	Type    string            `json:"type"`
	Token   string            `json:"token"`
	Details map[string]string `json:"details,omitempty"`
}

type createPaymentRequest struct {
	Amount         string            `json:"amount"`
	Asset          string            `json:"asset"`
	CaptureMethod  string            `json:"capture_method"`
	PaymentMethod  *paymentMethodDTO `json:"payment_method"`
	Confirm        bool              `json:"confirm"`
	Description    string            `json:"description"`
	ReturnURL      string            `json:"return_url"`
	Metadata       map[string]string `json:"metadata"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type confirmPaymentRequest struct {
	PaymentMethod *paymentMethodDTO `json:"payment_method"`
}

type capturePaymentRequest struct {
	Amount string `json:"amount"`
}

type cancelPaymentRequest struct {
	Reason string `json:"reason"`
}

type refundPaymentRequest struct {
	Amount         string `json:"amount"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

type nextActionDTO struct {
	Type        string `json:"type"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Address     string `json:"address,omitempty"`
	Amount      string `json:"amount,omitempty"`
	Asset       string `json:"asset,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type attemptDTO struct {
	ID                     string         `json:"id"`
	ConnectorCode          string         `json:"connector_code"`
	Status                 string         `json:"status"`
	ConnectorStatus        string         `json:"connector_status"`
	Amount                 string         `json:"amount"`
	Asset                  string         `json:"asset"`
	AmountToCapture        string         `json:"amount_to_capture,omitempty"`
	AmountCaptured         string         `json:"amount_captured"`
	AmountReceived         *string        `json:"amount_received,omitempty"`
	ReceivedAsset          string         `json:"received_asset,omitempty"`
	ConnectorTransactionID string         `json:"connector_transaction_id,omitempty"`
	SelectionReason        string         `json:"selection_reason,omitempty"`
	ErrorCode              string         `json:"error_code,omitempty"`
	ErrorMessage           string         `json:"error_message,omitempty"`
	NextAction             *nextActionDTO `json:"next_action,omitempty"`
	CreatedAt              string         `json:"created_at"`
	UpdatedAt              string         `json:"updated_at"`
}

type refundDTO struct {
	ID                string `json:"id"`
	PaymentID         string `json:"payment_id"`
	AttemptID         string `json:"attempt_id"`
	Status            string `json:"status"`
	ConnectorStatus   string `json:"connector_status"`
	Amount            string `json:"amount"`
	Asset             string `json:"asset"`
	ConnectorRefundID string `json:"connector_refund_id,omitempty"`
	Reason            string `json:"reason,omitempty"`
	ErrorCode         string `json:"error_code,omitempty"`
	ErrorMessage      string `json:"error_message,omitempty"`
	IdempotencyKey    string `json:"idempotency_key"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

type paymentDTO struct {
	ID                string            `json:"id"`
	Status            string            `json:"status"`
	Amount            string            `json:"amount"`
	Asset             string            `json:"asset"`
	AmountCaptured    string            `json:"amount_captured"`
	AmountRefunded    string            `json:"amount_refunded"`
	CaptureMethod     string            `json:"capture_method"`
	PaymentMethodType string            `json:"payment_method_type,omitempty"`
	ConnectorCode     string            `json:"connector_code,omitempty"`
	ActiveAttemptID   string            `json:"active_attempt_id,omitempty"`
	Description       string            `json:"description,omitempty"`
	ReturnURL         string            `json:"return_url,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	NextAction        *nextActionDTO    `json:"next_action,omitempty"`
	ErrorCode         string            `json:"error_code,omitempty"`
	ErrorMessage      string            `json:"error_message,omitempty"`
	IdempotencyKey    string            `json:"idempotency_key"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
	Attempts          []attemptDTO      `json:"attempts,omitempty"`
	Refunds           []refundDTO       `json:"refunds,omitempty"`
}

type paymentEnvelope struct {
	Payment paymentDTO `json:"payment"`
}

type refundEnvelope struct {
	Refund refundDTO `json:"refund"`
}

type webhookEnvelope struct {
	EventID   string `json:"event_id"`
	PaymentID string `json:"payment_id,omitempty"`
	Ignored   bool   `json:"ignored"`
	Reason    string `json:"reason,omitempty"`
}

type switchErrorEnvelope struct {
	Error switchErrorBody `json:"error"`
}

type switchErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	PaymentID string `json:"payment_id,omitempty"`
}

func (h *paymentSwitchHandler) create(c *gin.Context) {
	merchant, platform, ok := switchIdentity(c)
	if !ok {
		return
	}
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "body is not valid JSON for this request")
		return
	}
	amount, err := parseAmount(req.Amount)
	if err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "amount must be a decimal string")
		return
	}
	cmd := paymentswitch.CreateCommand{
		MerchantID:     merchant,
		PlatformID:     platform,
		IdempotencyKey: firstNonEmpty(c.GetHeader("Idempotency-Key"), req.IdempotencyKey),
		Money:          paymentswitch.Money{Amount: amount, Asset: req.Asset},
		CaptureMethod:  connectors.CaptureMethod(req.CaptureMethod),
		PaymentMethod:  toPaymentMethod(req.PaymentMethod),
		Confirm:        req.Confirm,
		Description:    req.Description,
		ReturnURL:      req.ReturnURL,
		Metadata:       req.Metadata,
	}
	intent, err := h.svc.Create(c.Request.Context(), cmd)
	if err != nil {
		// The intent may exist although its confirm failed; the caller must learn its id (M4).
		writeSwitchServiceErrorFor(c, err, intent.ID)
		return
	}
	c.JSON(http.StatusCreated, paymentEnvelope{Payment: toPaymentDTO(intent, nil, nil)})
}

func (h *paymentSwitchHandler) get(c *gin.Context) {
	merchant, _, ok := switchIdentity(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	if c.Query("sync") == "true" {
		if _, err := h.svc.Sync(ctx, merchant, c.Param("id")); err != nil {
			writeSwitchServiceError(c, err)
			return
		}
	}
	view, err := h.svc.Get(ctx, merchant, c.Param("id"))
	if err != nil {
		writeSwitchServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, paymentEnvelope{Payment: toPaymentDTO(view.Intent, view.Attempts, view.Refunds)})
}

func (h *paymentSwitchHandler) confirm(c *gin.Context) {
	merchant, _, ok := switchIdentity(c)
	if !ok {
		return
	}
	var req confirmPaymentRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "body is not valid JSON for this request")
		return
	}
	intent, err := h.svc.Confirm(c.Request.Context(), merchant, c.Param("id"), paymentswitch.ConfirmCommand{PaymentMethod: toPaymentMethod(req.PaymentMethod)})
	h.respondIntent(c, intent, err)
}

func (h *paymentSwitchHandler) capture(c *gin.Context) {
	merchant, _, ok := switchIdentity(c)
	if !ok {
		return
	}
	var req capturePaymentRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "body is not valid JSON for this request")
		return
	}
	cmd := paymentswitch.CaptureCommand{}
	if req.Amount != "" {
		amount, err := parseAmount(req.Amount)
		if err != nil {
			writeSwitchError(c, http.StatusBadRequest, "invalid_request", "amount must be a decimal string")
			return
		}
		cmd.Amount = &amount
	}
	intent, err := h.svc.Capture(c.Request.Context(), merchant, c.Param("id"), cmd)
	h.respondIntent(c, intent, err)
}

func (h *paymentSwitchHandler) cancel(c *gin.Context) {
	merchant, _, ok := switchIdentity(c)
	if !ok {
		return
	}
	var req cancelPaymentRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "body is not valid JSON for this request")
		return
	}
	intent, err := h.svc.Cancel(c.Request.Context(), merchant, c.Param("id"), paymentswitch.CancelCommand{Reason: req.Reason})
	h.respondIntent(c, intent, err)
}

func (h *paymentSwitchHandler) refund(c *gin.Context) {
	merchant, _, ok := switchIdentity(c)
	if !ok {
		return
	}
	var req refundPaymentRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "body is not valid JSON for this request")
		return
	}
	cmd := paymentswitch.RefundCommand{Reason: req.Reason, IdempotencyKey: firstNonEmpty(c.GetHeader("Idempotency-Key"), req.IdempotencyKey)}
	if req.Amount != "" {
		amount, err := parseAmount(req.Amount)
		if err != nil {
			writeSwitchError(c, http.StatusBadRequest, "invalid_request", "amount must be a decimal string")
			return
		}
		cmd.Amount = &amount
	}
	refund, err := h.svc.Refund(c.Request.Context(), merchant, c.Param("id"), cmd)
	if err != nil {
		writeSwitchServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, refundEnvelope{Refund: toRefundDTO(refund)})
}

func (h *paymentSwitchHandler) webhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		writeSwitchError(c, http.StatusBadRequest, "invalid_request", "could not read body")
		return
	}
	result, err := h.svc.HandleWebhook(c.Request.Context(), connectors.Code(c.Param("connector")), c.Request.Header, body)
	if err != nil {
		writeSwitchServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, webhookEnvelope{EventID: result.EventID, PaymentID: result.IntentID, Ignored: result.Ignored, Reason: result.IgnoreWhy})
}

func (h *paymentSwitchHandler) respondIntent(c *gin.Context, intent paymentswitch.Intent, err error) {
	if err != nil {
		writeSwitchServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, paymentEnvelope{Payment: toPaymentDTO(intent, nil, nil)})
}

// switchIdentity reads what the auth middleware set; the merchant is the member, the platform the tenant.
func switchIdentity(c *gin.Context) (merchant, platform string, ok bool) {
	memberID, hasMember := c.Get("memberID")
	platformID, hasPlatform := c.Get("externalPlatformID")
	m, okM := memberID.(uint)
	p, okP := platformID.(uint)
	if !hasMember || !hasPlatform || !okM || !okP || m == 0 {
		writeSwitchError(c, http.StatusUnauthorized, "unauthenticated", "merchant identity missing")
		c.Abort()
		return "", "", false
	}
	return strconv.FormatUint(uint64(m), 10), strconv.FormatUint(uint64(p), 10), true
}

func bindOptionalJSON(c *gin.Context, v any) error {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return nil
	}
	return c.ShouldBindJSON(v)
}

func parseAmount(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, errors.New("empty amount")
	}
	return decimal.NewFromString(s)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func toPaymentMethod(d *paymentMethodDTO) *connectors.PaymentMethod {
	if d == nil {
		return nil
	}
	return &connectors.PaymentMethod{Type: connectors.Method(d.Type), Token: d.Token, Details: d.Details}
}

func toNextAction(a *connectors.NextAction) *nextActionDTO {
	if a == nil {
		return nil
	}
	return &nextActionDTO{Type: a.Type, RedirectURL: a.RedirectURL, Address: a.Address, Amount: a.Amount, Asset: a.Asset, ExpiresAt: a.ExpiresAt}
}

func toPaymentDTO(in paymentswitch.Intent, attempts []paymentswitch.Attempt, refunds []paymentswitch.Refund) paymentDTO {
	out := paymentDTO{
		ID:                in.ID,
		Status:            string(in.Status),
		Amount:            in.Money.Amount.String(),
		Asset:             in.Money.Asset,
		AmountCaptured:    in.AmountCaptured.String(),
		AmountRefunded:    in.AmountRefunded.String(),
		CaptureMethod:     string(in.CaptureMethod),
		PaymentMethodType: string(in.PaymentMethodType),
		ConnectorCode:     string(in.ConnectorCode),
		ActiveAttemptID:   in.ActiveAttemptID,
		Description:       in.Description,
		ReturnURL:         in.ReturnURL,
		Metadata:          in.Metadata,
		NextAction:        toNextAction(in.NextAction),
		ErrorCode:         in.LastErrorCode,
		ErrorMessage:      in.LastErrorMessage,
		IdempotencyKey:    in.IdempotencyKey,
		CreatedAt:         in.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt:         in.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
	for _, a := range attempts {
		dto := attemptDTO{
			ID:                     a.ID,
			ConnectorCode:          string(a.ConnectorCode),
			Status:                 string(a.Status),
			ConnectorStatus:        string(a.RawStatus),
			Amount:                 a.Money.Amount.String(),
			Asset:                  a.Money.Asset,
			AmountCaptured:         a.AmountCaptured.String(),
			ReceivedAsset:          a.ReceivedAsset,
			ConnectorTransactionID: a.ConnectorTransactionID,
			SelectionReason:        a.SelectionReason,
			ErrorCode:              a.ErrorCode,
			ErrorMessage:           a.ErrorMessage,
			NextAction:             toNextAction(a.NextAction),
			CreatedAt:              a.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			UpdatedAt:              a.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		}
		if a.AmountToCapture.IsPositive() {
			dto.AmountToCapture = a.AmountToCapture.String()
		}
		if a.AmountReceived != nil {
			received := a.AmountReceived.String()
			dto.AmountReceived = &received
		}
		out.Attempts = append(out.Attempts, dto)
	}
	for _, r := range refunds {
		out.Refunds = append(out.Refunds, toRefundDTO(r))
	}
	return out
}

func toRefundDTO(r paymentswitch.Refund) refundDTO {
	return refundDTO{
		ID:                r.ID,
		PaymentID:         r.IntentID,
		AttemptID:         r.AttemptID,
		Status:            string(r.Status),
		ConnectorStatus:   string(r.RawStatus),
		Amount:            r.Money.Amount.String(),
		Asset:             r.Money.Asset,
		ConnectorRefundID: r.ConnectorRefundID,
		Reason:            r.Reason,
		ErrorCode:         r.ErrorCode,
		ErrorMessage:      r.ErrorMessage,
		IdempotencyKey:    r.IdempotencyKey,
		CreatedAt:         r.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt:         r.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}

func writeSwitchError(c *gin.Context, status int, code, message string) {
	c.JSON(status, switchErrorEnvelope{Error: switchErrorBody{Code: code, Message: message}})
}

func writeSwitchServiceError(c *gin.Context, err error) { writeSwitchServiceErrorFor(c, err, "") }

// writeSwitchServiceErrorFor maps module errors onto HTTP; messages name the rule, never internals.
func writeSwitchServiceErrorFor(c *gin.Context, err error, paymentID string) {
	status, code, message := classifySwitchError(err)
	c.JSON(status, switchErrorEnvelope{Error: switchErrorBody{Code: code, Message: message, PaymentID: paymentID}})
}

func classifySwitchError(err error) (int, string, string) {
	switch {
	case errors.Is(err, paymentswitch.ErrNotFound):
		return http.StatusNotFound, "not_found", "payment not found"
	case errors.Is(err, paymentswitch.ErrIdempotencyConflict):
		return http.StatusConflict, "idempotency_conflict", "idempotency key was already used for a different request"
	case errors.Is(err, paymentswitch.ErrInvalidTransition):
		return http.StatusConflict, "invalid_state", err.Error()
	case errors.Is(err, paymentswitch.ErrConcurrentUpdate):
		return http.StatusConflict, "concurrent_update", "another request changed this payment first; retry"
	case errors.Is(err, paymentswitch.ErrAmountExceeds):
		return http.StatusUnprocessableEntity, "amount_exceeds", err.Error()
	case errors.Is(err, paymentswitch.ErrFeeRuleMissing):
		return http.StatusUnprocessableEntity, "no_fee_rule", "no fee rule prices this payment; configure one before accepting it"
	case errors.Is(err, paymentswitch.ErrPaymentRecord):
		return http.StatusUnprocessableEntity, "payment_record", "this payment cannot be priced: " + err.Error()
	case errors.Is(err, paymentswitch.ErrNoConnector):
		return http.StatusUnprocessableEntity, "no_connector", "no connector is enabled for this payment method"
	case errors.Is(err, paymentswitch.ErrInvalid), errors.Is(err, connectors.ErrInvalidRequest):
		return http.StatusBadRequest, "invalid_request", err.Error()
	case errors.Is(err, connectors.ErrWebhookSignature):
		return http.StatusUnauthorized, "webhook_signature", "webhook signature invalid"
	case errors.Is(err, connectors.ErrWebhookStale):
		return http.StatusUnauthorized, "webhook_stale", "webhook timestamp outside the accepted window"
	case errors.Is(err, paymentswitch.ErrAmountUnknown):
		return http.StatusBadGateway, "connector_amount_unknown", "connector reported money in without an amount; recorded for review"
	case errors.Is(err, connectors.ErrWebhookMalformed):
		return http.StatusBadRequest, "webhook_malformed", "webhook body malformed"
	case errors.Is(err, connectors.ErrUnknownConnector):
		return http.StatusNotFound, "unknown_connector", "connector not registered"
	case errors.Is(err, connectors.ErrUnsupported):
		return http.StatusUnprocessableEntity, "unsupported", err.Error()
	case errors.Is(err, paymentswitch.ErrUnmappedStatus):
		return http.StatusBadGateway, "connector_status_unmapped", "connector reported a status the switch does not know"
	default:
		return http.StatusServiceUnavailable, "unavailable", "payment service is temporarily unavailable"
	}
}
