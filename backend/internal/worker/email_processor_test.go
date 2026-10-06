package worker

import (
	"context"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newEmailProcessorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.EEEvent{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestEmailProcessor_ProcessesPendingEvent(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Insert a pending email event.
	event := &models.EEEvent{
		EventType:   models.EventTypeEmailSend,
		Status:      models.EEEventStatusPending,
		Payload:     `{"to":"test@example.com","template":"welcome","subject":"Hi","data":{"Name":"Bob"}}`,
		Attempts:    0,
		MaxAttempts: 7,
	}
	if err := repo.Create(event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	proc := NewEmailProcessor(repo, emailSvc)
	proc.processBatch()

	// Event should now be processed.
	updated, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if updated.Status != models.EEEventStatusProcessed {
		t.Errorf("status = %q, want %q", updated.Status, models.EEEventStatusProcessed)
	}
}

func TestEmailProcessor_SkipsNonEmailEvents(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Insert a webhook event (should be skipped by email processor).
	event := &models.EEEvent{
		EventType:   models.EventTypeWebhookSend,
		Status:      models.EEEventStatusPending,
		Payload:     `{"webhookId":1}`,
		Attempts:    0,
		MaxAttempts: 7,
	}
	if err := repo.Create(event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	proc := NewEmailProcessor(repo, emailSvc)
	proc.processBatch()

	// Status should remain pending (not processed by email processor).
	updated, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if updated.Status != models.EEEventStatusPending {
		t.Errorf("status = %q, want pending (webhook events skipped)", updated.Status)
	}
}

func TestEmailProcessor_MarksFailedOnBadPayload(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Insert an event with invalid JSON payload.
	event := &models.EEEvent{
		EventType:   models.EventTypeEmailSend,
		Status:      models.EEEventStatusPending,
		Payload:     `{invalid json`,
		Attempts:    0,
		MaxAttempts: 7,
	}
	if err := repo.Create(event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	proc := NewEmailProcessor(repo, emailSvc)
	proc.processBatch()

	updated, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	// After a failure, MarkFailed keeps status='pending' (not 'failed') so the
	// event is re-fetched on the next tick when next_retry_at has passed.
	// The failure is recorded in next_retry_at and failure_reason.
	if updated.Status != models.EEEventStatusPending {
		t.Errorf("status = %q, want %q (MarkFailed keeps status pending for retry)", updated.Status, models.EEEventStatusPending)
	}
	if updated.Attempts != 1 {
		t.Errorf("attempts = %d, want 1 after one failure", updated.Attempts)
	}
	if updated.NextRetryAt == nil {
		t.Error("expected NextRetryAt to be set after failure")
	}
}

func TestEmailProcessor_DeadLetterAfterMaxAttempts(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Insert event already at max attempts.
	event := &models.EEEvent{
		EventType:   models.EventTypeEmailSend,
		Status:      models.EEEventStatusPending,
		Payload:     `{bad json`,
		Attempts:    emailMaxAttempts,
		MaxAttempts: emailMaxAttempts,
	}
	if err := repo.Create(event); err != nil {
		t.Fatalf("create event: %v", err)
	}

	proc := NewEmailProcessor(repo, emailSvc)
	proc.processBatch()

	updated, err := repo.GetByID(event.ID)
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if updated.Status != models.EEEventStatusDeadLetter {
		t.Errorf("status = %q, want dead_letter", updated.Status)
	}
}

// TestEmailProcessor_ProcessBatch_TypeFilter_NoStarvation creates 50 webhook
// events and 5 email events. After processBatch, all 5 email events must be
// processed and zero webhook events must be touched (status must remain pending).
func TestEmailProcessor_ProcessBatch_TypeFilter_NoStarvation(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Insert 50 webhook.send events.
	for range 50 {
		evt := &models.EEEvent{
			EventType:   models.EventTypeWebhookSend,
			Status:      models.EEEventStatusPending,
			Payload:     `{"webhookId":1}`,
			Attempts:    0,
			MaxAttempts: 7,
		}
		if err := repo.Create(evt); err != nil {
			t.Fatalf("create webhook event: %v", err)
		}
	}

	// Insert 5 email.send events with valid payloads.
	emailIDs := make([]uint, 5)
	for i := range 5 {
		evt := &models.EEEvent{
			EventType:   models.EventTypeEmailSend,
			Status:      models.EEEventStatusPending,
			Payload:     `{"to":"test@example.com","template":"welcome","subject":"Hi","data":{"Name":"Bob"}}`,
			Attempts:    0,
			MaxAttempts: 7,
		}
		if err := repo.Create(evt); err != nil {
			t.Fatalf("create email event: %v", err)
		}
		emailIDs[i] = evt.ID
	}

	proc := NewEmailProcessor(repo, emailSvc)
	proc.processBatch()

	// All 5 email events should now be processed.
	for _, id := range emailIDs {
		evt, err := repo.GetByID(id)
		if err != nil {
			t.Fatalf("GetByID email event %d: %v", id, err)
		}
		if evt.Status != models.EEEventStatusProcessed {
			t.Errorf("email event %d status = %q, want %q", id, evt.Status, models.EEEventStatusProcessed)
		}
	}

	// Zero webhook events should have been touched (all still pending).
	var count int64
	db.Model(&models.EEEvent{}).
		Where("event_type = ? AND status != ?", models.EventTypeWebhookSend, models.EEEventStatusPending).
		Count(&count)
	if count != 0 {
		t.Errorf("expected 0 webhook events touched by email processor, got %d", count)
	}
}

// TestEmailProcessor_Retry_OffByOne_CorrectDeadLetter verifies that an event
// becomes dead_letter after exactly emailMaxAttempts failures, not one more.
// This tests the off-by-one fix: nextAttempt := event.Attempts + 1.
func TestEmailProcessor_Retry_OffByOne_CorrectDeadLetter(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	// Create an event with a bad payload so it always fails.
	evt := &models.EEEvent{
		EventType:   models.EventTypeEmailSend,
		Status:      models.EEEventStatusPending,
		Payload:     `{bad json`,
		Attempts:    0,
		MaxAttempts: emailMaxAttempts,
	}
	if err := repo.Create(evt); err != nil {
		t.Fatalf("create event: %v", err)
	}

	proc := NewEmailProcessor(repo, emailSvc)

	// Run emailMaxAttempts - 1 times; the event should still be pending (not dead letter).
	for i := range emailMaxAttempts - 1 {
		// Reset next_retry_at so ListPendingByType picks it up again.
		db.Model(&models.EEEvent{}).Where("id = ?", evt.ID).Update("next_retry_at", nil)
		proc.processBatch()

		updated, err := repo.GetByID(evt.ID)
		if err != nil {
			t.Fatalf("iteration %d: GetByID: %v", i, err)
		}
		if updated.Status == models.EEEventStatusDeadLetter {
			t.Errorf("iteration %d: event became dead_letter too early (attempts=%d, max=%d)",
				i, updated.Attempts, emailMaxAttempts)
		}
	}

	// On the Nth attempt the event should move to dead_letter.
	db.Model(&models.EEEvent{}).Where("id = ?", evt.ID).Update("next_retry_at", nil)
	proc.processBatch()

	final, err := repo.GetByID(evt.ID)
	if err != nil {
		t.Fatalf("final GetByID: %v", err)
	}
	if final.Status != models.EEEventStatusDeadLetter {
		t.Errorf("expected dead_letter after %d attempts, got status=%q attempts=%d",
			emailMaxAttempts, final.Status, final.Attempts)
	}
}

func TestEmailProcessor_Start_CancelsViaContext(t *testing.T) {
	db := newEmailProcessorDB(t)
	repo := repository.NewEEEventRepository(db)
	emailSvc := service.NewEmailService(nil)

	proc := NewEmailProcessor(repo, emailSvc)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- proc.Start(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned non-nil error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start did not return after context cancellation")
	}
}
