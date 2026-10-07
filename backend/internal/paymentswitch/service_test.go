package paymentswitch_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	merchant      = "merchant-1"
	chainMerchant = "7"
)

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

type recordingEvents struct {
	mu     sync.Mutex
	events []paymentswitch.Event
}

func (r *recordingEvents) Emit(_ context.Context, ev paymentswitch.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
	return nil
}

func (r *recordingEvents) ofType(t string) []paymentswitch.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []paymentswitch.Event
	for _, ev := range r.events {
		if ev.Type == t {
			out = append(out, ev)
		}
	}
	return out
}

// flaky wraps the mock so a test can script one raw, untyped error on Authorize (I2).
type flaky struct {
	connectors.Connector
	mu           sync.Mutex
	authorizeErr error
}

func (f *flaky) Authorize(ctx context.Context, req connectors.AuthorizeRequest) (connectors.AuthorizeResponse, error) {
	f.mu.Lock()
	err := f.authorizeErr
	f.authorizeErr = nil
	f.mu.Unlock()
	resp, callErr := f.Connector.Authorize(ctx, req)
	if err != nil {
		return connectors.AuthorizeResponse{}, err
	}
	return resp, callErr
}

type fixture struct {
	t      *testing.T
	db     *gorm.DB
	svc    *paymentswitch.Service
	mock   *mock.Connector
	flaky  *flaky
	chain  *chaindeposit.MemoryBackend
	ledger *recordingLedger
	events *recordingEvents
	logs   []string
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
	for _, table := range []string{"switch_payment_intents", "switch_payment_attempts", "switch_refunds", "switch_webhook_events", "switch_status_transitions", "switch_anomalies", "ledger_lines", "ledger_journals", "ledger_accounts"} {
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
	fl := &flaky{Connector: m}
	if err := reg.Register(fl); err != nil {
		t.Fatalf("register mock: %v", err)
	}
	chain := chaindeposit.NewMemoryBackend()
	if err := reg.Register(chaindeposit.New(chain)); err != nil {
		t.Fatalf("register chaindeposit: %v", err)
	}
	led := &recordingLedger{inner: ledger.New(db)}
	events := &recordingEvents{}
	f := &fixture{t: t, db: db, mock: m, flaky: fl, chain: chain, ledger: led, events: events, ctx: context.Background()}
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors{mock.Code, chaindeposit.Code}, Connectors: reg}
	f.svc = paymentswitch.New(db, reg, selector, led, paymentswitch.WithEvents(events), paymentswitch.WithLogger(func(format string, args ...any) {
		f.logs = append(f.logs, fmt.Sprintf(format, args...))
	}))
	return f
}

func usd(n int64) paymentswitch.Money {
	return paymentswitch.Money{Amount: decimal.NewFromInt(n), Asset: "USD"}
}

func card(scenario string) *connectors.PaymentMethod {
	return &connectors.PaymentMethod{Type: connectors.MethodCard, Token: scenario}
}

