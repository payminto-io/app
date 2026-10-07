package links

import (
	"context"
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// Store sentinels; storetest asserts both stores return them.
var (
	ErrStoreNotFound  = errors.New("links: not found")
	ErrStale          = errors.New("links: row changed since it was read")
	ErrShortCodeTaken = errors.New("links: short code taken")
)

const (
	paymentPending = "pending"
	paymentCreated = "created"
)

type Answer struct {
	QuestionKey   string
	QuestionLabel string
	Value         string
}

// LinkPayment is one use of a link: reserved (pending) before the payment exists, created once it does.
type LinkPayment struct {
	ID               string
	LinkID           string
	IdempotencyKey   string
	RequestHash      string
	Status           string
	Environment      Environment
	Method           MethodSpec
	Connector        string
	Amount           decimal.Decimal
	Currency         string
	Fee              *decimal.Decimal
	Tax              *decimal.Decimal
	CustomerTotal    decimal.Decimal
	FeeBearer        fees.FeeBearer
	FeeRuleID        *uint
	FeeRuleVersion   *int
	CustomerName     string
	CustomerEmail    string
	CustomerPhone    string
	BillingAddress   *Address
	ShippingAddress  *Address
	Answers          []Answer
	PaymentReference string
	Processor        *CreatedPayment
	// ClientKey is a hash of the payer's IP, for the per-client open-payment cap.
	ClientKey string
	// ReservedUntil is the lease of a pending use; after it the resolver may look the payment up.
	ReservedUntil time.Time
	// OpenUntil is when this use stops counting as an open payment: the lease, then the payment's expiry.
	OpenUntil time.Time
	CreatedAt time.Time
}

// ReserveLimits caps open payments on a multi-use link, so an anonymous payer cannot drain the address pool.
type ReserveLimits struct {
	MaxOpen          int
	MaxOpenPerClient int
}

// Store persists links. Implementations: memStore (tests) and pgStore (Postgres); both must pass store_contract_test.go.
type Store interface {
	Insert(ctx context.Context, l Link) (Link, error)
	Get(ctx context.Context, platformID uint, id string) (Link, error)
	GetByShortCode(ctx context.Context, code string) (Link, error)
	List(ctx context.Context, platformID uint, f ListFilter) ([]Link, int64, error)
	// Save replaces the editable fields, Total and children if the stored revision is still l.Revision (else errStale).
	Save(ctx context.Context, l Link) (Link, error)
	// SetStatus moves the link to `to` if its revision is unchanged; a non-empty shortCode is set when it has none.
	SetStatus(ctx context.Context, platformID uint, id string, revision int, to Status, shortCode string, now time.Time) (Link, error)
	DeleteDraft(ctx context.Context, platformID uint, id string, revision int) error
	WebhookExists(ctx context.Context, platformID, webhookID uint) (bool, error)
	MerchantName(ctx context.Context, platformID uint) (string, error)
	FindPayment(ctx context.Context, linkID, key string) (*LinkPayment, error)
	// Reserve atomically re-checks status, expiry, fee bearer, use limit and open-payment caps under a lock and
	// takes one use. When the key already exists it returns that payment as existing instead.
	Reserve(ctx context.Context, p LinkPayment, now time.Time, limits ReserveLimits) (reserved LinkPayment, existing *LinkPayment, err error)
	// Complete records the created payment on a pending use (else ErrStale).
	Complete(ctx context.Context, paymentID string, created CreatedPayment, openUntil time.Time) error
	// Release deletes a pending use and gives it back; only for a payment that definitively does not exist.
	Release(ctx context.Context, paymentID string) error
	// ExpiredPending lists pending uses whose lease ended before now, oldest first.
	ExpiredPending(ctx context.Context, now time.Time, limit int) ([]LinkPayment, error)
}

// openPaymentsError is the shared cap check; open counts uses still pending or not yet expired.
func openPaymentsError(l Link, limits ReserveLimits, open, openForClient int) error {
	if !l.MultiUse {
		return nil
	}
	if limits.MaxOpen > 0 && open >= limits.MaxOpen {
		return newErr(CodeOpenPaymentsLimit, "", "this link has too many unpaid payments open; try again later")
	}
	if limits.MaxOpenPerClient > 0 && openForClient >= limits.MaxOpenPerClient {
		return newErr(CodeOpenPaymentsLimit, "", "you have too many unpaid payments open on this link; finish or wait for one to expire")
	}
	return nil
}

// availability is the shared rule both stores apply under their lock.
func availability(l Link, now time.Time) error {
	switch l.Status {
	case StatusActive:
	case StatusPaused:
		return newErr(CodePaused, "", "this link is paused")
	case StatusArchived:
		return newErr(CodeArchived, "", "this link is no longer available")
	default:
		return newErr(CodeNotFound, "", "payment link not found")
	}
	if l.ExpiresAt != nil && !now.Before(*l.ExpiresAt) {
		return newErr(CodeExpired, "", "this link has expired")
	}
	if limit := l.EffectiveUseLimit(); limit != nil && l.UsesCount >= *limit {
		return newErr(CodeUseLimitReached, "", "this link has taken all the payments it allows")
	}
	return nil
}
