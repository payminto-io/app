package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MemberService manages member CRUD operations and account state transitions.
type MemberService struct {
	memberRepo repository.MemberRepository
	mepRoleSvc *MemberExternalPlatformRoleService
}

// NewMemberService constructs a MemberService.
func NewMemberService(memberRepo repository.MemberRepository, mepRoleSvc *MemberExternalPlatformRoleService) *MemberService {
	return &MemberService{memberRepo: memberRepo, mepRoleSvc: mepRoleSvc}
}

// CreateMemberInput carries caller-supplied fields for a new member.
type CreateMemberInput struct {
	Name       string
	Email      string
	Password   string
	MemberType string // "merchant" | "customer" | "admin"
	Username   string
}

// CreateMember creates a new member. Returns ErrDuplicateEmail when the
// email is already registered.
func (s *MemberService) CreateMember(_ context.Context, input CreateMemberInput) (*models.Member, error) {
	// Duplicate-email guard.
	if input.Email != "" {
		_, err := s.memberRepo.GetByEmail(input.Email)
		if err == nil {
			return nil, fmt.Errorf("member with email %q already exists", input.Email)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("check email: %w", err)
		}
	}

	memberType := input.MemberType
	if memberType == "" {
		memberType = "customer"
	}

	m := &models.Member{
		Name:       input.Name,
		MemberType: memberType,
		State:      "active",
	}
	if input.Email != "" {
		email := input.Email
		m.Email = &email
	}
	if input.Username != "" {
		u := input.Username
		m.Username = &u
	}
	if input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		h := string(hashed)
		m.Password = &h
	}

	if err := s.memberRepo.Create(m); err != nil {
		return nil, fmt.Errorf("create member: %w", err)
	}
	return m, nil
}

// GetByID returns a member by primary key.
func (s *MemberService) GetByID(_ context.Context, id uint) (*models.Member, error) {
	return s.memberRepo.GetByID(id)
}

// GetByEmail returns a member by email.
func (s *MemberService) GetByEmail(_ context.Context, email string) (*models.Member, error) {
	return s.memberRepo.GetByEmail(email)
}

// ListByPlatform returns all members assigned to the given external platform.
// The join is via member_external_platform_roles.
func (s *MemberService) ListByPlatform(ctx context.Context, platformID uint) ([]models.Member, error) {
	recs, err := s.mepRoleSvc.ListByPlatform(ctx, platformID)
	if err != nil {
		return nil, err
	}
	seen := make(map[uint]struct{})
	members := make([]models.Member, 0, len(recs))
	for _, rec := range recs {
		if rec.Member == nil {
			continue
		}
		if _, dup := seen[rec.MemberID]; dup {
			continue
		}
		seen[rec.MemberID] = struct{}{}
		members = append(members, *rec.Member)
	}
	return members, nil
}

// ListCustomersFilter configures pagination for ListCustomers.
type ListCustomersFilter struct {
	Search string
	Limit  int
	Offset int
}

// ListCustomers returns paginated members where member_type = "customer",
// optionally filtered by a search query (matches name or email). Returns
// the matching customers and the total count for pagination UI.
func (s *MemberService) ListCustomers(_ context.Context, filter ListCustomersFilter) ([]models.Member, int64, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	// When a search query is present, filter in-memory after loading.
	// This is acceptable for the dashboard combobox (small result set).
	if filter.Search != "" {
		all, err := s.memberRepo.ListByType("customer")
		if err != nil {
			return nil, 0, fmt.Errorf("list customers: %w", err)
		}
		q := strings.ToLower(filter.Search)
		filtered := make([]models.Member, 0, len(all))
		for _, c := range all {
			if strings.Contains(strings.ToLower(c.Name), q) {
				filtered = append(filtered, c)
				continue
			}
			if c.Email != nil && strings.Contains(strings.ToLower(*c.Email), q) {
				filtered = append(filtered, c)
			}
		}
		total := int64(len(filtered))
		// Apply limit + offset in memory.
		start := filter.Offset
		if start > len(filtered) {
			start = len(filtered)
		}
		end := start + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		return filtered[start:end], total, nil
	}

	total, err := s.memberRepo.CountByType("customer")
	if err != nil {
		return nil, 0, fmt.Errorf("count customers: %w", err)
	}

	opts := []repository.QueryOption{
		repository.WithLimit(limit),
		repository.WithOffset(filter.Offset),
	}
	customers, err := s.memberRepo.ListByType("customer", opts...)
	if err != nil {
		return nil, 0, fmt.Errorf("list customers: %w", err)
	}

	return customers, total, nil
}

// UpdateMemberInput carries updatable fields. Zero-value fields are ignored.
type UpdateMemberInput struct {
	Name     string
	Username string
}

// Update applies non-zero fields to the member record.
func (s *MemberService) Update(_ context.Context, id uint, input UpdateMemberInput) (*models.Member, error) {
	m, err := s.memberRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("get member: %w", err)
	}
	if input.Name != "" {
		m.Name = input.Name
	}
	if input.Username != "" {
		u := input.Username
		m.Username = &u
	}
	if err := s.memberRepo.Update(m); err != nil {
		return nil, fmt.Errorf("update member: %w", err)
	}
	return m, nil
}

// Activate sets the member state to "active".
func (s *MemberService) Activate(_ context.Context, id uint) error {
	m, err := s.memberRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	m.State = "active"
	return s.memberRepo.Update(m)
}

// Deactivate sets the member state to "inactive".
func (s *MemberService) Deactivate(_ context.Context, id uint) error {
	m, err := s.memberRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	m.State = "inactive"
	return s.memberRepo.Update(m)
}

// ChangePassword updates a member's password after verifying the current one.
func (s *MemberService) ChangePassword(_ context.Context, id uint, currentPassword, newPassword string) error {
	m, err := s.memberRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("get member: %w", err)
	}
	if m.Password == nil {
		return errors.New("member has no password set")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*m.Password), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	h := string(hashed)
	m.Password = &h
	return s.memberRepo.Update(m)
}