func usdcOnETH() *connectors.PaymentMethod {
	return &connectors.PaymentMethod{Type: connectors.MethodChain, Token: "usdc", Details: map[string]string{chaindeposit.DetailChain: "ETH", chaindeposit.DetailAsset: "USDC"}}
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

func (f *fixture) createChain(amount int64) paymentswitch.Intent {
	f.t.Helper()
	in, err := f.svc.Create(f.ctx, paymentswitch.CreateCommand{MerchantID: chainMerchant, PlatformID: "3", Money: usd(amount), PaymentMethod: usdcOnETH(), Confirm: true})
	if err != nil {
		f.t.Fatalf("Create chain: %v", err)
	}
	return in
}

func (f *fixture) getAs(merchantID, id string) paymentswitch.View {
	f.t.Helper()
	v, err := f.svc.Get(f.ctx, merchantID, id)
	if err != nil {
		f.t.Fatalf("Get: %v", err)
	}
	return v
}

func (f *fixture) get(id string) paymentswitch.View { return f.getAs(merchant, id) }

func (f *fixture) wantStatus(in paymentswitch.Intent, want paymentswitch.IntentStatus) {
	f.t.Helper()
	if in.Status != want {
		f.t.Fatalf("intent %s status = %s, want %s (error %s %s)", in.ID, in.Status, want, in.LastErrorCode, in.LastErrorMessage)
	}
}

func (f *fixture) wantAttemptAs(merchantID, id string, want paymentswitch.AttemptStatus) paymentswitch.Attempt {
	f.t.Helper()
	v := f.getAs(merchantID, id)
	if len(v.Attempts) == 0 {
		f.t.Fatalf("intent %s has no attempts", id)
	}
	a := v.Attempts[len(v.Attempts)-1]
	if a.Status != want {
		f.t.Fatalf("attempt %s status = %s (raw %s), want %s", a.ID, a.Status, a.RawStatus, want)
	}
	return a
}

func (f *fixture) wantAttempt(id string, want paymentswitch.AttemptStatus) paymentswitch.Attempt {
	f.t.Helper()
	return f.wantAttemptAs(merchant, id, want)
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

func (f *fixture) anomalies(id string) []paymentswitch.AnomalyRow {
	f.t.Helper()
	var rows []paymentswitch.AnomalyRow
	f.db.Where("intent_id = ?", id).Order("id").Find(&rows)
	return rows
}

func (f *fixture) sync(merchantID, id string) paymentswitch.Intent {
	f.t.Helper()
	in, err := f.svc.Sync(f.ctx, merchantID, id)
	if err != nil {
		f.t.Fatalf("Sync: %v", err)
	}
	return in
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

	tooFine, _ := decimal.NewFromString("1.0000000000000000001")
	for name, cmd := range map[string]paymentswitch.CreateCommand{
		"zero amount":        {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.Zero, Asset: "USD"}},
		"no asset":           {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.NewFromInt(1)}},
		"no merchant":        {Money: usd(1)},
		"confirm without pm": {MerchantID: merchant, Money: usd(1), Confirm: true},
		"bad capture method": {MerchantID: merchant, Money: usd(1), CaptureMethod: "later"},
		"pm without type":    {MerchantID: merchant, Money: usd(1), PaymentMethod: &connectors.PaymentMethod{Token: "x"}},
		"key too long":       {MerchantID: merchant, Money: usd(1), IdempotencyKey: string(make([]byte, 129))},
		"padded asset":       {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.NewFromInt(1), Asset: " USD"}},
		"19 decimals":        {MerchantID: merchant, Money: paymentswitch.Money{Amount: tooFine, Asset: "USD"}},
		"too large":          {MerchantID: merchant, Money: paymentswitch.Money{Amount: decimal.New(1, 20), Asset: "USD"}},
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

// M4: a confirm that fails before any claim still hands back the intent, and the replay re-runs the confirm.
func TestCreate_ConfirmFailureBeforeClaimReturnsTheIntentAndReplayRetries(t *testing.T) {
	f := newFixture(t)
	cmd := paymentswitch.CreateCommand{MerchantID: merchant, IdempotencyKey: "k-bank", Money: usd(10), PaymentMethod: &connectors.PaymentMethod{Type: "carrier_pigeon", Token: "x"}, Confirm: true}
	in, err := f.svc.Create(f.ctx, cmd)
	if !errors.Is(err, paymentswitch.ErrNoConnector) || in.ID == "" {
		t.Fatalf("create = %+v, %v; want the intent id alongside ErrNoConnector", in, err)
	}
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentRequiresConfirmation)
	again, err := f.svc.Create(f.ctx, cmd)
	if !errors.Is(err, paymentswitch.ErrNoConnector) || again.ID != in.ID {
		t.Fatalf("replay must re-run the confirm: %+v, %v", again, err)
	}
	if n := len(f.get(in.ID).Attempts); n != 0 {
		t.Fatalf("attempts = %d, want 0", n)
	}
}

func TestFlow_CaptureNow(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentSucceeded)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	if a.ConnectorTransactionID == "" || !a.AmountCaptured.Equal(decimal.NewFromInt(100)) || !in.AmountCaptured.Equal(decimal.NewFromInt(100)) || a.AmountReceived != nil {
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
	if got := f.events.ofType(paymentswitch.EventPaymentSucceeded); len(got) != 1 || got[0].IntentID != in.ID || got[0].Payload["amount_captured"] != "100" {
		t.Fatalf("succeeded events = %+v", got)
	}
	for _, op := range []func() error{
		func() error {
			_, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{})
			return err
		},
		func() error {
			_, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
			return err
		},
		func() error {
			_, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{})
			return err
		},
	} {
		if err := op(); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
			t.Fatalf("operation after success err = %v", err)
		}
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
	a := f.wantAttempt(in.ID, paymentswitch.AttemptAuthorized)
	if a.AmountToCapture.IsPositive() {
		t.Fatalf("a manual authorization claims no capture amount yet, got %s", a.AmountToCapture)
	}
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
	part := decimal.NewFromInt(40)
	in, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &part})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(in, paymentswitch.IntentPartiallyCaptured)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPartialCharged)
	if !a.AmountCaptured.Equal(part) || !in.AmountCaptured.Equal(part) || !a.AmountToCapture.Equal(part) {
		t.Fatalf("captured = %s / %s (claimed %s), want 40", a.AmountCaptured, in.AmountCaptured, a.AmountToCapture)
	}
	f.wantPayments(1)
	if !f.ledger.ofKind(ledger.KindPayment)[0].Lines[0].Amount.Equal(part) {
		t.Fatalf("journal must carry the captured amount")
	}
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("second capture err = %v", err)
	}
}

