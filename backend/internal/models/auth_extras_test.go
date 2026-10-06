package models

import "testing"

func TestAuthRefreshToken_TableName(t *testing.T) {
	if (AuthRefreshToken{}).TableName() != "auth_refresh_tokens" {
		t.Error("wrong table name")
	}
}

func TestOTP_TableName(t *testing.T) {
	if (OTP{}).TableName() != "otps" {
		t.Error("wrong table name")
	}
}

func TestWebSocketToken_TableName(t *testing.T) {
	if (WebSocketToken{}).TableName() != "web_socket_tokens" {
		t.Error("wrong table name")
	}
}

func TestOTPPurposeConstants(t *testing.T) {
	if OTPPurposeWithdrawalApproval != "withdrawal_approval" {
		t.Error("wrong constant")
	}
	if OTPPurposePasswordReset != "password_reset" {
		t.Error("wrong constant")
	}
	if OTPPurposeEmailVerification != "email_verification" {
		t.Error("wrong constant")
	}
}
