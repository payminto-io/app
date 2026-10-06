package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// AuthRefreshTokenRepository defines all data access operations for refresh
// tokens with rotation-family reuse detection.
type AuthRefreshTokenRepository interface {
	Create(t *models.AuthRefreshToken) error
	GetByTokenHash(hash string) (*models.AuthRefreshToken, error)
	RevokeByID(id uint) error
	RevokeFamilyByRotationFamily(family string) error
	RevokeAllByMemberID(memberID uint) error
	DeleteExpired(before time.Time) (int64, error)
}

// AuthRefreshTokenRepositoryImpl is the GORM-backed implementation.
type AuthRefreshTokenRepositoryImpl struct {
	db *gorm.DB
}

// NewAuthRefreshTokenRepository constructs an AuthRefreshTokenRepositoryImpl.
func NewAuthRefreshTokenRepository(db *gorm.DB) AuthRefreshTokenRepository {
	return &AuthRefreshTokenRepositoryImpl{db: db}
}

// Create inserts a new AuthRefreshToken row.
func (r *AuthRefreshTokenRepositoryImpl) Create(t *models.AuthRefreshToken) error {
	return r.db.Create(t).Error
}

// GetByTokenHash fetches the token whose hash matches. Returns gorm.ErrRecordNotFound
// if no un-revoked row exists.
func (r *AuthRefreshTokenRepositoryImpl) GetByTokenHash(hash string) (*models.AuthRefreshToken, error) {
	var t models.AuthRefreshToken
	err := r.db.Where("token_hash = ?", hash).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// RevokeByID marks a single token as revoked.
func (r *AuthRefreshTokenRepositoryImpl) RevokeByID(id uint) error {
	now := time.Now()
	return r.db.Model(&models.AuthRefreshToken{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now).Error
}

// RevokeFamilyByRotationFamily revokes every token in the given rotation family.
// Called when reuse of an old token is detected.
func (r *AuthRefreshTokenRepositoryImpl) RevokeFamilyByRotationFamily(family string) error {
	now := time.Now()
	return r.db.Model(&models.AuthRefreshToken{}).
		Where("rotation_family = ? AND revoked_at IS NULL", family).
		Update("revoked_at", now).Error
}

// RevokeAllByMemberID revokes every non-revoked token for the member.
func (r *AuthRefreshTokenRepositoryImpl) RevokeAllByMemberID(memberID uint) error {
	now := time.Now()
	return r.db.Model(&models.AuthRefreshToken{}).
		Where("member_id = ? AND revoked_at IS NULL", memberID).
		Update("revoked_at", now).Error
}

// DeleteExpired hard-deletes expired tokens older than before. Returns the
// number of rows deleted.
func (r *AuthRefreshTokenRepositoryImpl) DeleteExpired(before time.Time) (int64, error) {
	result := r.db.Unscoped().
		Where("expires_at < ?", before).
		Delete(&models.AuthRefreshToken{})
	return result.RowsAffected, result.Error
}
