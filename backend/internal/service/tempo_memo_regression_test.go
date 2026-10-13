package service

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

func TestTempoPollRequiresIssuedPaymentLeg(t *testing.T) {
	for _, change := range []struct {
		name string
		sql  string
	}{
		{"unassigned invoice", "DELETE FROM deposit_addresses"},
		{"different token", "UPDATE deposit_addresses SET blockchain_currency_id = 99"},
		{"different merchant", "UPDATE deposit_addresses SET member_id = 99"},
		{"different recipient", "UPDATE deposit_addresses SET address = '0x0000000000000000000000000000000000000001'"},
		{"disabled token", "UPDATE blockchain_currencies SET deposit_enabled = false"},
		{"disabled chain", "UPDATE blockchains SET status = 'inactive'"},
		{"disabled currency", "UPDATE currencies SET deposit_enabled = false"},
		{"changed token address", "UPDATE blockchain_currencies SET address = '0x0000000000000000000000000000000000000001'"},
		{"changed token decimals", "UPDATE blockchain_currencies SET wallet_precision = 18"},
		{"changed chain id", "UPDATE blockchains SET chain_id = 1"},
	} {
		t.Run(change.name, func(t *testing.T) {
			reader := &fakeTempoReader{head: 120}
			svc, db, p := newTempoFixture(t, reader)
			if err := db.Exec(change.sql).Error; err != nil {
				t.Fatal(err)
			}
			reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xe1", 0, 110)}
			if n, err := svc.PollOnce(context.Background()); err != nil || n != 0 {
				t.Fatalf("unissued/disabled leg credited: n=%d, err=%v", n, err)
			}
			var count int64
			if err := db.Model(&models.Deposit{}).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("deposits=%d, err=%v", count, err)
			}
		})
	}
}

func TestTempoFinalizationFailureRemainsRetryable(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xe2", 0, 110)}
	if err := db.Callback().Update().Before("gorm:update").Register("test:payment-write-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "payment_requests" {
			_ = tx.AddError(errors.New("temporary payment write failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Update().Remove("test:payment-write-failure"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got models.PaymentRequest
	if err := db.First(&got, p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.State != models.PaymentStateFilled {
		t.Fatalf("payment remains %s after successful retry, want FILLED", got.State)
	}
}

func TestTempoCursorWriteFailureReturnsError(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, _ := newTempoFixture(t, reader)
	if err := db.Callback().Update().Before("gorm:update").Register("test:cursor-write-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "blockchains" {
			_ = tx.AddError(errors.New("cursor write failed"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PollOnce(context.Background()); err == nil {
		t.Fatal("poll reported success although cursor write failed")
	}
}

func TestTempoPollRejectsWrongRPCChainBeforeCrediting(t *testing.T) {
	reader := &fakeTempoReader{head: 120, chainID: big.NewInt(1)}
	svc, db, p := newTempoFixture(t, reader)
	reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xe4", 0, 110)}
	if _, err := svc.PollOnce(context.Background()); err == nil {
		t.Fatal("wrong chain accepted")
	}
	var count int64
	if err := db.Model(&models.Deposit{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("deposits=%d, err=%v", count, err)
	}
	if len(reader.ranges) != 0 {
		t.Fatal("scanned wrong chain")
	}
}

func TestTempoAdditionalTransferUpdatesOverpayment(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	reader.transfers = []tempo.MemoTransfer{
		memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xe5", 0, 110),
		memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 1_000_000, "0xe5", 1, 110),
	}
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 2 {
		t.Fatalf("poll=%d, %v", n, err)
	}
	var got models.PaymentRequest
	if err := db.First(&got, p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.State != models.PaymentStateOverFilled {
		t.Fatalf("state=%s, want OVER_FILLED for 3 received against 2 due", got.State)
	}
}

func TestTempoAmbiguousMemoNeverChoosesAnInvoice(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	compact, _ := tempo.MemoForReference(p.ReferenceID)
	other := &models.PaymentRequest{ReferenceID: compact, MemberID: p.MemberID, ExternalPlatformID: p.ExternalPlatformID, AmountInUSD: p.AmountInUSD, State: models.PaymentStateOpen}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	var leg models.DepositAddress
	if err := db.First(&leg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.DepositAddress{Address: leg.Address, MemberID: other.MemberID, BlockchainCurrencyID: leg.BlockchainCurrencyID, PaymentRequestID: &other.ID}).Error; err != nil {
		t.Fatal(err)
	}
	reader.transfers = []tempo.MemoTransfer{memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 2_000_000, "0xe6", 0, 110)}
	if n, err := svc.PollOnce(context.Background()); err != nil || n != 0 {
		t.Fatalf("ambiguous memo credited: %d, %v", n, err)
	}
}

func TestTempoSweptPartialDepositStillCountsTowardInvoice(t *testing.T) {
	reader := &fakeTempoReader{head: 120}
	svc, db, p := newTempoFixture(t, reader)
	first := memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 750_000, "0xe7", 0, 110)
	if won, err := svc.handle(context.Background(), first); err != nil || !won {
		t.Fatalf("first=%t, %v", won, err)
	}
	if err := db.Model(&models.Deposit{}).Where("payment_request_id = ?", p.ID).Update("status", models.DepositStatusSwept).Error; err != nil {
		t.Fatal(err)
	}
	second := memoTransfer(p.ReferenceID, tempo.AlphaUSD, tempoMerchant, 1_250_000, "0xe8", 0, 111)
	if won, err := svc.handle(context.Background(), second); err != nil || !won {
		t.Fatalf("second=%t, %v", won, err)
	}
	var got models.PaymentRequest
	if err := db.First(&got, p.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.State != models.PaymentStateFilled {
		t.Fatalf("swept partial lost from total: %s", got.State)
	}
}
