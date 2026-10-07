// Package links owns the payment link: the six-step form's object, its lifecycle, the public render
// model and paying a link through a PaymentCreator (README.md).
package links

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

type Status string

const (
	StatusDraft    Status = "draft"
	StatusActive   Status = "active"
	StatusPaused   Status = "paused"
	StatusArchived Status = "archived"
)

// Published is true once a link has been live; amount, currency, methods and line items are then fixed.
func (s Status) Published() bool { return s == StatusActive || s == StatusPaused }

type AmountMode string

const (
	AmountFixed     AmountMode = "fixed"
	AmountCustomer  AmountMode = "customer"
	AmountLineItems AmountMode = "line_items"
)

// Environment is the process environment a link belongs to (internal/environment).
type Environment = environment.Environment

const (
	EnvLive = environment.Live
	EnvTest = environment.Test
)

type FieldMode string

const (
	FieldRequired FieldMode = "required"
	FieldOptional FieldMode = "optional"
	FieldHidden   FieldMode = "hidden"
)

type QuestionType string

const (
	QuestionText     QuestionType = "text"
	QuestionSelect   QuestionType = "select"
	QuestionCheckbox QuestionType = "checkbox"
)

const (
	CaptureAutomatic = "automatic"
	CaptureManual    = "manual"
	ThreeDSInherit   = "inherit"
	ThreeDSForce     = "force"
	SuccessMessage   = "message"
	SuccessRedirect  = "redirect"
	TimingCycle      = "cycle"
	TimingImmediate  = "immediate"
	SettleFiat       = "fiat"
	SettleCrypto     = "crypto"
)

// FieldRule says whether checkout asks for a customer field; Prefill is the merchant's value for it.
type FieldRule struct {
	Mode    FieldMode `json:"mode"`
	Prefill string    `json:"prefill,omitempty"`
}

type CustomerFieldPolicy struct {
	Name  FieldRule `json:"name"`
	Email FieldRule `json:"email"`
	Phone FieldRule `json:"phone"`
}

// MethodSpec is one way to pay. Fiat methods carry neither chain nor asset; crypto carries both.
type MethodSpec struct {
	Method fees.Method `json:"method"`
	Chain  string      `json:"chain,omitempty"`
	Asset  string      `json:"asset,omitempty"`
}

func (m MethodSpec) String() string {
	if m.Method == fees.MethodCrypto {
		return fmt.Sprintf("crypto:%s@%s", m.Asset, m.Chain)
	}
	return string(m.Method)
}

// FeeCurrency is the currency a fee rule for this method is written in: the asset for crypto, else the link's.
func (m MethodSpec) FeeCurrency(linkCurrency string) string {
	if m.Method == fees.MethodCrypto {
		return m.Asset
	}
	return linkCurrency
}

type LineItem struct {
	Name      string          `json:"name"`
	Quantity  int             `json:"quantity"`
	UnitPrice decimal.Decimal `json:"unit_price"`
	// TaxRate is in percent (18 means 18%).
	TaxRate decimal.Decimal `json:"tax_rate"`
}

type Question struct {
	Key      string       `json:"key"`
	Label    string       `json:"label"`
	Type     QuestionType `json:"type"`
	Options  []string     `json:"options,omitempty"`
	Required bool         `json:"required"`
	// PerOrder false asks once per customer email on this link; true asks on every payment.
	PerOrder bool `json:"per_order"`
}

type SettlementOverride struct {
	Kind          string `json:"kind"`
	DestinationID string `json:"destination_id"`
	Chain         string `json:"chain,omitempty"`
	Asset         string `json:"asset,omitempty"`
}

