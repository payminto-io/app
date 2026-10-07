// Package conformance is the test suite every connector provider must pass (MODULES.md rule 10).
// A provider's own test calls Run with a Harness that knows how to script its outcomes.
package conformance

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

// Outcome is what the harness is asked to make the provider do.
type Outcome string

const (
	OutcomeSettled    Outcome = "settled"    // money in synchronously
	OutcomeAuthorized Outcome = "authorized" // held, waiting for Capture
	OutcomeDeclined   Outcome = "declined"
	OutcomePending    Outcome = "pending" // no answer yet; Settle then Sync or webhook finishes it
)

// Class is the semantic bucket of a raw status, supplied by the harness so the suite stays free of the status map.
type Class string

const (
	ClassSettled        Class = "settled"
	ClassAuthorized     Class = "authorized"
	ClassPending        Class = "pending"
	ClassActionRequired Class = "action_required"
	ClassFailed         Class = "failed"
	ClassVoided         Class = "voided"
)

type Harness struct {
	New    func(t *testing.T) connectors.Connector
	Method connectors.Method
	Money  connectors.Money
	// PaymentMethod returns a method producing the outcome, or ok=false when the provider cannot do it synchronously.
	PaymentMethod func(outcome Outcome) (pm connectors.PaymentMethod, ok bool)
	// Classify maps the provider's raw payment statuses onto classes.
	Classify func(raw connectors.RawStatus) Class
	// Settle drives the provider side so a pending transaction reaches settled; nil when not scriptable.
	Settle func(t *testing.T, c connectors.Connector, attemptID, connectorTransactionID string)
	// Webhook builds a signed delivery announcing the transaction settled; nil when the provider has no webhooks.
	Webhook func(t *testing.T, c connectors.Connector, eventID, connectorTransactionID string) (http.Header, []byte)
}

var seq int

func nextAttempt() string {
	seq++
	return "conf_pa_" + decimal.NewFromInt(int64(seq)).String()
}

func (h Harness) authorize(t *testing.T, c connectors.Connector, outcome Outcome, capture connectors.CaptureMethod) (connectors.AuthorizeRequest, connectors.AuthorizeResponse, bool) {
	t.Helper()
	pm, ok := h.PaymentMethod(outcome)
	if !ok {
		return connectors.AuthorizeRequest{}, connectors.AuthorizeResponse{}, false
	}
	req := connectors.AuthorizeRequest{AttemptID: nextAttempt(), IntentID: "conf_pi", MerchantID: "1", PlatformID: "1", Money: h.Money, CaptureMethod: capture, PaymentMethod: pm, IdempotencyKey: "k"}
	resp, err := c.Authorize(context.Background(), req)
	if err != nil {
		t.Fatalf("Authorize(%s): %v", outcome, err)
	}
	if resp.ConnectorTransactionID == "" {
		t.Fatalf("Authorize(%s) returned no connector transaction id", outcome)
	}
	declared(t, c, resp.RawStatus)
	return req, resp, true
}

func declared(t *testing.T, c connectors.Connector, raw connectors.RawStatus) {
	t.Helper()
	if !slices.Contains(c.Capabilities().RawStatuses, raw) {
		t.Fatalf("connector reported status %q that Capabilities().RawStatuses does not declare", raw)
	}
}

func declaredRefund(t *testing.T, c connectors.Connector, raw connectors.RawStatus) {
	t.Helper()
	if !slices.Contains(c.Capabilities().RawRefundStatuses, raw) {
		t.Fatalf("connector reported refund status %q that Capabilities().RawRefundStatuses does not declare", raw)
	}
}

