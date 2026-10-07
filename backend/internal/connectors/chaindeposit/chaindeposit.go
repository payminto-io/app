// Package chaindeposit wraps Payminto's crypto deposit flow as a connector: a payment request with a deposit
// address is the authorization, the customer must act (send funds), and the attempt settles when the chain
// confirms. Payminto's own APIs are untouched; this package only reads and drives them through Backend.
package chaindeposit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

const Code connectors.Code = "chaindeposit"

// PricingAsset is the only asset Payminto's payment_requests price in (amount_in_usd).
const PricingAsset = "USD"

// Raw statuses are Payminto payment_requests.state in lower case, plus cancelled_underpaid for a request
// closed with confirmed funds short of the amount.
const (
	StatusOpen               connectors.RawStatus = "open"
	StatusPartiallyFilled    connectors.RawStatus = "partially_filled"
	StatusFilled             connectors.RawStatus = "filled"
	StatusOverFilled         connectors.RawStatus = "over_filled"
	StatusCancelled          connectors.RawStatus = "cancelled"
	StatusCancelledUnderpaid connectors.RawStatus = "cancelled_underpaid"
)

var RawStatuses = []connectors.RawStatus{StatusOpen, StatusPartiallyFilled, StatusFilled, StatusOverFilled, StatusCancelled, StatusCancelledUnderpaid}

// Payment method details the merchant supplies; both name Payminto blockchain and currency codes.
const (
	DetailChain = "chain"
	DetailAsset = "asset"
)

// StableUSD lists the currency codes a USD-priced deposit may be paid in until conversion (ticket 10) lands:
// Payminto's finalizer treats one token unit as one dollar, which is only honest for USD-pegged stablecoins.
var StableUSD = []string{"USDC", "USDT", "DAI", "PYUSD"}

// Backend is the slice of the deposit flow the connector needs; payminto.go adapts the real services and
// memory.go is the test double. Both must pass RunBackendSuite.
type Backend interface {
	OpenPayment(ctx context.Context, req OpenRequest) (OpenResult, error)
	PaymentStatus(ctx context.Context, reference string) (PaymentStatus, error)
	// PaymentStatusByAttempt resolves by the attempt id persisted at OpenPayment, so a crash after the
	// request committed still finds it.
	PaymentStatusByAttempt(ctx context.Context, attemptID string) (PaymentStatus, error)
	CancelPayment(ctx context.Context, reference string) error
}

type OpenRequest struct {
	MerchantMemberID uint
	PlatformID       uint
	AmountInUSD      decimal.Decimal
	ChainCode        string
	CurrencyCode     string
	// AttemptID is persisted on the payment request (Payminto's invoice_id) in the same write that creates it.
	AttemptID string
}

type OpenResult struct {
	Reference string
	Address   string
	ExpiresAt *time.Time
}

// PaymentStatus is what the deposit flow knows: Payminto's state, the confirmed sum so far and where it sits.
type PaymentStatus struct {
	Reference    string
	State        string
	AmountInUSD  decimal.Decimal
	Received     decimal.Decimal
	ChainCode    string
	CurrencyCode string
	Address      string
	ExpiresAt    *time.Time
}

var (
	ErrBackendNotFound = errors.New("chaindeposit: payment request not found")
	// ErrNotCancellable means the request left OPEN/PARTIALLY_FILLED before the cancel could take it.
	ErrNotCancellable = errors.New("chaindeposit: payment request is no longer cancellable")
)

type Connector struct {
	backend Backend
}

func New(backend Backend) *Connector { return &Connector{backend: backend} }

func (c *Connector) Code() connectors.Code { return Code }

func (c *Connector) Capabilities() connectors.Capabilities {
	return connectors.Capabilities{
		Methods:              []connectors.Method{connectors.MethodChain},
		Void:                 true,
		Sync:                 true,
		WatchesAfterTerminal: true,
		RawStatuses:          RawStatuses,
	}
}

// ReceivedAsset is the ledger's chain-qualified code for a token on a chain (USDC.SOLANA).
func ReceivedAsset(currencyCode, chainCode string) string {
	return strings.ToUpper(strings.TrimSpace(currencyCode)) + "." + strings.ToUpper(strings.TrimSpace(chainCode))
}

func (c *Connector) Authorize(ctx context.Context, req connectors.AuthorizeRequest) (connectors.AuthorizeResponse, error) {
	if req.PaymentMethod.Type != connectors.MethodChain {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: method %q", connectors.ErrUnsupported, req.PaymentMethod.Type)
	}
	if !req.Money.Amount.IsPositive() {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: amount must be positive", connectors.ErrInvalidRequest)
	}
	if req.Money.Asset != PricingAsset {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: chain deposits are priced in %s, got %s", connectors.ErrInvalidRequest, PricingAsset, req.Money.Asset)
	}
	if req.CaptureMethod == connectors.CaptureManual {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: a deposit cannot be held for later capture", connectors.ErrUnsupported)
	}
	chain, asset := strings.TrimSpace(req.PaymentMethod.Details[DetailChain]), strings.TrimSpace(req.PaymentMethod.Details[DetailAsset])
	if chain == "" || asset == "" {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: payment method details need %s and %s", connectors.ErrInvalidRequest, DetailChain, DetailAsset)
	}
	if !slices.Contains(StableUSD, strings.ToUpper(asset)) {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: %s is not a USD stablecoin; conversion is not available yet", connectors.ErrUnsupported, asset)
	}
	member, err := parseID(req.MerchantID)
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: merchant id: %v", connectors.ErrInvalidRequest, err)
	}
	platform, err := parseID(req.PlatformID)
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: platform id: %v", connectors.ErrInvalidRequest, err)
	}
	if req.AttemptID == "" {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: attempt id required", connectors.ErrInvalidRequest)
	}
	opened, err := c.backend.OpenPayment(ctx, OpenRequest{
		MerchantMemberID: member, PlatformID: platform, AmountInUSD: req.Money.Amount,
		ChainCode: chain, CurrencyCode: asset, AttemptID: req.AttemptID,
	})
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("chaindeposit: open payment: %w", err)
	}
	return connectors.AuthorizeResponse{ConnectorTransactionID: opened.Reference, RawStatus: StatusOpen, NextAction: payToAddress(opened.Address, req.Money.Amount, asset, opened.ExpiresAt)}, nil
}

