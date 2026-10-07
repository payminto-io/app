package service

import (
	"context"
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

func newJournalingLedgerService(t *testing.T) (*LedgerService, *ledger.Service) {
	t.Helper()
	db := newLedgerSvcTestDB(t)
	if err := ledger.Migrate(db); err != nil {
		t.Fatalf("ledger.Migrate: %v", err)
	}
	journal := ledger.New(db)
	resolver := func(currencyID uint) (string, error) {
		switch currencyID {
		case 1:
			return "USDC", nil
		case 2:
			return "ETH", nil
		}
		return "", errors.New("unknown currency")
	}
	return NewLedgerService(repository.NewAccountRepository(db), WithJournal(journal, resolver)), journal
}

func TestLedgerService_DualWritesEveryEventAsABalancedJournal(t *testing.T) {
	svc, journal := newJournalingLedgerService(t)
	ctx := context.Background()
	amount, gas := decimal.NewFromInt(100), decimal.NewFromInt(3)

	steps := []struct {
		name string
		call func() error
		key  string
		kind ledger.JournalKind
	}{
		{"deposit", func() error { return svc.RecordPaymentDeposit(1, 1, amount) }, "payminto:payment:1", ledger.KindPayment},
		{"sweep", func() error { return svc.RecordSweep(2, 1, amount, gas) }, "payminto:sweep:2", ledger.KindTransfer},
		{"withdrawal", func() error { return svc.RecordWithdrawal(3, 1, amount, gas) }, "payminto:withdrawal:3", ledger.KindSettlement},
		{"gas fee", func() error { return svc.RecordGasFee(4, 2, gas) }, "payminto:gas_fee:4", ledger.KindFee},
		{"duplicate deposit", func() error { return svc.RecordDuplicateDeposit(5, 1, amount) }, "payminto:duplicate_deposit:5", ledger.KindAdjustment},
		{"referral payout", func() error { return svc.RecordReferralPayout(6, 1, amount) }, "payminto:referral_reward:6", ledger.KindSettlement},
		{"address deployment", func() error { return svc.RecordAddressDeployment(7, 2, gas) }, "payminto:address_deployment:7", ledger.KindFee},
	}
	for _, step := range steps {
		if err := step.call(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		var row ledger.JournalRow
		if err := journalDB(journal).Where("idempotency_key = ?", step.key).First(&row).Error; err != nil {
			t.Fatalf("%s: journal %q not written: %v", step.name, step.key, err)
		}
		if row.Kind != step.kind {
			t.Errorf("%s: kind = %s, want %s", step.name, row.Kind, step.kind)
		}
	}

	var totals []struct {
		Asset string
		Total decimal.Decimal
	}
	if err := journalDB(journal).Model(&ledger.LineRow{}).Select("asset, COALESCE(SUM(amount), 0) AS total").Group("asset").Scan(&totals).Error; err != nil {
		t.Fatal(err)
	}
	if len(totals) != 2 {
		t.Fatalf("assets posted = %v, want USDC and ETH", totals)
	}
	for _, r := range totals {
		if !r.Total.IsZero() {
			t.Errorf("asset %s sums to %s", r.Asset, r.Total)
		}
	}

	hot, err := journal.AccountID(ctx, ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "crypto_assets", Asset: "USDC", Kind: ledger.KindAsset})
	if err != nil {
		t.Fatal(err)
	}
	bal, err := journal.Balance(ctx, hot)
	if err != nil {
		t.Fatal(err)
	}
	// +100 deposit -103 sweep -103 withdrawal +100 duplicate -100 referral = -106
	if want := decimal.NewFromInt(-106); !bal.Equal(want) {
		t.Fatalf("crypto_assets USDC balance = %s, want %s", bal, want)
	}
}

func TestLedgerService_ReplayWritesNeitherJournalNorLegacyRows(t *testing.T) {
	svc, journal := newJournalingLedgerService(t)
	for range 3 {
		if err := svc.RecordPaymentDeposit(9, 1, decimal.NewFromInt(5)); err != nil {
			t.Fatal(err)
		}
	}
	var journals, lines, assets, liabilities int64
	db := journalDB(journal)
	db.Model(&ledger.JournalRow{}).Count(&journals)
	db.Model(&ledger.LineRow{}).Count(&lines)
	db.Model(&models.Asset{}).Count(&assets)
	db.Model(&models.Liability{}).Count(&liabilities)
	if journals != 1 || lines != 2 || assets != 1 || liabilities != 1 {
		t.Fatalf("journals=%d lines=%d assets=%d liabilities=%d, want 1/2/1/1", journals, lines, assets, liabilities)
	}
}

func TestLedgerService_UnresolvableAssetWritesNothing(t *testing.T) {
	svc, journal := newJournalingLedgerService(t)
	err := svc.RecordPaymentDeposit(10, 99, decimal.NewFromInt(5))
	if err == nil {
		t.Fatal("expected an error for an unknown currency")
	}
	var journals, assets int64
	journalDB(journal).Model(&ledger.JournalRow{}).Count(&journals)
	journalDB(journal).Model(&models.Asset{}).Count(&assets)
	if journals != 0 || assets != 0 {
		t.Fatalf("journals=%d assets=%d after failed post, want 0/0", journals, assets)
	}
}

func TestLedgerService_ReconcileReportsDriftWithoutFixing(t *testing.T) {
	svc, journal := newJournalingLedgerService(t)
	ctx := context.Background()
	if _, err := journal.Post(ctx, ledger.Journal{
		Kind:           ledger.KindPayment,
		IdempotencyKey: "member-credit",
		Lines: []ledger.Line{
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerPlatform, OwnerID: "crypto_assets", Asset: "USDC", Kind: ledger.KindAsset}, Amount: decimal.NewFromInt(40)},
			{Account: ledger.AccountKey{OwnerType: ledger.OwnerMember, OwnerID: "7", Asset: "USDC", Kind: ledger.KindLiability}, Amount: decimal.NewFromInt(-40)},
		},
	}); err != nil {
		t.Fatal(err)
	}
	stored := []models.Account{
		{MemberID: 7, CurrencyID: 1, Balance: decimal.NewFromInt(40)},
		{MemberID: 7, CurrencyID: 2, Balance: decimal.NewFromInt(1)},
		{MemberID: 8, CurrencyID: 1, Balance: decimal.NewFromInt(12)},
	}
	drifts, err := svc.ReconcileStoredBalances(ctx, stored)
	if err != nil {
		t.Fatal(err)
	}
	if len(drifts) != 2 {
		t.Fatalf("drifts = %+v, want 2", drifts)
	}
	for _, d := range drifts {
		if d.Account.OwnerType != ledger.OwnerMember || !d.Derived.IsZero() || d.Stored.IsZero() {
			t.Errorf("unexpected drift %+v", d)
		}
	}
	if stored[1].Balance.IsZero() || stored[2].Balance.IsZero() {
		t.Fatal("reconcile must not mutate stored balances")
	}
}
