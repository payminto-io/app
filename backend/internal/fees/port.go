// Package fees owns versioned fee rules: resolution, computation, preview and the
// per-payment snapshot. Rules are never mutated; an edit is version n+1 (README.md).
package fees

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Method string

const (
	MethodCard   Method = "card"
	MethodUPI    Method = "upi"
	MethodBank   Method = "bank"
	MethodCrypto Method = "crypto"
)

var Methods = []Method{MethodCard, MethodUPI, MethodBank, MethodCrypto}

type CardType string

const (
	CardCredit  CardType = "credit"
	CardDebit   CardType = "debit"
	CardPrepaid CardType = "prepaid"
)

var CardTypes = []CardType{CardCredit, CardDebit, CardPrepaid}

// FeeBearer says who pays the fee: the customer on top of the amount (a surcharge) or the merchant out of it.
type FeeBearer string

const (
	BearerMerchant FeeBearer = "merchant"
	BearerCustomer FeeBearer = "customer"
)

// Specificity ranks a rule's scope; Resolve picks the highest active match.
type Specificity int

const (
	SpecMethodDefault Specificity = iota
	SpecConnector
	SpecConnectorCardType
	SpecConnectorCardTypeRegion
)

// Scope is the identity of a lineage: every version of a rule shares it.
type Scope struct {
	Method    Method
	Connector *string
	CardType  *CardType
	Region    *string
	Currency  string
}

func (s Scope) Specificity() Specificity {
	switch {
	case s.Region != nil:
		return SpecConnectorCardTypeRegion
	case s.CardType != nil:
		return SpecConnectorCardType
	case s.Connector != nil:
		return SpecConnector
	default:
		return SpecMethodDefault
	}
}

// Slab prices amounts in (previous UpTo, UpTo]; a nil UpTo is the open-ended last slab.
type Slab struct {
	UpTo    *decimal.Decimal `json:"up_to"`
	Percent decimal.Decimal  `json:"percent"`
	Flat    decimal.Decimal  `json:"flat"`
}

// Pricing is everything a new version may change. Percentages are in percent (2.9 means 2.9%).
type Pricing struct {
	Percent       decimal.Decimal
	Flat          decimal.Decimal
	Slabs         []Slab
	MinFee        *decimal.Decimal
	MaxFee        *decimal.Decimal
	Taxable       bool
	TaxPercent    decimal.Decimal
	FeeBearer     FeeBearer
	EffectiveFrom *time.Time
	EffectiveTo   *time.Time
}

type RuleInput struct {
	Scope
	Pricing
}

// Rule is one immutable version of a lineage.
type Rule struct {
	ID        uint
	LineageID string
	Version   int
	Scope
	// MinorUnits is the currency precision fixed when the version was written; Compute rounds to it.
	MinorUnits    int32
	Percent       decimal.Decimal
	Flat          decimal.Decimal
	Slabs         []Slab
	MinFee        *decimal.Decimal
	MaxFee        *decimal.Decimal
	Taxable       bool
	TaxPercent    decimal.Decimal
	FeeBearer     FeeBearer
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	CreatedBy     string
	CreatedAt     time.Time
}

// ActiveAt reports whether at falls in [EffectiveFrom, EffectiveTo).
func (r Rule) ActiveAt(at time.Time) bool {
	return !at.Before(r.EffectiveFrom) && (r.EffectiveTo == nil || at.Before(*r.EffectiveTo))
}

// Query describes a payment to price. At defaults to now. Chain names the network of an on-chain
// asset and is required by Snapshot for method crypto (it picks the ledger asset, e.g. USDC.BASE).
type Query struct {
	Method    Method
	Connector string
	CardType  CardType
	Region    string
	Currency  string
	Chain     string
	At        time.Time
}

// Breakdown is the result of applying one rule to one amount.
type Breakdown struct {
	RuleID        uint
	Version       int
	Currency      string
	FeeBearer     FeeBearer
	Amount        decimal.Decimal
	Fee           decimal.Decimal
	Tax           decimal.Decimal
	CustomerTotal decimal.Decimal
	MerchantNet   decimal.Decimal
}