// C2: the capture claim is exclusive; a second capture while one is in flight is refused unless it is the same retry.
func TestCapture_ClaimIsExclusiveAndRetriesReuseTheKey(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureTimeoutLand), Confirm: true})
	forty := decimal.NewFromInt(40)
	got, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &forty})
	if err != nil {
		t.Fatalf("Capture (timeout): %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentProcessing)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCaptureInitiated)
	if a.ErrorCode != paymentswitch.ErrorCodeTimeout || strings.Contains(a.ErrorMessage, "mock") {
		t.Fatalf("attempt error = %s %q; want the typed timeout code and a redacted message", a.ErrorCode, a.ErrorMessage)
	}
	f.wantPayments(0)
	sixty := decimal.NewFromInt(60)
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &sixty}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("different amount while in flight err = %v", err)
	}
	if _, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("cancel while capture is in flight err = %v", err)
	}
	retried, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &forty})
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	f.wantStatus(retried, paymentswitch.IntentPartiallyCaptured)
	if f.mock.Calls("capture") != 2 {
		t.Fatalf("capture calls = %d, want 2 (one timed out, one retry under the same key)", f.mock.Calls("capture"))
	}
	a = f.wantAttempt(in.ID, paymentswitch.AttemptPartialCharged)
	if !a.AmountCaptured.Equal(forty) {
		t.Fatalf("captured = %s, want 40", a.AmountCaptured)
	}
	f.wantPayments(1)
}

// C1: a capture that landed at the connector but errored on the way back is recovered by Sync, with one journal.
func TestCapture_ErrorAfterLandingIsRecoveredBySync(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureErrorLand), Confirm: true})
	got, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentProcessing)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCaptureInitiated)
	if a.ErrorCode != paymentswitch.ErrorCodeConnector || strings.Contains(a.ErrorMessage, "502") {
		t.Fatalf("error = %s %q", a.ErrorCode, a.ErrorMessage)
	}
	f.wantPayments(0)
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	f.wantPayments(1)
	f.sync(merchant, in.ID)
	f.wantPayments(1)
}

// C1 scenario A: a declined capture whose transaction the connector later reports captured.
func TestCapture_DeclinedThenConnectorEvidenceOfCaptureWins(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureDeclined), Confirm: true})
	got, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentRequiresCapture)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCaptureFailed)
	if a.ErrorCode != paymentswitch.ErrorCodeDeclined {
		t.Fatalf("error code = %s", a.ErrorCode)
	}
	if err := f.mock.Settle(a.ConnectorTransactionID, mock.StatusCaptured); err != nil {
		t.Fatal(err)
	}
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	f.wantPayments(1)
}

// I1: a lost capture or void leaves the authorization; Sync brings the attempt back to authorized.
func TestCapture_TimeoutLostThenSyncRestoresAuthorizationAndCaptureSucceeds(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureTimeoutLost), Confirm: true})
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f.wantAttempt(in.ID, paymentswitch.AttemptCaptureInitiated)
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentRequiresCapture)
	f.wantAttempt(in.ID, paymentswitch.AttemptAuthorized)
	f.wantPayments(0)
	again, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{})
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}
	f.wantStatus(again, paymentswitch.IntentSucceeded)
	f.wantPayments(1)
}

func TestCapture_TimeoutLandedThenSyncCharges(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureTimeoutLand), Confirm: true})
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	f.wantPayments(1)
}

func TestCancel_VoidTimeoutLostThenSyncThenCancelAgain(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioVoidTimeoutLost), Confirm: true})
	got, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentProcessing)
	f.wantAttempt(in.ID, paymentswitch.AttemptVoidInitiated)
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentRequiresCapture)
	f.wantAttempt(in.ID, paymentswitch.AttemptAuthorized)
	again, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{})
	if err != nil {
		t.Fatalf("second cancel: %v", err)
	}
	f.wantStatus(again, paymentswitch.IntentCancelled)
	f.wantAttempt(in.ID, paymentswitch.AttemptVoided)
}

