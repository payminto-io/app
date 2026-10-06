package models

import "time"

// EEEvent is an event-emitter event. Services emit these to a queue table
// and the consumer worker (email processor, webhook processor) picks them up
// asynchronously. Uses BaseModel because it's a queue/audit table.
type EEEvent struct {
	BaseModel
	EventType     string     `gorm:"type:varchar(100);not null;index" json:"eventType"`
	Status        string     `gorm:"type:varchar(20);default:'pending';not null;index" json:"status"`
	Payload       string     `gorm:"type:text;not null" json:"payload"`
	Attempts      int        `gorm:"default:0" json:"attempts"`
	MaxAttempts   int        `gorm:"default:5" json:"maxAttempts"`
	NextRetryAt   *time.Time `json:"nextRetryAt,omitempty"`
	ProcessedAt   *time.Time `json:"processedAt,omitempty"`
	FailureReason *string    `gorm:"type:text" json:"failureReason,omitempty"`
}

func (EEEvent) TableName() string { return "ee_events" }

// Well-known event types.
const (
	EventTypeEmailSend        = "email.send"
	EventTypeWebhookSend      = "webhook.send"
	EventTypeNotificationSend = "notification.send"
	EventTypePaymentConfirmed = "payment.confirmed"
	EventTypeSweepCompleted   = "sweep.completed"
	EventTypeWithdrawalSent   = "withdrawal.sent"
)

// Event statuses.
const (
	EEEventStatusPending    = "pending"
	EEEventStatusProcessing = "processing"
	EEEventStatusProcessed  = "processed"
	EEEventStatusFailed     = "failed"
	EEEventStatusDeadLetter = "dead_letter"
)