type Address struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// Input is everything the form edits; it is also the JSON body of create and update.
type Input struct {
	Title                string              `json:"title"`
	Description          string              `json:"description"`
	AmountMode           AmountMode          `json:"amount_mode"`
	Amount               *decimal.Decimal    `json:"amount"`
	AmountMin            *decimal.Decimal    `json:"amount_min"`
	AmountMax            *decimal.Decimal    `json:"amount_max"`
	Currency             string              `json:"currency"`
	ReferenceID          string              `json:"reference_id"`
	Metadata             map[string]string   `json:"metadata"`
	Category             string              `json:"category"`
	CustomerFields       CustomerFieldPolicy `json:"customer_field_policy"`
	BillingRequired      bool                `json:"billing_required"`
	ShippingRequired     bool                `json:"shipping_required"`
	MultiUse             bool                `json:"multi_use"`
	UseLimit             *int                `json:"use_limit"`
	Methods              []MethodSpec        `json:"methods"`
	CaptureMode          string              `json:"capture_mode"`
	ThreeDSPolicy        string              `json:"three_ds_policy"`
	ChainToleranceBps    int                 `json:"chain_tolerance_bps"`
	QuoteExpirySeconds   int                 `json:"quote_expiry_seconds"`
	FeeBearer            fees.FeeBearer      `json:"fee_bearer"`
	SuccessMode          string              `json:"success_mode"`
	SuccessURL           string              `json:"success_url"`
	SuccessMessage       string              `json:"success_message"`
	ReceiptEmail         bool                `json:"receipt_email"`
	ReceiptNote          string              `json:"receipt_note"`
	WebhookID            *uint               `json:"webhook_id"`
	FailureRetry         bool                `json:"failure_retry"`
	FailureMessage       string              `json:"failure_message"`
	SettlementOverride   *SettlementOverride `json:"settlement_override"`
	HoldInAsset          bool                `json:"hold_in_asset"`
	SettlementTiming     string              `json:"settlement_timing"`
	ExpiresAt            *time.Time          `json:"expires_at"`
	ExpiresAfterPayments *int                `json:"expires_after_payments"`
	LogoURL              string              `json:"logo_url"`
	AccentColor          string              `json:"accent_color"`
	Language             string              `json:"language"`
	LineItems            []LineItem          `json:"line_items"`
	Questions            []Question          `json:"questions"`
}

// DefaultInput is a new link's starting point; a create body overrides any of it.
func DefaultInput() Input {
	return Input{
		AmountMode:         AmountFixed,
		Metadata:           map[string]string{},
		CustomerFields:     CustomerFieldPolicy{Name: FieldRule{Mode: FieldOptional}, Email: FieldRule{Mode: FieldOptional}, Phone: FieldRule{Mode: FieldHidden}},
		Methods:            []MethodSpec{},
		CaptureMode:        CaptureAutomatic,
		ThreeDSPolicy:      ThreeDSInherit,
		QuoteExpirySeconds: 900,
		FeeBearer:          fees.BearerMerchant,
		SuccessMode:        SuccessMessage,
		FailureRetry:       true,
		SettlementTiming:   TimingCycle,
		Language:           "en",
		LineItems:          []LineItem{},
		Questions:          []Question{},
	}
}