func TestCancel_VoidTimeoutLandedThenRetryReusesKey(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioVoidTimeoutLand), Confirm: true})
	if _, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{}); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantAttempt(in.ID, paymentswitch.AttemptVoidInitiated)
	again, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{})
	if err != nil {
		t.Fatalf("retry cancel: %v", err)
	}
	f.wantStatus(again, paymentswitch.IntentCancelled)
	if f.mock.Calls("void") != 2 {
		t.Fatalf("void calls = %d", f.mock.Calls("void"))
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
	if got := f.events.ofType(paymentswitch.EventRefundSucceeded); len(got) != 2 || got[0].RefundID != r1.ID {
		t.Fatalf("refund events = %+v", got)
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
	res, err = f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || !res.Ignored || res.IgnoreWhy != "replay" {
		t.Fatalf("replay = %+v, %v; a replay is answered as ignored, never an error", res, err)
	}
	f.wantRefundJournals(1)
	if got := f.get(in.ID); got.Refunds[0].Status != paymentswitch.RefundSucceeded || !got.Intent.AmountRefunded.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("after webhook = %+v", got.Refunds[0])
	}
}

// I4: a pending refund on a connector without refund webhooks resolves through SyncRefund.
func TestFlow_RefundPendingResolvedBySync(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	r, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Reason: mock.ScenarioRefundAsync})
	if err != nil || r.Status != paymentswitch.RefundPending {
		t.Fatalf("async refund = %+v, %v", r, err)
	}
	f.sync(merchant, in.ID)
	f.wantRefundJournals(0)
	if err := f.mock.SettleRefund(r.ConnectorRefundID, mock.RefundDone); err != nil {
		t.Fatal(err)
	}
	f.sync(merchant, in.ID)
	f.wantRefundJournals(1)
	if got := f.get(in.ID); got.Refunds[0].Status != paymentswitch.RefundSucceeded {
		t.Fatalf("refund after sync = %+v", got.Refunds[0])
	}
}

// I2: a refund that landed but errored on the way back stays initiated, keeps the balance, and resolves by retry or sync.
func TestFlow_RefundUnknownOutcomeHoldsBalanceAndResolves(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	sixty := decimal.NewFromInt(60)
	r, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Amount: &sixty, Reason: mock.ScenarioRefundErrorLand})
	if err != nil || r.Status != paymentswitch.RefundInitiated || r.ErrorCode != paymentswitch.ErrorCodeConnector {
		t.Fatalf("refund = %+v, %v; an unknown outcome stays initiated", r, err)
	}
	f.wantRefundJournals(0)
	fifty := decimal.NewFromInt(50)
	if _, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r2", Amount: &fifty}); !errors.Is(err, paymentswitch.ErrAmountExceeds) {
		t.Fatalf("an initiated refund must hold the balance, err = %v", err)
	}
	again, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Amount: &sixty, Reason: mock.ScenarioRefundErrorLand})
	if err != nil || again.ID != r.ID || again.Status != paymentswitch.RefundSucceeded {
		t.Fatalf("retry under the same key = %+v, %v", again, err)
	}
	if f.mock.Calls("refund") != 2 {
		t.Fatalf("refund calls = %d", f.mock.Calls("refund"))
	}
	f.wantRefundJournals(1)

	r2, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r3", Reason: mock.ScenarioRefundTimeoutLand})
	if err != nil || r2.Status != paymentswitch.RefundInitiated {
		t.Fatalf("timeout refund = %+v, %v", r2, err)
	}
	f.sync(merchant, in.ID)
	f.wantRefundJournals(2)
	if got := f.get(in.ID); got.Refunds[1].Status != paymentswitch.RefundSucceeded || !got.Intent.AmountRefunded.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("after sync = %+v", got.Refunds[1])
	}
}

func TestFlow_RefundDeclinedAtConnector(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	r, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Reason: mock.ScenarioRefundFail})
	if err != nil || r.Status != paymentswitch.RefundFailed || r.ErrorCode != paymentswitch.ErrorCodeDeclined {
		t.Fatalf("declined refund = %+v, %v", r, err)
	}
	f.wantRefundJournals(0)
	r2, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r2"})
	if err != nil || !r2.Money.Amount.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("a declined refund must free the balance: %+v, %v", r2, err)
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
	if got := f.events.ofType(paymentswitch.EventPaymentFailed); len(got) != 1 || got[0].Payload["error_code"] != "do_not_honor" {
		t.Fatalf("failed events = %+v", got)
	}
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

