package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ReferralCampaign defines a referral program with reward rules. Mirrors
// PayRam's Campaign struct from PAYRAM_GO_MODELS.txt.
type ReferralCampaign struct {
	BaseModel
	Project                   string           `gorm:"type:varchar(100);not null;index" json:"project"`
	Name                      string           `gorm:"type:varchar(200);not null" json:"name"`
	RewardType                *string          `gorm:"type:varchar(50)" json:"rewardType,omitempty"`
	RewardValue               *decimal.Decimal `gorm:"type:numeric(38,18)" json:"rewardValue,omitempty"`
	CurrencyCode              string           `gorm:"type:varchar(20);not null" json:"currencyCode"`
	RewardCap                 *decimal.Decimal `gorm:"type:numeric(38,18)" json:"rewardCap,omitempty"`
	InviteeRewardType         *string          `gorm:"type:varchar(50)" json:"inviteeRewardType,omitempty"`
	InviteeRewardValue        *decimal.Decimal `gorm:"type:numeric(38,18)" json:"inviteeRewardValue,omitempty"`
	InviteeRewardCap          *decimal.Decimal `gorm:"type:numeric(38,18)" json:"inviteeRewardCap,omitempty"`
	Budget                    *decimal.Decimal `gorm:"type:numeric(38,18)" json:"budget,omitempty"`
	Description               *string          `gorm:"type:text" json:"description,omitempty"`
	StartDate                 *time.Time       `json:"startDate,omitempty"`
	EndDate                   *time.Time       `json:"endDate,omitempty"`
	Status                    string           `gorm:"type:varchar(20);default:'active';not null" json:"status"`
	IsDefault                 bool             `gorm:"default:false" json:"isDefault"`
	CampaignTypePerCustomer   string           `gorm:"type:varchar(50)" json:"campaignTypePerCustomer"`
	MaxOccurrencesPerCustomer *int64           `json:"maxOccurrencesPerCustomer,omitempty"`
	ValidityMonthsPerCustomer *int             `json:"validityMonthsPerCustomer,omitempty"`
	RewardCapPerCustomer      *decimal.Decimal `gorm:"type:numeric(38,18)" json:"rewardCapPerCustomer,omitempty"`
	ConsiderEventsFrom        time.Time        `gorm:"not null" json:"considerEventsFrom"`
}

func (ReferralCampaign) TableName() string { return "referral_campaigns" }

