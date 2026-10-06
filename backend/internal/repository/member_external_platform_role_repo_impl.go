package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// MemberExternalPlatformRoleRepository manages per-project role assignments
// (the member_external_platform_roles join table).
type MemberExternalPlatformRoleRepository interface {
	Assign(rec *models.MemberExternalPlatformRole) error
	Revoke(memberID, platformID, roleID uint) error
	ListByMember(memberID uint) ([]models.MemberExternalPlatformRole, error)
	ListByPlatform(platformID uint) ([]models.MemberExternalPlatformRole, error)
	// GetWithRole returns the join record with its Role (and Role.Permissions) preloaded.
	GetWithRole(memberID, platformID, roleID uint) (*models.MemberExternalPlatformRole, error)
	// GetRolesByMemberAndPlatform returns all Role records (with Permissions) assigned
	// to a member on a given platform.
	GetRolesByMemberAndPlatform(memberID, platformID uint) ([]models.Role, error)
}

// MemberExternalPlatformRoleRepositoryImpl is the GORM-backed implementation.
type MemberExternalPlatformRoleRepositoryImpl struct {
	db *gorm.DB
}

// NewMemberExternalPlatformRoleRepository constructs the repo.
func NewMemberExternalPlatformRoleRepository(db *gorm.DB) MemberExternalPlatformRoleRepository {
	return &MemberExternalPlatformRoleRepositoryImpl{db: db}
}

// Assign inserts a new member-platform-role triple. Silently ignores duplicate-key errors.
func (r *MemberExternalPlatformRoleRepositoryImpl) Assign(rec *models.MemberExternalPlatformRole) error {
	return r.db.
		Where(models.MemberExternalPlatformRole{
			MemberID:           rec.MemberID,
			ExternalPlatformID: rec.ExternalPlatformID,
			RoleID:             rec.RoleID,
		}).
		FirstOrCreate(rec).Error
}

// Revoke removes the matching triple.
func (r *MemberExternalPlatformRoleRepositoryImpl) Revoke(memberID, platformID, roleID uint) error {
	return r.db.
		Where("member_id = ? AND external_platform_id = ? AND role_id = ?", memberID, platformID, roleID).
		Delete(&models.MemberExternalPlatformRole{}).Error
}

// ListByMember returns all role assignments for a member across all platforms.
func (r *MemberExternalPlatformRoleRepositoryImpl) ListByMember(memberID uint) ([]models.MemberExternalPlatformRole, error) {
	var recs []models.MemberExternalPlatformRole
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("member_id = ?", memberID).
		Find(&recs).Error
	return recs, err
}

// ListByPlatform returns all role assignments for all members on a platform.
func (r *MemberExternalPlatformRoleRepositoryImpl) ListByPlatform(platformID uint) ([]models.MemberExternalPlatformRole, error) {
	var recs []models.MemberExternalPlatformRole
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("external_platform_id = ?", platformID).
		Find(&recs).Error
	return recs, err
}

// GetWithRole returns the join record with Role + Permissions preloaded.
func (r *MemberExternalPlatformRoleRepositoryImpl) GetWithRole(memberID, platformID, roleID uint) (*models.MemberExternalPlatformRole, error) {
	var rec models.MemberExternalPlatformRole
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("member_id = ? AND external_platform_id = ? AND role_id = ?", memberID, platformID, roleID).
		First(&rec).Error
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetRolesByMemberAndPlatform returns all Role records (with Permissions) assigned
// to the member on the given platform.
func (r *MemberExternalPlatformRoleRepositoryImpl) GetRolesByMemberAndPlatform(memberID, platformID uint) ([]models.Role, error) {
	var recs []models.MemberExternalPlatformRole
	err := r.db.
		Preload("Role").
		Preload("Role.Permissions").
		Where("member_id = ? AND external_platform_id = ?", memberID, platformID).
		Find(&recs).Error
	if err != nil {
		return nil, err
	}
	roles := make([]models.Role, 0, len(recs))
	for _, rec := range recs {
		if rec.Role != nil {
			roles = append(roles, *rec.Role)
		}
	}
	return roles, nil
}
