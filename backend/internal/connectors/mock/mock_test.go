package mock

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/shopspring/decimal"
)

func usd(n int64) connectors.Money {
	return connectors.Money{Amount: decimal.NewFromInt(n), Asset: "USD"}
}

var attemptSeq int

func authorize(t *testing.T, c *Connector, scenario string, capture connectors.CaptureMethod) (connectors.AuthorizeResponse, error) {
	t.Helper()
	attemptSeq++
	return c.Authorize(context.Background(), connectors.AuthorizeRequest{
		AttemptID:     fmt.Sprintf("pa_%d", attemptSeq),
		Money:         usd(100),
		CaptureMethod: capture,
		PaymentMethod: connectors.PaymentMethod{Type: connectors.MethodCard, Token: scenario},
	})
}

func TestAuthorize_Scenarios(t *testing.T) {
	cases := []struct {
		scenario string
		capture  connectors.CaptureMethod
		status   connectors.RawStatus
		err      error
	}{
		{ScenarioSuccess, connectors.CaptureAutomatic, StatusCaptured, nil},
		{"", connectors.CaptureAutomatic, StatusCaptured, nil},
		{ScenarioSuccess, connectors.CaptureManual, StatusAuthorized, nil},
		{ScenarioDecline, connectors.CaptureAutomatic, StatusDeclined, nil},
		{ScenarioRequiresAction, connectors.CaptureAutomatic, StatusActionRequired, nil},
		{ScenarioAsync, connectors.CaptureAutomatic, StatusPending, nil},
		{ScenarioTimeout, connectors.CaptureAutomatic, "", connectors.ErrTimeout},
		{"bogus", connectors.CaptureAutomatic, "", connectors.ErrInvalidRequest},
	}
	for _, tc := range cases {
		c := New()
		resp, err := authorize(t, c, tc.scenario, tc.capture)
		if !errors.Is(err, tc.err) {
			t.Fatalf("%s: err = %v, want %v", tc.scenario, err, tc.err)
		}
		if resp.RawStatus != tc.status {
			t.Fatalf("%s: status = %s, want %s", tc.scenario, resp.RawStatus, tc.status)
		}
	}
}

func TestTimeout_TransactionIsFoundBySync(t *testing.T) {
	c := New()
	if _, err := authorize(t, c, ScenarioTimeout, connectors.CaptureAutomatic); !errors.Is(err, connectors.ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
	sync, err := c.Sync(context.Background(), connectors.SyncRequest{AttemptID: fmt.Sprintf("pa_%d", attemptSeq)})
	if err != nil || sync.ConnectorTransactionID != c.LastTransactionID() || sync.RawStatus != StatusCaptured || sync.AmountCaptured == nil || !sync.AmountCaptured.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("sync = %+v, %v", sync, err)
	}
}

func TestCapture_FullAndPartial(t *testing.T) {
	c := New()
	auth, _ := authorize(t, c, ScenarioSuccess, connectors.CaptureManual)
	ctx := context.Background()
	if _, err := c.Capture(ctx, connectors.CaptureRequest{ConnectorTransactionID: auth.ConnectorTransactionID, Money: usd(101)}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("over-capture err = %v", err)
	}
	cap, err := c.Capture(ctx, connectors.CaptureRequest{ConnectorTransactionID: auth.ConnectorTransactionID, Money: usd(40)})
	if err != nil || cap.RawStatus != StatusPartiallyCaptured || !cap.AmountCaptured.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("partial capture = %+v, %v", cap, err)
	}
	if _, err := c.Capture(ctx, connectors.CaptureRequest{ConnectorTransactionID: auth.ConnectorTransactionID, Money: usd(60)}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("second capture must be refused, err = %v", err)
	}

	auth2, _ := authorize(t, c, ScenarioSuccess, connectors.CaptureManual)
	cap2, err := c.Capture(ctx, connectors.CaptureRequest{ConnectorTransactionID: auth2.ConnectorTransactionID, Money: usd(100)})
	if err != nil || cap2.RawStatus != StatusCaptured {
		t.Fatalf("full capture = %+v, %v", cap2, err)
	}
}

func TestVoidAndRefundRules(t *testing.T) {
	c := New()
	ctx := context.Background()
	auth, _ := authorize(t, c, ScenarioSuccess, connectors.CaptureManual)
	v, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: auth.ConnectorTransactionID})
	if err != nil || v.RawStatus != StatusVoided {
		t.Fatalf("void = %+v, %v", v, err)
	}
	if _, err := c.Refund(ctx, connectors.RefundRequest{ConnectorTransactionID: auth.ConnectorTransactionID, Money: usd(1)}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("refund of voided must fail, err = %v", err)
	}

	paid, _ := authorize(t, c, ScenarioSuccess, connectors.CaptureAutomatic)
	if _, err := c.Void(ctx, connectors.VoidRequest{ConnectorTransactionID: paid.ConnectorTransactionID}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("void of captured must fail, err = %v", err)
	}
	r1, err := c.Refund(ctx, connectors.RefundRequest{ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(30)})
	if err != nil || r1.RawStatus != RefundDone {
		t.Fatalf("refund = %+v, %v", r1, err)
	}
	if _, err := c.Refund(ctx, connectors.RefundRequest{ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(71)}); !errors.Is(err, connectors.ErrInvalidRequest) {
		t.Fatalf("refund over balance must fail, err = %v", err)
	}
	r2, err := c.Refund(ctx, connectors.RefundRequest{ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(70), Reason: ScenarioRefundAsync})
	if err != nil || r2.RawStatus != RefundPending {
		t.Fatalf("async refund = %+v, %v", r2, err)
	}
	r2, err = c.Refund(ctx, connectors.RefundRequest{ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(1), Reason: ScenarioRefundAsync})
	if err != nil || r2.RawStatus != RefundPending {
		t.Fatalf("async refund = %+v, %v", r2, err)
	}
	if _, err := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: "nope"}); !errors.Is(err, connectors.ErrNotFound) {
		t.Fatalf("sync unknown err = %v", err)
	}
}

