package service

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestLedgerService_RecordReferralPayout(t *testing.T) {
	svc, db := newLedgerService(t)

	amount := decimal.NewFromFloat(5.0)
	if err := svc.RecordReferralPayout(42, 1, amount); err != nil {
		t.Fatalf("RecordReferralPayout: %v", err)
	}

	assertBalance(t, db)

	// Liability debit should equal the payout amount.
	type sumRow struct{ Total decimal.Decimal }
	var liabSum sumRow
	db.Raw(`SELECT SUM(debit) AS total FROM liabilities WHERE reference = 'referral_reward'`).Scan(&liabSum)
	if !liabSum.Total.Equal(amount) {
		t.Errorf("liability debit: got %s want %s", liabSum.Total, amount)
	}

	// Asset credit should equal the payout amount.
	var assetSum sumRow
	db.Raw(`SELECT SUM(credit) AS total FROM assets WHERE reference = 'referral_reward'`).Scan(&assetSum)
	if !assetSum.Total.Equal(amount) {
		t.Errorf("asset credit: got %s want %s", assetSum.Total, amount)
	}
}
