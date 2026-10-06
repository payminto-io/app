package repository

import (
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ─── ReferralCampaign ────────────────────────────────────────────────────────

// ReferralCampaignRepository defines CRUD for referral campaigns.
type ReferralCampaignRepository interface {
	Create(c *models.ReferralCampaign) error
	Update(c *models.ReferralCampaign) error
	Delete(id uint) error
	GetByID(id uint) (*models.ReferralCampaign, error)
	ListByProject(project string) ([]models.ReferralCampaign, error)
	SetStatus(id uint, status string) error
}

// ReferralCampaignRepositoryImpl is the GORM-backed implementation.
type ReferralCampaignRepositoryImpl struct{ db *gorm.DB }

// NewReferralCampaignRepository constructs a ReferralCampaignRepository.
func NewReferralCampaignRepository(db *gorm.DB) ReferralCampaignRepository {
	return &ReferralCampaignRepositoryImpl{db: db}
}

// Create inserts a new ReferralCampaign.
func (r *ReferralCampaignRepositoryImpl) Create(c *models.ReferralCampaign) error {
	return r.db.Create(c).Error
}

// Update saves all fields of the given ReferralCampaign.
func (r *ReferralCampaignRepositoryImpl) Update(c *models.ReferralCampaign) error {
	return r.db.Save(c).Error
}

// Delete soft-deletes a ReferralCampaign.
func (r *ReferralCampaignRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.ReferralCampaign{}, id).Error
}

