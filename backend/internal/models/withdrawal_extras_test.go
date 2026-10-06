package models

import "testing"

func TestWithdrawal_TableName(t *testing.T) {
	if (Withdrawal{}).TableName() != "withdrawals" {
		t.Error("wrong table name")
	}
}

func TestWithdraw_TableName(t *testing.T) {
	if (Withdraw{}).TableName() != "withdraws" {
		t.Error("wrong table name")
	}
}

func TestWithdrawalStateConstants(t *testing.T) {
	cases := map[string]string{
		"pending-otp":      WithdrawalStatePendingOTP,
		"pending-approval": WithdrawalStatePendingApproval,
		"pending":          WithdrawalStatePending,
		"initiated":        WithdrawalStateInitiated,
		"sent":             WithdrawalStateSent,
		"processed":        WithdrawalStateProcessed,
		"failed":           WithdrawalStateFailed,
		"cancelled":        WithdrawalStateCancelled,
	}
	for want, got := range cases {
		if want != got {
			t.Errorf("expected %q, got %q", want, got)
		}
	}
}

func TestOnramperPayments_TableName(t *testing.T) {
	if (OnramperPayments{}).TableName() != "onramper_payments" {
		t.Error("wrong table name")
	}
}

func TestOnramperStatusConstants(t *testing.T) {
	if OnramperStatusCompleted != "completed" {
		t.Error("wrong constant")
	}
}

func TestRecipient_TableName(t *testing.T) {
	if (Recipient{}).TableName() != "recipients" {
		t.Error("wrong table name")
	}
}
