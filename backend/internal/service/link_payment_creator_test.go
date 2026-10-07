package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func linkCreatorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Blockchain{}, &models.BlockchainCurrency{}, &models.PaymentRequest{}, &models.Member{}, &models.DepositAddress{}); err != nil {
		t.Fatal(err)
	}
	for _, col := range []string{"fee_rule_id integer", "fee_rule_version integer"} {
		if err := db.Exec(`ALTER TABLE payment_requests ADD COLUMN ` + col).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func seedLinkChain(t *testing.T, db *gorm.DB, code, asset, status string, deposits bool) {
	t.Helper()
	chain := models.Blockchain{Code: code, Name: code, BlockchainFamilyID: 1, Status: status}
	if err := db.Create(&chain).Error; err != nil {
		t.Fatal(err)
	}
	bc := models.BlockchainCurrency{CurrencyCode: asset, BlockchainCode: code, BlockchainID: chain.ID, CurrencyID: 1}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&bc).Update("deposit_enabled", deposits).Error; err != nil {
		t.Fatal(err)
	}
}

func TestLinkPaymentCreatorOffersOnlyActiveCryptoInUSD(t *testing.T) {
	db := linkCreatorDB(t)
	seedLinkChain(t, db, "SOL", "USDC", "active", true)
	seedLinkChain(t, db, "TRX", "USDT", "inactive", true)
	seedLinkChain(t, db, "ETH", "USDC", "active", false)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "https://checkout.test/")
	ctx := context.Background()
	crypto := func(chain, asset string) links.MethodSpec {
		return links.MethodSpec{Method: fees.MethodCrypto, Chain: chain, Asset: asset}
	}
	cases := []struct {
		currency string
		m        links.MethodSpec
		offered  bool
	}{
		{"USD", crypto("SOL", "USDC"), true},
		{"EUR", crypto("SOL", "USDC"), false},
		{"USD", crypto("TRX", "USDT"), false},
		{"USD", crypto("ETH", "USDC"), false},
		{"USD", crypto("SOL", "USDT"), false},
		{"USD", links.MethodSpec{Method: fees.MethodCard}, false},
	}
	for _, tc := range cases {
		got, err := c.Connectors(ctx, links.EnvTest, tc.currency, tc.m)
		if err != nil || (len(got) > 0) != tc.offered {
			t.Errorf("%s %s: %v %v, want offered=%v", tc.currency, tc.m, got, err, tc.offered)
		}
	}
}

func linkPayReq(id string) links.PaymentRequest {
	return links.PaymentRequest{
		LinkPaymentID: id, MemberID: 4, PlatformID: 9, Method: links.MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"},
		Amount: decimal.RequireFromString("25"), CustomerTotal: decimal.RequireFromString("25.75"), Currency: "USD",
		CustomerEmail: "ada@example.test", ReferenceID: "order-42", QuoteExpirySeconds: 600, FeeRuleID: 3, FeeRuleVersion: 2,
	}
}

func TestLinkPaymentCreatorCreatesAPaymentRequestForTheCustomerTotal(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "https://checkout.test")
	before := time.Now()
	created, err := c.CreatePayment(context.Background(), linkPayReq("lp-1"))
	if err != nil {
		t.Fatal(err)
	}
	var pr struct {
		models.PaymentRequest
		FeeRuleID      *uint
		FeeRuleVersion *int
	}
	if err := db.Table("payment_requests").Where("reference_id = ?", "pl_lp-1").First(&pr).Error; err != nil {
		t.Fatal(err)
	}
	if created.Reference != "pl_lp-1" || !pr.AmountInUSD.Equal(decimal.RequireFromString("25.75")) || pr.MemberID != 4 || pr.ExternalPlatformID != 9 ||
		*pr.CustomerEmail != "ada@example.test" || *pr.InvoiceID != "order-42" || *pr.FeeRuleID != 3 || *pr.FeeRuleVersion != 2 {
		t.Fatalf("payment request %+v fee %v/%v", pr.PaymentRequest, pr.FeeRuleID, pr.FeeRuleVersion)
	}
	if exp := pr.ExpiresAt.Sub(before); exp < 590*time.Second || exp > 610*time.Second {
		t.Fatalf("expiry %v, want the link's 600s quote expiry", exp)
	}
	if created.CheckoutURL != "https://checkout.test/pay/pl_lp-1" || created.ExpiresAt == nil {
		t.Fatalf("created %+v", created)
	}
}

