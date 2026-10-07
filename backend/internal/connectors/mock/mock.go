// Package mock is the scriptable connector every test and `docker compose up` runs without keys.
// The payment method token picks the authorize scenario and what later Capture/Void calls do; the refund
// reason picks the refund scenario. See the Scenario constants.
package mock

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

const Code connectors.Code = "mock"

// Authorize scenarios, chosen by PaymentMethod.Token. The capture_* and void_* ones authorize normally and
// script the first Capture or Void on that transaction; "landed" means the provider did it before failing to answer.
const (
	ScenarioSuccess            = "success"
	ScenarioDecline            = "decline"
	ScenarioRequiresAction     = "requires_action"
	ScenarioTimeout            = "timeout"
	ScenarioTimeoutThenFail    = "timeout_then_decline"
	ScenarioAsync              = "async"
	ScenarioCaptureTimeoutLand = "capture_timeout_landed"
	ScenarioCaptureTimeoutLost = "capture_timeout_lost"
	ScenarioCaptureErrorLand   = "capture_error_landed"
	ScenarioCaptureDeclined    = "capture_declined"
	ScenarioVoidTimeoutLand    = "void_timeout_landed"
	ScenarioVoidTimeoutLost    = "void_timeout_lost"
)

// Refund scenarios, chosen by RefundRequest.Reason.
const (
	ScenarioRefundAsync       = "async"
	ScenarioRefundFail        = "fail"
	ScenarioRefundTimeoutLand = "refund_timeout_landed"
	ScenarioRefundTimeoutLost = "refund_timeout_lost"
	ScenarioRefundErrorLand   = "refund_error_landed"
)

const (
	SignatureHeader = "X-Mock-Signature"
	TimestampHeader = "X-Mock-Timestamp"
	// WebhookWindow is how far a delivery's timestamp may sit from the clock.
	WebhookWindow  = 5 * time.Minute
	declineCode    = "do_not_honor"
	declineMessage = "The issuing bank declined the payment"
	defaultSecret  = "mock-webhook-secret"
)

var errProviderGlitch = errors.New("mock: provider returned 502 after processing")

// Raw statuses the mock reports; the switch maps them in paymentswitch/status_map.go.
const (
	StatusAuthorized        connectors.RawStatus = "authorized"
	StatusCaptured          connectors.RawStatus = "captured"
	StatusPartiallyCaptured connectors.RawStatus = "partially_captured"
	StatusDeclined          connectors.RawStatus = "declined"
	StatusActionRequired    connectors.RawStatus = "action_required"
	StatusPending           connectors.RawStatus = "pending"
	StatusVoided            connectors.RawStatus = "voided"
	StatusFailed            connectors.RawStatus = "failed"

	RefundPending connectors.RawStatus = "refund_pending"
	RefundDone    connectors.RawStatus = "refunded"
	RefundFailed  connectors.RawStatus = "refund_failed"
)

var (
	RawStatuses = []connectors.RawStatus{
		StatusAuthorized, StatusCaptured, StatusPartiallyCaptured, StatusDeclined,
		StatusActionRequired, StatusPending, StatusVoided, StatusFailed,
	}
	RawRefundStatuses = []connectors.RawStatus{RefundPending, RefundDone, RefundFailed}
)

type transaction struct {
	id            string
	status        connectors.RawStatus
	amount        decimal.Decimal
	captured      decimal.Decimal
	refunded      decimal.Decimal
	asset         string
	refunds       map[string]*refund
	captureScript string
	voidScript    string
}

type refund struct {
	id     string
	amount decimal.Decimal
	status connectors.RawStatus
}

type captureOutcome struct {
	resp connectors.CaptureResponse
	err  error
}

type voidOutcome struct {
	resp connectors.VoidResponse
	err  error
}

type refundOutcome struct {
	resp connectors.RefundResponse
	err  error
}

// Connector is safe for concurrent use.
type Connector struct {
	secret []byte
	now    func() time.Time

	mu        sync.Mutex
	txs       map[string]*transaction
	byAttempt map[string]string
	byRefund  map[string]string
	captures  map[string]captureOutcome
	voids     map[string]voidOutcome
	refunds   map[string]refundOutcome
	seq       int
	rseq      int
	calls     map[string]int
}

type Option func(*Connector)

func WithSecret(secret string) Option { return func(c *Connector) { c.secret = []byte(secret) } }
func WithClock(now func() time.Time) Option {
	return func(c *Connector) { c.now = now }
}

