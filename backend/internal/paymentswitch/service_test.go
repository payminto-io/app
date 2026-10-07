package paymentswitch_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const merchant = "merchant-1"

// recordingLedger wraps the real ledger so tests assert what crossed the port, then check the rows too.
type recordingLedger struct {
	inner *ledger.Service
	mu    sync.Mutex
	posts []ledger.Journal
}

func (r *recordingLedger) PostIn(ctx context.Context, tx *gorm.DB, j ledger.Journal) (ledger.Receipt, error) {
	r.mu.Lock()
	r.posts = append(r.posts, j)
	r.mu.Unlock()
	return r.inner.PostIn(ctx, tx, j)
}

func (r *recordingLedger) ofKind(kind ledger.JournalKind) []ledger.Journal {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []ledger.Journal
	for _, j := range r.posts {
		if j.Kind == kind {
			out = append(out, j)
		}
	}
	return out
}

type fixture struct {
	t      *testing.T
	db     *gorm.DB
	svc    *paymentswitch.Service
	mock   *mock.Connector
	ledger *recordingLedger
	ctx    context.Context
}

// SQLite stores numeric as float, so unit tests use integer amounts; exactness is proven on Postgres.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for _, table := range []string{"switch_payment_intents", "switch_payment_attempts", "switch_refunds", "switch_webhook_events", "switch_status_transitions", "ledger_lines", "ledger_journals", "ledger_accounts"} {
		db.Exec("DROP TABLE IF EXISTS " + table)
	}
	if err := ledger.Migrate(db); err != nil {
		t.Fatalf("ledger migrate: %v", err)
	}
	if err := paymentswitch.Migrate(db); err != nil {
		t.Fatalf("switch migrate: %v", err)
	}
	reg := connectors.NewRegistry()
	m := mock.New()
	if err := reg.Register(m); err != nil {
		t.Fatalf("register mock: %v", err)
	}
	led := &recordingLedger{inner: ledger.New(db)}
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors{mock.Code}, Connectors: reg}
	svc := paymentswitch.New(db, reg, selector, led)
	return &fixture{t: t, db: db, svc: svc, mock: m, ledger: led, ctx: context.Background()}
}

func usd(n int64) paymentswitch.Money {
	return paymentswitch.Money{Amount: decimal.NewFromInt(n), Asset: "USD"}
}

func card(scenario string) *connectors.PaymentMethod {
	return &connectors.PaymentMethod{Type: connectors.MethodCard, Token: scenario}
}

func (f *fixture) create(cmd paymentswitch.CreateCommand) paymentswitch.Intent {
	f.t.Helper()
	if cmd.MerchantID == "" {
		cmd.MerchantID = merchant
	}
	if cmd.Money.Asset == "" {
		cmd.Money = usd(100)
	}
	in, err := f.svc.Create(f.ctx, cmd)
	if err != nil {
		f.t.Fatalf("Create: %v", err)
	}
	return in
}

func (f *fixture) get(id string) paymentswitch.View {
	f.t.Helper()
	v, err := f.svc.Get(f.ctx, merchant, id)
	if err != nil {
		f.t.Fatalf("Get: %v", err)
	}
	return v
}

func (f *fixture) wantStatus(in paymentswitch.Intent, want paymentswitch.IntentStatus) {
	f.t.Helper()
	if in.Status != want {
		f.t.Fatalf("intent %s status = %s, want %s (error %s %s)", in.ID, in.Status, want, in.LastErrorCode, in.LastErrorMessage)
	}
}

func (f *fixture) wantAttempt(id string, want paymentswitch.AttemptStatus) paymentswitch.Attempt {
	f.t.Helper()
	v := f.get(id)
	if len(v.Attempts) == 0 {
		f.t.Fatalf("intent %s has no attempts", id)
	}
	a := v.Attempts[len(v.Attempts)-1]
	if a.Status != want {
		f.t.Fatalf("attempt %s status = %s (raw %s), want %s", a.ID, a.Status, a.RawStatus, want)
	}
	return a
}

func (f *fixture) wantPayments(n int) {
	f.t.Helper()
	if got := len(f.ledger.ofKind(ledger.KindPayment)); got != n {
		f.t.Fatalf("payment journals posted = %d, want %d", got, n)
	}
	var rows int64
	f.db.Model(&ledger.JournalRow{}).Where("kind = ?", ledger.KindPayment).Count(&rows)
	if int(rows) != n {
		f.t.Fatalf("payment journal rows = %d, want %d", rows, n)
	}
}