func TestLinkPaymentCreatorIsIdempotentAndFindsByLinkPaymentID(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "")
	ctx := context.Background()
	first, err := c.CreatePayment(ctx, linkPayReq("lp-2"))
	if err != nil {
		t.Fatal(err)
	}
	again, err := c.CreatePayment(ctx, linkPayReq("lp-2"))
	if err != nil || again.Reference != first.Reference {
		t.Fatalf("second call %+v %v", again, err)
	}
	var n int64
	db.Model(&models.PaymentRequest{}).Count(&n)
	if n != 1 {
		t.Fatalf("%d payments for one link payment id", n)
	}
	got, found, err := c.FencePayment(ctx, linkPayReq("lp-2"))
	if err != nil || !found || got.Reference != first.Reference {
		t.Fatalf("fence of a live payment %+v %v %v", got, found, err)
	}
	if open, err := c.OpenPayments(ctx, []string{"lp-2", "lp-none"}); err != nil || !open["lp-2"] || open["lp-none"] {
		t.Fatalf("open %v %v", open, err)
	}
	if err := c.CancelPayment(ctx, "lp-2"); err != nil {
		t.Fatal(err)
	}
	if open, _ := c.OpenPayments(ctx, []string{"lp-2"}); open["lp-2"] {
		t.Fatal("a cancelled payment is open")
	}
}

func TestLinkPaymentCreatorFenceMakesALateCreationFail(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "")
	ctx := context.Background()
	if _, found, err := c.FencePayment(ctx, linkPayReq("lp-6")); found || err != nil {
		t.Fatalf("fence before creation: %v %v", found, err)
	}
	if _, found, err := c.FencePayment(ctx, linkPayReq("lp-6")); found || err != nil {
		t.Fatalf("second fence: %v %v", found, err)
	}
	if _, err := c.CreatePayment(ctx, linkPayReq("lp-6")); !errors.Is(err, links.ErrNotCreated) {
		t.Fatalf("creation after the fence: %v", err)
	}
	// The stalled path: the payment service itself inserting after the fence hits the unique index.
	if _, err := c.payments.CreatePayment(CreatePaymentInput{AmountInUSD: decimal.NewFromInt(1), ReferenceID: LinkPaymentReference("lp-6")}, 4, 9); err == nil {
		t.Fatal("a second payment took a fenced reference")
	}
	var live int64
	db.Model(&models.PaymentRequest{}).Where("reference_id = ? AND state <> ?", "pl_lp-6", models.PaymentStateCancelled).Count(&live)
	if live != 0 {
		t.Fatalf("%d live payments behind a fence", live)
	}
}

// failingAfterInsert writes the payment and then fails, as a payment service whose rollback also failed would.
type failingAfterInsert struct {
	repository.PaymentRepository
	db    *gorm.DB
	state string
}

func (r failingAfterInsert) Create(p *models.PaymentRequest) error {
	if err := r.PaymentRepository.Create(p); err != nil {
		return err
	}
	r.db.Model(p).Update("state", r.state)
	return errors.New("address pool exhausted")
}

func TestLinkPaymentCreatorCancelsAHalfCreatedPayment(t *testing.T) {
	for _, tc := range []struct {
		state      string
		notCreated bool
	}{{models.PaymentStateOpen, true}, {models.PaymentStatePartiallyFilled, false}} {
		db := linkCreatorDB(t)
		svc := NewPaymentService(failingAfterInsert{repository.NewPaymentRepository(db), db, tc.state})
		c := NewLinkPaymentCreator(svc, db, "")
		_, err := c.CreatePayment(context.Background(), linkPayReq("lp-7"))
		if errors.Is(err, links.ErrNotCreated) != tc.notCreated || err == nil {
			t.Fatalf("%s: %v", tc.state, err)
		}
		if _, found, _ := c.FencePayment(context.Background(), linkPayReq("lp-7")); found == tc.notCreated {
			t.Fatalf("%s: half-created payment live=%v", tc.state, found)
		}
	}
}

func TestLinkPaymentCreatorReportsNotCreatedOnlyWhenNothingExists(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "")
	bad := linkPayReq("lp-3")
	bad.CustomerTotal = decimal.Zero
	if _, err := c.CreatePayment(context.Background(), bad); !errors.Is(err, links.ErrNotCreated) {
		t.Fatalf("refused amount: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.CreatePayment(ctx, linkPayReq("lp-4")); !errors.Is(err, links.ErrNotCreated) {
		t.Fatalf("cancelled context: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()
	if _, err := c.CreatePayment(context.Background(), linkPayReq("lp-5")); err == nil || errors.Is(err, links.ErrNotCreated) {
		t.Fatalf("a database that cannot answer must be ambiguous, got %v", err)
	}
}

func TestLinkPaymentCreatorRefusesWhatPaymintoCannotTake(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "")
	for _, req := range []links.PaymentRequest{
		{Method: links.MethodSpec{Method: fees.MethodCard}, Currency: "USD", CustomerTotal: decimal.NewFromInt(1)},
		{Method: links.MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"}, Currency: "EUR", CustomerTotal: decimal.NewFromInt(1)},
	} {
		if _, err := c.CreatePayment(context.Background(), req); links.CodeOf(err) != links.CodeMethodUnavailable {
			t.Errorf("%+v: %v", req.Method, err)
		}
	}
	var n int64
	db.Model(&models.PaymentRequest{}).Count(&n)
	if n != 0 {
		t.Fatalf("%d payments created for refused requests", n)
	}
}