func New(opts ...Option) *Connector {
	c := &Connector{
		secret: []byte(defaultSecret), now: time.Now,
		txs: map[string]*transaction{}, byAttempt: map[string]string{}, byRefund: map[string]string{},
		captures: map[string]captureOutcome{}, voids: map[string]voidOutcome{}, refunds: map[string]refundOutcome{},
		calls: map[string]int{},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

func (c *Connector) Code() connectors.Code { return Code }

func (c *Connector) Capabilities() connectors.Capabilities {
	return connectors.Capabilities{
		Methods:           []connectors.Method{connectors.MethodCard, connectors.MethodBank},
		ManualCapture:     true,
		PartialCapture:    true,
		Void:              true,
		Refund:            true,
		PartialRefund:     true,
		Webhooks:          true,
		Sync:              true,
		RefundSync:        true,
		RawStatuses:       RawStatuses,
		RawRefundStatuses: RawRefundStatuses,
	}
}

// Calls counts provider calls per operation ("authorize", "capture", "void", "refund"); tests assert exactly-once.
func (c *Connector) Calls(op string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[op]
}

func (c *Connector) Authorize(_ context.Context, req connectors.AuthorizeRequest) (connectors.AuthorizeResponse, error) {
	if req.Money.Amount.IsNegative() || req.Money.Amount.IsZero() || req.Money.Asset == "" {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: amount and asset required", connectors.ErrInvalidRequest)
	}
	if !c.Capabilities().Supports(req.PaymentMethod.Type) {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: method %q", connectors.ErrUnsupported, req.PaymentMethod.Type)
	}
	scenario := req.PaymentMethod.Token
	if scenario == "" {
		scenario = ScenarioSuccess
	}
	settled := StatusCaptured
	if req.CaptureMethod == connectors.CaptureManual {
		settled = StatusAuthorized
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls["authorize"]++
	if existing, ok := c.byAttempt[req.AttemptID]; ok && req.AttemptID != "" {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: attempt %q already authorized as %s", connectors.ErrInvalidRequest, req.AttemptID, existing)
	}
	c.seq++
	tx := &transaction{
		id:      fmt.Sprintf("mock_tx_%d", c.seq),
		amount:  req.Money.Amount,
		asset:   req.Money.Asset,
		refunds: map[string]*refund{},
	}
	if req.AttemptID != "" {
		c.byAttempt[req.AttemptID] = tx.id
	}
	resp := connectors.AuthorizeResponse{ConnectorTransactionID: tx.id}
	switch scenario {
	case ScenarioSuccess, ScenarioCaptureTimeoutLand, ScenarioCaptureTimeoutLost, ScenarioCaptureErrorLand, ScenarioCaptureDeclined,
		ScenarioVoidTimeoutLand, ScenarioVoidTimeoutLost:
		tx.status = settled
		tx.captureScript, tx.voidScript = scenario, scenario
	case ScenarioDecline:
		tx.status = StatusDeclined
		resp.ErrorCode, resp.ErrorMessage = declineCode, declineMessage
	case ScenarioRequiresAction:
		tx.status = StatusActionRequired
		resp.NextAction = &connectors.NextAction{Type: "redirect", RedirectURL: "https://mock.invalid/3ds/" + tx.id}
	case ScenarioAsync:
		tx.status = StatusPending
	case ScenarioTimeout:
		tx.status = settled
		if settled == StatusCaptured {
			tx.captured = tx.amount
		}
		c.txs[tx.id] = tx
		return connectors.AuthorizeResponse{}, connectors.ErrTimeout
	case ScenarioTimeoutThenFail:
		tx.status = StatusDeclined
		c.txs[tx.id] = tx
		return connectors.AuthorizeResponse{}, connectors.ErrTimeout
	default:
		delete(c.byAttempt, req.AttemptID)
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: unknown mock scenario %q", connectors.ErrInvalidRequest, scenario)
	}
	if tx.status == StatusCaptured {
		tx.captured = tx.amount
	}
	c.txs[tx.id] = tx
	resp.RawStatus = tx.status
	return resp, nil
}

// unknown reports an error that leaves the outcome open: anything but a definitive refusal or a bad request.
func unknown(err error) bool {
	return err != nil && !connectors.Definitive(err) && !errors.Is(err, connectors.ErrInvalidRequest) && !errors.Is(err, connectors.ErrNotFound)
}

func (c *Connector) Capture(_ context.Context, req connectors.CaptureRequest) (connectors.CaptureResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls["capture"]++
	if req.IdempotencyKey != "" {
		if prior, ok := c.captures[req.IdempotencyKey]; ok {
			return prior.resp, prior.err
		}
	}
	out := c.capture(req)
	if req.IdempotencyKey != "" {
		// A provider answers a repeat with what it did, so a landed-but-unanswered call replays as its result.
		stored := out
		if unknown(out.err) {
			if tx, ok := c.txs[req.ConnectorTransactionID]; ok && (tx.status == StatusCaptured || tx.status == StatusPartiallyCaptured) {
				stored = captureOutcome{resp: connectors.CaptureResponse{ConnectorCaptureID: tx.id + "_cap", RawStatus: tx.status, AmountCaptured: tx.captured}}
			}
		}
		c.captures[req.IdempotencyKey] = stored
	}
	return out.resp, out.err
}

func (c *Connector) capture(req connectors.CaptureRequest) captureOutcome {
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return captureOutcome{err: connectors.ErrNotFound}
	}
	if tx.status != StatusAuthorized {
		return captureOutcome{err: fmt.Errorf("%w: capture needs an authorized transaction, status is %s", connectors.ErrInvalidRequest, tx.status)}
	}
	if req.Money.Asset != tx.asset || req.Money.Amount.IsZero() || req.Money.Amount.IsNegative() || req.Money.Amount.GreaterThan(tx.amount) {
		return captureOutcome{err: fmt.Errorf("%w: capture amount %s %s not within authorization", connectors.ErrInvalidRequest, req.Money.Amount, req.Money.Asset)}
	}
	script := tx.captureScript
	tx.captureScript = ""
	switch script {
	case ScenarioCaptureTimeoutLost:
		return captureOutcome{err: connectors.ErrTimeout}
	case ScenarioCaptureDeclined:
		return captureOutcome{err: fmt.Errorf("%w: capture refused by issuer", connectors.ErrDeclined)}
	}
	tx.captured = req.Money.Amount
	if req.Money.Amount.Equal(tx.amount) {
		tx.status = StatusCaptured
	} else {
		tx.status = StatusPartiallyCaptured
	}
	switch script {
	case ScenarioCaptureTimeoutLand:
		return captureOutcome{err: connectors.ErrTimeout}
	case ScenarioCaptureErrorLand:
		return captureOutcome{err: errProviderGlitch}
	}
	return captureOutcome{resp: connectors.CaptureResponse{ConnectorCaptureID: tx.id + "_cap", RawStatus: tx.status, AmountCaptured: tx.captured}}
}

func (c *Connector) Void(_ context.Context, req connectors.VoidRequest) (connectors.VoidResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls["void"]++
	if req.IdempotencyKey != "" {
		if prior, ok := c.voids[req.IdempotencyKey]; ok {
			return prior.resp, prior.err
		}
	}
	out := c.void(req)
	if req.IdempotencyKey != "" {
		stored := out
		if unknown(out.err) {
			if tx, ok := c.txs[req.ConnectorTransactionID]; ok && tx.status == StatusVoided {
				stored = voidOutcome{resp: connectors.VoidResponse{RawStatus: StatusVoided}}
			}
		}
		c.voids[req.IdempotencyKey] = stored
	}
	return out.resp, out.err
}

func (c *Connector) void(req connectors.VoidRequest) voidOutcome {
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return voidOutcome{err: connectors.ErrNotFound}
	}
	switch tx.status {
	case StatusAuthorized, StatusActionRequired, StatusPending:
	default:
		return voidOutcome{err: fmt.Errorf("%w: cannot void a transaction in status %s", connectors.ErrInvalidRequest, tx.status)}
	}
	script := tx.voidScript
	tx.voidScript = ""
	if script == ScenarioVoidTimeoutLost {
		return voidOutcome{err: connectors.ErrTimeout}
	}
	tx.status = StatusVoided
	if script == ScenarioVoidTimeoutLand {
		return voidOutcome{err: connectors.ErrTimeout}
	}
	return voidOutcome{resp: connectors.VoidResponse{RawStatus: StatusVoided}}
}

func (c *Connector) Refund(_ context.Context, req connectors.RefundRequest) (connectors.RefundResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls["refund"]++
	if req.IdempotencyKey != "" {
		if prior, ok := c.refunds[req.IdempotencyKey]; ok {
			return prior.resp, prior.err
		}
	}
	out := c.refund(req)
	if req.IdempotencyKey != "" {
		stored := out
		if unknown(out.err) {
			if id, ok := c.byRefund[req.RefundID]; ok {
				if tx, ok := c.txs[req.ConnectorTransactionID]; ok {
					if r, ok := tx.refunds[id]; ok {
						stored = refundOutcome{resp: connectors.RefundResponse{ConnectorRefundID: r.id, RawStatus: r.status}}
					}
				}
			}
		}
		c.refunds[req.IdempotencyKey] = stored
	}
	return out.resp, out.err
}

func (c *Connector) refund(req connectors.RefundRequest) refundOutcome {
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return refundOutcome{err: connectors.ErrNotFound}
	}
	if tx.status != StatusCaptured && tx.status != StatusPartiallyCaptured {
		return refundOutcome{err: fmt.Errorf("%w: refund needs captured funds, status is %s", connectors.ErrInvalidRequest, tx.status)}
	}
	if req.Money.Asset != tx.asset || req.Money.Amount.IsZero() || req.Money.Amount.IsNegative() || tx.refunded.Add(req.Money.Amount).GreaterThan(tx.captured) {
		return refundOutcome{err: fmt.Errorf("%w: refund %s %s exceeds captured balance", connectors.ErrInvalidRequest, req.Money.Amount, req.Money.Asset)}
	}
	scenario := strings.ToLower(req.Reason)
	switch scenario {
	case ScenarioRefundTimeoutLost:
		return refundOutcome{err: connectors.ErrTimeout}
	case ScenarioRefundFail:
		return refundOutcome{err: fmt.Errorf("%w: refund refused by issuer", connectors.ErrDeclined)}
	}
	c.rseq++
	r := &refund{id: fmt.Sprintf("mock_re_%d", c.rseq), amount: req.Money.Amount}
	if scenario == ScenarioRefundAsync {
		r.status = RefundPending
	} else {
		r.status = RefundDone
		tx.refunded = tx.refunded.Add(r.amount)
	}
	tx.refunds[r.id] = r
	if req.RefundID != "" {
		c.byRefund[req.RefundID] = r.id
	}
	switch scenario {
	case ScenarioRefundTimeoutLand:
		return refundOutcome{err: connectors.ErrTimeout}
	case ScenarioRefundErrorLand:
		return refundOutcome{err: errProviderGlitch}
	}
	return refundOutcome{resp: connectors.RefundResponse{ConnectorRefundID: r.id, RawStatus: r.status}}
}

