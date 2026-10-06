package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ActivityLogRepository defines storage operations for request audit logs.
type ActivityLogRepository interface {
	Create(log *models.ActivityLog) error
	BulkCreate(logs []models.ActivityLog) error
	ListByMember(memberID uint, limit int) ([]models.ActivityLog, error)
}

// ActivityLogRepositoryImpl is the GORM-backed implementation.
type ActivityLogRepositoryImpl struct {
	db *gorm.DB
}

// NewActivityLogRepository constructs an ActivityLogRepositoryImpl.
func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &ActivityLogRepositoryImpl{db: db}
}

// Create inserts a single ActivityLog row.
func (r *ActivityLogRepositoryImpl) Create(log *models.ActivityLog) error {
	return r.db.Create(log).Error
}

// BulkCreate inserts multiple ActivityLog rows in a single DB round-trip.
// The slice is split into batches of 100 to avoid overwhelming the driver.
func (r *ActivityLogRepositoryImpl) BulkCreate(logs []models.ActivityLog) error {
	if len(logs) == 0 {
		return nil
	}
	return r.db.CreateInBatches(logs, 100).Error
}

// ListByMember returns the most recent audit entries for a given member.
func (r *ActivityLogRepositoryImpl) ListByMember(memberID uint, limit int) ([]models.ActivityLog, error) {
	var entries []models.ActivityLog
	err := r.db.
		Where("member_id = ?", memberID).
		Order("created_at DESC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}
