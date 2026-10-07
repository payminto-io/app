// Package connectors is the slot module for payment processors: card, bank and chain.
// The switch (internal/paymentswitch) depends only on this file; providers live in subfolders.
// Design: .scratch/payments-v1/issues/05-switch-core.md, docs/architecture/MODULES.md.
package connectors

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// Code identifies a provider in configuration and in the status map ("mock", "chaindeposit", "stripe").
type Code string

// RawStatus is a status string exactly as the provider reports it; the switch maps it, never interprets it.
type RawStatus string

// Method is the payment method family a connector can take.
type Method string

const (
	MethodCard  Method = "card"
	MethodBank  Method = "bank"
	MethodChain Method = "chain"
)

// CaptureMethod says whether an authorization is captured immediately or by a later Capture call.
type CaptureMethod string

const (
	CaptureAutomatic CaptureMethod = "automatic"
	CaptureManual    CaptureMethod = "manual"
)

var (
	ErrUnsupported      = errors.New("connectors: operation not supported by this connector")
	ErrNotFound         = errors.New("connectors: transaction not known to the connector")
	ErrTimeout          = errors.New("connectors: no definitive answer from the connector; sync later")
	ErrInvalidRequest   = errors.New("connectors: invalid request")
	ErrWebhookSignature = errors.New("connectors: webhook signature invalid")
	ErrWebhookMalformed = errors.New("connectors: webhook body malformed")
	ErrUnknownConnector = errors.New("connectors: connector not registered")
)

// Money is an exact amount in one asset (ISO currency or token code). Never a float.
type Money struct {
	Amount decimal.Decimal
	Asset  string
}

// Capabilities declares what a provider can do. The switch refuses operations the provider cannot do
// before calling it, and verifies every RawStatus has a status-map entry at registration.
type Capabilities struct {
	Methods        []Method
	ManualCapture  bool
	PartialCapture bool
	Void           bool
	Refund         bool
	PartialRefund  bool
	Webhooks       bool
	Sync           bool
	// RawStatuses is every status this provider can report for payments; RawRefundStatuses for refunds.
	RawStatuses       []RawStatus
	RawRefundStatuses []RawStatus
}

// PaymentMethod is a token or reference, never raw card data (CLAUDE.md: no card data on these servers).
type PaymentMethod struct {
	Type  Method
	Token string
	// Details carries provider-specific, non-sensitive hints (chain code, asset code, wallet address).
	Details map[string]string
}

// NextAction tells the customer what to do before the payment can proceed.
type NextAction struct {
	Type string `json:"type"`
	// RedirectURL for 3DS or hosted pages; Address, Amount and Asset for a chain deposit.
	RedirectURL string `json:"redirect_url,omitempty"`
	Address     string `json:"address,omitempty"`
	Amount      string `json:"amount,omitempty"`
	Asset       string `json:"asset,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type AuthorizeRequest struct {
	AttemptID  string
	IntentID   string
	MerchantID string
	// PlatformID is the tenant the merchant acts under (Payminto external platform); chain deposits need it.
	PlatformID     string
	Money          Money
	CaptureMethod  CaptureMethod
	PaymentMethod  PaymentMethod
	ReturnURL      string
	Description    string
	IdempotencyKey string
	Metadata       map[string]string
}

type AuthorizeResponse struct {
	ConnectorTransactionID string
	RawStatus              RawStatus
	// AmountReceived is set when the provider reports an amount that differs from the request (chain deposits).
	AmountReceived *decimal.Decimal
	NextAction     *NextAction
	ErrorCode      string
	ErrorMessage   string
}

type CaptureRequest struct {
	AttemptID              string
	ConnectorTransactionID string
	Money                  Money
	IdempotencyKey         string
}

type CaptureResponse struct {
	ConnectorCaptureID string
	RawStatus          RawStatus
	AmountCaptured     decimal.Decimal
	ErrorCode          string
	ErrorMessage       string
}

type VoidRequest struct {
	AttemptID              string
	ConnectorTransactionID string
	Reason                 string
	IdempotencyKey         string
}

type VoidResponse struct {
	RawStatus    RawStatus
	ErrorCode    string
	ErrorMessage string
}

type RefundRequest struct {
	RefundID               string
	AttemptID              string
	ConnectorTransactionID string
	Money                  Money
	Reason                 string
	IdempotencyKey         string
}

type RefundResponse struct {
	ConnectorRefundID string
	RawStatus         RawStatus
	ErrorCode         string
	ErrorMessage      string
}

// SyncRequest carries our attempt id as well, because after ErrTimeout the switch holds no connector id.
type SyncRequest struct {
	AttemptID              string
	ConnectorTransactionID string
}

type SyncResponse struct {
	ConnectorTransactionID string
	RawStatus              RawStatus
	AmountCaptured         *decimal.Decimal
	AmountReceived         *decimal.Decimal
	NextAction             *NextAction
	ErrorCode              string
	ErrorMessage           string
}

// WebhookKind says which object a webhook event is about.
type WebhookKind string

const (
	WebhookPayment WebhookKind = "payment"
	WebhookRefund  WebhookKind = "refund"
)

// WebhookEvent is a verified, decoded provider notification.
type WebhookEvent struct {
	// EventID is the provider's unique delivery id. The switch persists (Code, EventID) and rejects a
	// second delivery with the same id, so a connector must never return an empty or derived id.
	EventID                string
	Kind                   WebhookKind
	ConnectorTransactionID string
	ConnectorRefundID      string
	RawStatus              RawStatus
	AmountCaptured         *decimal.Decimal
	AmountReceived         *decimal.Decimal
	OccurredAt             time.Time
}

// Connector is the port every provider implements. Operations the provider cannot do return ErrUnsupported.
// A call that reached the provider but got no definitive answer returns ErrTimeout and leaves the switch to Sync.
// VerifyWebhook checks authenticity (signature, timestamp window) and decodes the event; replay protection is the
// switch's job via WebhookEvent.EventID.
type Connector interface {
	Code() Code
	Capabilities() Capabilities
	Authorize(ctx context.Context, req AuthorizeRequest) (AuthorizeResponse, error)
	Capture(ctx context.Context, req CaptureRequest) (CaptureResponse, error)
	Void(ctx context.Context, req VoidRequest) (VoidResponse, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
	Sync(ctx context.Context, req SyncRequest) (SyncResponse, error)
	VerifyWebhook(ctx context.Context, headers http.Header, body []byte) (WebhookEvent, error)
}

// Lookup is the read side of the registry the switch depends on.
type Lookup interface {
	Get(code Code) (Connector, bool)
	Codes() []Code
}