func (f *fixture) wantRefundJournals(n int) {
	f.t.Helper()
	if got := len(f.ledger.ofKind(ledger.KindRefund)); got != n {
		f.t.Fatalf("refund journals posted = %d, want %d", got, n)
	}
	var rows int64
	f.db.Model(&ledger.JournalRow{}).Where("kind = ?", ledger.KindRefund).Count(&rows)
	if int(rows) != n {
		f.t.Fatalf("refund journal rows = %d, want %d", rows, n)
	}
}

func TestCreate_StatusDependsOnPaymentMethod(t *testing.T) {
	f := newFixture(t)
	bare := f.create(paymentswitch.CreateCommand{})
	f.wantStatus(bare, paymentswitch.IntentRequiresPaymentMethod)
	if bare.IdempotencyKey == "" || bare.CaptureMethod != connectors.CaptureAutomatic {
		t.Fatalf("defaults not applied: %+v", bare)
	}
	withPM := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess)})
	f.wantStatus(withPM, paymentswitch.IntentRequiresConfirmation)

	for name, cmd := range map[string]paymentswitch.CreateCommand{
		"zero amount":        {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.Zero, Asset: "USD"}},
		"no asset":           {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.NewFromInt(1)}},
		"no merchant":        {Money: usd(1)},
		"confirm without pm": {MerchantID: merchant, Money: usd(1), Confirm: true},
		"bad capture method": {MerchantID: merchant, Money: usd(1), CaptureMethod: "later"},
		"pm without type":    {MerchantID: merchant, Money: usd(1), PaymentMethod: &connectors.PaymentMethod{Token: "x"}},
		"key too long":       {MerchantID: merchant, Money: usd(1), IdempotencyKey: string(make([]byte, 129))},
		"padded asset":       {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.NewFromInt(1), Asset: " USD"}},
	} {
		if _, err := f.svc.Create(f.ctx, cmd); !errors.Is(err, paymentswitch.ErrInvalid) {
			t.Errorf("%s: err = %v, want ErrInvalid", name, err)
		}
	}
}

func TestCreate_IdempotentOnMerchantAndKey(t *testing.T) {
	f := newFixture(t)
	cmd := paymentswitch.CreateCommand{MerchantID: merchant, IdempotencyKey: "order-42", Money: usd(100), PaymentMethod: card(mock.ScenarioSuccess), Metadata: map[string]string{"order": "42"}}
	first := f.create(cmd)
	again := f.create(cmd)
	if again.ID != first.ID {
		t.Fatalf("replay returned %s, want %s", again.ID, first.ID)
	}
	var count int64
	f.db.Model(&paymentswitch.IntentRow{}).Count(&count)
	if count != 1 {
		t.Fatalf("intents = %d, want 1", count)
	}

	changed := cmd
	changed.Money = usd(101)
	if _, err := f.svc.Create(f.ctx, changed); !errors.Is(err, paymentswitch.ErrIdempotencyConflict) {
		t.Fatalf("different body err = %v, want ErrIdempotencyConflict", err)
	}
	other := cmd
	other.MerchantID = "merchant-2"
	if in := f.create(other); in.ID == first.ID {
		t.Fatalf("another merchant must get its own intent")
	}
}

func TestCreate_ConfirmReplayDoesNotRunASecondAttempt(t *testing.T) {
	f := newFixture(t)
	cmd := paymentswitch.CreateCommand{IdempotencyKey: "k1", PaymentMethod: card(mock.ScenarioSuccess), Confirm: true}
	first := f.create(cmd)
	f.wantStatus(first, paymentswitch.IntentSucceeded)
	again := f.create(cmd)
	if again.ID != first.ID || again.Status != paymentswitch.IntentSucceeded {
		t.Fatalf("replay = %+v", again)
	}
	if n := len(f.get(first.ID).Attempts); n != 1 {
		t.Fatalf("attempts = %d, want 1", n)
	}
	f.wantPayments(1)
}

func TestFlow_CaptureNow(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentSucceeded)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	if a.ConnectorTransactionID == "" || !a.AmountCaptured.Equal(decimal.NewFromInt(100)) || !in.AmountCaptured.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("attempt = %+v intent captured = %s", a, in.AmountCaptured)
	}
	f.wantPayments(1)
	j := f.ledger.ofKind(ledger.KindPayment)[0]
	if j.IdempotencyKey != "switch.payment."+a.ID || len(j.Lines) != 2 {
		t.Fatalf("journal = %+v", j)
	}
	if j.Lines[0].Account.OwnerType != ledger.OwnerConnector || j.Lines[1].Account.OwnerID != merchant || !j.Lines[1].Amount.Equal(decimal.NewFromInt(-100)) {
		t.Fatalf("journal lines = %+v", j.Lines)
	}
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("confirm after success err = %v", err)
	}
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("capture after success err = %v", err)
	}
	if _, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("cancel after success err = %v", err)
	}
	f.wantPayments(1)
}

