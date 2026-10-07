package links

import (
	"context"
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

var (
	errStoreNotFound  = errors.New("links: not found")
	errStale          = errors.New("links: link changed since it was read")
	errShortCodeTaken = errors.New("links: short code taken")
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
	CreatedAt        time.Time
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
	// AnsweredBefore reports which of keys this email answered on an earlier created payment of the link.
	AnsweredBefore(ctx context.Context, linkID, email string, keys []string) (map[string]bool, error)
	FindPayment(ctx context.Context, linkID, key string) (*LinkPayment, error)
	// Reserve atomically re-checks status, expiry and use limit under a lock and takes one use. When the key
	// already exists it returns that payment as existing instead.
	Reserve(ctx context.Context, p LinkPayment, now time.Time) (reserved LinkPayment, existing *LinkPayment, err error)
	Complete(ctx context.Context, paymentID string, created CreatedPayment) error
	// Release deletes a pending reservation and gives its use back; the payment was never created.
	Release(ctx context.Context, paymentID string) error
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
