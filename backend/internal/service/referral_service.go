package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ErrReferralNotFound is returned when a referral code cannot be resolved.
var ErrReferralNotFound = errors.New("referral code not found")

// ErrAlreadyReferred is returned when the member already has a referrer.
var ErrAlreadyReferred = errors.New("member already has a referrer")

// ReferralService handles referral code registration and reward tracking.
type ReferralService struct {
	memberRepo   repository.ReferralMemberRepository
	rewardRepo   repository.ReferralRewardRepository
	campaignRepo repository.ReferralCampaignRepository
	ledgerSvc    *LedgerService
}

// NewReferralService constructs a ReferralService.
func NewReferralService(
	memberRepo repository.ReferralMemberRepository,
	rewardRepo repository.ReferralRewardRepository,
	campaignRepo repository.ReferralCampaignRepository,
	ledgerSvc *LedgerService,
) *ReferralService {
	return &ReferralService{
		memberRepo:   memberRepo,
		rewardRepo:   rewardRepo,
		campaignRepo: campaignRepo,
		ledgerSvc:    ledgerSvc,
	}
}

// generateReferralCode produces a short unique referral code.
func generateReferralCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)[:10]), nil
}

// EnsureReferralMember returns the existing ReferralMember for memberID or creates
// one if none exists. The project field scopes the code namespace.
func (s *ReferralService) EnsureReferralMember(memberID uint, project string) (*models.ReferralMember, error) {
	existing, err := s.memberRepo.GetByMemberID(memberID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	code, err := generateReferralCode()
	if err != nil {
		return nil, fmt.Errorf("generate referral code: %w", err)
	}

	rm := &models.ReferralMember{
		MemberID:     memberID,
		ReferralCode: code,
		Project:      project,
		Status:       "active",
	}
	if err := s.memberRepo.Create(rm); err != nil {
		return nil, fmt.Errorf("create referral member: %w", err)
	}
	return rm, nil
}

// ErrSelfReferral is returned when a member presents their own referral code.
var ErrSelfReferral = errors.New("self-referral not allowed")

// RegisterReferral links refereeMemberID to the referrer identified by referralCode.
// Creates a ReferralMember record for the referee if needed and sets ReferredBy.
//
// Rejects self-referral (referee == referrer) to prevent reward farming.
func (s *ReferralService) RegisterReferral(referralCode string, refereeMemberID uint, project string) error {
	referrer, err := s.memberRepo.GetByCode(referralCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrReferralNotFound
		}
		return err
	}

	// Self-referral guard — a member cannot present their own code.
	if referrer.MemberID == refereeMemberID {
		return ErrSelfReferral
	}

	// Check the referee doesn't already have a referrer.
	existing, err := s.memberRepo.GetByMemberID(refereeMemberID)
	if err == nil && existing.ReferredBy != nil {
		return ErrAlreadyReferred
	}

	if existing == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		// Create a new referral member record for the referee.
		code, err := generateReferralCode()
		if err != nil {
			return fmt.Errorf("generate code for referee: %w", err)
		}
		rm := &models.ReferralMember{
			MemberID:     refereeMemberID,
			ReferralCode: code,
			ReferredBy:   &referrer.MemberID,
			Project:      project,
			Status:       "active",
		}
		return s.memberRepo.Create(rm)
	}

	// Update the existing referee's ReferredBy.
	existing.ReferredBy = &referrer.MemberID
	return s.memberRepo.Update(existing)
}

// RecordPayment is called when a referred merchant completes a payment. It
// looks up the default active campaign for the project, calculates the reward,
// and creates a ReferralReward in pending state.
//
// Returns nil (no reward) if the referee has no referrer or no active campaign exists.
func (s *ReferralService) RecordPayment(refereeMemberID uint, project string, amountUSD decimal.Decimal) error {
	rm, err := s.memberRepo.GetByMemberID(refereeMemberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // no referrer registered
		}
		return err
	}
	if rm.ReferredBy == nil {
		return nil // not a referred member
	}

	campaigns, err := s.campaignRepo.ListByProject(project)
	if err != nil {
		return err
	}

	// Find the first active campaign.
	var campaign *models.ReferralCampaign
	for i := range campaigns {
		if campaigns[i].Status == "active" {
			campaign = &campaigns[i]
			break
		}
	}
	if campaign == nil {
		return nil // no active campaign — no reward
	}

	// Calculate reward amount.
	var rewardAmount decimal.Decimal
	if campaign.RewardType != nil && *campaign.RewardType == "percentage" && campaign.RewardValue != nil {
		rewardAmount = amountUSD.Mul(*campaign.RewardValue).Div(decimal.NewFromInt(100))
	} else if campaign.RewardValue != nil {
		rewardAmount = *campaign.RewardValue
	} else {
		return nil // no reward value configured
	}

	if rewardAmount.IsZero() {
		return nil
	}

	// Cap to budget if set.
	if campaign.RewardCap != nil && rewardAmount.GreaterThan(*campaign.RewardCap) {
		rewardAmount = *campaign.RewardCap
	}

	referrerMemberID := *rm.ReferredBy
	reward := &models.ReferralReward{
		CampaignID:   campaign.ID,
		MemberID:     referrerMemberID,
		Project:      project,
		Amount:       rewardAmount,
		CurrencyCode: campaign.CurrencyCode,
		Status:       "pending",
	}
	if err := s.rewardRepo.Create(reward); err != nil {
		return fmt.Errorf("create referral reward: %w", err)
	}
	return nil
}

// ProcessPendingRewards is a deliberate no-op stub deferred to Phase K.
//
// The previous implementation flipped reward status to "paid" via a
// non-conditional UPDATE, never wrote to the ledger, and skipped any
// idempotency or on-chain payout. That meant LedgerService.RecordReferralPayout
// was dead code and the worker loop would silently desync the ledger.
//
// The proper fulfilment flow lands in Phase K alongside SecretsVault
// transaction signing. Until then this method does nothing so the wired-in
// AccountProcessor sub-loop can keep ticking without corrupting state.
//
// TODO(phase-k-payout): atomic conditional UPDATE on status, ledger write
// inside the same transaction, real on-chain payout via SecretsVault + chain
// adapter, idempotency by reward ID.
func (s *ReferralService) ProcessPendingRewards() error {
	return nil
}

// RetryFailedRewards is a deliberate no-op stub deferred to Phase K.
//
// TODO(phase-k-payout): once ProcessPendingRewards lands a real fulfilment
// path, this method walks rewards in 'failed' state and retries them with
// exponential backoff.
func (s *ReferralService) RetryFailedRewards() error {
	return nil
}

// ListEventsByMember returns all rewards for a member (referrer view).
func (s *ReferralService) ListEventsByMember(memberID uint) ([]models.ReferralReward, error) {
	return s.rewardRepo.ListByMember(memberID)
}

// GetMemberReferralCode returns the referral code for a member, creating one if needed.
func (s *ReferralService) GetMemberReferralCode(memberID uint, project string) (string, error) {
	rm, err := s.EnsureReferralMember(memberID, project)
	if err != nil {
		return "", err
	}
	return rm.ReferralCode, nil
}