// Link is one stored payment link.
type Link struct {
	Input
	// Total is the amount column: the fixed amount or the server's line-item sum; nil for customer-entered.
	Total              *decimal.Decimal
	ID                 string
	MemberID           uint
	ExternalPlatformID uint
	Environment        Environment
	Status             Status
	ShortCode          string
	UsesCount          int
	Revision           int
	PublishedAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// EffectiveUseLimit is the most payments the link may take; nil is unlimited.
func (l Link) EffectiveUseLimit() *int {
	if !l.MultiUse {
		one := 1
		return &one
	}
	var limit *int
	for _, v := range []*int{l.Input.UseLimit, l.ExpiresAfterPayments} {
		if v != nil && (limit == nil || *v < *limit) {
			n := *v
			limit = &n
		}
	}
	return limit
}

type Actor struct {
	MemberID   uint
	PlatformID uint
}

type ListFilter struct {
	Status Status
	Limit  int
	Offset int
}

// CustomerInput is what the payer typed; nil means the field was not sent.
type CustomerInput struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type PayRequest struct {
	IdempotencyKey  string            `json:"-"`
	Method          MethodSpec        `json:"method"`
	Amount          *decimal.Decimal  `json:"amount"`
	Customer        CustomerInput     `json:"customer"`
	BillingAddress  *Address          `json:"billing_address"`
	ShippingAddress *Address          `json:"shipping_address"`
	Answers         map[string]string `json:"answers"`
	// ClientIP is the payer's address as the trusted-proxy policy resolves it; only its hash is stored.
	ClientIP string `json:"-"`
}

// PayResult is the outcome of paying a link; replaying the idempotency key returns the same one.
type PayResult struct {
	PaymentReference   string
	Amount             decimal.Decimal
	Currency           string
	Fee                *decimal.Decimal
	Tax                *decimal.Decimal
	CustomerTotal      decimal.Decimal
	FeeBearer          fees.FeeBearer
	Method             MethodSpec
	CheckoutURL        string
	DepositAddress     string
	ExpiresAt          *time.Time
	SuccessRedirectURL string
	Replayed           bool
}

// Port is what the HTTP layer depends on.
type Port interface {
	Create(ctx context.Context, actor Actor, in Input) (Link, error)
	Get(ctx context.Context, platformID uint, id string) (Link, error)
	List(ctx context.Context, platformID uint, f ListFilter) ([]Link, int64, error)
	// Update saves in over the link if its revision is still `revision` (else link_conflict).
	Update(ctx context.Context, platformID uint, id string, revision int, in Input) (Link, error)
	Delete(ctx context.Context, platformID uint, id string) error
	Publish(ctx context.Context, platformID uint, id string) (Link, error)
	Pause(ctx context.Context, platformID uint, id string) (Link, error)
	Archive(ctx context.Context, platformID uint, id string) (Link, error)
	Duplicate(ctx context.Context, actor Actor, id string) (Link, error)
	// FeePreview prices every method of the link for the merchant: connector, rule and breakdown.
	FeePreview(ctx context.Context, l Link) []MethodPreview
	Render(ctx context.Context, shortCode string) (RenderModel, error)
	// Preview renders an unsaved form exactly as the public endpoint would, without storing it.
	Preview(ctx context.Context, platformID uint, in Input, linkID string) (PreviewResult, error)
	// Options lists the methods and currencies the form may offer in the process environment.
	Options(ctx context.Context) (Options, error)
	MerchantName(ctx context.Context, platformID uint) (string, error)
	Pay(ctx context.Context, shortCode string, req PayRequest) (PayResult, error)
	URL(shortCode string) string
}

// FeeQuoter is the slice of fees.Port links needs: rule existence and previews, never postings.
type FeeQuoter interface {
	Resolve(ctx context.Context, q fees.Query) (fees.Rule, error)
	Preview(ctx context.Context, req fees.PreviewRequest) (fees.Breakdown, error)
}

// PaymentCreator turns a paid link into a payment. The default calls Payminto's payment service; the switch (ticket 05) replaces it.
type PaymentCreator interface {
	// Connectors names the connectors that can take m for currency in env; empty means unavailable, "" is unscoped.
	Connectors(ctx context.Context, env Environment, currency string, m MethodSpec) ([]string, error)
	// CreatePayment is idempotent on req.LinkPaymentID, which must be the payment's unique reference: a second call
	// returns the payment the first made, and a call after FencePayment fails with ErrNotCreated.
	// Only an error wrapping ErrNotCreated, or a *Error, promises that no payment exists; any other error is ambiguous.
	CreatePayment(ctx context.Context, req PaymentRequest) (CreatedPayment, error)
	// FencePayment settles a use whose outcome is unknown: it returns the live payment for req.LinkPaymentID
	// (found), or atomically makes one impossible to create from then on (found=false). An error leaves both open.
	FencePayment(ctx context.Context, req PaymentRequest) (created CreatedPayment, found bool, err error)
	// CancelPayment cancels the payment of a use the link no longer tracks; nil when there is none.
	CancelPayment(ctx context.Context, linkPaymentID string) error
	// OpenPayments reports which of these uses' payments are still open and unpaid (not paid, cancelled or expired).
	OpenPayments(ctx context.Context, linkPaymentIDs []string) (map[string]bool, error)
}

// ErrNotCreated is what a PaymentCreator wraps when it is certain no payment exists; only then is a use released.
var ErrNotCreated = errors.New("links: payment definitively not created")

// PaymentRequest is one reserved use of a link; LinkPaymentID is unique and stable across retries of the reservation.
type PaymentRequest struct {
	LinkID        string
	LinkPaymentID string
	MemberID      uint
	PlatformID    uint
	Environment   Environment
	Method        MethodSpec
	Connector     string
	Amount        decimal.Decimal
	Currency      string
	CustomerTotal decimal.Decimal
	FeeBearer     fees.FeeBearer
	// FeeRuleID and FeeRuleVersion are the rule the link priced this payment under; zero when none applied.
	FeeRuleID          uint
	FeeRuleVersion     int
	CustomerName       string
	CustomerEmail      string
	CustomerPhone      string
	BillingAddress     *Address
	ShippingAddress    *Address
	ReferenceID        string
	Metadata           map[string]string
	CaptureMode        string
	ThreeDSPolicy      string
	ChainToleranceBps  int
	QuoteExpirySeconds int
	WebhookID          *uint
}

type CreatedPayment struct {
	Reference      string     `json:"reference"`
	CheckoutURL    string     `json:"checkout_url,omitempty"`
	DepositAddress string     `json:"deposit_address,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

// DestinationVerifier answers whether a settlement destination is verified for the merchant (ticket 11 provides one).
type DestinationVerifier interface {
	Verified(ctx context.Context, platformID uint, o SettlementOverride) (bool, error)
}

// NoDestinations is the verifier until settlement destinations exist: nothing is verified, so overrides cannot publish.
type NoDestinations struct{}

func (NoDestinations) Verified(context.Context, uint, SettlementOverride) (bool, error) {
	return false, nil
}

// Code is a typed error code; the HTTP layer maps it to a status (routes_links.go).
type Code string

const (
	CodeInvalidRequest          Code = "invalid_request"
	CodeTitleRequired           Code = "title_required"
	CodeTitleTooLong            Code = "title_too_long"
	CodeDescriptionTooLong      Code = "description_too_long"
	CodeAmountModeInvalid       Code = "amount_mode_invalid"
	CodeAmountRequired          Code = "amount_required"
	CodeAmountNotAllowed        Code = "amount_not_allowed"
	CodeAmountInvalid           Code = "amount_invalid"
	CodeAmountBoundsNotAllowed  Code = "amount_bounds_not_allowed"
	CodeAmountRangeInvalid      Code = "amount_range_invalid"
	CodeAmountOutOfRange        Code = "amount_out_of_range"
	CodeLineItemsRequired       Code = "line_items_required"
	CodeLineItemsNotAllowed     Code = "line_items_not_allowed"
	CodeLineItemInvalid         Code = "line_item_invalid"
	CodeCurrencyRequired        Code = "currency_required"
	CodeCurrencyUnsupported     Code = "currency_unsupported"
	CodeReferenceTooLong        Code = "reference_id_too_long"
	CodeCategoryTooLong         Code = "category_too_long"
	CodeMetadataInvalid         Code = "metadata_invalid"
	CodeCustomerPolicyInvalid   Code = "customer_field_policy_invalid"
	CodeQuestionInvalid         Code = "question_invalid"
	CodeQuestionKeyDuplicate    Code = "question_key_duplicate"
	CodeUseLimitInvalid         Code = "use_limit_invalid"
	CodeUseLimitNeedsMultiUse   Code = "use_limit_requires_multi_use"
	CodeMethodsRequired         Code = "methods_required"
	CodeMethodUnknown           Code = "method_unknown"
	CodeMethodInvalid           Code = "method_invalid"
	CodeMethodChainRequired     Code = "method_chain_required"
	CodeMethodAssetUnsupported  Code = "method_asset_unsupported"
	CodeMethodDuplicate         Code = "method_duplicate"
	CodeMethodNoConnector       Code = "method_no_connector"
	CodeMethodNoFeeRule         Code = "method_no_fee_rule"
	CodeFeeRuleAmbiguous        Code = "fee_rule_ambiguous"
	CodeSurchargeForbidden      Code = "surcharge_forbidden"
	CodeSurchargeNeedsQuote     Code = "surcharge_needs_quote"
	CodeFeeExceedsAmount        Code = "fee_exceeds_amount"
	CodeCaptureModeInvalid      Code = "capture_mode_invalid"
	CodeCaptureModeNeedsCard    Code = "capture_mode_requires_card"
	CodeThreeDSInvalid          Code = "three_ds_policy_invalid"
	CodeChainToleranceInvalid   Code = "chain_tolerance_invalid"
	CodeQuoteExpiryInvalid      Code = "quote_expiry_invalid"
	CodeFeeBearerInvalid        Code = "fee_bearer_invalid"
	CodeSuccessModeInvalid      Code = "success_mode_invalid"
	CodeSuccessURLInvalid       Code = "success_url_invalid"
	CodeSuccessURLRequired      Code = "success_url_required"
	CodeSuccessMessageRequired  Code = "success_message_required"
	CodeTextTooLong             Code = "text_too_long"
	CodeReceiptNeedsEmail       Code = "receipt_requires_email"
	CodeWebhookNotFound         Code = "webhook_not_found"
	CodeSettlementInvalid       Code = "settlement_override_invalid"
	CodeDestinationUnverified   Code = "settlement_destination_unverified"
	CodeHoldInAssetNeedsCrypto  Code = "hold_in_asset_requires_crypto"
	CodeSettlementTimingInvalid Code = "settlement_timing_invalid"
	CodeExpiresAtInPast         Code = "expires_at_in_past"
	CodeLogoURLInvalid          Code = "logo_url_invalid"
	CodeAccentColorInvalid      Code = "accent_color_invalid"
	CodeLanguageInvalid         Code = "language_invalid"
	CodeNotFound                Code = "link_not_found"
	CodeInvalidTransition       Code = "invalid_transition"
	CodePublishedImmutable      Code = "link_published_immutable"
	CodeNotEditable             Code = "link_not_editable"
	CodeNotDeletable            Code = "link_not_deletable"
	CodeConflict                Code = "link_conflict"
	CodeArchived                Code = "link_archived"
	CodePaused                  Code = "link_paused"
	CodeExpired                 Code = "link_expired"
	CodeUseLimitReached         Code = "link_use_limit_reached"
	CodeMethodNotEnabled        Code = "method_not_enabled"
	CodeMethodUnavailable       Code = "method_unavailable"
	CodeCustomerFieldRequired   Code = "customer_field_required"
	CodeCustomerFieldHidden     Code = "customer_field_hidden"
	CodeCustomerEmailInvalid    Code = "customer_email_invalid"
	CodeCustomerPhoneInvalid    Code = "customer_phone_invalid"
	CodeCustomerNameInvalid     Code = "customer_name_invalid"
	CodeAddressRequired         Code = "address_required"
	CodeAddressInvalid          Code = "address_invalid"
	CodeAnswerRequired          Code = "answer_required"
	CodeAnswerUnknownQuestion   Code = "answer_unknown_question"
	CodeAnswerInvalid           Code = "answer_invalid"
	CodeIdempotencyKeyRequired  Code = "idempotency_key_required"
	CodeIdempotencyKeyReused    Code = "idempotency_key_reused"
	CodePaymentInProgress       Code = "payment_in_progress"
	CodePaymentCreationFailed   Code = "payment_creation_failed"
	CodeShortCodeExhausted      Code = "short_code_exhausted"
	CodeUseLimitBelowUses       Code = "use_limit_below_uses"
	CodeEnvironmentMismatch     Code = "link_environment_mismatch"
	CodeOpenPaymentsLimit       Code = "open_payments_limit"
)

// Error is a typed refusal naming the field it concerns; Errors carries every refusal when there are several.
type Error struct {
	Code    Code
	Field   string
	Message string
	Errors  []*Error
	// RetryAfter is set on payment_in_progress: when the reservation's lease ends and the retry can resolve it.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("links: %s (%s): %s", e.Code, e.Field, e.Message)
	}
	return fmt.Sprintf("links: %s: %s", e.Code, e.Message)
}

func newErr(code Code, field, format string, args ...any) *Error {
	return &Error{Code: code, Field: field, Message: fmt.Sprintf(format, args...)}
}

// joinErrs returns nil for none, the error itself for one, and the first carrying all of them otherwise.
func joinErrs(errs []*Error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	first := *errs[0]
	first.Errors = errs
	return &first
}

// CodeOf returns the typed code of err, or "" when err is not a links error.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// Codes lists every code in err, for tests and multi-error responses.
func Codes(err error) []Code {
	var e *Error
	if !errors.As(err, &e) {
		return nil
	}
	if len(e.Errors) == 0 {
		return []Code{e.Code}
	}
	out := make([]Code, len(e.Errors))
	for i, x := range e.Errors {
		out[i] = x.Code
	}
	return out
}

func normCode(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