func TestFlow_AuthorizeThenCapture(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess)})
	f.wantStatus(in, paymentswitch.IntentRequiresConfirmation)
	in, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentRequiresCapture)
	f.wantAttempt(in.ID, paymentswitch.AttemptAuthorized)
	f.wantPayments(0)

	too := decimal.NewFromInt(101)
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &too}); !errors.Is(err, paymentswitch.ErrAmountExceeds) {
		t.Fatalf("over-capture err = %v", err)
	}
	in, err = f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentSucceeded)
	f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	f.wantPayments(1)
}

func TestFlow_PartialCapture(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentRequiresCapture)
	part := decimal.NewFromInt(40)
	in, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &part})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentPartiallyCaptured)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPartialCharged)
	if !a.AmountCaptured.Equal(part) || !in.AmountCaptured.Equal(part) {
		t.Fatalf("captured = %s / %s, want 40", a.AmountCaptured, in.AmountCaptured)
	}
	f.wantPayments(1)
	if !f.ledger.ofKind(ledger.KindPayment)[0].Lines[0].Amount.Equal(part) {
		t.Fatalf("journal must carry the captured amount, got %s", f.ledger.ofKind(ledger.KindPayment)[0].Lines[0].Amount)
	}
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("second capture err = %v", err)
	}
}

func TestFlow_Void(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	in, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{Reason: "customer changed mind"})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentCancelled)
	f.wantAttempt(in.ID, paymentswitch.AttemptVoided)
	f.wantPayments(0)
	var transitions []paymentswitch.TransitionRow
	f.db.Where("entity_id = ?", in.ActiveAttemptID).Order("id").Find(&transitions)
	var seq []string
	for _, tr := range transitions {
		seq = append(seq, tr.ToStatus)
	}
	if fmt.Sprint(seq) != "[started authorized void_initiated voided]" {
		t.Fatalf("attempt transitions = %v", seq)
	}
	if transitions[len(transitions)-1].Reason != "cancel: customer changed mind" {
		t.Fatalf("reason = %q", transitions[len(transitions)-1].Reason)
	}
}

func TestFlow_CancelBeforeAnyAttempt(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess)})
	in, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentCancelled)
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("confirm after cancel err = %v", err)
	}
}

func TestFlow_RefundFullAndPartial(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	thirty := decimal.NewFromInt(30)
	r1, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Amount: &thirty})
	if err != nil || r1.Status != paymentswitch.RefundSucceeded || r1.ConnectorRefundID == "" {
		t.Fatalf("partial refund = %+v, %v", r1, err)
	}
	f.wantRefundJournals(1)
	again, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Amount: &thirty})
	if err != nil || again.ID != r1.ID {
		t.Fatalf("replayed refund = %+v, %v", again, err)
	}
	f.wantRefundJournals(1)
	forty := decimal.NewFromInt(40)
	if _, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Amount: &forty}); !errors.Is(err, paymentswitch.ErrIdempotencyConflict) {
		t.Fatalf("same key different amount err = %v", err)
	}
	eighty := decimal.NewFromInt(80)
	if _, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r2", Amount: &eighty}); !errors.Is(err, paymentswitch.ErrAmountExceeds) {
		t.Fatalf("over-refund err = %v", err)
	}
	r3, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r3"})
	if err != nil || !r3.Money.Amount.Equal(decimal.NewFromInt(70)) || r3.Status != paymentswitch.RefundSucceeded {
		t.Fatalf("remainder refund = %+v, %v", r3, err)
	}
	f.wantRefundJournals(2)
	v := f.get(in.ID)
	if !v.Intent.AmountRefunded.Equal(decimal.NewFromInt(100)) || v.Intent.Status != paymentswitch.IntentSucceeded || len(v.Refunds) != 2 {
		t.Fatalf("intent after refunds = %+v, refunds %d", v.Intent, len(v.Refunds))
	}
	if _, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r4"}); !errors.Is(err, paymentswitch.ErrAmountExceeds) {
		t.Fatalf("refund of nothing left err = %v", err)
	}
	j := f.ledger.ofKind(ledger.KindRefund)[0]
	if j.IdempotencyKey != "switch.refund."+r1.ID || !j.Lines[0].Amount.Equal(thirty) || j.Lines[0].Account.OwnerType != ledger.OwnerMember {
		t.Fatalf("refund journal = %+v", j)
	}
}

