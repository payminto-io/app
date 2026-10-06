package quote_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
	quoteadapter "github.com/payminto/payminto/backend/internal/paymentlifecycle/quote"
)

func TestAdapterIssuesExactCeilingQuoteFromTenantEnabledAssetAndFreshPrice(t *testing.T) {
	now := time.Date(2026, 8, 27, 11, 0, 0, 0, time.UTC)
	catalog := quoteadapter.NewMemoryAssetCatalog([]quoteadapter.Asset{
		{TenantID: "tenant-acme", ChainID: "eip155:1", AssetID: usdtAsset, Decimals: 6, Enabled: true},
	})
	prices := quoteadapter.NewMemoryPriceSource([]quoteadapter.Price{
		{
			Currency: "USD", ChainID: "eip155:1", AssetID: usdtAsset,
			MinorUnitsPerAtomicNumerator: "3", MinorUnitsPerAtomicDenominator: "20000",
			Source: "merchant-oracle", ObservedAt: now.Add(-30 * time.Second),
		},
	})
	adapter, err := quoteadapter.New(catalog, prices, quoteadapter.Config{
		Now: func() time.Time { return now }, QuoteTTL: 5 * time.Minute, MaxPriceAge: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got, err := adapter.Quote(context.Background(), validRequest(now))
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}

	// ceil(1250 * 20000 / 3) = 8,333,334 atomic units.
	if got.RequiredAtomicUnits != "8333334" {
		t.Fatalf("RequiredAtomicUnits = %q", got.RequiredAtomicUnits)
	}
	if got.RateNumerator != "3" || got.RateDenominator != "20000" || got.Rounding != paymentlifecycle.RoundingCeil {
		t.Fatalf("rate/rounding = %s/%s %s", got.RateNumerator, got.RateDenominator, got.Rounding)
	}
	if got.QuotedAt != now.Add(-30*time.Second) || got.ExpiresAt != now.Add(90*time.Second) {
		t.Fatalf("quote timeline = %s..%s", got.QuotedAt, got.ExpiresAt)
	}
	if !strings.HasPrefix(string(got.ID), "quote-") || len(got.ID) != 70 {
		t.Fatalf("Quote ID = %q", got.ID)
	}
}

const usdtAsset = paymentlifecycle.AssetID("eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7")

func validRequest(now time.Time) paymentlifecycle.QuoteRequest {
	return paymentlifecycle.QuoteRequest{
		TenantID:      "tenant-acme",
		InvoiceAmount: paymentlifecycle.FiatAmount{Currency: "USD", MinorUnits: "1250"},
		PaymentMethod: paymentlifecycle.PaymentMethod{ChainID: "eip155:1", AssetID: usdtAsset},
		InvoiceExpiry: now.Add(10 * time.Minute),
		RequestedAt:   now,
	}
}
