// Package paymentswitch is the switch core: payment intents, attempts, refunds, the status vocabulary
// and the mapping from each connector's raw statuses onto it. The Go keyword "switch" forces the name.
// Shape follows Hyperswitch's payment_intent / payment_attempt; design: .scratch/payments-v1/issues/05-switch-core.md.
package paymentswitch

import (
	"context"
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// IntentStatus is the merchant-facing state of a payment.
type IntentStatus string

const (
	IntentRequiresPaymentMethod IntentStatus = "requires_payment_method"
	IntentRequiresConfirmation  IntentStatus = "requires_confirmation"
	IntentRequiresAction        IntentStatus = "requires_action"
	IntentProcessing            IntentStatus = "processing"
	IntentRequiresCapture       IntentStatus = "requires_capture"
	IntentPartiallyCaptured     IntentStatus = "partially_captured"
	IntentSucceeded             IntentStatus = "succeeded"
	IntentFailed                IntentStatus = "failed"
	IntentCancelled             IntentStatus = "cancelled"
)

// AttemptStatus is one connector attempt's state; an intent derives its status from its active attempt.
type AttemptStatus string

const (
	AttemptStarted               AttemptStatus = "started"
	AttemptPending               AttemptStatus = "pending"
	AttemptAuthenticationPending AttemptStatus = "authentication_pending"
	AttemptAuthorized            AttemptStatus = "authorized"
	AttemptCaptureInitiated      AttemptStatus = "capture_initiated"
	AttemptCharged               AttemptStatus = "charged"
	AttemptPartialCharged        AttemptStatus = "partial_charged"
	AttemptPartiallyPaid         AttemptStatus = "partially_paid"
	AttemptOverpaid              AttemptStatus = "overpaid"
	AttemptCaptureFailed         AttemptStatus = "capture_failed"
	AttemptAuthorizationFailed   AttemptStatus = "authorization_failed"
	AttemptVoidInitiated         AttemptStatus = "void_initiated"
	AttemptVoided                AttemptStatus = "voided"
	AttemptVoidFailed            AttemptStatus = "void_failed"
	AttemptFailure               AttemptStatus = "failure"
)

// RefundStatus is one refund's state.
type RefundStatus string

const (
	RefundPending   RefundStatus = "pending"
	RefundSucceeded RefundStatus = "succeeded"
	RefundFailed    RefundStatus = "failed"
)

var (
	ErrInvalid             = errors.New("paymentswitch: invalid request")
	ErrNotFound            = errors.New("paymentswitch: not found")
	ErrInvalidTransition   = errors.New("paymentswitch: status transition not allowed")
	ErrIdempotencyConflict = errors.New("paymentswitch: idempotency key reused with a different request")
	ErrNoConnector         = errors.New("paymentswitch: no connector available for this payment")
	ErrUnmappedStatus      = errors.New("paymentswitch: connector status has no mapping")
	ErrWebhookReplay       = errors.New("paymentswitch: webhook event already processed")
	ErrAmountExceeds       = errors.New("paymentswitch: amount exceeds what is available")
	ErrConcurrentUpdate    = errors.New("paymentswitch: another request changed this payment first")
)

type Money = connectors.Money

// Intent is the merchant-facing payment record.
type Intent struct {
	ID                string
	MerchantID        string
	PlatformID        string
	IdempotencyKey    string
	Status            IntentStatus
	Money             Money
	AmountCaptured    decimal.Decimal
	AmountRefunded    decimal.Decimal
	CaptureMethod     connectors.CaptureMethod
	PaymentMethodType connectors.Method
	ConnectorCode     connectors.Code
	ActiveAttemptID   string
	Description       string
	ReturnURL         string
	Metadata          map[string]string
	NextAction        *connectors.NextAction
	LastErrorCode     string
	LastErrorMessage  string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Attempt is one try at one connector.
type Attempt struct {
	ID                     string
	IntentID               string
	MerchantID             string
	ConnectorCode          connectors.Code
	Status                 AttemptStatus
	RawStatus              connectors.RawStatus
	Money                  Money
	AmountCaptured         decimal.Decimal
	AmountReceived         decimal.Decimal
	ConnectorTransactionID string
	SelectionReason        string
	ErrorCode              string
	ErrorMessage           string
	NextAction             *connectors.NextAction
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Refund struct {
	ID                string
	IntentID          string
	AttemptID         string
	MerchantID        string
	ConnectorCode     connectors.Code
	IdempotencyKey    string
	Status            RefundStatus
	RawStatus         connectors.RawStatus
	Money             Money
	ConnectorRefundID string
	Reason            string
	ErrorCode         string
	ErrorMessage      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// View is what GET returns: the intent with its history.
type View struct {
	Intent   Intent
	Attempts []Attempt
	Refunds  []Refund
}

// SelectionRequest is what the switch hands the router (ticket 06) to pick a connector.
type SelectionRequest struct {
	MerchantID string
	Money      Money
	Method     connectors.Method
	Metadata   map[string]string
}

type Selection struct {
	Code   connectors.Code
	Reason string
}

// ConnectorSelector chooses the connector for an attempt. The default picks the merchant's first enabled
// connector that supports the method; ticket 06 plugs the routing engine in here.
type ConnectorSelector interface {
	Select(ctx context.Context, req SelectionRequest) (Selection, error)
}

// MerchantConnectors lists the connectors a merchant may use, in priority order.
type MerchantConnectors interface {
	EnabledConnectors(ctx context.Context, merchantID string) ([]connectors.Code, error)
}

// Ledger is the only money-recording port the switch uses; satisfied by *ledger.Service.
type Ledger interface {
	PostIn(ctx context.Context, tx *gorm.DB, j ledger.Journal) (ledger.Receipt, error)
}