// I2: an untyped error from Authorize is an unknown outcome: pending, no new attempt, redacted message, resolved by Sync.
func TestConfirm_UntypedErrorIsUnknownNotFailed(t *testing.T) {
	f := newFixture(t)
	f.flaky.authorizeErr = errors.New("read tcp: connection reset by peer after request was sent")
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentProcessing)
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPending)
	if a.ErrorCode != paymentswitch.ErrorCodeConnector || strings.Contains(a.ErrorMessage, "tcp") || strings.Contains(in.LastErrorMessage, "tcp") {
		t.Fatalf("raw error leaked: %s %q / %q", a.ErrorCode, a.ErrorMessage, in.LastErrorMessage)
	}
	var tr []paymentswitch.TransitionRow
	f.db.Where("entity_id = ? AND reason LIKE ?", a.ID, "%connection reset%").Find(&tr)
	if len(tr) != 1 {
		t.Fatalf("the raw error must be in the audit trail, found %d rows", len(tr))
	}
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("a second attempt must not open while one is unknown, err = %v", err)
	}
	if len(f.events.ofType(paymentswitch.EventPaymentFailed)) != 0 {
		t.Fatal("no failed event for an unknown outcome")
	}
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	if n := len(f.get(in.ID).Attempts); n != 1 || f.mock.Calls("authorize") != 1 {
		t.Fatalf("attempts = %d authorize calls = %d", n, f.mock.Calls("authorize"))
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

	// I5: a pending webhook does not erase the redirect the customer still needs.
	hp, bp := f.mock.SignWebhook(mock.Event{EventID: "evt_0", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusPending)})
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, hp, bp); err != nil {
		t.Fatalf("pending webhook: %v", err)
	}
	if got := f.get(in.ID).Intent; got.NextAction == nil || got.NextAction.RedirectURL == "" {
		t.Fatalf("next action erased by a pending webhook: %+v", got.NextAction)
	}
	synced := f.sync(merchant, in.ID)
	if synced.NextAction == nil {
		t.Fatal("next action erased by sync")
	}

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
	res, err = f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || !res.Ignored || res.IgnoreWhy != "replay" {
		t.Fatalf("replay = %+v, %v", res, err)
	}
	f.wantPayments(1)
	h2, b2 := f.mock.SignWebhook(mock.Event{EventID: "evt_2", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusAuthorized)})
	res2, err := f.svc.HandleWebhook(f.ctx, mock.Code, h2, b2)
	if err != nil || !res2.Ignored {
		t.Fatalf("out of order webhook = %+v, %v", res2, err)
	}
	f.wantPayments(1)
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentSucceeded)
	if an := f.anomalies(in.ID); len(an) != 1 || an[0].Kind != paymentswitch.AnomalyEvidenceAfterTerminal {
		t.Fatalf("evidence after terminal must be an anomaly, got %+v", an)
	}
}

// M10: money-in evidence after a void is recorded as an anomaly, not swallowed.
func TestWebhook_CapturedAfterVoidIsAnAnomaly(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	if _, err := f.svc.Cancel(f.ctx, merchant, in.ID, paymentswitch.CancelCommand{}); err != nil {
		t.Fatal(err)
	}
	a := f.wantAttempt(in.ID, paymentswitch.AttemptVoided)
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_late", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusCaptured), AmountCaptured: "100"})
	res, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || !res.Ignored {
		t.Fatalf("late capture webhook = %+v, %v", res, err)
	}
	f.wantPayments(0)
	an := f.anomalies(in.ID)
	if len(an) != 1 || an[0].Kind != paymentswitch.AnomalyEvidenceAfterTerminal || !strings.Contains(an[0].Detail, "captured") {
		t.Fatalf("anomalies = %+v", an)
	}
	if !strings.Contains(strings.Join(f.logs, "\n"), "ERROR anomaly") {
		t.Fatal("anomaly must be logged at error level")
	}
	got, err := f.svc.Anomalies(f.ctx, merchant, in.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("Anomalies = %+v, %v", got, err)
	}
}

