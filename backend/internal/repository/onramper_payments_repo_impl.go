package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// OnramperPaymentsRepository persists Onramper card-to-crypto session rows.
type OnramperPaymentsRepository interface {
	Create(o *models.OnramperPayments) error
	GetByID(id uint) (*models.OnramperPayments, error)
	GetBySessionID(sessionID string) (*models.OnramperPayments, error)
	ListByPlatform(platformID uint, opts ...QueryOption) ([]models.OnramperPayments, error)
	UpdateStatus(id uint, status string) error
	// MarkCompleted is an atomic conditional UPDATE that only fires when the
	// row is currently in 'created' or 'pending' state. Returns rows-affected.
	MarkCompleted(id uint, providerTxID, onchainTxHash string, cryptoAmount string) (int64, error)
	// MarkFailed is an atomic conditional UPDATE that only fires when the row
	// is not already in a terminal state. Returns rows-affected.
	MarkFailed(id uint, reason string) (int64, error)
	// MarkExpired flips pending sessions older than the given threshold to
	// 'expired'. Returns rows-affected for the cleanup worker.
	MarkExpired(olderThan time.Time) (int64, error)
	// ListPendingExpired returns sessions in 'created' or 'pending' state
	// older than the given threshold for inspection by the cleanup worker.
	ListPendingExpired(olderThan time.Time, limit int) ([]models.OnramperPayments, error)
}

// OnramperPaymentsRepositoryImpl is the GORM implementation.
type OnramperPaymentsRepositoryImpl struct {
	db *gorm.DB
}

// NewOnramperPaymentsRepository constructs the repo.
func NewOnramperPaymentsRepository(db *gorm.DB) OnramperPaymentsRepository {
	return &OnramperPaymentsRepositoryImpl{db: db}
}

// Create persists a new session row.
func (r *OnramperPaymentsRepositoryImpl) Create(o *models.OnramperPayments) error {
	return r.db.Create(o).Error
}

// GetByID returns a session by primary key.
func (r *OnramperPaymentsRepositoryImpl) GetByID(id uint) (*models.OnramperPayments, error) {
	var o models.OnramperPayments
	if err := r.db.First(&o, id).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// GetBySessionID returns a session by its provider-generated session ID.
func (r *OnramperPaymentsRepositoryImpl) GetBySessionID(sessionID string) (*models.OnramperPayments, error) {
	var o models.OnramperPayments
	if err := r.db.Where("session_id = ?", sessionID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

// ListByPlatform returns sessions for a platform with optional pagination.
func (r *OnramperPaymentsRepositoryImpl) ListByPlatform(platformID uint, opts ...QueryOption) ([]models.OnramperPayments, error) {
	var rows []models.OnramperPayments
	q := Apply(r.db.Where("external_platform_id = ?", platformID).Order("created_at DESC"), opts...)
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpdateStatus is the unguarded status setter — callers should prefer the
// conditional MarkCompleted / MarkFailed / MarkExpired methods.
func (r *OnramperPaymentsRepositoryImpl) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.OnramperPayments{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// MarkCompleted atomically transitions the session to 'completed' if and
// only if it is currently 'created' or 'pending'.
func (r *OnramperPaymentsRepositoryImpl) MarkCompleted(id uint, providerTxID, onchainTxHash, cryptoAmount string) (int64, error) {
	now := time.Now()
	updates := map[string]any{
		"status":          models.OnramperStatusCompleted,
		"provider_tx_id":  providerTxID,
		"onchain_tx_hash": onchainTxHash,
		"completed_at":    now,
		"updated_at":      now,
	}
	if cryptoAmount != "" {
		updates["crypto_amount"] = cryptoAmount
	}
	res := r.db.Model(&models.OnramperPayments{}).
		Where("id = ? AND status IN ?", id, []string{models.OnramperStatusCreated, models.OnramperStatusPending}).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// MarkFailed atomically transitions the session to 'failed' if it is not
// already in a terminal state.
func (r *OnramperPaymentsRepositoryImpl) MarkFailed(id uint, reason string) (int64, error) {
	now := time.Now()
	updates := map[string]any{
		"status":         models.OnramperStatusFailed,
		"failure_reason": reason,
		"failed_at":      now,
		"updated_at":     now,
	}
	res := r.db.Model(&models.OnramperPayments{}).
		Where("id = ? AND status IN ?", id, []string{models.OnramperStatusCreated, models.OnramperStatusPending}).
		Updates(updates)
	return res.RowsAffected, res.Error
}

// MarkExpired bulk-flips stale pending sessions to 'expired'.
func (r *OnramperPaymentsRepositoryImpl) MarkExpired(olderThan time.Time) (int64, error) {
	res := r.db.Model(&models.OnramperPayments{}).
		Where("status IN ? AND created_at < ?",
			[]string{models.OnramperStatusCreated, models.OnramperStatusPending},
			olderThan,
		).
		Updates(map[string]any{"status": models.OnramperStatusExpired, "updated_at": time.Now()})
	return res.RowsAffected, res.Error
}

// ListPendingExpired returns up to limit pending+created sessions older than
// olderThan, ordered by created_at ASC.
func (r *OnramperPaymentsRepositoryImpl) ListPendingExpired(olderThan time.Time, limit int) ([]models.OnramperPayments, error) {
	var rows []models.OnramperPayments
	q := r.db.Where(
		"status IN ? AND created_at < ?",
		[]string{models.OnramperStatusCreated, models.OnramperStatusPending},
		olderThan,
	).Order("created_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
