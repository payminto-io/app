// Package chaindeposit wraps Payminto's crypto deposit flow as a connector: a payment request with a deposit
// address is the authorization, and the attempt stays pending until the chain confirms the deposit.
// Payminto's own APIs are untouched; this package only reads and drives them through Backend.
package chaindeposit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

const Code connectors.Code = "chaindeposit"

// PricingAsset is the only asset Payminto's payment_requests price in (amount_in_usd).
const PricingAsset = "USD"

// Raw statuses are Payminto payment_requests.state in lower case.
const (
	StatusOpen            connectors.RawStatus = "open"
	StatusPartiallyFilled connectors.RawStatus = "partially_filled"
	StatusFilled          connectors.RawStatus = "filled"
	StatusOverFilled      connectors.RawStatus = "over_filled"
	StatusCancelled       connectors.RawStatus = "cancelled"
)

var RawStatuses = []connectors.RawStatus{StatusOpen, StatusPartiallyFilled, StatusFilled, StatusOverFilled, StatusCancelled}

// Payment method details the merchant supplies; both name Payminto blockchain and currency codes.
const (
	DetailChain = "chain"
	DetailAsset = "asset"
)

// Backend is the slice of the deposit flow the connector needs; payminto.go adapts the real services and
// tests use an in-memory one.
type Backend interface {
	OpenPayment(ctx context.Context, req OpenRequest) (OpenResult, error)
	PaymentStatus(ctx context.Context, reference string) (PaymentStatus, error)
	CancelPayment(ctx context.Context, reference string) error
}

type OpenRequest struct {
	MerchantMemberID uint
	PlatformID       uint
	AmountInUSD      decimal.Decimal
	ChainCode        string
	CurrencyCode     string
	Reference        string
}

type OpenResult struct {
	Reference string
	Address   string
	ExpiresAt *time.Time
}

// PaymentStatus is what the deposit flow knows: Payminto's state and the confirmed sum so far.
type PaymentStatus struct {
	State       string
	AmountInUSD decimal.Decimal
	Received    decimal.Decimal
}

var ErrBackendNotFound = errors.New("chaindeposit: payment request not found")

type Connector struct {
	backend Backend
}

func New(backend Backend) *Connector { return &Connector{backend: backend} }

func (c *Connector) Code() connectors.Code { return Code }

func (c *Connector) Capabilities() connectors.Capabilities {
	return connectors.Capabilities{
		Methods:     []connectors.Method{connectors.MethodChain},
		Void:        true,
		Sync:        true,
		RawStatuses: RawStatuses,
	}
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
	chain, asset := req.PaymentMethod.Details[DetailChain], req.PaymentMethod.Details[DetailAsset]
	if chain == "" || asset == "" {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: payment method details need %s and %s", connectors.ErrInvalidRequest, DetailChain, DetailAsset)
	}
	member, err := parseID(req.MerchantID)
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: merchant id: %v", connectors.ErrInvalidRequest, err)
	}
	platform, err := parseID(req.PlatformID)
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("%w: platform id: %v", connectors.ErrInvalidRequest, err)
	}
	opened, err := c.backend.OpenPayment(ctx, OpenRequest{
		MerchantMemberID: member, PlatformID: platform, AmountInUSD: req.Money.Amount,
		ChainCode: chain, CurrencyCode: asset, Reference: req.AttemptID,
	})
	if err != nil {
		return connectors.AuthorizeResponse{}, fmt.Errorf("chaindeposit: open payment: %w", err)
	}
	next := &connectors.NextAction{Type: "pay_to_address", Address: opened.Address, Amount: req.Money.Amount.String(), Asset: asset}
	if opened.ExpiresAt != nil {
		next.ExpiresAt = opened.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return connectors.AuthorizeResponse{ConnectorTransactionID: opened.Reference, RawStatus: StatusOpen, NextAction: next}, nil
}

func (c *Connector) Capture(context.Context, connectors.CaptureRequest) (connectors.CaptureResponse, error) {
	return connectors.CaptureResponse{}, fmt.Errorf("%w: deposits settle on confirmation", connectors.ErrUnsupported)
}

// Void cancels an open payment request so later deposits are not claimed for it.
func (c *Connector) Void(ctx context.Context, req connectors.VoidRequest) (connectors.VoidResponse, error) {
	st, err := c.backend.PaymentStatus(ctx, req.ConnectorTransactionID)
	if err != nil {
		return connectors.VoidResponse{}, mapBackendErr(err)
	}
	if raw(st.State) != StatusOpen {
		return connectors.VoidResponse{}, fmt.Errorf("%w: cannot cancel a payment request in state %s", connectors.ErrInvalidRequest, st.State)
	}
	if err := c.backend.CancelPayment(ctx, req.ConnectorTransactionID); err != nil {
		return connectors.VoidResponse{}, mapBackendErr(err)
	}
	return connectors.VoidResponse{RawStatus: StatusCancelled}, nil
}

func (c *Connector) Refund(context.Context, connectors.RefundRequest) (connectors.RefundResponse, error) {
	return connectors.RefundResponse{}, fmt.Errorf("%w: on-chain refunds are a withdrawal, not a connector refund", connectors.ErrUnsupported)
}

func (c *Connector) Sync(ctx context.Context, req connectors.SyncRequest) (connectors.SyncResponse, error) {
	ref := req.ConnectorTransactionID
	if ref == "" {
		ref = req.AttemptID
	}
	st, err := c.backend.PaymentStatus(ctx, ref)
	if err != nil {
		return connectors.SyncResponse{}, mapBackendErr(err)
	}
	status := raw(st.State)
	resp := connectors.SyncResponse{ConnectorTransactionID: ref, RawStatus: status}
	if st.Received.IsPositive() {
		received := st.Received
		resp.AmountReceived = &received
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
