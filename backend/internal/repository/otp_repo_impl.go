package repository

import (
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ErrOTPNotFound is returned when no OTP row can be located by GetByID.
var ErrOTPNotFound = errors.New("OTP not found")

// OTPRepository defines all data access operations for OTPs.
type OTPRepository interface {
	Create(o *models.OTP) error
	GetByID(id uint) (*models.OTP, error)
	GetLatestByMemberAndPurpose(memberID uint, purpose string) (*models.OTP, error)
	// ClaimAttempt atomically increments attempts only if the OTP is not yet
	// used/expired/maxed. Returns rowsAffected == 1 on success, 0 if the claim
	// was rejected (caller should re-read to give the right sentinel error).
	ClaimAttempt(id uint) (int64, error)
	// IncrementAttempts is kept for backward compatibility but deprecated in
	// favour of ClaimAttempt. New code should use ClaimAttempt instead.
	//
	// Deprecated: use ClaimAttempt which is atomic and bounded.
	IncrementAttempts(id uint) error
	MarkUsed(id uint) error
	InvalidateByMemberAndPurpose(memberID uint, purpose string) error
	DeleteExpired(before time.Time) (int64, error)
}

// OTPRepositoryImpl is the GORM-backed implementation of OTPRepository.
type OTPRepositoryImpl struct {
	db *gorm.DB
}

// NewOTPRepository constructs an OTPRepositoryImpl.
func NewOTPRepository(db *gorm.DB) OTPRepository {
	return &OTPRepositoryImpl{db: db}
}

// Create inserts a new OTP row.
func (r *OTPRepositoryImpl) Create(o *models.OTP) error {
	return r.db.Create(o).Error
}

// GetByID fetches an OTP by primary key.
func (r *OTPRepositoryImpl) GetByID(id uint) (*models.OTP, error) {
	var o models.OTP
	err := r.db.First(&o, id).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// GetLatestByMemberAndPurpose returns the most recently created unused OTP for
// the given member and purpose, or gorm.ErrRecordNotFound if none exists.
func (r *OTPRepositoryImpl) GetLatestByMemberAndPurpose(memberID uint, purpose string) (*models.OTP, error) {
	var o models.OTP
	err := r.db.
		Where("member_id = ? AND purpose = ? AND used_at IS NULL", memberID, purpose).
		Order("created_at DESC").
		First(&o).Error
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// ClaimAttempt atomically increments the attempts counter only when the OTP is
// still valid (not used, not expired, not maxed). Returns (rowsAffected, error).
// rowsAffected == 0 means the claim was rejected — caller re-reads with GetByID
// to return the correct sentinel error.
func (r *OTPRepositoryImpl) ClaimAttempt(id uint) (int64, error) {
	result := r.db.Model(&models.OTP{}).
		Where(`id = ?
		  AND used_at IS NULL
		  AND attempts < max_attempts
		  AND expires_at > ?`, id, time.Now()).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"updated_at": time.Now(),
		})
	return result.RowsAffected, result.Error
}

// IncrementAttempts atomically increments the attempts counter.
//
// Deprecated: use ClaimAttempt which is atomic and bounded.
func (r *OTPRepositoryImpl) IncrementAttempts(id uint) error {
	return r.db.Model(&models.OTP{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

// MarkUsed sets used_at to now for the given OTP row.
func (r *OTPRepositoryImpl) MarkUsed(id uint) error {
	now := time.Now()
	return r.db.Model(&models.OTP{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", now).Error
}

// InvalidateByMemberAndPurpose marks all pending OTPs for the given
// member+purpose as used, effectively invalidating them.
func (r *OTPRepositoryImpl) InvalidateByMemberAndPurpose(memberID uint, purpose string) error {
	now := time.Now()
	return r.db.Model(&models.OTP{}).
		Where("member_id = ? AND purpose = ? AND used_at IS NULL", memberID, purpose).
		Update("used_at", now).Error
}

// DeleteExpired hard-deletes OTPs that have expired before the given time.
func (r *OTPRepositoryImpl) DeleteExpired(before time.Time) (int64, error) {
	result := r.db.Unscoped().
		Where("expires_at < ?", before).
		Delete(&models.OTP{})
	return result.RowsAffected, result.Error
}
