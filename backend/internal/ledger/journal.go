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
	"slices"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
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
	ErrIdempotencyConflict = errors.New("ledger: idempotency key reused with a different journal")
	ErrAccountNotFound     = errors.New("ledger: account not found")
)

const (
	maxOwnerIDLen        = 128
	maxAssetLen          = 16
	maxIdempotencyKeyLen = 128
	maxReferenceLen      = 128
)

var (
	ownerTypes   = []OwnerType{OwnerMember, OwnerPlatform, OwnerConnector, OwnerChain, OwnerFees, OwnerReserve}
	accountKinds = []AccountKind{KindAsset, KindLiability, KindIncome, KindExpense}
	journalKinds = []JournalKind{KindPayment, KindFee, KindConversion, KindSettlement, KindRefund, KindAdjustment, KindTransfer}
)

// AccountKey identifies a ledger account; accounts are created on first use.
// Environment may be left empty: it then resolves to the context's environment, then the service default.
type AccountKey struct {
	OwnerType   OwnerType
	OwnerID     string
	Asset       string
	Kind        AccountKind
	Environment environment.Environment
}

func (k AccountKey) validate() error {
	if k.Environment != "" && !k.Environment.Valid() {
		return fmt.Errorf("%w: environment %q", ErrInvalid, k.Environment)
	}
	if !slices.Contains(ownerTypes, k.OwnerType) {
		return fmt.Errorf("%w: owner type %q", ErrInvalid, k.OwnerType)
	}
	if !slices.Contains(accountKinds, k.Kind) {
		return fmt.Errorf("%w: account kind %q", ErrInvalid, k.Kind)
	}
	if n := len(strings.TrimSpace(k.OwnerID)); n == 0 || n > maxOwnerIDLen || n != len(k.OwnerID) {
		return fmt.Errorf("%w: owner id %q", ErrInvalid, k.OwnerID)
	}
	if n := len(strings.TrimSpace(k.Asset)); n == 0 || n > maxAssetLen || n != len(k.Asset) {
		return fmt.Errorf("%w: asset %q", ErrInvalid, k.Asset)
	}
	return nil
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
	if _, err := j.explicitEnvironment(); err != nil {
		return err
	}
	sums := map[string]decimal.Decimal{}
	for i, l := range j.Lines {
		if err := l.Account.validate(); err != nil {
			return fmt.Errorf("line %d: %w", i, err)
		}
		if l.Amount.IsZero() {
			return fmt.Errorf("line %d: %w", i, ErrZeroAmount)
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

// explicitEnvironment is the one environment the lines name, or "" when none does; mixing is invalid.
func (j Journal) explicitEnvironment() (environment.Environment, error) {
	var env environment.Environment
	for i, l := range j.Lines {
		if l.Account.Environment == "" {
			continue
		}
		if env == "" {
			env = l.Account.Environment
		} else if l.Account.Environment != env {
			return "", fmt.Errorf("%w: line %d is %s but line 0 is %s", ErrInvalid, i, l.Account.Environment, env)
		}
	}
	return env, nil
}

// canonicalLine omits an empty environment so hashes stored before the column existed still match.
type canonicalLine struct {
	OwnerType   OwnerType               `json:"owner_type"`
	OwnerID     string                  `json:"owner_id"`
	Asset       string                  `json:"asset"`
	Kind        AccountKind             `json:"kind"`
	Environment environment.Environment `json:"environment,omitempty"`
	Amount      string                  `json:"amount"`
}

type canonicalJournal struct {
	Kind      JournalKind     `json:"kind"`
	Reference Reference       `json:"reference"`
	Metadata  map[string]any  `json:"metadata,omitempty"`
	Lines     []canonicalLine `json:"lines"`
}

// requestHash is the payload fingerprint stored with the journal so a replayed key can be checked against its original.
func (j Journal) requestHash() string {
	lines := make([]canonicalLine, 0, len(j.Lines))
	for _, l := range j.Lines {
		lines = append(lines, canonicalLine{
			OwnerType:   l.Account.OwnerType,
			OwnerID:     l.Account.OwnerID,
			Asset:       l.Account.Asset,
			Kind:        l.Account.Kind,
			Environment: l.Account.Environment,
			Amount:      l.Amount.String(),
		})
	}
	slices.SortFunc(lines, func(a, b canonicalLine) int {
		return strings.Compare(
			fmt.Sprint(a.OwnerType, "|", a.OwnerID, "|", a.Asset, "|", a.Kind, "|", a.Environment, "|", a.Amount),
			fmt.Sprint(b.OwnerType, "|", b.OwnerID, "|", b.Asset, "|", b.Kind, "|", b.Environment, "|", b.Amount),
		)
	})
	raw, err := json.Marshal(canonicalJournal{Kind: j.Kind, Reference: j.Reference, Metadata: j.Metadata, Lines: lines})
	if err != nil {
		// Metadata is the only field that can fail to marshal; fold the failure into the hash so it never matches a good payload.
		raw = []byte("unmarshalable:" + err.Error())
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
