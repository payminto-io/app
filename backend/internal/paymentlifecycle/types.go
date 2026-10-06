package paymentlifecycle

import "time"

type TenantID string
type InvoiceID string
type QuoteID string
type ChainID string
type AssetID string
type AddressAssignmentID string
type LifecycleEventID string
type OutboxEventID string
type IdempotencyKey string
type MerchantReference string

type FiatAmount struct {
	Currency   string
	MinorUnits string
}

type PaymentMethod struct {
	ChainID ChainID
	AssetID AssetID
}

type OpenPayment struct {
	TenantID          TenantID
	IdempotencyKey    IdempotencyKey
	MerchantReference MerchantReference
	InvoiceAmount     FiatAmount
	PaymentMethod     PaymentMethod
	ExpiresAt         time.Time
}

type Quote struct {
	ID                  QuoteID
	InvoiceCurrency     string
	InvoiceMinorUnits   string
	ChainID             ChainID
	AssetID             AssetID
	RequiredAtomicUnits string
	AssetDecimals       uint8
	RateNumerator       string
	RateDenominator     string
	Source              string
	QuotedAt            time.Time
	ExpiresAt           time.Time
	Rounding            string
}

type InvoiceState string

const InvoiceOpen InvoiceState = "open"

type DepositAddress struct {
	AssignmentID AddressAssignmentID
	Address      string
}

type PaymentView struct {
	InvoiceID         InvoiceID
	TenantID          TenantID
	MerchantReference MerchantReference
	InvoiceAmount     FiatAmount
	PaymentMethod     PaymentMethod
	Quote             Quote
	DepositAddress    DepositAddress
	State             InvoiceState
	Revision          uint64
	OpenedAt          time.Time
	ExpiresAt         time.Time
}

type QuoteRequest struct {
	TenantID      TenantID
	InvoiceAmount FiatAmount
	PaymentMethod PaymentMethod
	InvoiceExpiry time.Time
	RequestedAt   time.Time
}

// OpenRecord is the complete validated input to one atomic persistence call.
// The Store adapter owns ID/address allocation and commits every Open effect.
type OpenRecord struct {
	Operation      string
	RequestHash    string
	Command        OpenPayment
	Quote          Quote
	State          InvoiceState
	Revision       uint64
	LifecycleEvent LifecycleEventSpec
	OutboxEvent    OutboxEventSpec
	OpenedAt       time.Time
}

type LifecycleEventSpec struct {
	Type       string
	Revision   uint64
	OccurredAt time.Time
}

type OutboxEventSpec struct {
	Type          string
	SchemaVersion uint16
	OccurredAt    time.Time
}

const (
	OperationOpen    = "payment_lifecycle.open"
	EventInvoiceOpen = "invoice.opened"
	RoundingCeil     = "ceil"
	RoundingFloor    = "floor"
	RoundingHalfUp   = "half_up"
	RoundingHalfEven = "half_even"
)

type StoreDisposition string

const (
	StoreCreated  StoreDisposition = "created"
	StoreReplayed StoreDisposition = "replayed"
)

type StoreResult struct {
	Disposition StoreDisposition
	View        PaymentView
}

type ReceiptLookup struct {
	TenantID       TenantID
	Operation      string
	IdempotencyKey IdempotencyKey
	RequestHash    string
}

type ReceiptDisposition string

const (
	ReceiptMiss     ReceiptDisposition = "miss"
	ReceiptReplayed ReceiptDisposition = "replayed"
)

type ReceiptResult struct {
	Disposition ReceiptDisposition
	View        PaymentView
}

type AvailableDepositAddress struct {
	TenantID TenantID
	ChainID  ChainID
	AssetID  AssetID
	Address  string
}

type MemoryStoreSnapshot struct {
	Invoices            int
	AddressAssignments  int
	LifecycleEvents     int
	OutboxEvents        int
	Receipts            int
	LifecycleEventSpecs []LifecycleEventSpec
	OutboxEventSpecs    []OutboxEventSpec
}