// I6: a captured event without an amount after a partial claim settles the claimed amount, never the authorization.
func TestWebhook_CapturedWithoutAmountUsesTheClaimedCaptureAmount(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureTimeoutLost), Confirm: true})
	forty := decimal.NewFromInt(40)
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &forty}); err != nil {
		t.Fatal(err)
	}
	a := f.wantAttempt(in.ID, paymentswitch.AttemptCaptureInitiated)
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_cap", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusCaptured)})
	res, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
	if err != nil || res.Ignored {
		t.Fatalf("webhook = %+v, %v", res, err)
	}
	v := f.get(in.ID)
	if !v.Intent.AmountCaptured.Equal(forty) || !f.ledger.ofKind(ledger.KindPayment)[0].Lines[0].Amount.Equal(forty) {
		t.Fatalf("captured = %s, want the claimed 40", v.Intent.AmountCaptured)
	}

	// With neither a claim nor a connector amount the switch refuses to post and records why.
	in2 := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	a2 := f.wantAttempt(in2.ID, paymentswitch.AttemptAuthorized)
	h2, b2 := f.mock.SignWebhook(mock.Event{EventID: "evt_cap2", TransactionID: a2.ConnectorTransactionID, Status: string(mock.StatusCaptured)})
	res2, err := f.svc.HandleWebhook(f.ctx, mock.Code, h2, b2)
	if err != nil || !res2.Ignored {
		t.Fatalf("amount-less webhook = %+v, %v", res2, err)
	}
	f.wantPayments(1)
	f.wantAttempt(in2.ID, paymentswitch.AttemptAuthorized)
	if an := f.anomalies(in2.ID); len(an) != 1 || an[0].Kind != paymentswitch.AnomalyAmountUnknown {
		t.Fatalf("anomalies = %+v", an)
	}
}

func TestWebhook_RejectsBadSignatureStaleUnknownTransactionAndUnknownConnector(t *testing.T) {
	f := newFixture(t)
	headers, body := f.mock.SignWebhook(mock.Event{EventID: "evt_x", TransactionID: "mock_tx_999", Status: "captured"})
	bad := http.Header{}
	bad.Set(mock.SignatureHeader, "0000")
	bad.Set(mock.TimestampHeader, headers.Get(mock.TimestampHeader))
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, bad, body); !errors.Is(err, connectors.ErrWebhookSignature) {
		t.Fatalf("bad signature err = %v", err)
	}
	stale, sbody := f.mock.SignWebhookAt(mock.Event{EventID: "evt_s", TransactionID: "mock_tx_999", Status: "captured"}, time.Now().Add(-time.Hour))
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, stale, sbody); !errors.Is(err, connectors.ErrWebhookStale) {
		t.Fatalf("stale err = %v", err)
	}
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, headers, body); !errors.Is(err, paymentswitch.ErrNotFound) {
		t.Fatalf("unknown transaction err = %v", err)
	}
	if _, err := f.svc.HandleWebhook(f.ctx, "stripe", headers, body); !errors.Is(err, connectors.ErrUnknownConnector) {
		t.Fatalf("unknown connector err = %v", err)
	}
	hk, bk := f.mock.SignWebhook(mock.Event{EventID: "evt_k", Kind: "dispute", TransactionID: "mock_tx_999", Status: "captured"})
	if _, err := f.svc.HandleWebhook(f.ctx, mock.Code, hk, bk); !errors.Is(err, connectors.ErrWebhookMalformed) {
		t.Fatalf("unknown kind err = %v", err)
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
	if a.ConnectorTransactionID != "" || a.ErrorCode != paymentswitch.ErrorCodeTimeout {
		t.Fatalf("timeout must leave no connector id and a typed code, got %q %s", a.ConnectorTransactionID, a.ErrorCode)
	}
	f.wantPayments(0)
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); !errors.Is(err, paymentswitch.ErrInvalid) && !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("capture while pending err = %v", err)
	}
	synced := f.sync(merchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	a = f.wantAttempt(in.ID, paymentswitch.AttemptCharged)
	if a.ConnectorTransactionID == "" || a.ErrorCode != "" {
		t.Fatalf("sync must store the connector transaction id and clear the error, got %+v", a)
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
	synced := f.sync(merchant, in.ID)
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
	pigeon := &connectors.PaymentMethod{Type: "carrier_pigeon", Token: "x"}
	if _, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{PaymentMethod: pigeon}); !errors.Is(err, paymentswitch.ErrNoConnector) {
		t.Fatalf("no connector err = %v", err)
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

// I7, I9: a chain deposit through the switch posts received token amounts into custody by delta, keeps its address
// while open, and closes short as partially_paid with the shortfall recorded.
func TestChainDeposit_PartialThenFilledPostsByDeltaInTheReceivedAsset(t *testing.T) {
	f := newFixture(t)
	in := f.createChain(100)
	f.wantStatus(in, paymentswitch.IntentRequiresAction)
	if in.NextAction == nil || in.NextAction.Type != "pay_to_address" || in.NextAction.Address == "" || in.NextAction.Asset != "USDC" {
		t.Fatalf("next action = %+v", in.NextAction)
	}
	a := f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptAuthenticationPending)
	address := in.NextAction.Address

	synced := f.sync(chainMerchant, in.ID)
	if synced.NextAction == nil || synced.NextAction.Address != address {
		t.Fatalf("an open deposit must keep its address after sync: %+v", synced.NextAction)
	}
	f.wantPayments(0)

	if err := f.chain.Deposit(a.ConnectorTransactionID, decimal.NewFromInt(40)); err != nil {
		t.Fatal(err)
	}
	synced = f.sync(chainMerchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentRequiresAction)
	a = f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptPartiallyPaid)
	if a.AmountReceived == nil || !a.AmountReceived.Equal(decimal.NewFromInt(40)) || a.ReceivedAsset != "USDC.ETH" || !synced.AmountCaptured.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("partially paid attempt = %+v intent captured %s", a, synced.AmountCaptured)
	}
	if synced.NextAction == nil || synced.NextAction.Address != address {
		t.Fatalf("a partially paid deposit still needs its address: %+v", synced.NextAction)
	}
	f.wantPayments(1)
	j := f.ledger.ofKind(ledger.KindPayment)[0]
	if j.Lines[0].Account.OwnerType != ledger.OwnerPlatform || j.Lines[0].Account.OwnerID != "crypto_assets" || j.Lines[0].Account.Asset != "USDC.ETH" || !j.Lines[0].Amount.Equal(decimal.NewFromInt(40)) {
		t.Fatalf("custody line = %+v", j.Lines[0])
	}
	if j.Lines[1].Account.OwnerType != ledger.OwnerMember || j.Lines[1].Account.OwnerID != chainMerchant || j.Lines[1].Account.Asset != "USDC.ETH" {
		t.Fatalf("merchant line = %+v", j.Lines[1])
	}
	if j.Metadata["priced_asset"] != "USD" || j.Metadata["received_asset"] != "USDC.ETH" {
		t.Fatalf("journal metadata = %+v", j.Metadata)
	}
	f.sync(chainMerchant, in.ID)
	f.wantPayments(1)

	if err := f.chain.Deposit(a.ConnectorTransactionID, decimal.NewFromInt(60)); err != nil {
		t.Fatal(err)
	}
	synced = f.sync(chainMerchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptCharged)
	f.wantPayments(2)
	if !f.ledger.ofKind(ledger.KindPayment)[1].Lines[0].Amount.Equal(decimal.NewFromInt(60)) {
		t.Fatalf("second journal must carry the delta")
	}
	if synced.NextAction != nil || !synced.AmountCaptured.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("after fill = %+v", synced)
	}
	bal, err := f.ledger.inner.Balances(f.ctx, ledger.OwnerMember, chainMerchant)
	if err != nil || !bal["USDC.ETH"].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("merchant balance = %v, %v", bal, err)
	}
}

