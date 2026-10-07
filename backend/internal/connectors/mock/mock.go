// Package mock is the scriptable connector every test and `docker compose up` runs without keys.
// The payment method token picks the scenario; see Scenario constants.
package mock

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

const Code connectors.Code = "mock"

// Scenarios, chosen by PaymentMethod.Token.
const (
	ScenarioSuccess         = "success"
	ScenarioDecline         = "decline"
	ScenarioRequiresAction  = "requires_action"
	ScenarioTimeout         = "timeout"
	ScenarioTimeoutThenFail = "timeout_then_decline"
	ScenarioAsync           = "async"
	ScenarioRefundAsync     = "async"
	ScenarioRefundFail      = "fail"
	SignatureHeader         = "X-Mock-Signature"
	declineCode             = "do_not_honor"
	declineMessage          = "The issuing bank declined the payment"
	defaultSecret           = "mock-webhook-secret"
)

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
	captureMethod connectors.CaptureMethod
	refunds       map[string]*refund
}

type refund struct {
	id     string
	amount decimal.Decimal
	status connectors.RawStatus
}

// Connector is safe for concurrent use.
type Connector struct {
	secret []byte
	now    func() time.Time

	mu   sync.Mutex
	txs  map[string]*transaction
	seq  int
	rseq int
}

type Option func(*Connector)

func WithSecret(secret string) Option { return func(c *Connector) { c.secret = []byte(secret) } }
func WithClock(now func() time.Time) Option {
	return func(c *Connector) { c.now = now }
}

func New(opts ...Option) *Connector {
	c := &Connector{secret: []byte(defaultSecret), now: time.Now, txs: map[string]*transaction{}}
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
		RawStatuses:       RawStatuses,
		RawRefundStatuses: RawRefundStatuses,
	}
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
	c.seq++
	tx := &transaction{
		id:            fmt.Sprintf("mock_tx_%d", c.seq),
		amount:        req.Money.Amount,
		asset:         req.Money.Asset,
		captureMethod: req.CaptureMethod,
		refunds:       map[string]*refund{},
	}
	resp := connectors.AuthorizeResponse{ConnectorTransactionID: tx.id}
	switch scenario {
	case ScenarioSuccess:
		tx.status = settled
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
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: unknown mock scenario %q", connectors.ErrInvalidRequest, scenario)
	}
	if tx.status == StatusCaptured {
		tx.captured = tx.amount
	}
	c.txs[tx.id] = tx
	resp.RawStatus = tx.status
	return resp, nil
}

func (c *Connector) Capture(_ context.Context, req connectors.CaptureRequest) (connectors.CaptureResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return connectors.CaptureResponse{}, connectors.ErrNotFound
	}
	if tx.status != StatusAuthorized {
		return connectors.CaptureResponse{}, fmt.Errorf("%w: capture needs an authorized transaction, status is %s", connectors.ErrInvalidRequest, tx.status)
	}
	if req.Money.Asset != tx.asset || req.Money.Amount.IsZero() || req.Money.Amount.IsNegative() || req.Money.Amount.GreaterThan(tx.amount) {
		return connectors.CaptureResponse{}, fmt.Errorf("%w: capture amount %s %s not within authorization", connectors.ErrInvalidRequest, req.Money.Amount, req.Money.Asset)
	}
	tx.captured = req.Money.Amount
	if req.Money.Amount.Equal(tx.amount) {
		tx.status = StatusCaptured
	} else {
		tx.status = StatusPartiallyCaptured
	}
	return connectors.CaptureResponse{ConnectorCaptureID: tx.id + "_cap", RawStatus: tx.status, AmountCaptured: tx.captured}, nil
}

func (c *Connector) Void(_ context.Context, req connectors.VoidRequest) (connectors.VoidResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return connectors.VoidResponse{}, connectors.ErrNotFound
	}
	switch tx.status {
	case StatusAuthorized, StatusActionRequired, StatusPending:
		tx.status = StatusVoided
		return connectors.VoidResponse{RawStatus: StatusVoided}, nil
	default:
		return connectors.VoidResponse{}, fmt.Errorf("%w: cannot void a transaction in status %s", connectors.ErrInvalidRequest, tx.status)
	}
}

func (c *Connector) Refund(_ context.Context, req connectors.RefundRequest) (connectors.RefundResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return connectors.RefundResponse{}, connectors.ErrNotFound
	}
	if tx.status != StatusCaptured && tx.status != StatusPartiallyCaptured {
		return connectors.RefundResponse{}, fmt.Errorf("%w: refund needs captured funds, status is %s", connectors.ErrInvalidRequest, tx.status)
	}
	if req.Money.Asset != tx.asset || req.Money.Amount.IsZero() || req.Money.Amount.IsNegative() || tx.refunded.Add(req.Money.Amount).GreaterThan(tx.captured) {
		return connectors.RefundResponse{}, fmt.Errorf("%w: refund %s %s exceeds captured balance", connectors.ErrInvalidRequest, req.Money.Amount, req.Money.Asset)
	}
	c.rseq++
	r := &refund{id: fmt.Sprintf("mock_re_%d", c.rseq), amount: req.Money.Amount}
	switch strings.ToLower(req.Reason) {
	case ScenarioRefundAsync:
		r.status = RefundPending
	case ScenarioRefundFail:
		r.status = RefundFailed
	default:
		r.status = RefundDone
		tx.refunded = tx.refunded.Add(r.amount)
	}
	tx.refunds[r.id] = r
	return connectors.RefundResponse{ConnectorRefundID: r.id, RawStatus: r.status}, nil
}

func (c *Connector) Sync(_ context.Context, req connectors.SyncRequest) (connectors.SyncResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, ok := c.txs[req.ConnectorTransactionID]
	if !ok {
		return connectors.SyncResponse{}, connectors.ErrNotFound
	}
	resp := connectors.SyncResponse{RawStatus: tx.status}
	if tx.status == StatusCaptured || tx.status == StatusPartiallyCaptured {
		captured := tx.captured
		resp.AmountCaptured = &captured
	}
	if tx.status == StatusDeclined {
		resp.ErrorCode, resp.ErrorMessage = declineCode, declineMessage
	}
	return resp, nil
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

// SignWebhook returns the headers and body a real delivery would carry.
func (c *Connector) SignWebhook(ev Event) (http.Header, []byte) {
	body, _ := json.Marshal(ev)
	h := http.Header{}
	h.Set(SignatureHeader, c.sign(body))
	h.Set("Content-Type", "application/json")
	return h, body
}

func (c *Connector) sign(body []byte) string {
	mac := hmac.New(sha256.New, c.secret)
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhook checks the HMAC, decodes the event and applies it to the mock's own state so a later Sync agrees.
func (c *Connector) VerifyWebhook(_ context.Context, headers http.Header, body []byte) (connectors.WebhookEvent, error) {
	sig := headers.Get(SignatureHeader)
	if sig == "" || !hmac.Equal([]byte(sig), []byte(c.sign(body))) {
		return connectors.WebhookEvent{}, connectors.ErrWebhookSignature
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
		OccurredAt:             c.now(),
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
		if ts, err := time.Parse(time.RFC3339Nano, ev.OccurredAt); err == nil {
			out.OccurredAt = ts
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
		} else if ev.RawStatus == StatusCaptured {
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
	if status == StatusCaptured {
		tx.captured = tx.amount
	}
	return nil
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
