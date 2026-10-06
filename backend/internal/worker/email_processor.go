package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/gorm"
)

const (
	emailProcessorBatchSize   = 50
	emailProcessorTickInterval = 10 * time.Second
	emailMaxAttempts          = 7
)

// emailRetrySchedule defines the back-off delays for failed email events.
// Matches the webhook processor schedule: 30m, 1h, 2h, 4h, 8h, 24h, 48h.
var emailRetrySchedule = []time.Duration{
	30 * time.Minute,
	1 * time.Hour,
	2 * time.Hour,
	4 * time.Hour,
	8 * time.Hour,
	24 * time.Hour,
	48 * time.Hour,
}

// EmailProcessor is a background worker that consumes ee_events of type
// "email.send" from the queue, renders them via EmailService, and marks them
// processed or failed. Retries with exponential backoff up to emailMaxAttempts;
// then moves to dead letter.
type EmailProcessor struct {
	eeEventRepo repository.EEEventRepository
	emailSvc    *service.EmailService
}

// NewEmailProcessor constructs an EmailProcessor.
func NewEmailProcessor(eeEventRepo repository.EEEventRepository, emailSvc *service.EmailService) *EmailProcessor {
	return &EmailProcessor{eeEventRepo: eeEventRepo, emailSvc: emailSvc}
}

// Name implements Worker and returns the human-readable identifier.
func (p *EmailProcessor) Name() string { return "email_processor" }

// Start implements Worker. It polls ee_events on a 10s ticker, processing up
// to emailProcessorBatchSize events per tick. Runs until ctx is cancelled.
func (p *EmailProcessor) Start(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[email_processor] panic: %v\n%s", r, debug.Stack())
		}
	}()

	ticker := time.NewTicker(emailProcessorTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			p.processBatch()
		}
	}
}

// processBatch fetches pending email events and dispatches each one.
// Uses ListPendingByType to filter in SQL — avoids fetching webhook/notification
// events that would be discarded in Go, preventing type starvation.
func (p *EmailProcessor) processBatch() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[email_processor] batch panic: %v\n%s", r, debug.Stack())
		}
	}()

	events, err := p.eeEventRepo.ListPendingByType(models.EventTypeEmailSend, emailProcessorBatchSize)
	if err != nil {
		log.Printf("[email_processor] list pending email events: %v", err)
		return
	}

	for _, event := range events {
		p.processEvent(event)
	}
}

// processEvent claims a single event atomically and delivers the email.
func (p *EmailProcessor) processEvent(event models.EEEvent) {
	// Atomic claim: transition pending → processing.
	if err := p.eeEventRepo.MarkProcessing(event.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return // another worker claimed it
		}
		log.Printf("[email_processor] mark processing event %d: %v", event.ID, err)
		return
	}

	if err := p.deliver(event); err != nil {
		p.handleFailure(event, err)
		return
	}

	if err := p.eeEventRepo.MarkProcessed(event.ID); err != nil {
		log.Printf("[email_processor] mark processed event %d: %v", event.ID, err)
	}
}

// deliver deserialises the payload and sends the email.
func (p *EmailProcessor) deliver(event models.EEEvent) error {
	var payload service.EmailPayload
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	msg := service.EmailMessage{
		To:       payload.To,
		Subject:  payload.Subject,
		Template: payload.Template,
		Data:     payload.Data,
	}
	return p.emailSvc.Send(msg)
}

// handleFailure increments the failure count and either schedules a retry or
// moves the event to dead letter.
func (p *EmailProcessor) handleFailure(event models.EEEvent, err error) {
	log.Printf("[email_processor] deliver event %d: %v", event.ID, err)

	// nextAttempt is what the counter will be AFTER MarkFailed increments it.
	// Computing this before MarkFailed prevents the off-by-one where the dead
	// letter check fires one retry late.
	nextAttempt := event.Attempts + 1
	if nextAttempt >= emailMaxAttempts {
		if dlErr := p.eeEventRepo.MarkDeadLetter(event.ID, err.Error()); dlErr != nil {
			log.Printf("[email_processor] mark dead letter event %d: %v", event.ID, dlErr)
		}
		return
	}

	scheduleIdx := nextAttempt - 1 // index into zero-based emailRetrySchedule
	if scheduleIdx < 0 {
		scheduleIdx = 0
	}
	if scheduleIdx >= len(emailRetrySchedule) {
		scheduleIdx = len(emailRetrySchedule) - 1
	}
	nextRetry := time.Now().Add(emailRetrySchedule[scheduleIdx])

	if rErr := p.eeEventRepo.MarkFailed(event.ID, err.Error(), nextRetry); rErr != nil {
		log.Printf("[email_processor] mark failed event %d: %v", event.ID, rErr)
	}
}