func TestChainDeposit_OverpaidPostsWhatArrived(t *testing.T) {
	f := newFixture(t)
	in := f.createChain(100)
	a := f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptAuthenticationPending)
	if err := f.chain.Deposit(a.ConnectorTransactionID, decimal.NewFromInt(130)); err != nil {
		t.Fatal(err)
	}
	synced := f.sync(chainMerchant, in.ID)
	f.wantStatus(synced, paymentswitch.IntentSucceeded)
	a = f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptOverpaid)
	if !a.AmountReceived.Equal(decimal.NewFromInt(130)) || !synced.AmountCaptured.Equal(decimal.NewFromInt(130)) {
		t.Fatalf("overpaid = %+v / %s", a, synced.AmountCaptured)
	}
	f.wantPayments(1)
	if !f.ledger.ofKind(ledger.KindPayment)[0].Lines[0].Amount.Equal(decimal.NewFromInt(130)) {
		t.Fatal("journal must carry the received amount")
	}
}

func TestChainDeposit_CancelWhilePartiallyPaidClosesShortWithFundsOnTheBooks(t *testing.T) {
	f := newFixture(t)
	in := f.createChain(100)
	a := f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptAuthenticationPending)
	if err := f.chain.Deposit(a.ConnectorTransactionID, decimal.NewFromInt(25)); err != nil {
		t.Fatal(err)
	}
	f.sync(chainMerchant, in.ID)
	f.wantPayments(1)
	got, err := f.svc.Cancel(f.ctx, chainMerchant, in.ID, paymentswitch.CancelCommand{Reason: "customer walked away"})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentPartiallyPaid)
	a = f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptUnderpaid)
	if !a.AmountReceived.Equal(decimal.NewFromInt(25)) || got.Metadata["shortfall"] != "75" || got.Metadata["received_asset"] != "USDC.ETH" {
		t.Fatalf("underpaid = %+v metadata %v", a, got.Metadata)
	}
	f.wantPayments(1)
	var tr []paymentswitch.TransitionRow
	f.db.Where("entity_id = ? AND to_status = ?", a.ID, "void_initiated").Find(&tr)
	if len(tr) != 1 || !strings.Contains(tr[0].Reason, "await refund") {
		t.Fatalf("cancel reason must say the funds await refund: %+v", tr)
	}
	if _, err := f.svc.Refund(f.ctx, chainMerchant, in.ID, paymentswitch.RefundCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) && !errors.Is(err, paymentswitch.ErrInvalid) {
		t.Fatalf("refund of a chain deposit is ticket 11, err = %v", err)
	}
	if _, err := f.svc.Cancel(f.ctx, chainMerchant, in.ID, paymentswitch.CancelCommand{}); !errors.Is(err, paymentswitch.ErrInvalidTransition) {
		t.Fatalf("cancel twice err = %v", err)
	}
}

