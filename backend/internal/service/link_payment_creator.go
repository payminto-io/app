package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

var _ links.Catalog = (*LinkPaymentCreator)(nil)

// Offerings lists crypto in USD for every asset an active chain accepts deposits of, the same rows Connectors checks.
func (c *LinkPaymentCreator) Offerings(ctx context.Context, _ links.Environment) ([]links.Offering, error) {
	var rows []struct{ Chain, Asset string }
	err := c.db.WithContext(ctx).Raw(`SELECT DISTINCT upper(bc.blockchain_code) AS chain, upper(bc.currency_code) AS asset
		FROM blockchain_currencies bc JOIN blockchains b ON b.id = bc.blockchain_id
		WHERE bc.deleted_at IS NULL AND b.deleted_at IS NULL AND b.status = 'active' AND bc.deposit_enabled
		ORDER BY 1, 2`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("links: payminto offerings: %w", err)
	}
	out := make([]links.Offering, len(rows))
	for i, r := range rows {
		out[i] = links.Offering{Currency: paymintoLinkCurrency, Method: links.MethodSpec{Method: fees.MethodCrypto, Chain: r.Chain, Asset: r.Asset}}
	}
	return out, nil
}

// lookupAny reads the payment_requests row carrying a link use's reference, live or cancelled.
func (c *LinkPaymentCreator) lookupAny(ctx context.Context, linkPaymentID string) (*models.PaymentRequest, error) {
	var rows []models.PaymentRequest
	err := c.db.WithContext(ctx).Where("reference_id = ?", LinkPaymentReference(linkPaymentID)).Limit(1).Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("links: look up payment %s: %w", linkPaymentID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

// lookup reads the live payment a link use made; a cancelled payment (or a fence) counts as none.
func (c *LinkPaymentCreator) lookup(ctx context.Context, linkPaymentID string) (*models.PaymentRequest, error) {
	pr, err := c.lookupAny(ctx, linkPaymentID)
	if err != nil || pr == nil || pr.State == models.PaymentStateCancelled {
		return nil, err
	}
	return pr, nil
}

func (c *LinkPaymentCreator) created(ctx context.Context, pr *models.PaymentRequest) links.CreatedPayment {
	out := links.CreatedPayment{Reference: pr.ReferenceID, CheckoutURL: c.checkoutBase + "/pay/" + pr.ReferenceID, ExpiresAt: pr.ExpiresAt}
	var addrs []string
	if err := c.db.WithContext(ctx).Raw(`SELECT address FROM deposit_addresses WHERE payment_request_id = ? ORDER BY id LIMIT 1`, pr.ID).Scan(&addrs).Error; err == nil && len(addrs) > 0 {
		out.DepositAddress = addrs[0]
	}
	return out
}

// FencePayment claims the use's reference with a cancelled placeholder unless a payment already holds it, so a
// stalled CreatePayment that arrives later hits the reference_id unique index instead of creating a live payment.
func (c *LinkPaymentCreator) FencePayment(ctx context.Context, req links.PaymentRequest) (links.CreatedPayment, bool, error) {
	fence := models.PaymentRequest{
		ReferenceID: LinkPaymentReference(req.LinkPaymentID), AmountInUSD: req.CustomerTotal, State: models.PaymentStateCancelled,
		MemberID: req.MemberID, ExternalPlatformID: req.PlatformID,
	}
	if err := c.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "reference_id"}}, DoNothing: true}).
		Create(&fence).Error; err != nil {
		return links.CreatedPayment{}, false, fmt.Errorf("links: fence payment %s: %w", req.LinkPaymentID, err)
	}
	pr, err := c.lookupAny(ctx, req.LinkPaymentID)
	switch {
	case err != nil:
		return links.CreatedPayment{}, false, err
	case pr == nil:
		return links.CreatedPayment{}, false, fmt.Errorf("links: fence for %s neither written nor found", req.LinkPaymentID)
	case pr.State == models.PaymentStateCancelled:
		return links.CreatedPayment{}, false, nil
	}
	return c.created(ctx, pr), true, nil
}

// CancelPayment cancels the use's payment while nothing has been paid into it; a payment already receiving funds is
// an error for the caller to record.
func (c *LinkPaymentCreator) CancelPayment(ctx context.Context, linkPaymentID string) error {
	pr, err := c.lookupAny(ctx, linkPaymentID)
	if err != nil || pr == nil || pr.State == models.PaymentStateCancelled {
		return err
	}
	res := c.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("id = ? AND state = ?", pr.ID, models.PaymentStateOpen).Update("state", models.PaymentStateCancelled)
	if res.Error != nil {
		return fmt.Errorf("links: cancel payment %s: %w", pr.ReferenceID, res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("links: payment %s is %s and cannot be cancelled", pr.ReferenceID, pr.State)
	}
	return nil
}

// OpenPayments reports which uses' payments are still open: not filled, cancelled or past their expiry.
func (c *LinkPaymentCreator) OpenPayments(ctx context.Context, linkPaymentIDs []string) (map[string]bool, error) {
	out := make(map[string]bool, len(linkPaymentIDs))
	if len(linkPaymentIDs) == 0 {
		return out, nil
	}
	refs := make([]string, len(linkPaymentIDs))
	for i, id := range linkPaymentIDs {
		refs[i] = LinkPaymentReference(id)
	}
	var open []string
	err := c.db.WithContext(ctx).Model(&models.PaymentRequest{}).
		Where("reference_id IN ? AND state IN ? AND (expires_at IS NULL OR expires_at > ?)", refs,
			[]string{models.PaymentStateOpen, models.PaymentStatePartiallyFilled}, time.Now()).
		Pluck("reference_id", &open).Error
	if err != nil {
		return nil, fmt.Errorf("links: open payments: %w", err)
	}
	for _, ref := range open {
		out[strings.TrimPrefix(ref, linkReferencePrefix)] = true
	}
	return out, nil
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
	case createErr != nil:
		// The service failed after writing (its own rollback failed too): cancel the half-made payment or say so.
		if cerr := c.CancelPayment(settled, req.LinkPaymentID); cerr != nil {
			slog.Error("links: anomaly: half-created payment could not be cancelled", "payment_reference", pr.ReferenceID,
				"create_error", createErr, "cancel_error", cerr)
			return links.CreatedPayment{}, errors.Join(createErr, cerr)
		}
		return links.CreatedPayment{}, fmt.Errorf("%w: %v", links.ErrNotCreated, createErr)
	}
	if req.FeeRuleID != 0 {
		if err := c.db.WithContext(settled).Exec(`UPDATE payment_requests SET fee_rule_id = ?, fee_rule_version = ?
			WHERE id = ? AND fee_rule_id IS NULL`, req.FeeRuleID, req.FeeRuleVersion, pr.ID).Error; err != nil {
			return links.CreatedPayment{}, fmt.Errorf("links: record fee rule on payment %s: %w", pr.ReferenceID, err)
		}
	}
	return c.created(settled, pr), nil
}
