package models

import "testing"

func TestReferralCampaign_TableName(t *testing.T) {
	if (ReferralCampaign{}).TableName() != "referral_campaigns" {
		t.Error("wrong table name")
	}
}

func TestReferralEvent_TableName(t *testing.T) {
	if (ReferralEvent{}).TableName() != "referral_events" {
		t.Error("wrong table name")
	}
}

func TestReferralMember_TableName(t *testing.T) {
	if (ReferralMember{}).TableName() != "referral_members" {
		t.Error("wrong table name")
	}
}

func TestReferralReward_TableName(t *testing.T) {
	if (ReferralReward{}).TableName() != "referral_rewards" {
		t.Error("wrong table name")
	}
}

func TestReferralEventLog_TableName(t *testing.T) {
	if (ReferralEventLog{}).TableName() != "referral_event_logs" {
		t.Error("wrong table name")
	}
}

func TestReferralCampaignEvent_TableName(t *testing.T) {
	if (ReferralCampaignEvent{}).TableName() != "referral_campaign_events" {
		t.Error("wrong table name")
	}
}

func TestReferralCampaignEventLog_TableName(t *testing.T) {
	if (ReferralCampaignEventLog{}).TableName() != "referral_campaign_event_logs" {
		t.Error("wrong table name")
	}
}

func TestReferralMemberCampaign_TableName(t *testing.T) {
	if (ReferralMemberCampaign{}).TableName() != "referral_member_campaigns" {
		t.Error("wrong table name")
	}
}

func TestProcessedReward_TableName(t *testing.T) {
	if (ProcessedReward{}).TableName() != "processed_rewards" {
		t.Error("wrong table name")
	}
}