func TestChainDeposit_CancelWhileOpen(t *testing.T) {
	f := newFixture(t)
	in := f.createChain(100)
	got, err := f.svc.Cancel(f.ctx, chainMerchant, in.ID, paymentswitch.CancelCommand{})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	f.wantStatus(got, paymentswitch.IntentCancelled)
	f.wantAttemptAs(chainMerchant, in.ID, paymentswitch.AttemptVoided)
	f.wantPayments(0)
}

// Reconciler: due rows are synced with backoff; rows older than MaxAge become anomalies, never a guessed status.
func TestReconciler_SyncsInFlightRowsAndFlagsStaleOnes(t *testing.T) {
	f := newFixture(t)
	in := f.create(paymentswitch.CreateCommand{CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioCaptureTimeoutLand), Confirm: true})
	if _, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{}); err != nil {
		t.Fatal(err)
	}
	paid := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	r, err := f.svc.Refund(f.ctx, merchant, paid.ID, paymentswitch.RefundCommand{IdempotencyKey: "r1", Reason: mock.ScenarioRefundAsync})
	if err != nil {
		t.Fatal(err)
	}
	rec := paymentswitch.NewReconciler(f.svc)
	f.db.Model(&paymentswitch.AttemptRow{}).Where("intent_id = ?", in.ID).Update("next_sync_at", nil)
	n, err := rec.RunOnce(f.ctx)
	if err != nil || n != 2 {
		t.Fatalf("RunOnce = %d, %v", n, err)
	}
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentSucceeded)
	f.wantPayments(2)
	if got := f.get(paid.ID).Refunds[0]; got.Status != paymentswitch.RefundPending {
		t.Fatalf("refund still pending at the provider, got %s", got.Status)
	}
	n, err = rec.RunOnce(f.ctx)
	if err != nil || n != 0 {
		t.Fatalf("second run must honour the backoff: %d, %v", n, err)
	}
	if err := f.mock.SettleRefund(r.ConnectorRefundID, mock.RefundDone); err != nil {
		t.Fatal(err)
	}
	f.db.Model(&paymentswitch.RefundRow{}).Where("id = ?", r.ID).Update("next_sync_at", nil)
	if _, err := rec.RunOnce(f.ctx); err != nil {
		t.Fatal(err)
	}
	if got := f.get(paid.ID).Refunds[0]; got.Status != paymentswitch.RefundSucceeded {
		t.Fatalf("refund after reconcile = %s", got.Status)
	}

	stuck := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioAsync), Confirm: true})
	old := time.Now().Add(-48 * time.Hour)
	f.db.Model(&paymentswitch.AttemptRow{}).Where("intent_id = ?", stuck.ID).Updates(map[string]any{"created_at": old, "next_sync_at": nil})
	rec.MaxAge = 24 * time.Hour
	if _, err := rec.RunOnce(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.wantStatus(f.get(stuck.ID).Intent, paymentswitch.IntentProcessing)
	an := f.anomalies(stuck.ID)
	if len(an) != 1 || an[0].Kind != paymentswitch.AnomalyStaleInFlight {
		t.Fatalf("stale anomaly = %+v", an)
	}
	if !strings.Contains(strings.Join(f.logs, "\n"), "stale_in_flight") {
		t.Fatal("stale rows must be logged at error level")
	}
}
