package chaindeposit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type openStub struct {
	result *service.CreatePaymentResult
	err    error
}

func (o openStub) CreatePayment(service.CreatePaymentInput, uint, uint) (*service.CreatePaymentResult, error) {
	return o.result, o.err
}

func paymintoFixture(t *testing.T) (*gorm.DB, *chaindeposit.PaymintoBackend) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.PaymentRequest{}, &models.Deposit{}); err != nil {
		t.Fatal(err)
	}
	payment := &models.PaymentRequest{ReferenceID: "ref-1", AmountInUSD: decimal.NewFromInt(100), State: models.PaymentStateOpen, MemberID: 7, ExternalPlatformID: 3}
	if err := db.Create(payment).Error; err != nil {
		t.Fatal(err)
	}
	addr := &models.DepositAddress{Address: "0xabc"}
	stub := openStub{result: &service.CreatePaymentResult{Payment: payment, DepositAddress: addr}}
	return db, chaindeposit.NewPaymintoBackend(stub, repository.NewPaymentRepository(db), repository.NewDepositRepository(db))
}

func TestPaymintoBackend_StatusSumsOnlyConfirmedDeposits(t *testing.T) {
	db, b := paymintoFixture(t)
	ctx := context.Background()
	for _, d := range []models.Deposit{
		{TxID: "a", Amount: decimal.NewFromInt(30), Status: models.DepositStatusConfirmed, ToAddress: "0xabc", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: ptr(1)},
		{TxID: "b", Amount: decimal.NewFromInt(50), Status: models.DepositStatusPending, ToAddress: "0xabc", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: ptr(1)},
		{TxID: "c", Amount: decimal.NewFromInt(20), Status: models.DepositStatusSwept, ToAddress: "0xabc", BlockchainCurrencyID: 1, MemberID: 7, PaymentRequestID: ptr(1)},
	} {
		if err := db.Create(&d).Error; err != nil {
			t.Fatal(err)
		}
	}
	st, err := b.PaymentStatus(ctx, "ref-1")
	if err != nil || st.State != "OPEN" || !st.Received.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("status = %+v, %v", st, err)
	}
	if _, err := b.PaymentStatus(ctx, "nope"); !errors.Is(err, chaindeposit.ErrBackendNotFound) {
		t.Fatalf("missing err = %v", err)
	}
}

func TestPaymintoBackend_CancelOnlyOpen(t *testing.T) {
	db, b := paymintoFixture(t)
	ctx := context.Background()
	if err := b.CancelPayment(ctx, "ref-1"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	var p models.PaymentRequest
	db.Where("reference_id = ?", "ref-1").First(&p)
	if p.State != models.PaymentStateCancelled {
		t.Fatalf("state = %s", p.State)
	}
	if err := b.CancelPayment(ctx, "ref-1"); err == nil {
		t.Fatal("cancelling twice must fail")
	}
	if err := b.CancelPayment(ctx, "nope"); !errors.Is(err, chaindeposit.ErrBackendNotFound) {
		t.Fatalf("missing err = %v", err)
	}
}

func TestPaymintoBackend_OpenRequiresADepositAddress(t *testing.T) {
	_, b := paymintoFixture(t)
	got, err := b.OpenPayment(context.Background(), chaindeposit.OpenRequest{MerchantMemberID: 7, PlatformID: 3, AmountInUSD: decimal.NewFromInt(100), ChainCode: "ETH", CurrencyCode: "USDC"})
	if err != nil || got.Reference != "ref-1" || got.Address != "0xabc" {
		t.Fatalf("open = %+v, %v", got, err)
	}
	noAddr := chaindeposit.NewPaymintoBackend(openStub{result: &service.CreatePaymentResult{Payment: &models.PaymentRequest{ReferenceID: "ref-2"}}}, nil, nil)
	if _, err := noAddr.OpenPayment(context.Background(), chaindeposit.OpenRequest{}); err == nil {
		t.Fatal("a payment without a deposit address cannot be paid")
	}
	c := chaindeposit.New(b)
	resp, err := c.Authorize(context.Background(), connectors.AuthorizeRequest{AttemptID: "pa", MerchantID: "7", PlatformID: "3", Money: connectors.Money{Amount: decimal.NewFromInt(100), Asset: "USD"}, CaptureMethod: connectors.CaptureAutomatic, PaymentMethod: usdc()})
	if err != nil || resp.ConnectorTransactionID != "ref-1" || resp.NextAction.Address != "0xabc" {
		t.Fatalf("authorize through payminto = %+v, %v", resp, err)
	}
}

func ptr(v uint) *uint { return &v }
