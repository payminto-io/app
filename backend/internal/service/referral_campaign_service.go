package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ErrCampaignNotFound is returned when a campaign ID cannot be resolved.
var ErrCampaignNotFound = errors.New("referral campaign not found")

// CreateCampaignInput contains the fields required to create a campaign.
type CreateCampaignInput struct {
	Project      string
	Name         string
	Description  string
	RewardType   string
	RewardValue  decimal.Decimal
	CurrencyCode string
	Budget       *decimal.Decimal
	StartDate    *time.Time
	EndDate      *time.Time
}

// ReferralCampaignService manages referral campaign lifecycle.
type ReferralCampaignService struct {
	repo repository.ReferralCampaignRepository
}

// NewReferralCampaignService constructs a ReferralCampaignService.
func NewReferralCampaignService(repo repository.ReferralCampaignRepository) *ReferralCampaignService {
	return &ReferralCampaignService{repo: repo}
}

// CreateCampaign creates a new referral campaign with status "active".
func (s *ReferralCampaignService) CreateCampaign(input CreateCampaignInput) (*models.ReferralCampaign, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if input.Project == "" {
		return nil, fmt.Errorf("campaign project is required")
	}

	now := time.Now()
	rewardType := input.RewardType
	if rewardType == "" {
		rewardType = "fixed"
	}
	rewardVal := input.RewardValue
	cc := input.CurrencyCode
	if cc == "" {
		cc = "USDT"
	}

	c := &models.ReferralCampaign{
		Project:            input.Project,
		Name:               input.Name,
		Description:        toStringPtr(input.Description),
		RewardType:         &rewardType,
		RewardValue:        &rewardVal,
		CurrencyCode:       cc,
		Budget:             input.Budget,
		StartDate:          input.StartDate,
		EndDate:            input.EndDate,
		Status:             "active",
		ConsiderEventsFrom: now,
	}

	if err := s.repo.Create(c); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}
	return c, nil
}

// GetCampaign returns a campaign by ID.
func (s *ReferralCampaignService) GetCampaign(id uint) (*models.ReferralCampaign, error) {
	c, err := s.repo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return c, nil
}

// ListCampaigns returns all campaigns for a project.
func (s *ReferralCampaignService) ListCampaigns(project string) ([]models.ReferralCampaign, error) {
	return s.repo.ListByProject(project)
}

// UpdateCampaign applies allowed mutations to an existing campaign.
func (s *ReferralCampaignService) UpdateCampaign(id uint, input CreateCampaignInput) (*models.ReferralCampaign, error) {
	c, err := s.GetCampaign(id)
	if err != nil {
		return nil, err
	}
	if input.Name != "" {
		c.Name = input.Name
	}
	if input.Description != "" {
		c.Description = toStringPtr(input.Description)
	}
	if input.StartDate != nil {
		c.StartDate = input.StartDate
	}
	if input.EndDate != nil {
		c.EndDate = input.EndDate
	}
	if !input.RewardValue.IsZero() {
		rv := input.RewardValue
		c.RewardValue = &rv
	}
	if input.Budget != nil {
		c.Budget = input.Budget
	}
	if err := s.repo.Update(c); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	return c, nil
}

// DeleteCampaign soft-deletes a campaign.
func (s *ReferralCampaignService) DeleteCampaign(id uint) error {
	if _, err := s.GetCampaign(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// ActivateCampaign sets campaign status to "active".
func (s *ReferralCampaignService) ActivateCampaign(id uint) error {
	if _, err := s.GetCampaign(id); err != nil {
		return err
	}
	return s.repo.SetStatus(id, "active")
}

// DeactivateCampaign sets campaign status to "inactive".
func (s *ReferralCampaignService) DeactivateCampaign(id uint) error {
	if _, err := s.GetCampaign(id); err != nil {
		return err
	}
	return s.repo.SetStatus(id, "inactive")
}

// toStringPtr converts a string to a *string for optional fields. Returns nil for empty strings.
func toStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
