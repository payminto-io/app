// Package paymentswitch is the switch core: payment intents, attempts, refunds, the status vocabulary
// and the mapping from each connector's raw statuses onto it. The Go keyword "switch" forces the name.
// Shape follows Hyperswitch's payment_intent / payment_attempt; design: .scratch/payments-v1/issues/05-switch-core.md.
package paymentswitch

import (
	"context"
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/environment"
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
	// IntentPartiallyPaid is terminal: a chain deposit closed short; the received funds are on the books and await refund (ticket 11).
	IntentPartiallyPaid IntentStatus = "partially_paid"
	IntentSucceeded     IntentStatus = "succeeded"
	IntentFailed        IntentStatus = "failed"
	IntentCancelled     IntentStatus = "cancelled"
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
	AttemptUnderpaid             AttemptStatus = "underpaid"
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
	// RefundInitiated is the claim: the connector call is in flight or its outcome is unknown; Sync resolves it.
	RefundInitiated RefundStatus = "initiated"
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
	ErrOutcomeUnknown      = errors.New("paymentswitch: connector outcome unknown; the operation is in flight and will be synced")
	ErrAmountUnknown       = errors.New("paymentswitch: connector reported money in without an amount and none was claimed")
	ErrFeeRuleMissing      = errors.New("paymentswitch: no fee rule prices this payment")
	ErrPaymentRecord       = errors.New("paymentswitch: could not open the payment record this intent is priced against")
)

// Error codes exposed to merchants; raw connector and backend errors go to the audit trail and logs (M7).
const (
	ErrorCodeDeclined     = "declined"
	ErrorCodeConnector    = "connector_error"
	ErrorCodeTimeout      = "connector_timeout"
	ErrorCodeNotFound     = "not_found_at_connector"
	ErrorCodeAmountUnkown = "amount_unknown"
)

// Anomaly kinds: facts the switch recorded rather than acted on, each needing an operator's eye.
const (
	AnomalyEvidenceAfterTerminal = "evidence_after_terminal"
	AnomalyStaleInFlight         = "stale_in_flight"
	AnomalyAmountUnknown         = "amount_unknown"
	AnomalyUnmappedStatus        = "unmapped_status"
	// AnomalyLateReceipt: funds arrived on a chain address after the attempt closed; booked to unallocated receipts.
	AnomalyLateReceipt = "late_receipt"
	// AnomalyReceivedDecreased: a connector reported a cumulative below the watermark; nothing was changed.
	AnomalyReceivedDecreased = "received_decreased"
	// AnomalyFeeAlreadyPosted: fees refused a second successful attempt on one payment; the money was still booked.
	AnomalyFeeAlreadyPosted = "fee_already_posted"
)

// UnallocatedReceiptsOwner is the platform liability that holds late money until an operator refunds or applies it.
const UnallocatedReceiptsOwner = "unallocated_receipts"

// Domain events, named and versioned (MODULES.md rule 9).
const (
	EventPaymentSucceeded = "switch.payment.succeeded.v1"
	EventPaymentFailed    = "switch.payment.failed.v1"
	EventRefundSucceeded  = "switch.refund.succeeded.v1"
)

type Event struct {
	Type       string
	MerchantID string
	IntentID   string
	AttemptID  string
	RefundID   string
	Payload    map[string]any
}

// Events is where the switch publishes domain events after commit; wired to the existing event emitter.
type Events interface {
	Emit(ctx context.Context, ev Event) error
}

// NoEvents drops events; tests and tools that do not care use it.
type NoEvents struct{}

func (NoEvents) Emit(context.Context, Event) error { return nil }

type Money = connectors.Money

// Intent is the merchant-facing payment record.
type Intent struct {
	ID                string
	MerchantID        string
	PlatformID        string
	Environment       environment.Environment
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
	ID              string
	IntentID        string
	MerchantID      string
	Environment     environment.Environment
	ConnectorCode   connectors.Code
	Status          AttemptStatus
	RawStatus       connectors.RawStatus
	Money           Money
	AmountToCapture decimal.Decimal
	AmountCaptured  decimal.Decimal
	// AmountReceived and ReceivedAsset are set only when a connector reported funds arriving (chain deposits).
	AmountReceived         *decimal.Decimal
	ReceivedAsset          string
	ConnectorTransactionID string
	SelectionReason        string
	ErrorCode              string
	ErrorMessage           string
	NextAction             *connectors.NextAction
	ClaimedUntil           *time.Time
	StatusChangedAt        time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Refund struct {
	ID                string
	IntentID          string
	AttemptID         string
	MerchantID        string
	Environment       environment.Environment
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

// Anomaly is a recorded contradiction or stall; it is never turned into a status.
type Anomaly struct {
	ID        uint
	Entity    string
	EntityID  string
	IntentID  string
	Kind      string
	Detail    string
	CreatedAt time.Time
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

// FeeRef names the attempt and the Payminto payment record the fees module prices it against.
type FeeRef struct {
	PaymentRecordID uint
	AttemptID       string
}

// FeeQuery is what fees needs to resolve a rule; Method is the fees vocabulary (card, bank, upi, crypto).
type FeeQuery struct {
	Method    string
	Connector string
	Currency  string
	Chain     string
}

// Fees is the fee-rules port (internal/fees, ticket 02): Snapshot in the attempt's creating transaction, PostFee in
// the transaction that books the money. ErrFeeRuleMissing when no rule matches; ErrFeeAlreadyPosted when the
// payment already carries a fee from another attempt (a duplicate success to reconcile, not a retry).
type Fees interface {
	Snapshot(ctx context.Context, tx *gorm.DB, ref FeeRef, q FeeQuery) error
	PostFee(ctx context.Context, tx *gorm.DB, ref FeeRef, captured decimal.Decimal) error
}

var ErrFeeAlreadyPosted = errors.New("paymentswitch: a fee is already posted for this payment by another attempt")

// PaymentRecord is the Payminto payment_requests row an intent is priced against (fees reads the merchant there).
type PaymentRecord struct {
	IntentID    string
	MerchantID  string
	PlatformID  string
	Money       Money
	Description string
}

// PaymentRecords opens that row in the intent's creating transaction and returns its id.
type PaymentRecords interface {
	Open(ctx context.Context, tx *gorm.DB, rec PaymentRecord) (uint, error)
}