func TestFlow_RefundPendingThenWebhookPostsOnce(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	r, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Reason: mock.ScenarioRefundAsync})
	if err != nil || r.Status != paymentswitch.RefundPending {
		t.Fatalf("async refund = %+v, %v", r, err)
	}
	f.wantRefundJournals(0)
	a := f.get(in.ID).Attempts[0]
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_r1", Kind: "refund", TransactionID: a.ConnectorTransactionID, RefundID: r.ConnectorRefundID, Status: string(mock.RefundDone)})
	res, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || res.Ignored || res.IntentID != in.ID {
		t.Fatalf("refund webhook = %+v, %v", res, err)
	}
	f.wantRefundJournals(1)
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body); !errors.Is(err, paymentswitch.ErrWebhookReplay) {
		t.Fatalf("replay err = %v", err)
	}
	f.wantRefundJournals(1)
	if got := f.get(in.ID); got.Refunds[0].Status != paymentswitch.RefundSucceeded || !got.Intent.AmountRefunded.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("after webhook = %+v", got.Refunds[0])
	}
	failed, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r2"})
	if !errors.Is(err, paymentswitch.ErrAmountExceeds) {
		t.Fatalf("nothing left err = %v (%+v)", err, failed)
	}
}

func TestFlow_RefundFailedAtConnector(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	r, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Reason: mock.ScenarioRefundFail})
	if err != nil || r.Status != paymentswitch.RefundFailed {
		t.Fatalf("failed refund = %+v, %v", r, err)
	}
	f.wantRefundJournals(0)
	r2, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r2"})
	if err != nil || !r2.Money.Amount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("a failed refund must free the balance: %+v, %v", r2, err)
	}
}

func TestFlow_Decline(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioDecline), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentFailed)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptAuthorizationFailed)
	if a.ErrorCode != "do_not_honor" || in.LastErrorCode != "do_not_honor" || a.RawStatus != mock.StatusDeclined {
		t.Fatalf("decline detail missing: attempt %+v intent %+v", a, in)
	}
	f.wantPayments(0)

	retried, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{PaymentMethod: card(mock.ScenarioSuccess)})
	if err != nil {
		t.Fatalf("retry after decline: %v", err)
	}
	f.wantStatus(retried, paymentswitch.IntentSucceeded)
	if n := len(f.get(in.ID).Attempts); n != 2 {
		t.Fatalf("attempts after retry = %d, want 2", n)
	}
	f.wantPayments(1)
}

func TestFlow_RequiresActionThenWebhookSuccess(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioRequiresAction), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentRequiresAction)
	if in.NextAction == nil || in.NextAction.Type != "redirect" || in.NextAction.RedirectURL == "" {
		t.Fatalf("next action = %+v", in.NextAction)
	}
	a := f.wantAttempt(in.ID, paymentswitch.AttemptAuthenticationPending)

	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_1", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusCaptured)})
	res, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || res.Ignored || res.IntentID != in.ID {
		t.Fatalf("webhook = %+v, %v", res, err)
	}
	v := f.get(in.ID)
	f.wantStatus(v.Intent, paymentswitch.IntentSucceeded)
	if v.Intent.NextAction != nil {
		t.Fatalf("next action must clear after success")
	}
	f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	f.wantPayments(1)

	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body); !errors.Is(err, paymentswitch.ErrWebhookReplay) {
		t.Fatalf("replay err = %v, want ErrWebhookReplay", err)
	}
	f.wantPayments(1)

	// A late "authorized" after "captured" is out of order: recorded, ignored, nothing posted.
	h2, b2 := f.mock.SignWebhook(mock.Event{EventID: "evt_2", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusAuthorized)})
	res2, err := f.svc.HandleWebhook(f.ctx, mock.Code, h2, b2)
	if err != nil || !res2.Ignored {
		t.Fatalf("out of order webhook = %+v, %v", res2, err)
	}
	f.wantPayments(1)
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentSucceeded)
}

func TestWebhook_RejectsBadSignatureUnknownTransactionAndUnknownConnector(t *testing.T) {
	f := newFixture(t)
	_, body := f.mock.SignWebhook(mock.Event{EventID: "evt_x", TransactionID: "mock_tx_999", Status: "captured"})
	bad := http.Header{}
	bad.Set(mock.SignatureHeader, "0000")
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, bad, body); !errors.Is(err, connectors.ErrWebhookSignature) {
		t.Fatalf("bad signature err = %v", err)
	}
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_x", TransactionID: "mock_tx_999", Status: "captured"})
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body); !errors.Is(err, paymentswitch.ErrNotFound) {
		t.Fatalf("unknown transaction err = %v", err)
	}
	if _, err := f.svc.HandleWebhook(f.ctx, "stripe", h, body); !errors.Is(err, connectors.ErrUnknownConnector) {
		t.Fatalf("unknown connector err = %v", err)
	}
	var events int64
	f.db.Model(&paymentswitch.WebhookEventRow{}).Count(&events)
	if events != 0 {
		t.Fatalf("a failed delivery must not be recorded as processed, rows = %d", events)
	}
}

