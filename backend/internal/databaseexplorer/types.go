package databaseexplorer

import "time"

// Dataset is an allowlisted, versioned read projection. It is not a table name.
type Dataset string

const DatasetPaymentLifecycle Dataset = "payment_lifecycle_v1"

type Role string

const (
	RoleMerchant Role = "merchant"
	RoleOperator Role = "operator"
)

type Actor struct {
	Role     Role
	TenantID string
}

type Field string

const (
	FieldPaymentReference          Field = "payment_reference"
	FieldInvoiceID                 Field = "invoice_id"
	FieldInvoiceState              Field = "invoice_state"
	FieldInvoiceAmountFiat         Field = "invoice_amount_fiat"
	FieldFiatCurrency              Field = "fiat_currency"
	FieldQuoteRequiredAtomicAmount Field = "quote_required_atomic_amount"
	FieldDepositAddress            Field = "deposit_address"
	FieldLifecycleStatus           Field = "lifecycle_status"
	FieldOutboxStatus              Field = "outbox_status"
	FieldIdempotencyFingerprint    Field = "idempotency_fingerprint"
	FieldCustomerEmail             Field = "customer_email"
	FieldCreatedAt                 Field = "created_at"
	FieldRecordID                  Field = "record_id"

	// FieldWebhookSecret is intentionally absent from every dataset definition.
	// It lets adapters prove that an accidentally supplied secret is discarded.
	FieldWebhookSecret Field = "webhook_secret"
)

type Classification string

const (
	ClassificationInternal  Classification = "internal"
	ClassificationFinancial Classification = "financial"
	ClassificationPII       Classification = "pii"
	ClassificationSecret    Classification = "secret"
)

// FieldValue preserves its source representation exactly. Money is never
// converted to float64; atomic and decimal amounts remain strings.
type FieldValue struct {
	Value          string
	Classification Classification
	Null           bool
	Redacted       bool
}

type Row map[Field]FieldValue

type Operator string

const (
	OperatorEqual              Operator = "equal"
	OperatorIn                 Operator = "in"
	OperatorGreaterThanOrEqual Operator = "greater_than_or_equal"
	OperatorLessThan           Operator = "less_than"
)

type Filter struct {
	Field    Field
	Operator Operator
	Value    string
	Values   []string
}

type SortDirection string

const (
	SortAscending  SortDirection = "ascending"
	SortDescending SortDirection = "descending"
)

type Sort struct {
	Field     Field
	Direction SortDirection
}

type Query struct {
	Dataset  Dataset
	Actor    Actor
	TenantID string
	Filters  []Filter
	Sort     Sort
	Cursor   string
	Limit    int
}

type Page struct {
	Rows []Row
	Meta PageMetadata
}

type PageMetadata struct {
	Dataset    Dataset
	Limit      int
	Returned   int
	HasMore    bool
	NextCursor string
	ReadAt     time.Time
	DataAsOf   time.Time
	Source     string
}

type Config struct {
	MaxRows      int
	QueryTimeout time.Duration
	Now          func() time.Time
}