// ReferralEvent is a registered event type that can trigger reward
// calculation (e.g., "first_payment", "kyc_completed", "deposit_above_100").
type ReferralEvent struct {
	BaseModel
	Project     string  `gorm:"type:varchar(100);not null;index" json:"project"`
	Key         string  `gorm:"type:varchar(100);not null" json:"key"`
	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	EventType   string  `gorm:"type:varchar(50);not null" json:"eventType"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
}

func (ReferralEvent) TableName() string { return "referral_events" }

// ReferralMember is the junction between a Member and the referral system,
// holding their referral code and link.
type ReferralMember struct {
	BaseModel
	MemberID     uint    `gorm:"not null;uniqueIndex" json:"memberID"`
	ReferralCode string  `gorm:"type:varchar(50);not null;uniqueIndex" json:"referralCode"`
	ReferredBy   *uint   `json:"referredBy,omitempty"`
	Project      string  `gorm:"type:varchar(100);not null;index" json:"project"`
	Status       string  `gorm:"type:varchar(20);default:'active';not null" json:"status"`
	Notes        *string `gorm:"type:text" json:"notes,omitempty"`

	Member *Member `gorm:"foreignKey:MemberID" json:"-"`
}

func (ReferralMember) TableName() string { return "referral_members" }

// ReferralReward is a single reward earned by a member for triggering an
// event under a campaign.
type ReferralReward struct {
	BaseModel
	CampaignID    uint            `gorm:"not null;index" json:"campaignID"`
	MemberID      uint            `gorm:"not null;index" json:"memberID"`
	EventID       *uint           `json:"eventID,omitempty"`
	Project       string          `gorm:"type:varchar(100);not null;index" json:"project"`
	Amount        decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	CurrencyCode  string          `gorm:"type:varchar(20);not null" json:"currencyCode"`
	Status        string          `gorm:"type:varchar(20);default:'pending';not null" json:"status"`
	ProcessedAt   *time.Time      `json:"processedAt,omitempty"`
	FailureReason *string         `gorm:"type:text" json:"failureReason,omitempty"`

	Campaign *ReferralCampaign `gorm:"foreignKey:CampaignID" json:"-"`
	Member   *Member           `gorm:"foreignKey:MemberID" json:"-"`
	Event    *ReferralEvent    `gorm:"foreignKey:EventID" json:"-"`
}

func (ReferralReward) TableName() string { return "referral_rewards" }

// ReferralEventLog records every event firing — even if no reward was issued
// — so analytics can show conversion funnels.
type ReferralEventLog struct {
	BaseModel
	EventID   uint    `gorm:"not null;index" json:"eventID"`
	MemberID  uint    `gorm:"not null;index" json:"memberID"`
	Project   string  `gorm:"type:varchar(100);not null;index" json:"project"`
	Payload   *string `gorm:"type:text" json:"payload,omitempty"`
	IPAddress *string `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`

	Event  *ReferralEvent `gorm:"foreignKey:EventID" json:"-"`
	Member *Member        `gorm:"foreignKey:MemberID" json:"-"`
}

func (ReferralEventLog) TableName() string { return "referral_event_logs" }

// ReferralCampaignEvent associates an event type with a campaign.
type ReferralCampaignEvent struct {
	BaseModel
	CampaignID uint `gorm:"not null;index" json:"campaignID"`
	EventID    uint `gorm:"not null;index" json:"eventID"`

	Campaign *ReferralCampaign `gorm:"foreignKey:CampaignID" json:"-"`
	Event    *ReferralEvent    `gorm:"foreignKey:EventID" json:"-"`
}

func (ReferralCampaignEvent) TableName() string { return "referral_campaign_events" }

// ReferralCampaignEventLog logs every event firing within a campaign context.
type ReferralCampaignEventLog struct {
	BaseModel
	CampaignID uint    `gorm:"not null;index" json:"campaignID"`
	EventID    uint    `gorm:"not null;index" json:"eventID"`
	MemberID   uint    `gorm:"not null;index" json:"memberID"`
	RewardID   *uint   `json:"rewardID,omitempty"`
	Successful bool    `gorm:"default:true" json:"successful"`
	Notes      *string `gorm:"type:text" json:"notes,omitempty"`

	Campaign *ReferralCampaign `gorm:"foreignKey:CampaignID" json:"-"`
	Event    *ReferralEvent    `gorm:"foreignKey:EventID" json:"-"`
	Member   *Member           `gorm:"foreignKey:MemberID" json:"-"`
	Reward   *ReferralReward   `gorm:"foreignKey:RewardID" json:"-"`
}

func (ReferralCampaignEventLog) TableName() string { return "referral_campaign_event_logs" }

// ReferralMemberCampaign is the per-member opt-in/eligibility record for a
// campaign — used to enforce per-customer caps and validity windows.
type ReferralMemberCampaign struct {
	BaseModel
	MemberID    uint            `gorm:"not null;index" json:"memberID"`
	CampaignID  uint            `gorm:"not null;index" json:"campaignID"`
	JoinedAt    time.Time       `gorm:"not null" json:"joinedAt"`
	ExpiresAt   *time.Time      `json:"expiresAt,omitempty"`
	Occurrences int64           `gorm:"default:0" json:"occurrences"`
	TotalReward decimal.Decimal `gorm:"type:numeric(38,18);default:0" json:"totalReward"`
	Status      string          `gorm:"type:varchar(20);default:'active';not null" json:"status"`

	Member   *Member           `gorm:"foreignKey:MemberID" json:"-"`
	Campaign *ReferralCampaign `gorm:"foreignKey:CampaignID" json:"-"`
}

func (ReferralMemberCampaign) TableName() string { return "referral_member_campaigns" }

// ProcessedReward records that a referral reward has been settled into a
// member's account_reward balance. Prevents double-payment by uniqueness on
// (reward_id).
type ProcessedReward struct {
	BaseModel
	RewardID     uint            `gorm:"not null;uniqueIndex" json:"rewardID"`
	MemberID     uint            `gorm:"not null;index" json:"memberID"`
	Amount       decimal.Decimal `gorm:"type:numeric(38,18);not null" json:"amount"`
	CurrencyCode string          `gorm:"type:varchar(20);not null" json:"currencyCode"`
	ProcessedAt  time.Time       `gorm:"not null" json:"processedAt"`

	Reward *ReferralReward `gorm:"foreignKey:RewardID" json:"-"`
	Member *Member         `gorm:"foreignKey:MemberID" json:"-"`
}

func (ProcessedReward) TableName() string { return "processed_rewards" }
