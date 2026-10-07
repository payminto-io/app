// Package ledger is the multi-asset, append-only, double-entry ledger.
// Balances are derived by summing lines; nothing stores a balance.
// Amounts are signed: a positive amount is a debit, a negative one a credit.
// Design: .scratch/payments-v1/issues/01-ledger.md.
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type OwnerType string

const (
	OwnerMember    OwnerType = "member"
	OwnerPlatform  OwnerType = "platform"
	OwnerConnector OwnerType = "connector"
	OwnerChain     OwnerType = "chain"
	OwnerFees      OwnerType = "fees"
	OwnerReserve   OwnerType = "reserve"
)

type AccountKind string

const (
	KindAsset     AccountKind = "asset"
	KindLiability AccountKind = "liability"
	KindIncome    AccountKind = "income"
	KindExpense   AccountKind = "expense"
)

type JournalKind string

const (
	KindPayment    JournalKind = "payment"
	KindFee        JournalKind = "fee"
	KindConversion JournalKind = "conversion"
	KindSettlement JournalKind = "settlement"
	KindRefund     JournalKind = "refund"
	KindAdjustment JournalKind = "adjustment"
	// KindTransfer moves value between the platform's own accounts (sweeps); not in the ticket list, added for honesty.
	KindTransfer JournalKind = "transfer"
)

var (
	ErrInvalid             = errors.New("ledger: invalid journal")
	ErrEmptyJournal        = fmt.Errorf("%w: no lines", ErrInvalid)
	ErrZeroAmount          = fmt.Errorf("%w: zero-amount line", ErrInvalid)
	ErrUnbalanced          = fmt.Errorf("%w: lines do not sum to zero per asset", ErrInvalid)
	ErrScale               = fmt.Errorf("%w: amount has more than 18 decimal places", ErrInvalid)
	ErrMagnitude           = fmt.Errorf("%w: amount magnitude must be below 1e20", ErrInvalid)
	ErrPostedAt            = fmt.Errorf("%w: posted_at outside the allowed window", ErrInvalid)
	ErrMixedKinds          = errors.New("ledger: owner holds more than one account kind in one asset; use AccountBalances")
	ErrIdempotencyConflict = errors.New("ledger: idempotency key reused with a different journal")
	ErrAccountNotFound     = errors.New("ledger: account not found")
)

// numeric(38,18) column: anything finer is rounded by Postgres and anything larger overflows; both are refused here.
const (
	maxScale = 18
)

var maxMagnitude = decimal.New(1, 20)

const (
	maxOwnerIDLen        = 128
	maxAssetLen          = 32
	maxIdempotencyKeyLen = 128
	maxReferenceLen      = 128
)

var (
	ownerTypes   = []OwnerType{OwnerMember, OwnerPlatform, OwnerConnector, OwnerChain, OwnerFees, OwnerReserve}
	accountKinds = []AccountKind{KindAsset, KindLiability, KindIncome, KindExpense}
	journalKinds = []JournalKind{KindPayment, KindFee, KindConversion, KindSettlement, KindRefund, KindAdjustment, KindTransfer}
)

// AccountKey identifies a ledger account; accounts are created on first use.
type AccountKey struct {
	OwnerType OwnerType
	OwnerID   string
	Asset     string
	Kind      AccountKind
}

func (k AccountKey) validate() error {
	if !slices.Contains(ownerTypes, k.OwnerType) {
		return fmt.Errorf("%w: owner type %q", ErrInvalid, k.OwnerType)
	}
	if !slices.Contains(accountKinds, k.Kind) {
		return fmt.Errorf("%w: account kind %q", ErrInvalid, k.Kind)
	}
	if n := len(strings.TrimSpace(k.OwnerID)); n == 0 || n > maxOwnerIDLen || n != len(k.OwnerID) {
		return fmt.Errorf("%w: owner id %q", ErrInvalid, k.OwnerID)
	}
	if len(k.Asset) > maxAssetLen || !assetPattern.MatchString(k.Asset) {
		return fmt.Errorf("%w: asset %q", ErrInvalid, k.Asset)
	}
	return nil
}

// assetPattern: upper-case code segments joined by single dots, e.g. USDC, USDC.BASE, USDC.E.AVALANCHE.
// The last segment of a chain-qualified asset is the chain; CurrencyOfAsset strips it.
var assetPattern = regexp.MustCompile(`^[A-Z0-9_-]+(\.[A-Z0-9_-]+)*$`)

