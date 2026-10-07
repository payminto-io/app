package repository

import (
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
)

func TestGetRevenueBreakdown_CountsOnlyPaidDeposits(t *testing.T) {
	db := openTestDB(t)
	platformID, memberID := seedPlatformAndMember(t, db)

	p := models.PaymentRequest{ReferenceID: "rev-1", AmountInUSD: decimal.RequireFromString("15"), State: models.PaymentStateFilled, MemberID: memberID, ExternalPlatformID: platformID}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	for i, d := range []struct{ amount, status string }{
		{"10", models.DepositStatusConfirmed},
		{"5", models.DepositStatusSwept},
		{"100", models.DepositStatusFailed},
		{"7", models.DepositStatusPending},
		{"3", models.DepositStatusConfirming},
	} {
		dep := models.Deposit{TxID: "tx-" + string(rune('a'+i)), Amount: decimal.RequireFromString(d.amount), Status: d.status, ToAddress: "addr", BlockchainCurrencyID: 1, PaymentRequestID: &p.ID, MemberID: memberID}
		if err := db.Create(&dep).Error; err != nil {
			t.Fatal(err)
		}
	}

	rows, err := NewAnalyticsRepository(db).GetRevenueBreakdown(platformID, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1: %+v", len(rows), rows)
	}
	if !rows[0].TotalAmount.Equal(decimal.RequireFromString("15")) {
		t.Fatalf("total = %s, want 15 (confirmed + swept only)", rows[0].TotalAmount)
	}
}

func TestGetDashboardSummary_HasNoInstanceWideSweepCount(t *testing.T) {
	db := openTestDB(t)
	if err := db.AutoMigrate(&models.Sweep{}, &models.Withdrawal{}, &models.Webhook{}); err != nil {
		t.Fatal(err)
	}
	platformID, _ := seedPlatformAndMember(t, db)
	if _, err := NewAnalyticsRepository(db).GetDashboardSummary(platformID); err != nil {
		t.Fatal(err)
	}
}