func TestFlow_TimeoutThenSync(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioTimeout), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentProcessing)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPending)
	if a.ConnectorTransactionID != "" {
		t.Fatalf("timeout must leave no connector id, got %q", a.ConnectorTransactionID)
	}
	f.wantPayments(0)
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("capture while processing err = %v", err)
	}

	synced, err := f.svc.Sync(f.ctx, merchant, in.ID)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	a = f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	if a.ConnectorTransactionID == "" {
		t.Fatalf("sync must store the connector transaction id")
	}
	f.wantPayments(1)
	if again, err := f.svc.Sync(f.ctx, merchant, in.ID); err != nil || again.Status != paymentswitch.IntentSucceeded {
		t.Fatalf("second sync = %+v, %v", again, err)
	}
	f.wantPayments(1)
}

func TestFlow_TimeoutThenSyncDecline(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioTimeoutThenFail), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentProcessing)
	synced, err := f.svc.Sync(f.ctx, merchant, in.ID)
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}
	f.wantStatus(synced, paymentswitch.IntentFailed)
	if synced.LastErrorCode != "do_not_honor" {
		t.Fatalf("decline detail from sync missing: %+v", synced)
	}
	f.wantPayments(0)
}

func TestFlow_AsyncThenWebhookAuthorizedThenCapture(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioAsync), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentProcessing)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPending)
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_a", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusAuthorized)})
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body); err != nil {
		t.Fatalf("webhook: %v", err)
	}
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentRequiresCapture)
	in, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentSucceeded)
	f.wantPayments(1)
}

func TestConfirm_SecondConfirmOnOneIntentLoses(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioAsync), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentProcessing)
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("second confirm err = %v", err)
	}
	if n := len(f.get(in.ID).Attempts); n != 1 {
		t.Fatalf("attempts = %d, want 1", n)
	}
}

func TestConfirm_RequiresAPaymentMethodAndAConnector(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{})
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{}); !errors.Is(err, paymentswitch.ErrInvalid) {
		t.Fatalf("confirm without pm err = %v", err)
	}
	chain := &connectors.PaymentMethod{Type: connectors.MethodChain, Token: "usdc"}
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{PaymentMethod: chain}); !errors.Is(err, paymentswitch.ErrNoConnector) {
		t.Fatalf("no connector for chain err = %v", err)
	}
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentRequiresPaymentMethod)
	done, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{PaymentMethod: card(mock.ScenarioSuccess)})
	if err != nil {
		t.Fatalf("confirm with pm: %v", err)
	}
	f.wantStatus(done, paymentswitch.IntentSucceeded)
	if done.PaymentMethodType != connectors.MethodCard || done.ConnectorCode != mock.Code {
		t.Fatalf("intent = %+v", done)
	}
}

func TestGet_IsMerchantScoped(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{})
	if _, err := f.svc.Get(f.ctx, "merchant-2", in.ID); !errors.Is(err, paymentswitch.ErrNotFound) {
		t.Fatalf("other merchant err = %v", err)
	}
	if _, err := f.svc.Cancel(f.ctx, "merchant-2", in.ID, paymentswitch.CancelCommand{}); !errors.Is(err, paymentswitch.ErrNotFound) {
		t.Fatalf("other merchant cancel err = %v", err)
	}
	if _, err := f.svc.Get(f.ctx, merchant, "pi_missing"); !errors.Is(err, paymentswitch.ErrNotFound) {
		t.Fatalf("missing err = %v", err)
	}
}

func TestSelector_FirstEnabledSupportingMethod(t *testing.T) {
	reg := connectors.NewRegistry()
	_ = reg.Register(mock.New())
	sel := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors{"ghost", mock.Code}, Connectors: reg}
	got, err := sel.Select(context.Background(), paymentswitch.SelectionRequest{MerchantID: merchant, Method: connectors.MethodCard})
	if err != nil || got.Code != mock.Code || got.Reason == "" {
		t.Fatalf("select = %+v, %v", got, err)
	}
	if _, err := sel.Select(context.Background(), paymentswitch.SelectionRequest{MerchantID: merchant, Method: connectors.MethodChain}); !errors.Is(err, paymentswitch.ErrNoConnector) {
		t.Fatalf("chain err = %v", err)
	}
}