type PreviewRequest struct {
	Query
	Amount    decimal.Decimal
	FeeBearer *FeeBearer
}

type RuleFilter struct {
	Method    Method
	Currency  string
	LineageID string
}

// AttemptRef names one payment attempt; a retry on another connector is a new attempt.
type AttemptRef struct {
	PaymentRequestID uint
	AttemptID        string
}

// Snapshot is the rule version an attempt is priced under, with the merchant and ledger asset read from stored rows.
type Snapshot struct {
	ID               uint
	AttemptID        string
	PaymentRequestID uint
	MerchantID       uint
	RuleID           uint
	RuleVersion      int
	Currency         string
	LedgerAsset      string
	FeeBearer        FeeBearer
	CreatedAt        time.Time
}

// Port is what other modules and the HTTP layer depend on. Call points: README "Payment path".
type Port interface {
	Resolve(ctx context.Context, q Query) (Rule, error)
	Preview(ctx context.Context, req PreviewRequest) (Breakdown, error)
	CreateRule(ctx context.Context, in RuleInput, actor string) (Rule, error)
	NewVersion(ctx context.Context, ruleID uint, p Pricing, actor string) (Rule, error)
	GetRule(ctx context.Context, id uint) (Rule, error)
	ListRules(ctx context.Context, f RuleFilter) ([]Rule, error)
	// Snapshot resolves the rule for q at database time (q.At is ignored) and records it against the attempt,
	// in the caller's transaction, at attempt creation.
	Snapshot(ctx context.Context, tx *gorm.DB, ref AttemptRef, q Query) (Snapshot, error)
	// PostFee recomputes the fee from the attempt's snapshot and posts it, in the caller's transaction. captured is
	// the base for a merchant-borne rule and the customer's gross for a customer-borne one (README "Payment path").
	PostFee(ctx context.Context, tx *gorm.DB, ref AttemptRef, captured decimal.Decimal) (Breakdown, error)
}

var (
	ErrNoRule             = errors.New("fees: no active fee rule matches")
	ErrNotFound           = errors.New("fees: fee rule not found")
	ErrStaleVersion       = errors.New("fees: rule is not the latest version of its lineage")
	ErrSurchargeForbidden = errors.New("fees: method does not allow a customer surcharge")
	ErrSnapshotConflict   = errors.New("fees: attempt is already snapshotted for another payment")
	ErrPaymentNotFound    = errors.New("fees: payment request not found")
	ErrSnapshotNotFound   = errors.New("fees: no fee snapshot for this attempt")
	ErrFeeExceedsAmount   = errors.New("fees: fee and tax exceed the amount")
	ErrGrossMismatch      = errors.New("fees: captured gross is not base plus surcharge under the snapshotted rule")
	ErrFeeAlreadyPosted   = errors.New("fees: a fee is already posted for this payment by another attempt")
	ErrPostingConflict    = errors.New("fees: attempt already posted with a different captured amount")
)

// ValidationError names the offending field so the API can point at it.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("fees: invalid %s: %s", e.Field, e.Reason)
}

func invalid(field, format string, args ...any) error {
	return &ValidationError{Field: field, Reason: fmt.Sprintf(format, args...)}
}

// AmbiguousRuleError is a configuration error: two active rules tie at the top specificity.
type AmbiguousRuleError struct {
	Specificity Specificity
	RuleIDs     []uint
}

func (e *AmbiguousRuleError) Error() string {
	ids := make([]string, len(e.RuleIDs))
	for i, id := range e.RuleIDs {
		ids[i] = fmt.Sprint(id)
	}
	return fmt.Sprintf("fees: ambiguous configuration, rules %s tie at specificity %d", strings.Join(ids, ","), e.Specificity)
}

// OverlapError is a write refused because another active rule already covers the same scope in that window.
type OverlapError struct {
	RuleIDs []uint
}

func (e *OverlapError) Error() string {
	ids := make([]string, len(e.RuleIDs))
	for i, id := range e.RuleIDs {
		ids[i] = fmt.Sprint(id)
	}
	return fmt.Sprintf("fees: overlaps active rule(s) %s for the same scope", strings.Join(ids, ","))
}
