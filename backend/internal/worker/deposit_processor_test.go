package worker

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/realtime"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDepositProcessorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Deposit{}, &models.PaymentRequest{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestDepositProcessor_ConfirmsAndPublishes verifies that a CONFIRMING deposit
// with enough confirmations is promoted to CONFIRMED, its payment flipped to
// FILLED, and a real-time event published on the payment's topic.
func TestDepositProcessor_ConfirmsAndPublishes(t *testing.T) {
	db := setupDepositProcessorDB(t)
	broker := realtime.NewMemoryBroker()

	pr := models.PaymentRequest{ReferenceID: "ref-abc", State: models.PaymentStateOpen}
	if err := db.Create(&pr).Error; err != nil {
		t.Fatalf("create payment: %v", err)
	}
	dep := models.Deposit{
		PaymentRequestID:      &pr.ID,
		Status:                models.DepositStatusConfirming,
		Confirmations:         12,
		RequiredConfirmations: 12,
		TxID:                  "0xabc",
	}
	if err := db.Create(&dep).Error; err != nil {
		t.Fatalf("create deposit: %v", err)
	}

	// Subscribe before processing.
	ch, unsub := broker.Subscribe(broker.PaymentTopic("ref-abc"))
	defer unsub()

	dp := NewDepositProcessor(db, broker)
	dp.processConfirmingDeposits()

	// Deposit + payment state must have advanced.
	var got models.Deposit
	db.First(&got, dep.ID)
	if got.Status != models.DepositStatusConfirmed {
		t.Errorf("deposit status = %q, want confirmed", got.Status)
	}
	var gotPR models.PaymentRequest
	db.First(&gotPR, pr.ID)
	if gotPR.State != models.PaymentStateFilled {
		t.Errorf("payment state = %q, want FILLED", gotPR.State)
	}

	// A FILLED event must have been published.
	select {
	case ev := <-ch:
		if ev.ReferenceID != "ref-abc" || ev.State != models.PaymentStateFilled {
			t.Errorf("event = %+v, want ref-abc/FILLED", ev)
		}
		if ev.TxID != "0xabc" {
			t.Errorf("event txID = %q, want 0xabc", ev.TxID)
		}
	case <-time.After(time.Second):
		t.Fatal("no real-time event published on confirmation")
	}
}

// TestDepositProcessor_NilBrokerSafe ensures the processor works without a
// broker wired (events simply not published).
func TestDepositProcessor_NilBrokerSafe(t *testing.T) {
	db := setupDepositProcessorDB(t)
	pr := models.PaymentRequest{ReferenceID: "ref-nil", State: models.PaymentStateOpen}
	db.Create(&pr)
	db.Create(&models.Deposit{
		PaymentRequestID: &pr.ID, Status: models.DepositStatusConfirming,
		Confirmations: 3, RequiredConfirmations: 3, TxID: "0xnil",
	})

	dp := NewDepositProcessor(db, nil)
	dp.processConfirmingDeposits() // must not panic

	var gotPR models.PaymentRequest
	db.First(&gotPR, pr.ID)
	if gotPR.State != models.PaymentStateFilled {
		t.Errorf("payment state = %q, want FILLED", gotPR.State)
	}
}
