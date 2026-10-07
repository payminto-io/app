package service

import (
	"encoding/json"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// EmailPayload is the structured payload for email.send events.
type EmailPayload struct {
	To       string         `json:"to"`
	Template string         `json:"template"`
	Subject  string         `json:"subject"`
	Data     map[string]any `json:"data,omitempty"`
}

// EEWebhookPayload is the structured payload for webhook.send events in the
// EventEmitter queue. Distinct from service.WebhookPayload (the outbound body).
type EEWebhookPayload struct {
	WebhookID  uint           `json:"webhookId"`
	Event      string         `json:"event"`
	Payload    map[string]any `json:"payload"`
	TargetURL  string         `json:"targetUrl"`
	SignSecret string         `json:"signSecret,omitempty"`
}

// NotificationPayload is the structured payload for notification.send events.
type NotificationPayload struct {
	MemberID uint           `json:"memberId"`
	Title    string         `json:"title"`
	Body     string         `json:"body"`
	Data     map[string]any `json:"data,omitempty"`
}

// EventEmitterService publishes domain events to the ee_events queue for
// asynchronous processing by Consumer workers. Supports email dispatch,
// webhook delivery, and notification pushes via typed event payloads.
type EventEmitterService struct {
	eeEventRepo repository.EEEventRepository
}

// NewEventEmitterService constructs an EventEmitterService.
func NewEventEmitterService(eeEventRepo repository.EEEventRepository) *EventEmitterService {
	return &EventEmitterService{eeEventRepo: eeEventRepo}
}

// EmitEmail writes an ee_events row with event_type="email.send" and the
// serialised EmailPayload. The EmailProcessor worker will pick it up.
func (s *EventEmitterService) EmitEmail(payload EmailPayload) error {
	return s.emit(models.EventTypeEmailSend, payload)
}

// EmitWebhook writes an ee_events row with event_type="webhook.send".
func (s *EventEmitterService) EmitWebhook(payload EEWebhookPayload) error {
	return s.emit(models.EventTypeWebhookSend, payload)
}

// EmitNotification writes an ee_events row with event_type="notification.send".
func (s *EventEmitterService) EmitNotification(payload NotificationPayload) error {
	return s.emit(models.EventTypeNotificationSend, payload)
}

// EmitNamed publishes a module event (for example cre.attestation.recorded.v1) through the same queue.
func (s *EventEmitterService) EmitNamed(eventType string, payload map[string]any) error {
	return s.emit(eventType, payload)
}

// emit serialises the payload and creates an EEEvent row.
func (s *EventEmitterService) emit(eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	event := &models.EEEvent{
		EventType:   eventType,
		Status:      models.EEEventStatusPending,
		Payload:     string(data),
		Attempts:    0,
		MaxAttempts: 7,
		NextRetryAt: nil,
	}

	if err := s.eeEventRepo.Create(event); err != nil {
		return fmt.Errorf("create ee_event(%s): %w", eventType, err)
	}
	return nil
}
