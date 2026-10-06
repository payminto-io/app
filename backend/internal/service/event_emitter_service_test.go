package service

import (
	"encoding/json"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEETestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&models.EEEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestEventEmitterService_EmitEmail(t *testing.T) {
	db := newEETestDB(t)
	repo := repository.NewEEEventRepository(db)
	svc := NewEventEmitterService(repo)

	payload := EmailPayload{
		To:       "user@example.com",
		Template: "welcome",
		Subject:  "Welcome",
		Data:     map[string]any{"Name": "Alice"},
	}
	if err := svc.EmitEmail(payload); err != nil {
		t.Fatalf("EmitEmail: %v", err)
	}

	events, err := repo.ListPending(10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.EventType != models.EventTypeEmailSend {
		t.Errorf("event_type = %q, want %q", e.EventType, models.EventTypeEmailSend)
	}
	if e.Status != models.EEEventStatusPending {
		t.Errorf("status = %q, want %q", e.Status, models.EEEventStatusPending)
	}

	var got EmailPayload
	if err := json.Unmarshal([]byte(e.Payload), &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.To != payload.To {
		t.Errorf("payload.To = %q, want %q", got.To, payload.To)
	}
	if got.Template != payload.Template {
		t.Errorf("payload.Template = %q, want %q", got.Template, payload.Template)
	}
}

func TestEventEmitterService_EmitWebhook(t *testing.T) {
	db := newEETestDB(t)
	repo := repository.NewEEEventRepository(db)
	svc := NewEventEmitterService(repo)

	payload := EEWebhookPayload{
		WebhookID: 1,
		Event:     "payment.confirmed",
		TargetURL: "https://example.com/hook",
	}
	if err := svc.EmitWebhook(payload); err != nil {
		t.Fatalf("EmitWebhook: %v", err)
	}

	events, err := repo.ListPending(10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != models.EventTypeWebhookSend {
		t.Errorf("event_type = %q, want %q", events[0].EventType, models.EventTypeWebhookSend)
	}
}

func TestEventEmitterService_EmitNotification(t *testing.T) {
	db := newEETestDB(t)
	repo := repository.NewEEEventRepository(db)
	svc := NewEventEmitterService(repo)

	payload := NotificationPayload{
		MemberID: 42,
		Title:    "Deposit received",
		Body:     "0.5 ETH confirmed",
	}
	if err := svc.EmitNotification(payload); err != nil {
		t.Fatalf("EmitNotification: %v", err)
	}

	events, err := repo.ListPending(10)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != models.EventTypeNotificationSend {
		t.Errorf("event_type = %q, want %q", events[0].EventType, models.EventTypeNotificationSend)
	}
}
