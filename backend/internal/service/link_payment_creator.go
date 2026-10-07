package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"gorm.io/gorm"
)

// paymintoLinkCurrency is the only currency Payminto's payment_requests carry (amount_in_usd).
const paymintoLinkCurrency = "USD"

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

func (c *LinkPaymentCreator) CreatePayment(_ context.Context, req links.PaymentRequest) (links.CreatedPayment, error) {
	if req.Method.Method != fees.MethodCrypto || req.Currency != paymintoLinkCurrency {
		return links.CreatedPayment{}, &links.Error{Code: links.CodeMethodUnavailable, Field: "method",
			Message: fmt.Sprintf("this server takes %s only for crypto in %s until the switch is installed", req.Method, paymintoLinkCurrency)}
	}
	in := CreatePaymentInput{
		AmountInUSD:    req.CustomerTotal,
		BlockchainCode: req.Method.Chain,
		CurrencyCode:   req.Method.Asset,
	}
	if req.CustomerEmail != "" {
		email := req.CustomerEmail
		in.CustomerEmail = &email
	}
	if req.ReferenceID != "" {
		ref := req.ReferenceID
		in.InvoiceID = &ref
	}
	res, err := c.payments.CreatePayment(in, req.MemberID, req.PlatformID)
	if err != nil {
		return links.CreatedPayment{}, err
	}
	out := links.CreatedPayment{
		Reference:   res.Payment.ReferenceID,
		CheckoutURL: c.checkoutBase + "/pay/" + res.Payment.ReferenceID,
		ExpiresAt:   res.Payment.ExpiresAt,
	}
	if res.DepositAddress != nil {
		out.DepositAddress = res.DepositAddress.Address
	}
	return out, nil
}
