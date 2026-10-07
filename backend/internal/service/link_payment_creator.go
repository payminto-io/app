package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// paymintoLinkCurrency is the only currency Payminto's payment_requests carry (amount_in_usd).
const paymintoLinkCurrency = "USD"

// linkReferencePrefix makes a link payment's reference a function of its LinkPaymentID, so creation is idempotent
// through the payment_requests reference_id unique index and lookups need no extra column.
const linkReferencePrefix = "pl_"

// LinkPaymentCreator is the default links.PaymentCreator: a paid link becomes a Payminto payment request
// with a deposit address. It takes crypto in USD links only; the switch (ticket 05) replaces it (internal/links/README.md).
type LinkPaymentCreator struct {
	payments     *PaymentService
	db           *gorm.DB
	checkoutBase string
}

var _ links.PaymentCreator = (*LinkPaymentCreator)(nil)

func NewLinkPaymentCreator(payments *PaymentService, db *gorm.DB, checkoutBase string) *LinkPaymentCreator {
	return &LinkPaymentCreator{payments: payments, db: db, checkoutBase: strings.TrimRight(checkoutBase, "/")}
}

// LinkPaymentReference is the payment_requests reference of a link use.
func LinkPaymentReference(linkPaymentID string) string { return linkReferencePrefix + linkPaymentID }

// Connectors offers the unscoped connector "" for a crypto asset that an active chain accepts deposits of.
func (c *LinkPaymentCreator) Connectors(ctx context.Context, _ links.Environment, currency string, m links.MethodSpec) ([]string, error) {
	if m.Method != fees.MethodCrypto || currency != paymintoLinkCurrency {
		return nil, nil
	}
	var n int64
	err := c.db.WithContext(ctx).Raw(`SELECT count(*) FROM blockchain_currencies bc JOIN blockchains b ON b.id = bc.blockchain_id
		WHERE upper(bc.currency_code) = ? AND upper(bc.blockchain_code) = ?
		  AND bc.deleted_at IS NULL AND b.deleted_at IS NULL AND b.status = 'active' AND bc.deposit_enabled`,
		m.Asset, m.Chain).Scan(&n).Error
	if err != nil {
		return nil, fmt.Errorf("links: payminto connectors for %s: %w", m, err)
	}
	if n == 0 {
		return nil, nil
	}
	return []string{""}, nil
}

// lookup reads the payment a link use made; a cancelled payment counts as not created.
func (c *LinkPaymentCreator) lookup(ctx context.Context, linkPaymentID string) (*models.PaymentRequest, error) {
	var rows []models.PaymentRequest
	err := c.db.WithContext(ctx).Where("reference_id = ?", LinkPaymentReference(linkPaymentID)).Limit(1).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("links: look up payment %s: %w", linkPaymentID, err)
	}
	if len(rows) == 0 || rows[0].State == models.PaymentStateCancelled {
		return nil, nil
	}
	return &rows[0], nil
}

func (c *LinkPaymentCreator) created(ctx context.Context, pr *models.PaymentRequest) links.CreatedPayment {
	out := links.CreatedPayment{Reference: pr.ReferenceID, CheckoutURL: c.checkoutBase + "/pay/" + pr.ReferenceID, ExpiresAt: pr.ExpiresAt}
	var addrs []string
	if err := c.db.WithContext(ctx).Raw(`SELECT address FROM deposit_addresses WHERE payment_request_id = ? ORDER BY id LIMIT 1`, pr.ID).Scan(&addrs).Error; err == nil && len(addrs) > 0 {
		out.DepositAddress = addrs[0]
	}
	return out
}

func (c *LinkPaymentCreator) FindPayment(ctx context.Context, linkPaymentID string) (links.CreatedPayment, bool, error) {
	pr, err := c.lookup(ctx, linkPaymentID)
	if err != nil || pr == nil {
		return links.CreatedPayment{}, false, err
	}
	return c.created(ctx, pr), true, nil
}

// CreatePayment returns the existing payment for req.LinkPaymentID, or creates it with the link's quote expiry
// and records the link's fee rule on it. It reports ErrNotCreated only when a lookup proves nothing exists.
func (c *LinkPaymentCreator) CreatePayment(ctx context.Context, req links.PaymentRequest) (links.CreatedPayment, error) {
	if req.Method.Method != fees.MethodCrypto || req.Currency != paymintoLinkCurrency {
		return links.CreatedPayment{}, &links.Error{Code: links.CodeMethodUnavailable, Field: "method",
			Message: fmt.Sprintf("this server takes %s only for crypto in %s until the switch is installed", req.Method, paymintoLinkCurrency)}
	}
	if err := ctx.Err(); err != nil {
		return links.CreatedPayment{}, fmt.Errorf("%w: %v", links.ErrNotCreated, err)
	}
	existing, err := c.lookup(ctx, req.LinkPaymentID)
	if err != nil {
		return links.CreatedPayment{}, err
	}
	if existing != nil {
		return c.created(ctx, existing), nil
	}
	in := CreatePaymentInput{
		AmountInUSD:    req.CustomerTotal,
		BlockchainCode: req.Method.Chain,
		CurrencyCode:   req.Method.Asset,
		ReferenceID:    LinkPaymentReference(req.LinkPaymentID),
		ExpiresIn:      time.Duration(req.QuoteExpirySeconds) * time.Second,
	}
	if req.CustomerEmail != "" {
		email := req.CustomerEmail
		in.CustomerEmail = &email
	}
	if req.ReferenceID != "" {
		ref := req.ReferenceID
		in.InvoiceID = &ref
	}
	_, createErr := c.payments.CreatePayment(in, req.MemberID, req.PlatformID)
	// The database, not the returned error, decides what exists: the service may write and then fail.
	settled := context.WithoutCancel(ctx)
	pr, err := c.lookup(settled, req.LinkPaymentID)
	switch {
	case err != nil:
		return links.CreatedPayment{}, errors.Join(createErr, err)
	case pr == nil && createErr != nil:
		return links.CreatedPayment{}, fmt.Errorf("%w: %v", links.ErrNotCreated, createErr)
	case pr == nil:
		return links.CreatedPayment{}, errors.New("links: payment service reported success but no payment exists")
	}
	if req.FeeRuleID != 0 {
		if err := c.db.WithContext(settled).Exec(`UPDATE payment_requests SET fee_rule_id = ?, fee_rule_version = ?
			WHERE id = ? AND fee_rule_id IS NULL`, req.FeeRuleID, req.FeeRuleVersion, pr.ID).Error; err != nil {
			return links.CreatedPayment{}, fmt.Errorf("links: record fee rule on payment %s: %w", pr.ReferenceID, err)
		}
	}
	return c.created(settled, pr), nil
}
