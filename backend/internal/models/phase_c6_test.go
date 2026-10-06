package models

import "testing"

func TestPaymentChannel_TableName(t *testing.T) {
	if (PaymentChannel{}).TableName() != "payment_channels" {
		t.Error("wrong table name")
	}
}

func TestDisabledPaymentChannelProject_TableName(t *testing.T) {
	if (DisabledPaymentChannelProject{}).TableName() != "disabled_payment_channel_projects" {
		t.Error("wrong table name")
	}
}

func TestPaymentsApp_TableName(t *testing.T) {
	if (PaymentsApp{}).TableName() != "payments_apps" {
		t.Error("wrong table name")
	}
}

func TestExternalPlatformBlockchainCurrency_TableName(t *testing.T) {
	if (ExternalPlatformBlockchainCurrency{}).TableName() != "external_platform_blockchain_currencies" {
		t.Error("wrong table name")
	}
}

func TestExternalPlatformWalletBlockchainFamily_TableName(t *testing.T) {
	if (ExternalPlatformWalletBlockchainFamily{}).TableName() != "external_platform_wallet_blockchain_families" {
		t.Error("wrong table name")
	}
}

func TestAccountReward_TableName(t *testing.T) {
	if (AccountReward{}).TableName() != "account_rewards" {
		t.Error("wrong table name")
	}
}

func TestSeederLog_TableName(t *testing.T) {
	if (SeederLog{}).TableName() != "seeder_logs" {
		t.Error("wrong table name")
	}
}

func TestGenericDataStore_TableName(t *testing.T) {
	if (GenericDataStore{}).TableName() != "generic_data_stores" {
		t.Error("wrong table name")
	}
}

func TestTag_TableName(t *testing.T) {
	if (Tag{}).TableName() != "tags" {
		t.Error("wrong table name")
	}
}

func TestActivityLog_TableName(t *testing.T) {
	if (ActivityLog{}).TableName() != "activity_logs" {
		t.Error("wrong table name")
	}
}

func TestEEEvent_TableName(t *testing.T) {
	if (EEEvent{}).TableName() != "ee_events" {
		t.Error("wrong table name")
	}
}

func TestEventTypeConstants(t *testing.T) {
	if EventTypeEmailSend != "email.send" {
		t.Error("wrong constant")
	}
	if EventTypePaymentConfirmed != "payment.confirmed" {
		t.Error("wrong constant")
	}
}

func TestEEEventStatusConstants(t *testing.T) {
	if EEEventStatusDeadLetter != "dead_letter" {
		t.Error("wrong constant")
	}
}