// CurrencyOfAsset returns the currency part of an asset code: everything before the last dot, or the code itself.
func CurrencyOfAsset(asset string) string {
	if i := strings.LastIndex(asset, "."); i >= 0 {
		return asset[:i]
	}
	return asset
}

// Reference ties a journal to the domain object that caused it.
type Reference struct {
	Type string
	ID   string
}

type Line struct {
	Account AccountKey
	Amount  decimal.Decimal
}

// Journal is the input to Post. PostedAt defaults to now.
type Journal struct {
	Kind           JournalKind
	Reference      Reference
	IdempotencyKey string
	PostedAt       time.Time
	Metadata       map[string]any
	Lines          []Line
}

// Validate checks everything the database would also reject, so callers fail before a round trip.
func (j Journal) Validate() error {
	if !slices.Contains(journalKinds, j.Kind) {
		return fmt.Errorf("%w: journal kind %q", ErrInvalid, j.Kind)
	}
	if n := len(j.IdempotencyKey); n == 0 || n > maxIdempotencyKeyLen {
		return fmt.Errorf("%w: idempotency key length %d", ErrInvalid, n)
	}
	if len(j.Reference.Type) > maxReferenceLen || len(j.Reference.ID) > maxReferenceLen {
		return fmt.Errorf("%w: reference too long", ErrInvalid)
	}
	if len(j.Lines) == 0 {
		return ErrEmptyJournal
	}
	sums := map[string]decimal.Decimal{}
	for i, l := range j.Lines {
		if err := l.Account.validate(); err != nil {
			return fmt.Errorf("line %d: %w", i, err)
		}
		if l.Amount.IsZero() {
			return fmt.Errorf("line %d: %w", i, ErrZeroAmount)
		}
		if !l.Amount.Equal(l.Amount.Truncate(maxScale)) {
			return fmt.Errorf("line %d: %w", i, ErrScale)
		}
		if l.Amount.Abs().GreaterThanOrEqual(maxMagnitude) {
			return fmt.Errorf("line %d: %w", i, ErrMagnitude)
		}
		sums[l.Account.Asset] = sums[l.Account.Asset].Add(l.Amount)
	}
	for asset, sum := range sums {
		if !sum.IsZero() {
			return fmt.Errorf("%w: %s sums to %s", ErrUnbalanced, asset, sum)
		}
	}
	return nil
}

type canonicalLine struct {
	OwnerType OwnerType   `json:"owner_type"`
	OwnerID   string      `json:"owner_id"`
	Asset     string      `json:"asset"`
	Kind      AccountKind `json:"kind"`
	Amount    string      `json:"amount"`
}

type canonicalJournal struct {
	Kind      JournalKind     `json:"kind"`
	Reference Reference       `json:"reference"`
	PostedAt  string          `json:"posted_at,omitempty"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	Lines     []canonicalLine `json:"lines"`
}

// requestHash is the payload fingerprint stored with the journal so a replayed key can be checked against its original.
func (j Journal) requestHash() string {
	lines := make([]canonicalLine, 0, len(j.Lines))
	for _, l := range j.Lines {
		lines = append(lines, canonicalLine{
			OwnerType: l.Account.OwnerType,
			OwnerID:   l.Account.OwnerID,
			Asset:     l.Account.Asset,
			Kind:      l.Account.Kind,
			Amount:    l.Amount.String(),
		})
	}
	slices.SortFunc(lines, func(a, b canonicalLine) int {
		return strings.Compare(
			fmt.Sprint(a.OwnerType, "|", a.OwnerID, "|", a.Asset, "|", a.Kind, "|", a.Amount),
			fmt.Sprint(b.OwnerType, "|", b.OwnerID, "|", b.Asset, "|", b.Kind, "|", b.Amount),
		)
	})
	postedAt := ""
	if !j.PostedAt.IsZero() {
		postedAt = j.PostedAt.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(canonicalJournal{Kind: j.Kind, Reference: j.Reference, PostedAt: postedAt, Metadata: j.Metadata, Lines: lines})
	if err != nil {
		// Metadata is the only field that can fail to marshal; fold the failure into the hash so it never matches a good payload.
		raw = []byte("unmarshalable:" + err.Error())
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