func payToAddress(address string, amount decimal.Decimal, asset string, expires *time.Time) *connectors.NextAction {
	next := &connectors.NextAction{Type: "pay_to_address", Address: address, Amount: amount.String(), Asset: strings.ToUpper(asset)}
	if expires != nil {
		next.ExpiresAt = expires.UTC().Format(time.RFC3339)
	}
	return next
}

func (c *Connector) Capture(context.Context, connectors.CaptureRequest) (connectors.CaptureResponse, error) {
	return connectors.CaptureResponse{}, fmt.Errorf("%w: deposits settle on confirmation", connectors.ErrUnsupported)
}

// Void cancels an open or partially filled request so later deposits are not claimed for it, then re-reads: the
// cancel is conditional, so a fill that won the race is reported as filled, not cancelled. Funds already confirmed
// stay on the books and the raw status says so.
func (c *Connector) Void(ctx context.Context, req connectors.VoidRequest) (connectors.VoidResponse, error) {
	st, err := c.backend.PaymentStatus(ctx, req.ConnectorTransactionID)
	if err != nil {
		return connectors.VoidResponse{}, mapBackendErr(err)
	}
	switch raw(st.State) {
	case StatusOpen, StatusPartiallyFilled:
	default:
		return connectors.VoidResponse{}, fmt.Errorf("%w: cannot cancel a payment request in state %s", connectors.ErrInvalidRequest, st.State)
	}
	if err := c.backend.CancelPayment(ctx, req.ConnectorTransactionID); err != nil && !errors.Is(err, ErrNotCancellable) {
		return connectors.VoidResponse{}, mapBackendErr(err)
	}
	after, err := c.backend.PaymentStatus(ctx, req.ConnectorTransactionID)
	if err != nil {
		return connectors.VoidResponse{}, mapBackendErr(err)
	}
	resp := connectors.VoidResponse{RawStatus: statusOf(after)}
	if after.Received.IsPositive() {
		received := after.Received
		resp.AmountReceived = &received
		resp.ReceivedAsset = ReceivedAsset(after.CurrencyCode, after.ChainCode)
	}
	return resp, nil
}

// statusOf is Payminto's state as a raw status; a cancelled request that received less than its amount is
// cancelled_underpaid, one that received its amount or more is simply cancelled with the money reported alongside.
func statusOf(st PaymentStatus) connectors.RawStatus {
	status := raw(st.State)
	if status == StatusCancelled && st.Received.IsPositive() && st.Received.LessThan(st.AmountInUSD) {
		return StatusCancelledUnderpaid
	}
	return status
}

func (c *Connector) Refund(context.Context, connectors.RefundRequest) (connectors.RefundResponse, error) {
	return connectors.RefundResponse{}, fmt.Errorf("%w: on-chain refunds are a withdrawal, not a connector refund", connectors.ErrUnsupported)
}

func (c *Connector) SyncRefund(context.Context, connectors.SyncRefundRequest) (connectors.SyncRefundResponse, error) {
	return connectors.SyncRefundResponse{}, fmt.Errorf("%w: no connector refunds", connectors.ErrUnsupported)
}

// Sync reports Payminto's state; the received amount is in the deposit token, named by ReceivedAsset.
func (c *Connector) Sync(ctx context.Context, req connectors.SyncRequest) (connectors.SyncResponse, error) {
	var st PaymentStatus
	var err error
	if req.ConnectorTransactionID != "" {
		st, err = c.backend.PaymentStatus(ctx, req.ConnectorTransactionID)
	} else {
		st, err = c.backend.PaymentStatusByAttempt(ctx, req.AttemptID)
	}
	if err != nil {
		return connectors.SyncResponse{}, mapBackendErr(err)
	}
	status := statusOf(st)
	resp := connectors.SyncResponse{ConnectorTransactionID: st.Reference, RawStatus: status}
	if st.Received.IsPositive() {
		received := st.Received
		resp.AmountReceived = &received
		resp.ReceivedAsset = ReceivedAsset(st.CurrencyCode, st.ChainCode)
	}
	if (status == StatusOpen || status == StatusPartiallyFilled) && st.Address != "" {
		resp.NextAction = payToAddress(st.Address, st.AmountInUSD, st.CurrencyCode, st.ExpiresAt)
	}
	return resp, nil
}

func (c *Connector) VerifyWebhook(context.Context, http.Header, []byte) (connectors.WebhookEvent, error) {
	return connectors.WebhookEvent{}, fmt.Errorf("%w: deposits are observed by the block processors; use Sync", connectors.ErrUnsupported)
}

func raw(state string) connectors.RawStatus {
	return connectors.RawStatus(strings.ToLower(strings.TrimSpace(state)))
}

func parseID(s string) (uint, error) {
	n, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("%q is not a Payminto id", s)
	}
	return uint(n), nil
}

func mapBackendErr(err error) error {
	if errors.Is(err, ErrBackendNotFound) {
		return connectors.ErrNotFound
	}
	return fmt.Errorf("chaindeposit: %w", err)
}
