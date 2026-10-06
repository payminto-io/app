package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// EEEventRepository defines data access operations for the ee_events queue table.
type EEEventRepository interface {
	Create(e *models.EEEvent) error
	GetByID(id uint) (*models.EEEvent, error)
	// ListPending returns events across all types that are ready for processing.
	ListPending(limit int) ([]models.EEEvent, error)
	// ListPendingByType returns only events of the given type that are ready
	// for processing. Avoids cross-type starvation when batching.
	ListPendingByType(eventType string, limit int) ([]models.EEEvent, error)
	MarkProcessing(id uint) error
	MarkProcessed(id uint) error
	// MarkFailed bumps the attempt counter and schedules the next retry.
	// Status stays 'pending' so ListPending picks it up again on the next tick.
	// Only after the final attempt should MarkDeadLetter be called.
	MarkFailed(id uint, reason string, nextRetryAt time.Time) error
	MarkDeadLetter(id uint, reason string) error
}

// EEEventRepositoryImpl is the GORM-backed implementation of EEEventRepository.
type EEEventRepositoryImpl struct {
	db *gorm.DB
}

// NewEEEventRepository constructs an EEEventRepositoryImpl.
func NewEEEventRepository(db *gorm.DB) EEEventRepository {
	return &EEEventRepositoryImpl{db: db}
}

// Create inserts a new EEEvent row.
func (r *EEEventRepositoryImpl) Create(e *models.EEEvent) error {
	return r.db.Create(e).Error
}

// GetByID fetches an EEEvent by primary key.
func (r *EEEventRepositoryImpl) GetByID(id uint) (*models.EEEvent, error) {
	var e models.EEEvent
	err := r.db.First(&e, id).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListPending returns up to limit events where status IN ('pending','processing') and
// next_retry_at <= NOW() or next_retry_at IS NULL, ordered by created_at ASC.
// Note: 'failed' events are kept at status='pending' with a future next_retry_at
// by MarkFailed, so they are automatically included here when the retry window passes.
func (r *EEEventRepositoryImpl) ListPending(limit int) ([]models.EEEvent, error) {
	var events []models.EEEvent
	err := r.db.
		Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			models.EEEventStatusPending, time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ListPendingByType returns up to limit events of the given type that are ready
// for processing. Filters in SQL to avoid fetching cross-type events that would
// be discarded in Go — prevents email/webhook/notification starvation.
func (r *EEEventRepositoryImpl) ListPendingByType(eventType string, limit int) ([]models.EEEvent, error) {
	var events []models.EEEvent
	err := r.db.
		Where("event_type = ? AND status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)",
			eventType, models.EEEventStatusPending, time.Now()).
		Order("created_at ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// MarkProcessing atomically transitions a pending event to processing.
// Uses a conditional UPDATE so concurrent workers don't double-process.
func (r *EEEventRepositoryImpl) MarkProcessing(id uint) error {
	result := r.db.Model(&models.EEEvent{}).
		Where("id = ? AND status = ?", id, models.EEEventStatusPending).
		Update("status", models.EEEventStatusProcessing)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkProcessed sets status to 'processed' and records the timestamp.
func (r *EEEventRepositoryImpl) MarkProcessed(id uint) error {
	now := time.Now()
	return r.db.Model(&models.EEEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":       models.EEEventStatusProcessed,
			"processed_at": now,
		}).Error
}

// MarkFailed bumps the attempt counter and schedules the next retry.
// Status is intentionally kept as 'pending' so that ListPending picks the event
// up again once the next_retry_at window has passed. Only after reaching
// max_attempts should MarkDeadLetter be called.
func (r *EEEventRepositoryImpl) MarkFailed(id uint, reason string, nextRetryAt time.Time) error {
	return r.db.Model(&models.EEEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         models.EEEventStatusPending,
			"failure_reason": reason,
			"next_retry_at":  nextRetryAt,
			"attempts":       gorm.Expr("attempts + 1"),
		}).Error
}

// MarkDeadLetter moves the event to dead_letter status after max attempts.
func (r *EEEventRepositoryImpl) MarkDeadLetter(id uint, reason string) error {
	return r.db.Model(&models.EEEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         models.EEEventStatusDeadLetter,
			"failure_reason": reason,
			"attempts":       gorm.Expr("attempts + 1"),
		}).Error
}
