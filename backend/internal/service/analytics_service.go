package service

import (
	"context"
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/repository"
)

// AnalyticsService provides read-only aggregation queries scoped to a platform.
// All data is fetched via AnalyticsRepository; this service layer adds input
// validation and default values.
type AnalyticsService struct {
	repo repository.AnalyticsRepository
}

// NewAnalyticsService constructs an AnalyticsService.
func NewAnalyticsService(repo repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// GetVolumeOverTime returns time-bucketed confirmed deposit volume for the platform.
// interval must be "day", "week", or "month"; defaults to "day".
// start and end are clamped so end > start.
func (s *AnalyticsService) GetVolumeOverTime(_ context.Context, platformID uint, start, end time.Time, interval string) ([]repository.VolumeBucket, error) {
	switch interval {
	case "day", "week", "month":
	default:
		interval = "day"
	}
	if !end.After(start) {
		return nil, fmt.Errorf("analytics: end must be after start")
	}
	return s.repo.GetVolumeOverTime(platformID, start, end, interval)
}

// GetTopCustomers returns the top N customers by confirmed deposit volume.
// limit is clamped to [1, 100].
func (s *AnalyticsService) GetTopCustomers(_ context.Context, platformID uint, limit int) ([]repository.CustomerSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.GetTopCustomers(platformID, limit)
}

// GetRevenueBreakdown returns payment totals grouped by chain, currency, and state.
func (s *AnalyticsService) GetRevenueBreakdown(_ context.Context, platformID uint, start, end time.Time) ([]repository.RevenueBreakdown, error) {
	if !end.After(start) {
		return nil, fmt.Errorf("analytics: end must be after start")
	}
	return s.repo.GetRevenueBreakdown(platformID, start, end)
}

// GetSweepStats returns aggregate sweep statistics for the given time window.
func (s *AnalyticsService) GetSweepStats(_ context.Context, platformID uint, start, end time.Time) (repository.SweepStats, error) {
	if !end.After(start) {
		return repository.SweepStats{}, fmt.Errorf("analytics: end must be after start")
	}
	return s.repo.GetSweepStats(platformID, start, end)
}

// GetWithdrawalStats returns aggregate withdrawal statistics for the given time window.
func (s *AnalyticsService) GetWithdrawalStats(_ context.Context, platformID uint, start, end time.Time) (repository.WithdrawalStats, error) {
	if !end.After(start) {
		return repository.WithdrawalStats{}, fmt.Errorf("analytics: end must be after start")
	}
	return s.repo.GetWithdrawalStats(platformID, start, end)
}

// GetDashboardSummary returns the all-up counts and totals for the dashboard home tile.
func (s *AnalyticsService) GetDashboardSummary(_ context.Context, platformID uint) (repository.DashboardSummary, error) {
	return s.repo.GetDashboardSummary(platformID)
}
