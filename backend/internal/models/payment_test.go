package models

import "testing"

func TestPaymentRequest_TableName(t *testing.T) {
	p := PaymentRequest{}
	if p.TableName() != "payment_requests" {
		t.Errorf("expected 'payment_requests', got '%s'", p.TableName())
	}
}

func TestDeposit_TableName(t *testing.T) {
	d := Deposit{}
	if d.TableName() != "deposits" {
		t.Errorf("expected 'deposits', got '%s'", d.TableName())
	}
}

func TestWallet_TableName(t *testing.T) {
	w := Wallet{}
	if w.TableName() != "wallets" {
		t.Errorf("expected 'wallets', got '%s'", w.TableName())
	}
}

func TestSweep_TableName(t *testing.T) {
	s := Sweep{}
	if s.TableName() != "sweeps" {
		t.Errorf("expected 'sweeps', got '%s'", s.TableName())
	}
}

func TestWebhook_TableName(t *testing.T) {
	w := Webhook{}
	if w.TableName() != "webhooks" {
		t.Errorf("expected 'webhooks', got '%s'", w.TableName())
	}
}

func TestAccount_TableName(t *testing.T) {
	a := Account{}
	if a.TableName() != "accounts" {
		t.Errorf("expected 'accounts', got '%s'", a.TableName())
	}
}
