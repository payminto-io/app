package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newReferralTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Member{},
		&models.Currency{},
		&models.Account{},
		&models.AccountReward{},
		&models.Asset{},
		&models.Liability{},
		&models.Revenue{},
		&models.Expense{},
		&models.ReferralCampaign{},
		&models.ReferralMember{},
		&models.ReferralReward{},
		&models.ReferralEvent{},
		&models.ReferralEventLog{},
		&models.ReferralCampaignEvent{},
		&models.ReferralCampaignEventLog{},
		&models.ReferralMemberCampaign{},
		&models.ProcessedReward{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newReferralSvc(t *testing.T) (*ReferralService, *ReferralCampaignService, *gorm.DB) {
	t.Helper()
	db := newReferralTestDB(t)

	campaignRepo := repository.NewReferralCampaignRepository(db)
	memberRepo := repository.NewReferralMemberRepository(db)
	rewardRepo := repository.NewReferralRewardRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	ledgerSvc := NewLedgerService(accountRepo)

	campaignSvc := NewReferralCampaignService(campaignRepo)
	referralSvc := NewReferralService(memberRepo, rewardRepo, campaignRepo, ledgerSvc)

	return referralSvc, campaignSvc, db
}

func TestReferralService_EnsureReferralMember_CreatesCode(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	rm, err := svc.EnsureReferralMember(100, "payminto")
	if err != nil {
		t.Fatalf("EnsureReferralMember: %v", err)
	}
	if rm.ReferralCode == "" {
		t.Error("expected non-empty referral code")
	}
	if rm.MemberID != 100 {
		t.Errorf("member ID: got %d want 100", rm.MemberID)
	}
}

func TestReferralService_EnsureReferralMember_Idempotent(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	rm1, _ := svc.EnsureReferralMember(200, "payminto")
	rm2, err := svc.EnsureReferralMember(200, "payminto")
	if err != nil {
		t.Fatalf("second EnsureReferralMember: %v", err)
	}
	if rm1.ReferralCode != rm2.ReferralCode {
		t.Error("idempotent call should return same code")
	}
}

func TestReferralService_RegisterReferral(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	// Referrer gets a code.
	referrer, _ := svc.EnsureReferralMember(10, "payminto")

	// Referee registers under that code.
	err := svc.RegisterReferral(referrer.ReferralCode, 20, "payminto")
	if err != nil {
		t.Fatalf("RegisterReferral: %v", err)
	}

	// Referee's record should have ReferredBy set.
	referee, err := svc.EnsureReferralMember(20, "payminto")
	if err != nil {
		t.Fatalf("get referee: %v", err)
	}
	if referee.ReferredBy == nil {
		t.Error("expected ReferredBy to be set")
	}
	if *referee.ReferredBy != referrer.MemberID {
		t.Errorf("ReferredBy: got %d want %d", *referee.ReferredBy, referrer.MemberID)
	}
}

func TestReferralService_RegisterReferral_InvalidCode(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	err := svc.RegisterReferral("BADCODE", 30, "payminto")
	if !errors.Is(err, ErrReferralNotFound) {
		t.Errorf("expected ErrReferralNotFound, got %v", err)
	}
}

func TestReferralService_RegisterReferral_AlreadyReferred(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	referrer1, _ := svc.EnsureReferralMember(10, "payminto")
	referrer2, _ := svc.EnsureReferralMember(11, "payminto")

	svc.RegisterReferral(referrer1.ReferralCode, 20, "payminto")

	// Second registration attempt should fail.
	err := svc.RegisterReferral(referrer2.ReferralCode, 20, "payminto")
	if !errors.Is(err, ErrAlreadyReferred) {
		t.Errorf("expected ErrAlreadyReferred, got %v", err)
	}
}

func TestReferralService_RecordPayment_CreatesReward(t *testing.T) {
	svc, campaignSvc, db := newReferralSvc(t)

	// Create an active campaign.
	rewardVal := decimal.NewFromFloat(5.0)
	campaignSvc.CreateCampaign(CreateCampaignInput{
		Project:      "payminto",
		Name:         "Test Campaign",
		RewardType:   "fixed",
		RewardValue:  rewardVal,
		CurrencyCode: "USDT",
	})

	// Set up referral relationship.
	referrer, _ := svc.EnsureReferralMember(50, "payminto")
	svc.RegisterReferral(referrer.ReferralCode, 60, "payminto")

	// Record a payment from the referee.
	if err := svc.RecordPayment(60, "payminto", decimal.NewFromFloat(100)); err != nil {
		t.Fatalf("RecordPayment: %v", err)
	}

	// Verify a reward was created for the referrer.
	var rewards []models.ReferralReward
	if err := db.Where("member_id = ?", referrer.MemberID).Find(&rewards).Error; err != nil {
		t.Fatalf("find rewards: %v", err)
	}
	if len(rewards) == 0 {
		t.Error("expected at least one reward for referrer")
	}
	if !rewards[0].Amount.Equal(rewardVal) {
		t.Errorf("reward amount: got %s want %s", rewards[0].Amount, rewardVal)
	}
}

func TestReferralService_RecordPayment_NoCampaign(t *testing.T) {
	svc, _, db := newReferralSvc(t)

	referrer, _ := svc.EnsureReferralMember(70, "payminto")
	svc.RegisterReferral(referrer.ReferralCode, 80, "payminto")

	// No campaign exists — should not error, and no reward is created.
	if err := svc.RecordPayment(80, "payminto", decimal.NewFromFloat(50)); err != nil {
		t.Fatalf("RecordPayment without campaign: %v", err)
	}

	var count int64
	db.Model(&models.ReferralReward{}).Where("member_id = ?", referrer.MemberID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rewards without campaign, got %d", count)
	}
}

func TestReferralService_RecordPayment_NoReferrer(t *testing.T) {
	svc, _, db := newReferralSvc(t)

	// Member 90 has no referrer — recording a payment should silently succeed with no reward.
	svc.EnsureReferralMember(90, "payminto")

	if err := svc.RecordPayment(90, "payminto", decimal.NewFromFloat(100)); err != nil {
		t.Fatalf("RecordPayment for unreferred member: %v", err)
	}

	var count int64
	db.Model(&models.ReferralReward{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 rewards for unreferred member, got %d", count)
	}
}

func TestReferralService_ProcessPendingRewards(t *testing.T) {
	svc, campaignSvc, _ := newReferralSvc(t)

	rewardVal := decimal.NewFromFloat(2.5)
	campaignSvc.CreateCampaign(CreateCampaignInput{
		Project:      "payminto",
		Name:         "Campaign",
		RewardType:   "fixed",
		RewardValue:  rewardVal,
		CurrencyCode: "USDT",
	})

	referrer, _ := svc.EnsureReferralMember(91, "payminto")
	svc.RegisterReferral(referrer.ReferralCode, 92, "payminto")
	svc.RecordPayment(92, "payminto", decimal.NewFromFloat(100))

	// Process pending — should mark reward as paid (Phase I stub).
	if err := svc.ProcessPendingRewards(); err != nil {
		t.Fatalf("ProcessPendingRewards: %v", err)
	}
}

func TestReferralService_GetMemberReferralCode(t *testing.T) {
	svc, _, _ := newReferralSvc(t)

	code, err := svc.GetMemberReferralCode(111, "payminto")
	if err != nil {
		t.Fatalf("GetMemberReferralCode: %v", err)
	}
	if code == "" {
		t.Error("expected non-empty code")
	}

	// Second call returns same code.
	code2, _ := svc.GetMemberReferralCode(111, "payminto")
	if code != code2 {
		t.Errorf("expected same code on repeated call: %s vs %s", code, code2)
	}
}
