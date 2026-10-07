//go:build integration

package paymentswitch_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/mock"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/shopspring/decimal"
)

func newPostgresFixture(t *testing.T) *fixture {
	t.Helper()
	db, cleanup := database.NewTestDB(t)
	t.Cleanup(cleanup)
	reg := connectors.NewRegistry()
	m := mock.New()
	if err := reg.Register(m); err != nil {
		t.Fatal(err)
	}
	led := &recordingLedger{inner: ledger.New(db)}
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors{mock.Code}, Connectors: reg}
	return &fixture{t: t, db: db, svc: paymentswitch.New(db, reg, selector, led), mock: m, ledger: led, ctx: context.Background()}
}

func TestIntegration_SchemaConvergesAndConstraintsHold(t *testing.T) {
	f := newPostgresFixture(t)
	if _, err := database.ApplyMigrations(context.Background(), f.db); err != nil {
		t.Fatalf("ApplyMigrations after dev-style migrate: %v", err)
	}
	for _, name := range []string{
		"switch_payment_intents_status_check", "switch_payment_attempts_status_check", "switch_refunds_status_check",
		"switch_payment_attempts_intent_id_fkey", "switch_refunds_attempt_id_fkey", "switch_payment_intents_amounts_check",
	} {
		var n int64
		if err := f.db.Raw(`SELECT count(*) FROM pg_constraint WHERE conname = ?`, name).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("constraint %s count = %d, want 1", name, n)
		}
	}
	for _, index := range []string{"switch_intents_merchant_idempotency_key", "switch_attempts_connector_tx_key", "switch_webhook_events_connector_event_key"} {
		var n int64
		if err := f.db.Raw(`SELECT count(*) FROM pg_indexes WHERE indexname = ?`, index).Scan(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("index %s count = %d, want 1", index, n)
		}
	}
	err := f.db.Exec(`INSERT INTO switch_payment_intents (id, merchant_id, idempotency_key, request_hash, status, amount, asset, capture_method, payment_method, metadata)
		VALUES ('pi_bad', 'm', 'k', repeat('a', 64), 'paid', 1, 'USD', 'automatic', '{}', '{}')`).Error
	if err == nil {
		t.Fatal("a status outside the vocabulary must be rejected by the database")
	}
}

func TestIntegration_PersistsExactAmountsAndHistory(t *testing.T) {
	f := newPostgresFixture(t)
	amount, _ := decimal.NewFromString("1234.567890123456789012")
	in := f.create(paymentswitch.CreateCommand{Money: paymentswitch.Money{Amount: amount, Asset: "USDC"}, CaptureMethod: connectors.CaptureManual, PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	f.wantStatus(in, paymentswitch.IntentRequiresCapture)
	part, _ := decimal.NewFromString("0.000000000000000001")
	in, err := f.svc.Capture(f.ctx, merchant, in.ID, paymentswitch.CaptureCommand{Amount: &part})
	if err != nil {
		t.Fatal(err)
	}
	v := f.get(in.ID)
	if !v.Intent.Money.Amount.Equal(amount) || !v.Intent.AmountCaptured.Equal(part) || v.Intent.Status != paymentswitch.IntentPartiallyCaptured {
		t.Fatalf("intent = %+v", v.Intent)
	}
	if len(v.Attempts) != 1 || !v.Attempts[0].AmountCaptured.Equal(part) {
		t.Fatalf("attempt = %+v", v.Attempts)
	}
	var transitions int64
	f.db.Model(&paymentswitch.TransitionRow{}).Where("entity_id = ?", v.Attempts[0].ID).Count(&transitions)
	if transitions != 4 {
		t.Fatalf("attempt transitions = %d, want started, authorized, capture_initiated, partial_charged", transitions)
	}
	bal, err := f.ledger.inner.Balances(f.ctx, ledger.OwnerMember, merchant)
	if err != nil || !bal["USDC"].Equal(part) {
		t.Fatalf("merchant USDC natural balance = %v, %v; want %s", bal, err, part)
	}
}

// Concurrent confirms on one intent: exactly one attempt is created and one authorization reaches the connector.
func TestIntegration_ConcurrentConfirm_OnlyOneAttemptWins(t *testing.T) {
	f := newPostgresFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess)})
	const racers = 12
	var wg sync.WaitGroup
	results := make(chan error, racers)
	for range racers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svc.Confirm(f.ctx, merchant, in.ID, paymentswitch.ConfirmCommand{})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	var wins, losses int
	for err := range results {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, paymentswitch.ErrInvalidTransition), errors.Is(err, paymentswitch.ErrConcurrentUpdate):
			losses++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if wins != 1 || losses != racers-1 {
		t.Fatalf("wins = %d losses = %d, want 1 and %d", wins, losses, racers-1)
	}
	v := f.get(in.ID)
	if len(v.Attempts) != 1 || v.Intent.Status != paymentswitch.IntentSucceeded {
		t.Fatalf("attempts = %d status = %s", len(v.Attempts), v.Intent.Status)
	}
	if f.mock.LastTransactionID() != "mock_tx_1" {
		t.Fatalf("connector saw more than one authorization: last = %s", f.mock.LastTransactionID())
	}
	f.wantPayments(1)
}

// The same webhook delivered many times at once posts the journal once; the unique event id serialises them.
func TestIntegration_ConcurrentWebhookDeliveries_PostOnce(t *testing.T) {
	f := newPostgresFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioAsync), Confirm: true})
	a := f.wantAttempt(in.ID, paymentswitch.AttemptPending)
	h, body := f.mock.SignWebhook(mock.Event{EventID: "evt_once", TransactionID: a.ConnectorTransactionID, Status: string(mock.StatusCaptured)})
	const deliveries = 10
	var wg sync.WaitGroup
	errs := make(chan error, deliveries)
	for range deliveries {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.svc.HandleWebhook(f.ctx, mock.Code, h, body)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	var ok, replays int
	for err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, paymentswitch.ErrWebhookReplay):
			replays++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 || replays != deliveries-1 {
		t.Fatalf("processed = %d replays = %d", ok, replays)
	}
	f.wantPayments(1)
	f.wantStatus(f.get(in.ID).Intent, paymentswitch.IntentSucceeded)
}

func TestIntegration_ConcurrentRefundsCannotOverRefund(t *testing.T) {
	f := newPostgresFixture(t)
	in := f.create(paymentswitch.CreateCommand{PaymentMethod: card(mock.ScenarioSuccess), Confirm: true})
	const racers = 8
	sixty := decimal.NewFromInt(60)
	var wg sync.WaitGroup
	errs := make(chan error, racers)
	for i := range racers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := f.svc.Refund(f.ctx, merchant, in.ID, paymentswitch.RefundCommand{IdempotencyKey: "r" + decimal.NewFromInt(int64(i)).String(), Amount: &sixty})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	var ok int
	for err := range errs {
		if err == nil {
			ok++
		} else if !errors.Is(err, paymentswitch.ErrAmountExceeds) && !errors.Is(err, paymentswitch.ErrConcurrentUpdate) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("refunds of 60 on 100 captured: succeeded = %d, want 1", ok)
	}
	f.wantRefundJournals(1)
	if v := f.get(in.ID); !v.Intent.AmountRefunded.Equal(sixty) {
		t.Fatalf("amount refunded = %s", v.Intent.AmountRefunded)
	}
}