func (c *Connector) Sync(_ context.Context, req connectors.SyncRequest) (connectors.SyncResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := req.ConnectorTransactionID
	if id == "" {
		id = c.byAttempt[req.AttemptID]
	}
	tx, ok := c.txs[id]
	if !ok {
		return connectors.SyncResponse{}, connectors.ErrNotFound
	}
	resp := connectors.SyncResponse{ConnectorTransactionID: tx.id, RawStatus: tx.status}
	if tx.status == StatusCaptured || tx.status == StatusPartiallyCaptured {
		captured := tx.captured
		resp.AmountCaptured = &captured
	}
	if tx.status == StatusActionRequired {
		resp.NextAction = &connectors.NextAction{Type: "redirect", RedirectURL: "https://mock.invalid/3ds/" + tx.id}
	}
	if tx.status == StatusDeclined {
		resp.ErrorCode, resp.ErrorMessage = declineCode, declineMessage
	}
	return resp, nil
}

func (c *Connector) SyncRefund(_ context.Context, req connectors.SyncRefundRequest) (connectors.SyncRefundResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := req.ConnectorRefundID
	if id == "" {
		id = c.byRefund[req.RefundID]
	}
	if id == "" {
		return connectors.SyncRefundResponse{}, connectors.ErrNotFound
	}
	for _, tx := range c.txs {
		if r, ok := tx.refunds[id]; ok {
			return connectors.SyncRefundResponse{ConnectorRefundID: r.id, RawStatus: r.status}, nil
		}
	}
	return connectors.SyncRefundResponse{}, connectors.ErrNotFound
}

