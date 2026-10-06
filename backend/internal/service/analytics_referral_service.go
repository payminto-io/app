package service

import (
	"context"

	"github.com/payminto/payminto/backend/internal/repository"
)

// AnalyticsReferralService provides referral-specific analytics queries.
type AnalyticsReferralService struct {
	repo repository.AnalyticsReferralRepository
}

// NewAnalyticsReferralService constructs an AnalyticsReferralService.
func NewAnalyticsReferralService(repo repository.AnalyticsReferralRepository) *AnalyticsReferralService {
	return &AnalyticsReferralService{repo: repo}
}

// GetReferralStats returns aggregate stats for a single referrer (total referrals,
// total earned, conversion rate).
func (s *AnalyticsReferralService) GetReferralStats(_ context.Context, memberID uint) (repository.ReferralStats, error) {
	return s.repo.GetReferralStats(memberID)
}

// GetTopReferrers returns the top N referrers sorted by number of referrals.
func (s *AnalyticsReferralService) GetTopReferrers(_ context.Context, limit int) ([]repository.TopReferrerRow, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.repo.GetTopReferrers(limit)
}

// GetCampaignPerformance returns aggregate referral and reward stats for one campaign.
func (s *AnalyticsReferralService) GetCampaignPerformance(_ context.Context, campaignID uint) (repository.CampaignPerformance, error) {
	return s.repo.GetCampaignPerformance(campaignID)
}