// GetByID fetches a ReferralCampaign by primary key.
func (r *ReferralCampaignRepositoryImpl) GetByID(id uint) (*models.ReferralCampaign, error) {
	var c models.ReferralCampaign
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// ListByProject returns all campaigns belonging to a project.
func (r *ReferralCampaignRepositoryImpl) ListByProject(project string) ([]models.ReferralCampaign, error) {
	var cs []models.ReferralCampaign
	if err := r.db.Where("project = ?", project).Find(&cs).Error; err != nil {
		return nil, err
	}
	return cs, nil
}

// SetStatus updates the campaign status.
func (r *ReferralCampaignRepositoryImpl) SetStatus(id uint, status string) error {
	return r.db.Model(&models.ReferralCampaign{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now()}).Error
}

// ─── ReferralMember ──────────────────────────────────────────────────────────

// ReferralMemberRepository manages referral member records (referral codes + referrer links).
type ReferralMemberRepository interface {
	Create(m *models.ReferralMember) error
	Update(m *models.ReferralMember) error
	GetByMemberID(memberID uint) (*models.ReferralMember, error)
	GetByCode(code string) (*models.ReferralMember, error)
}

// ReferralMemberRepositoryImpl is the GORM-backed implementation.
type ReferralMemberRepositoryImpl struct{ db *gorm.DB }

// NewReferralMemberRepository constructs a ReferralMemberRepository.
func NewReferralMemberRepository(db *gorm.DB) ReferralMemberRepository {
	return &ReferralMemberRepositoryImpl{db: db}
}

// Create inserts a new ReferralMember.
func (r *ReferralMemberRepositoryImpl) Create(m *models.ReferralMember) error {
	return r.db.Create(m).Error
}

// Update saves all fields of the given ReferralMember.
func (r *ReferralMemberRepositoryImpl) Update(m *models.ReferralMember) error {
	return r.db.Save(m).Error
}

// GetByMemberID fetches the ReferralMember for a given member.
func (r *ReferralMemberRepositoryImpl) GetByMemberID(memberID uint) (*models.ReferralMember, error) {
	var m models.ReferralMember
	if err := r.db.Where("member_id = ?", memberID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByCode fetches a ReferralMember by their referral code.
func (r *ReferralMemberRepositoryImpl) GetByCode(code string) (*models.ReferralMember, error) {
	var m models.ReferralMember
	if err := r.db.Where("referral_code = ?", code).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// ─── ReferralReward ──────────────────────────────────────────────────────────

// ReferralRewardRepository manages referral reward rows.
type ReferralRewardRepository interface {
	Create(r *models.ReferralReward) error
	Update(r *models.ReferralReward) error
	GetByID(id uint) (*models.ReferralReward, error)
	ListByMember(memberID uint) ([]models.ReferralReward, error)
	ListPendingByMember(memberID uint) ([]models.ReferralReward, error)
	ListAllPending() ([]models.ReferralReward, error)
	SetStatus(id uint, status string, processedAt *time.Time, reason *string) error
}

// ReferralRewardRepositoryImpl is the GORM-backed implementation.
type ReferralRewardRepositoryImpl struct{ db *gorm.DB }

// NewReferralRewardRepository constructs a ReferralRewardRepository.
func NewReferralRewardRepository(db *gorm.DB) ReferralRewardRepository {
	return &ReferralRewardRepositoryImpl{db: db}
}

// Create inserts a new ReferralReward.
func (r *ReferralRewardRepositoryImpl) Create(rw *models.ReferralReward) error {
	return r.db.Create(rw).Error
}

// Update saves all fields of the given ReferralReward.
func (r *ReferralRewardRepositoryImpl) Update(rw *models.ReferralReward) error {
	return r.db.Save(rw).Error
}

// GetByID fetches a ReferralReward by primary key.
func (r *ReferralRewardRepositoryImpl) GetByID(id uint) (*models.ReferralReward, error) {
	var rw models.ReferralReward
	if err := r.db.First(&rw, id).Error; err != nil {
		return nil, err
	}
	return &rw, nil
}

// ListByMember returns all rewards for a member.
func (r *ReferralRewardRepositoryImpl) ListByMember(memberID uint) ([]models.ReferralReward, error) {
	var rws []models.ReferralReward
	if err := r.db.Where("member_id = ?", memberID).Order("created_at DESC").Find(&rws).Error; err != nil {
		return nil, err
	}
	return rws, nil
}

// ListPendingByMember returns pending rewards for a member.
func (r *ReferralRewardRepositoryImpl) ListPendingByMember(memberID uint) ([]models.ReferralReward, error) {
	var rws []models.ReferralReward
	if err := r.db.Where("member_id = ? AND status = 'pending'", memberID).Find(&rws).Error; err != nil {
		return nil, err
	}
	return rws, nil
}

// ListAllPending returns all pending rewards across all members.
func (r *ReferralRewardRepositoryImpl) ListAllPending() ([]models.ReferralReward, error) {
	var rws []models.ReferralReward
	if err := r.db.Where("status = 'pending'").Find(&rws).Error; err != nil {
		return nil, err
	}
	return rws, nil
}

// SetStatus atomically updates the status field of a reward.
func (r *ReferralRewardRepositoryImpl) SetStatus(id uint, status string, processedAt *time.Time, reason *string) error {
	updates := map[string]any{
		"status":     status,
		"updated_at": time.Now(),
	}
	if processedAt != nil {
		updates["processed_at"] = processedAt
	}
	if reason != nil {
		updates["failure_reason"] = reason
	}
	return r.db.Model(&models.ReferralReward{}).Where("id = ?", id).Updates(updates).Error
}

// ─── Referral analytics ──────────────────────────────────────────────────────

// ReferralStats summarises a single referrer's performance.
type ReferralStats struct {
	// MemberID is the referrer.
	MemberID      uint
	// TotalReferrals is the count of members referred.
	TotalReferrals int
	// TotalEarned is the sum of all paid/pending rewards.
	TotalEarned   float64
	// ConversionRate is the fraction of referred members who made a qualifying payment.
	ConversionRate float64
}

// TopReferrerRow is one row in the top-referrers leaderboard.
type TopReferrerRow struct {
	// MemberID is the referrer's member ID.
	MemberID      uint
	// TotalReferrals is the count of referred members.
	TotalReferrals int
	// TotalEarned is the sum of rewards earned.
	TotalEarned   float64
}

// CampaignPerformance aggregates stats for one campaign.
type CampaignPerformance struct {
	// CampaignID identifies the campaign.
	CampaignID    uint
	// TotalReferrals is the count of referral events under this campaign.
	TotalReferrals int
	// TotalRewards is the total amount of rewards issued.
	TotalRewards  float64
}

// AnalyticsReferralRepository defines read-only aggregation queries for the
// referral subsystem.
type AnalyticsReferralRepository interface {
	GetReferralStats(memberID uint) (ReferralStats, error)
	GetTopReferrers(limit int) ([]TopReferrerRow, error)
	GetCampaignPerformance(campaignID uint) (CampaignPerformance, error)
}

// AnalyticsReferralRepositoryImpl is the GORM-backed implementation.
type AnalyticsReferralRepositoryImpl struct{ db *gorm.DB }

// NewAnalyticsReferralRepository constructs an AnalyticsReferralRepository.
func NewAnalyticsReferralRepository(db *gorm.DB) AnalyticsReferralRepository {
	return &AnalyticsReferralRepositoryImpl{db: db}
}

// GetReferralStats returns aggregate stats for a single referrer.
func (r *AnalyticsReferralRepositoryImpl) GetReferralStats(memberID uint) (ReferralStats, error) {
	type countRow struct{ Total int }
	var refCount countRow
	r.db.Raw(`SELECT COUNT(*) AS total FROM referral_members WHERE referred_by = ? AND deleted_at IS NULL`, memberID).Scan(&refCount)

	type earnRow struct{ Total float64 }
	var earned earnRow
	r.db.Raw(`
		SELECT COALESCE(SUM(CAST(amount AS REAL)), 0) AS total
		FROM referral_rewards
		WHERE member_id = ? AND (status = 'pending' OR status = 'paid') AND deleted_at IS NULL
	`, memberID).Scan(&earned)

	// Conversion = members who made at least one qualifying payment / total referred.
	type convRow struct{ Total int }
	var converted convRow
	r.db.Raw(`
		SELECT COUNT(DISTINCT rm.member_id) AS total
		FROM referral_members rm
		JOIN referral_rewards rr ON rr.member_id = rm.member_id AND rr.deleted_at IS NULL
		WHERE rm.referred_by = ? AND rm.deleted_at IS NULL
	`, memberID).Scan(&converted)

	var rate float64
	if refCount.Total > 0 {
		rate = float64(converted.Total) / float64(refCount.Total)
	}

	return ReferralStats{
		MemberID:       memberID,
		TotalReferrals: refCount.Total,
		TotalEarned:    earned.Total,
		ConversionRate: rate,
	}, nil
}

// GetTopReferrers returns the top N referrers by count of referred members.
func (r *AnalyticsReferralRepositoryImpl) GetTopReferrers(limit int) ([]TopReferrerRow, error) {
	if limit <= 0 {
		limit = 10
	}
	type row struct {
		MemberID      uint
		TotalReferrals int
		TotalEarned   float64
	}
	var rows []row
	err := r.db.Raw(`
		SELECT
			rm.referred_by AS member_id,
			COUNT(*) AS total_referrals,
			COALESCE(SUM(CAST(rr.amount AS REAL)), 0) AS total_earned
		FROM referral_members rm
		LEFT JOIN referral_rewards rr
			ON rr.member_id = rm.member_id
			AND (rr.status = 'pending' OR rr.status = 'paid')
			AND rr.deleted_at IS NULL
		WHERE rm.referred_by IS NOT NULL
		  AND rm.deleted_at IS NULL
		GROUP BY rm.referred_by
		ORDER BY total_referrals DESC
		LIMIT ?
	`, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]TopReferrerRow, len(rows))
	for i, r := range rows {
		result[i] = TopReferrerRow{
			MemberID:       r.MemberID,
			TotalReferrals: r.TotalReferrals,
			TotalEarned:    r.TotalEarned,
		}
	}
	return result, nil
}

// GetCampaignPerformance returns aggregate stats for a single campaign.
func (r *AnalyticsReferralRepositoryImpl) GetCampaignPerformance(campaignID uint) (CampaignPerformance, error) {
	type row struct {
		TotalReferrals int
		TotalRewards   float64
	}
	var result row
	err := r.db.Raw(`
		SELECT
			COUNT(DISTINCT rr.member_id) AS total_referrals,
			COALESCE(SUM(CAST(rr.amount AS REAL)), 0) AS total_rewards
		FROM referral_rewards rr
		WHERE rr.campaign_id = ?
		  AND rr.deleted_at IS NULL
	`, campaignID).Scan(&result).Error
	if err != nil {
		return CampaignPerformance{}, err
	}

	// Check existence.
	var count int64
	if err := r.db.Model(&models.ReferralCampaign{}).
		Where("id = ? AND deleted_at IS NULL", campaignID).
		Count(&count).Error; err != nil || count == 0 {
		if !errors.Is(err, gorm.ErrRecordNotFound) && count == 0 {
			return CampaignPerformance{}, gorm.ErrRecordNotFound
		}
	}

	return CampaignPerformance{
		CampaignID:     campaignID,
		TotalReferrals: result.TotalReferrals,
		TotalRewards:   result.TotalRewards,
	}, nil
}