func TestVerifyWebhook(t *testing.T) {
	c := New(WithSecret("s3cret"))
	ctx := context.Background()
	auth, _ := authorize(t, c, ScenarioAsync, connectors.CaptureAutomatic)
	headers, body := c.SignWebhook(Event{EventID: "evt_1", TransactionID: auth.ConnectorTransactionID, Status: string(StatusCaptured)})

	ev, err := c.VerifyWebhook(ctx, headers, body)
	if err != nil || ev.EventID != "evt_1" || ev.RawStatus != StatusCaptured || ev.Kind != connectors.WebhookPayment {
		t.Fatalf("verify = %+v, %v", ev, err)
	}
	sync, _ := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: auth.ConnectorTransactionID})
	if sync.RawStatus != StatusCaptured {
		t.Fatalf("webhook must move the mock's own state, sync = %s", sync.RawStatus)
	}

	bad := http.Header{}
	bad.Set(SignatureHeader, "deadbeef")
	bad.Set(TimestampHeader, headers.Get(TimestampHeader))
	if _, err := c.VerifyWebhook(ctx, bad, body); !errors.Is(err, connectors.ErrWebhookSignature) {
		t.Fatalf("bad signature err = %v", err)
	}
	if _, err := c.VerifyWebhook(ctx, http.Header{}, body); !errors.Is(err, connectors.ErrWebhookSignature) {
		t.Fatalf("missing signature err = %v", err)
	}
	other := New(WithSecret("other"))
	if _, err := other.VerifyWebhook(ctx, headers, body); !errors.Is(err, connectors.ErrWebhookSignature) {
		t.Fatalf("wrong secret err = %v", err)
	}
	h2, b2 := c.SignWebhook(Event{TransactionID: auth.ConnectorTransactionID, Status: "captured"})
	if _, err := c.VerifyWebhook(ctx, h2, b2); !errors.Is(err, connectors.ErrWebhookMalformed) {
		t.Fatalf("empty event id err = %v", err)
	}
	h3, b3 := c.SignWebhookAt(Event{EventID: "evt_old", TransactionID: auth.ConnectorTransactionID, Status: "captured"}, time.Now().Add(-WebhookWindow-time.Second))
	if _, err := c.VerifyWebhook(ctx, h3, b3); !errors.Is(err, connectors.ErrWebhookStale) {
		t.Fatalf("stale err = %v", err)
	}
}

func TestIdempotencyKeys_ReplayFirstOutcomeEvenAfterAnUnansweredCall(t *testing.T) {
	c := New()
	ctx := context.Background()
	auth, _ := authorize(t, c, ScenarioCaptureTimeoutLand, connectors.CaptureManual)
	req := connectors.CaptureRequest{AttemptID: "pa_x", ConnectorTransactionID: auth.ConnectorTransactionID, Money: usd(100), IdempotencyKey: "cap-1"}
	if _, err := c.Capture(ctx, req); !errors.Is(err, connectors.ErrTimeout) {
		t.Fatalf("first capture err = %v, want timeout", err)
	}
	again, err := c.Capture(ctx, req)
	if err != nil || again.RawStatus != StatusCaptured || !again.AmountCaptured.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("repeat = %+v, %v; the landed capture must replay as its result", again, err)
	}
	if c.Calls("capture") != 2 {
		t.Fatalf("capture calls = %d", c.Calls("capture"))
	}

	lost, _ := authorize(t, c, ScenarioCaptureTimeoutLost, connectors.CaptureManual)
	lostReq := connectors.CaptureRequest{AttemptID: "pa_y", ConnectorTransactionID: lost.ConnectorTransactionID, Money: usd(100), IdempotencyKey: "cap-2"}
	if _, err := c.Capture(ctx, lostReq); !errors.Is(err, connectors.ErrTimeout) {
		t.Fatalf("lost capture err = %v", err)
	}
	sync, _ := c.Sync(ctx, connectors.SyncRequest{ConnectorTransactionID: lost.ConnectorTransactionID})
	if sync.RawStatus != StatusAuthorized {
		t.Fatalf("a lost capture must leave the authorization, got %s", sync.RawStatus)
	}

	paid, _ := authorize(t, c, ScenarioSuccess, connectors.CaptureAutomatic)
	rreq := connectors.RefundRequest{RefundID: "re_1", ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(10), Reason: ScenarioRefundErrorLand, IdempotencyKey: "re_1"}
	if _, err := c.Refund(ctx, rreq); err == nil || errors.Is(err, connectors.ErrDeclined) {
		t.Fatalf("refund error-landed err = %v", err)
	}
	rreq.Reason = ""
	r, err := c.Refund(ctx, rreq)
	if err != nil || r.RawStatus != RefundDone {
		t.Fatalf("repeat refund = %+v, %v", r, err)
	}
	sr, err := c.SyncRefund(ctx, connectors.SyncRefundRequest{RefundID: "re_1"})
	if err != nil || sr.ConnectorRefundID != r.ConnectorRefundID {
		t.Fatalf("sync refund by our id = %+v, %v", sr, err)
	}
	if _, err := c.Refund(ctx, connectors.RefundRequest{RefundID: "re_2", ConnectorTransactionID: paid.ConnectorTransactionID, Money: usd(10), Reason: ScenarioRefundFail, IdempotencyKey: "re_2"}); !errors.Is(err, connectors.ErrDeclined) {
		t.Fatalf("refund fail must be a typed decline, err = %v", err)
	}
}