// Event is the mock's webhook body; tests build one and sign it with SignWebhook.
type Event struct {
	EventID        string `json:"event_id"`
	Kind           string `json:"kind"`
	TransactionID  string `json:"transaction_id"`
	RefundID       string `json:"refund_id,omitempty"`
	Status         string `json:"status"`
	AmountCaptured string `json:"amount_captured,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

// SignWebhook returns the headers and body a real delivery would carry, stamped with the mock's clock.
func (c *Connector) SignWebhook(ev Event) (http.Header, []byte) {
	return c.SignWebhookAt(ev, c.now())
}

// SignWebhookAt signs as if delivered at the given time; tests use it for stale deliveries.
func (c *Connector) SignWebhookAt(ev Event, at time.Time) (http.Header, []byte) {
	body, _ := json.Marshal(ev)
	ts := strconv.FormatInt(at.Unix(), 10)
	h := http.Header{}
	h.Set(TimestampHeader, ts)
	h.Set(SignatureHeader, c.sign(ts, body))
	h.Set("Content-Type", "application/json")
	return h, body
}

func (c *Connector) sign(ts string, body []byte) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook checks the HMAC over timestamp and body and the timestamp window, decodes the event, and applies
// it to the mock's own state so a later Sync agrees.
func (c *Connector) VerifyWebhook(_ context.Context, headers http.Header, body []byte) (connectors.WebhookEvent, error) {
	sig := headers.Get(SignatureHeader)
	ts := headers.Get(TimestampHeader)
	if sig == "" || ts == "" || !hmac.Equal([]byte(sig), []byte(c.sign(ts, body))) {
		return connectors.WebhookEvent{}, connectors.ErrWebhookSignature
	}
	unix, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return connectors.WebhookEvent{}, connectors.ErrWebhookMalformed
	}
	if at := time.Unix(unix, 0); c.now().Sub(at).Abs() > WebhookWindow {
		return connectors.WebhookEvent{}, connectors.ErrWebhookStale
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil || ev.EventID == "" || ev.TransactionID == "" || ev.Status == "" {
		return connectors.WebhookEvent{}, connectors.ErrWebhookMalformed
	}
	out := connectors.WebhookEvent{
		EventID:                ev.EventID,
		Kind:                   connectors.WebhookKind(ev.Kind),
		ConnectorTransactionID: ev.TransactionID,
		ConnectorRefundID:      ev.RefundID,
		RawStatus:              connectors.RawStatus(ev.Status),
		OccurredAt:             time.Unix(unix, 0).UTC(),
	}
	if out.Kind == "" {
		out.Kind = connectors.WebhookPayment
	}
	if ev.AmountCaptured != "" {
		amt, err := decimal.NewFromString(ev.AmountCaptured)
		if err != nil {
			return connectors.WebhookEvent{}, connectors.ErrWebhookMalformed
		}
		out.AmountCaptured = &amt
	}
	if ev.OccurredAt != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, ev.OccurredAt); err == nil {
			out.OccurredAt = parsed
		}
	}
	c.apply(out)
	return out, nil
}

func (c *Connector) apply(ev connectors.WebhookEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[ev.ConnectorTransactionID]
	if !ok {
		return
	}
	switch ev.Kind {
	case connectors.WebhookRefund:
		if r, ok := tx.refunds[ev.ConnectorRefundID]; ok {
			if r.status != RefundDone && ev.RawStatus == RefundDone {
				tx.refunded = tx.refunded.Add(r.amount)
			}
			r.status = ev.RawStatus
		}
	default:
		tx.status = ev.RawStatus
		if ev.AmountCaptured != nil {
			tx.captured = *ev.AmountCaptured
		} else if ev.RawStatus == StatusCaptured && tx.captured.IsZero() {
			tx.captured = tx.amount
		}
	}
}

// Settle scripts the provider side: a customer finished 3DS, an async authorization landed.
func (c *Connector) Settle(connectorTransactionID string, status connectors.RawStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[connectorTransactionID]
	if !ok {
		return connectors.ErrNotFound
	}
	tx.status = status
	if status == StatusCaptured && tx.captured.IsZero() {
		tx.captured = tx.amount
	}
	return nil
}

// SettleRefund scripts a pending refund finishing on the provider side.
func (c *Connector) SettleRefund(connectorRefundID string, status connectors.RawStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, tx := range c.txs {
		if r, ok := tx.refunds[connectorRefundID]; ok {
			if r.status != RefundDone && status == RefundDone {
				tx.refunded = tx.refunded.Add(r.amount)
			}
			r.status = status
			return nil
		}
	}
	return connectors.ErrNotFound
}

// LastTransactionID returns the id Authorize assigned most recently; tests use it after a timeout scenario.
func (c *Connector) LastTransactionID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seq == 0 {
		return ""
	}
	return fmt.Sprintf("mock_tx_%d", c.seq)
}
