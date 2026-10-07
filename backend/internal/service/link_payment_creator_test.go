package service

import (
	"context"
	"testing"

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
	if err := db.AutoMigrate(&models.Blockchain{}, &models.BlockchainCurrency{}, &models.PaymentRequest{}, &models.Member{}); err != nil {
		t.Fatal(err)
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

func TestLinkPaymentCreatorCreatesAPaymentRequestForTheCustomerTotal(t *testing.T) {
	db := linkCreatorDB(t)
	c := NewLinkPaymentCreator(NewPaymentService(repository.NewPaymentRepository(db)), db, "https://checkout.test")
	created, err := c.CreatePayment(context.Background(), links.PaymentRequest{
		MemberID: 4, PlatformID: 9, Method: links.MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDC"},
		Amount: decimal.RequireFromString("25"), CustomerTotal: decimal.RequireFromString("25.75"), Currency: "USD",
		CustomerEmail: "ada@example.test", ReferenceID: "order-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	var pr models.PaymentRequest
	if err := db.Where("reference_id = ?", created.Reference).First(&pr).Error; err != nil {
		t.Fatal(err)
	}
	if !pr.AmountInUSD.Equal(decimal.RequireFromString("25.75")) || pr.MemberID != 4 || pr.ExternalPlatformID != 9 ||
		*pr.CustomerEmail != "ada@example.test" || *pr.InvoiceID != "order-42" {
		t.Fatalf("payment request %+v", pr)
	}
	if created.CheckoutURL != "https://checkout.test/pay/"+created.Reference || created.ExpiresAt == nil {
		t.Fatalf("created %+v", created)
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