// Run executes the suite. Tests a provider cannot do are skipped by capability, visibly.
func Run(t *testing.T, h Harness) {
	t.Helper()
	ctx := context.Background()

	t.Run("capabilities", func(t *testing.T) {
		c := h.New(t)
		caps := c.Capabilities()
		if c.Code() == "" {
			t.Fatal("Code() is empty")
		}
		if len(caps.RawStatuses) == 0 {
			t.Fatal("RawStatuses is empty; the switch cannot verify its status map")
		}
		if !caps.Supports(h.Method) {
			t.Fatalf("harness method %s is not in Methods %v", h.Method, caps.Methods)
		}
		if caps.Refund && len(caps.RawRefundStatuses) == 0 {
			t.Fatal("Refund is supported but RawRefundStatuses is empty")
		}
		if caps.PartialCapture && !caps.ManualCapture {
			t.Fatal("PartialCapture without ManualCapture is contradictory")
		}
	})

	t.Run("authorize validation", func(t *testing.T) {
		c := h.New(t)
		pm, _ := h.PaymentMethod(OutcomePending)
		if p, ok := h.PaymentMethod(OutcomeSettled); ok {
			pm = p
		}
		_, err := c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: nextAttempt(), MerchantID: "1", PlatformID: "1", Money: connectors.Money{Amount: decimal.Zero, Asset: h.Money.Asset}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: pm})
		if !errors.Is(err, connectors.ErrInvalidRequest) {
			t.Fatalf("zero amount err = %v, want ErrInvalidRequest", err)
		}
		_, err = c.Authorize(ctx, connectors.AuthorizeRequest{AttemptID: nextAttempt(), MerchantID: "1", PlatformID: "1", Money: h.Money, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: connectors.PaymentMethod{Type: "carrier_pigeon", Token: "x"}})
		if !errors.Is(err, connectors.ErrUnsupported) && !errors.Is(err, connectors.ErrInvalidRequest) {
			t.Fatalf("unsupported method err = %v, want ErrUnsupported or ErrInvalidRequest", err)
		}
	})

	t.Run("settled synchronously", func(t *testing.T) {
		c := h.New(t)
		req, resp, ok := h.authorize(t, c, OutcomeSettled, connectors.CaptureAutomatic)
		if !ok {
			t.Skip("provider cannot settle synchronously")
		}
		if h.Classify(resp.RawStatus) != ClassSettled {
			t.Fatalf("status %q is not settled", resp.RawStatus)
		}
		if c.Capabilities().Sync {
			sync, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID})
			if err != nil || h.Classify(sync.RawStatus) != ClassSettled {
				t.Fatalf("Sync = %+v, %v", sync, err)
			}
			declared(t, c, sync.RawStatus)
		}
	})

	t.Run("declined", func(t *testing.T) {
		c := h.New(t)
		_, resp, ok := h.authorize(t, c, OutcomeDeclined, connectors.CaptureAutomatic)
		if !ok {
			t.Skip("provider cannot decline synchronously")
		}
		if h.Classify(resp.RawStatus) != ClassFailed {
			t.Fatalf("status %q is not failed", resp.RawStatus)
		}
		if resp.ErrorCode == "" {
			t.Fatal("a decline must carry the provider's error code")
		}
	})

	t.Run("manual capture", func(t *testing.T) {
		c := h.New(t)
		if !c.Capabilities().ManualCapture {
			_, err := c.Capture(ctx, connectors.CaptureRequest{AttemptID: "x", ConnectorTransactionID: "x", Money: h.Money})
			if !errors.Is(err, connectors.ErrUnsupported) && !errors.Is(err, connectors.ErrNotFound) {
				t.Fatalf("Capture without ManualCapture err = %v, want ErrUnsupported", err)
			}
			t.Skip("provider has no manual capture")
		}
		req, resp, ok := h.authorize(t, c, OutcomeAuthorized, connectors.CaptureManual)
		if !ok {
			t.Skip("harness cannot produce an authorization")
		}
		if h.Classify(resp.RawStatus) != ClassAuthorized {
			t.Fatalf("status %q is not authorized", resp.RawStatus)
		}
		over := connectors.Money{Amount: h.Money.Amount.Add(decimal.NewFromInt(1)), Asset: h.Money.Asset}
		if _, err := c.Capture(ctx, connectors.CaptureRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Money: over}); !errors.Is(err, connectors.ErrInvalidRequest) {
			t.Fatalf("over-capture err = %v, want ErrInvalidRequest", err)
		}
		capResp, err := c.Capture(ctx, connectors.CaptureRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Money: h.Money})
		if err != nil || h.Classify(capResp.RawStatus) != ClassSettled || !capResp.AmountCaptured.Equal(h.Money.Amount) {
			t.Fatalf("Capture = %+v, %v", capResp, err)
		}
		declared(t, c, capResp.RawStatus)
		if _, err := c.Capture(ctx, connectors.CaptureRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Money: h.Money}); err == nil {
			t.Fatal("a second capture must fail")
		}

		if c.Capabilities().PartialCapture {
			req2, resp2, _ := h.authorize(t, c, OutcomeAuthorized, connectors.CaptureManual)
			half := connectors.Money{Amount: h.Money.Amount.Div(decimal.NewFromInt(2)), Asset: h.Money.Asset}
			part, err := c.Capture(ctx, connectors.CaptureRequest{AttemptID: req2.AttemptID, ConnectorTransactionID: resp2.ConnectorTransactionID, Money: half})
			if err != nil || !part.AmountCaptured.Equal(half.Amount) {
				t.Fatalf("partial Capture = %+v, %v", part, err)
			}
			declared(t, c, part.RawStatus)
		}
	})

	t.Run("void", func(t *testing.T) {
		c := h.New(t)
		if !c.Capabilities().Void {
			if _, err := c.Void(ctx, connectors.VoidRequest{AttemptID: "x", ConnectorTransactionID: "x"}); !errors.Is(err, connectors.ErrUnsupported) && !errors.Is(err, connectors.ErrNotFound) {
				t.Fatalf("Void without capability err = %v, want ErrUnsupported", err)
			}
			t.Skip("provider cannot void")
		}
		outcome := OutcomeAuthorized
		capture := connectors.CaptureManual
		if !c.Capabilities().ManualCapture {
			outcome, capture = OutcomePending, connectors.CaptureAutomatic
		}
		req, resp, ok := h.authorize(t, c, outcome, capture)
		if !ok {
			t.Skip("harness cannot produce a voidable transaction")
		}
		v, err := c.Void(ctx, connectors.VoidRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Reason: "conformance"})
		if err != nil || h.Classify(v.RawStatus) != ClassVoided {
			t.Fatalf("Void = %+v, %v", v, err)
		}
		declared(t, c, v.RawStatus)
		if reqS, respS, ok := h.authorize(t, c, OutcomeSettled, connectors.CaptureAutomatic); ok {
			if _, err := c.Void(ctx, connectors.VoidRequest{AttemptID: reqS.AttemptID, ConnectorTransactionID: respS.ConnectorTransactionID}); err == nil {
				t.Fatal("voiding settled money must fail")
			}
		}
	})

	t.Run("refund", func(t *testing.T) {
		c := h.New(t)
		if !c.Capabilities().Refund {
			if _, err := c.Refund(ctx, connectors.RefundRequest{RefundID: "x", AttemptID: "x", ConnectorTransactionID: "x", Money: h.Money}); !errors.Is(err, connectors.ErrUnsupported) {
				t.Fatalf("Refund without capability err = %v, want ErrUnsupported", err)
			}
			t.Skip("provider cannot refund")
		}
		req, resp, ok := h.authorize(t, c, OutcomeSettled, connectors.CaptureAutomatic)
		if !ok {
			t.Skip("harness cannot produce settled money to refund")
		}
		over := connectors.Money{Amount: h.Money.Amount.Add(decimal.NewFromInt(1)), Asset: h.Money.Asset}
		if _, err := c.Refund(ctx, connectors.RefundRequest{RefundID: "r0", AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Money: over}); !errors.Is(err, connectors.ErrInvalidRequest) {
			t.Fatalf("over-refund err = %v, want ErrInvalidRequest", err)
		}
		amount := h.Money
		if c.Capabilities().PartialRefund {
			amount = connectors.Money{Amount: h.Money.Amount.Div(decimal.NewFromInt(2)), Asset: h.Money.Asset}
		}
		r, err := c.Refund(ctx, connectors.RefundRequest{RefundID: "r1", AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID, Money: amount})
		if err != nil || r.ConnectorRefundID == "" {
			t.Fatalf("Refund = %+v, %v", r, err)
		}
		declaredRefund(t, c, r.RawStatus)
	})

	t.Run("pending then sync", func(t *testing.T) {
		c := h.New(t)
		if !c.Capabilities().Sync {
			if _, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: "x"}); !errors.Is(err, connectors.ErrUnsupported) {
				t.Fatalf("Sync without capability err = %v, want ErrUnsupported", err)
			}
			t.Skip("provider cannot sync")
		}
		if _, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: "never", ConnectorTransactionID: "never"}); !errors.Is(err, connectors.ErrNotFound) {
			t.Fatalf("Sync of unknown err = %v, want ErrNotFound", err)
		}
		req, resp, ok := h.authorize(t, c, OutcomePending, connectors.CaptureAutomatic)
		if !ok {
			t.Skip("harness cannot produce a pending transaction")
		}
		if cls := h.Classify(resp.RawStatus); cls != ClassPending && cls != ClassActionRequired {
			t.Fatalf("status %q is not pending", resp.RawStatus)
		}
		if h.Settle == nil {
			t.Skip("harness cannot settle out of band")
		}
		h.Settle(t, c, req.AttemptID, resp.ConnectorTransactionID)
		sync, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID})
		if err != nil || h.Classify(sync.RawStatus) != ClassSettled {
			t.Fatalf("Sync after settle = %+v, %v", sync, err)
		}
		declared(t, c, sync.RawStatus)
		if sync.ConnectorTransactionID != "" && sync.ConnectorTransactionID != resp.ConnectorTransactionID {
			t.Fatalf("Sync returned transaction %q, want %q", sync.ConnectorTransactionID, resp.ConnectorTransactionID)
		}
	})

	t.Run("webhook", func(t *testing.T) {
		c := h.New(t)
		if !c.Capabilities().Webhooks {
			if _, err := c.VerifyWebhook(ctx, http.Header{}, []byte("{}")); !errors.Is(err, connectors.ErrUnsupported) {
				t.Fatalf("VerifyWebhook without capability err = %v, want ErrUnsupported", err)
			}
			t.Skip("provider has no webhooks")
		}
		if h.Webhook == nil {
			t.Fatal("provider declares webhooks but the harness cannot build one")
		}
		req, resp, ok := h.authorize(t, c, OutcomePending, connectors.CaptureAutomatic)
		if !ok {
			t.Skip("harness cannot produce a pending transaction")
		}
		headers, body := h.Webhook(t, c, "conf_evt_1", resp.ConnectorTransactionID)
		ev, err := c.VerifyWebhook(ctx, headers, body)
		if err != nil || ev.EventID != "conf_evt_1" || ev.ConnectorTransactionID != resp.ConnectorTransactionID {
			t.Fatalf("VerifyWebhook = %+v, %v", ev, err)
		}
		if h.Classify(ev.RawStatus) != ClassSettled {
			t.Fatalf("webhook status %q is not settled", ev.RawStatus)
		}
		declared(t, c, ev.RawStatus)
		again, err := c.VerifyWebhook(ctx, headers, body)
		if err != nil || again.EventID != ev.EventID {
			t.Fatalf("a repeated delivery must verify with the same event id (the switch rejects the replay): %+v, %v", again, err)
		}
		if _, err := c.VerifyWebhook(ctx, http.Header{}, body); !errors.Is(err, connectors.ErrWebhookSignature) {
			t.Fatalf("unsigned delivery err = %v, want ErrWebhookSignature", err)
		}
		tampered := append([]byte{}, body...)
		tampered[len(tampered)-1] ^= 0x01
		if _, err := c.VerifyWebhook(ctx, headers, tampered); !errors.Is(err, connectors.ErrWebhookSignature) {
			t.Fatalf("tampered delivery err = %v, want ErrWebhookSignature", err)
		}
		if c.Capabilities().Sync {
			sync, err := c.Sync(ctx, connectors.SyncRequest{AttemptID: req.AttemptID, ConnectorTransactionID: resp.ConnectorTransactionID})
			if err != nil || h.Classify(sync.RawStatus) != ClassSettled {
				t.Fatalf("Sync after webhook = %+v, %v", sync, err)
			}
		}
	})
}
