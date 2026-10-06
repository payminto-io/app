package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// MemberRepository defines all data access operations for the Member model.
type MemberRepository interface {
	Create(m *models.Member) error
	Update(m *models.Member) error
	Delete(id uint) error
	GetByID(id uint) (*models.Member, error)
	GetByEmail(email string) (*models.Member, error)
	GetByUsername(username string) (*models.Member, error)
	GetByResetPasswordToken(token string) (*models.Member, error)
	List(opts ...QueryOption) ([]models.Member, error)
	Count() (int64, error)
	ListByType(memberType string, opts ...QueryOption) ([]models.Member, error)
	CountByType(memberType string) (int64, error)
	GetByCustomerIDAndExternalPlatformID(customerID string, platformID uint) (*models.Member, error)
}

// MemberRepositoryImpl is the GORM-backed implementation of MemberRepository.
type MemberRepositoryImpl struct {
	db *gorm.DB
}

// NewMemberRepository constructs a MemberRepositoryImpl bound to the given DB handle.
func NewMemberRepository(db *gorm.DB) MemberRepository {
	return &MemberRepositoryImpl{db: db}
}

// Create inserts a new Member row.
func (r *MemberRepositoryImpl) Create(m *models.Member) error {
	return r.db.Create(m).Error
}

// Update saves all non-zero fields on the member (full save).
func (r *MemberRepositoryImpl) Update(m *models.Member) error {
	return r.db.Save(m).Error
}

// Delete soft-deletes the member with the given id.
func (r *MemberRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.Member{}, id).Error
}

// GetByID returns a Member with its Roles association preloaded.
func (r *MemberRepositoryImpl) GetByID(id uint) (*models.Member, error) {
	var m models.Member
	err := r.db.Preload("Roles").First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByEmail fetches a Member by email address.
func (r *MemberRepositoryImpl) GetByEmail(email string) (*models.Member, error) {
	var m models.Member
	err := r.db.Where("email = ?", email).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByUsername fetches a Member by their unique username.
func (r *MemberRepositoryImpl) GetByUsername(username string) (*models.Member, error) {
	var m models.Member
	err := r.db.Where("username = ?", username).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByResetPasswordToken fetches a Member by their password-reset token.
func (r *MemberRepositoryImpl) GetByResetPasswordToken(token string) (*models.Member, error) {
	var m models.Member
	err := r.db.Where("reset_password_token = ?", token).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// List returns members with optional query options applied (pagination, ordering, etc.).
func (r *MemberRepositoryImpl) List(opts ...QueryOption) ([]models.Member, error) {
	var members []models.Member
	q := Apply(r.db.Model(&models.Member{}), opts...)
	err := q.Find(&members).Error
	return members, err
}

// Count returns the total number of non-deleted members.
func (r *MemberRepositoryImpl) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Member{}).Count(&count).Error
	return count, err
}

// ListByType returns members filtered by member_type with optional query options.
func (r *MemberRepositoryImpl) ListByType(memberType string, opts ...QueryOption) ([]models.Member, error) {
	var members []models.Member
	q := Apply(r.db.Model(&models.Member{}).Where("member_type = ?", memberType), opts...)
	err := q.Find(&members).Error
	return members, err
}

// CountByType returns the total number of non-deleted members with the given type.
func (r *MemberRepositoryImpl) CountByType(memberType string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Member{}).Where("member_type = ?", memberType).Count(&count).Error
	return count, err
}

// GetByCustomerIDAndExternalPlatformID finds a member via the
// member_external_platform_roles join table.
func (r *MemberRepositoryImpl) GetByCustomerIDAndExternalPlatformID(customerID string, platformID uint) (*models.Member, error) {
	var m models.Member
	err := r.db.
		Joins("JOIN member_external_platform_roles ON member_external_platform_roles.member_id = members.id").
		Where("members.customer_id = ? AND member_external_platform_roles.external_platform_id = ?", customerID, platformID).
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}
