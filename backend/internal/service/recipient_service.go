package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// ErrRecipientNotOwned is returned when a member tries to access another
// member's recipient.
var ErrRecipientNotOwned = errors.New("recipient not found or not owned by member")

// RecipientService manages merchant payee book entries.
type RecipientService struct {
	repo repository.RecipientRepository
}

// NewRecipientService constructs a RecipientService.
func NewRecipientService(repo repository.RecipientRepository) *RecipientService {
	return &RecipientService{repo: repo}
}

// CreateRecipientInput carries caller-supplied fields for a new recipient.
type CreateRecipientInput struct {
	MemberID           uint
	ExternalPlatformID uint
	Name               string
	Email              string
	BlockchainCode     string
	CurrencyCode       string
	Address            string
	Memo               string
}

// Create adds a new recipient to the payee book.
func (s *RecipientService) Create(_ context.Context, input CreateRecipientInput) (*models.Recipient, error) {
	rec := &models.Recipient{
		MemberID:           input.MemberID,
		ExternalPlatformID: input.ExternalPlatformID,
		Name:               input.Name,
		BlockchainCode:     input.BlockchainCode,
		CurrencyCode:       input.CurrencyCode,
		Address:            input.Address,
	}
	if input.Email != "" {
		e := input.Email
		rec.Email = &e
	}
	if input.Memo != "" {
		m := input.Memo
		rec.Memo = &m
	}
	if err := s.repo.Create(rec); err != nil {
		return nil, fmt.Errorf("create recipient: %w", err)
	}
	return rec, nil
}

// GetByID returns a recipient, verifying it belongs to the given member.
func (s *RecipientService) GetByID(_ context.Context, id, memberID uint) (*models.Recipient, error) {
	rec, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrRecipientNotOwned
	}
	if rec.MemberID != memberID {
		return nil, ErrRecipientNotOwned
	}
	return rec, nil
}

// ListByMember returns all recipients owned by a member.
func (s *RecipientService) ListByMember(_ context.Context, memberID uint) ([]models.Recipient, error) {
	return s.repo.ListByMember(memberID)
}

// UpdateRecipientInput carries updatable fields.
type UpdateRecipientInput struct {
	Name    string
	Email   string
	Memo    string
	Address string
}

// Update applies non-zero fields to a recipient, verifying ownership.
func (s *RecipientService) Update(_ context.Context, id, memberID uint, input UpdateRecipientInput) (*models.Recipient, error) {
	rec, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrRecipientNotOwned
	}
	if rec.MemberID != memberID {
		return nil, ErrRecipientNotOwned
	}
	if input.Name != "" {
		rec.Name = input.Name
	}
	if input.Email != "" {
		e := input.Email
		rec.Email = &e
	}
	if input.Memo != "" {
		m := input.Memo
		rec.Memo = &m
	}
	if input.Address != "" {
		rec.Address = input.Address
	}
	if err := s.repo.Update(rec); err != nil {
		return nil, fmt.Errorf("update recipient: %w", err)
	}
	return rec, nil
}

// Delete removes a recipient, verifying ownership.
func (s *RecipientService) Delete(_ context.Context, id, memberID uint) error {
	rec, err := s.repo.GetByID(id)
	if err != nil {
		return ErrRecipientNotOwned
	}
	if rec.MemberID != memberID {
		return ErrRecipientNotOwned
	}
	return s.repo.Delete(id)
}
