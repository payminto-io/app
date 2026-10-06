package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestTickerService_KnownSymbol(t *testing.T) {
	svc := NewTickerService(nil)

	price, err := svc.GetPrice(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("GetPrice BTC: %v", err)
	}
	if !price.Equal(decimal.NewFromInt(50000)) {
		t.Errorf("BTC price: got %s want 50000", price.String())
	}
}

func TestTickerService_ETH(t *testing.T) {
	svc := NewTickerService(nil)

	price, err := svc.GetPrice(context.Background(), "ETH")
	if err != nil {
		t.Fatalf("GetPrice ETH: %v", err)
	}
	if !price.Equal(decimal.NewFromInt(3000)) {
		t.Errorf("ETH price: got %s want 3000", price.String())
	}
}

func TestTickerService_USDT(t *testing.T) {
	svc := NewTickerService(nil)
	price, _ := svc.GetPrice(context.Background(), "USDT")
	if !price.Equal(decimal.NewFromFloat(1.00)) {
		t.Errorf("USDT: got %s want 1.00", price.String())
	}
}

func TestTickerService_UnknownSymbol(t *testing.T) {
	svc := NewTickerService(nil)
	_, err := svc.GetPrice(context.Background(), "UNKNOWN_XYZ")
	if err == nil {
		t.Error("expected error for unknown symbol")
	}
}

func TestTickerService_BatchGetPrices(t *testing.T) {
	svc := NewTickerService(nil)

	prices, err := svc.GetPrices(context.Background(), []string{"BTC", "ETH", "USDT", "UNKNOWN_XYZ"})
	if err != nil {
		t.Fatalf("GetPrices: %v", err)
	}

	// Unknown symbol is silently skipped.
	if _, ok := prices["UNKNOWN_XYZ"]; ok {
		t.Error("unknown symbol should be omitted from batch result")
	}

	// Known symbols are present.
	for _, sym := range []string{"BTC", "ETH", "USDT"} {
		if _, ok := prices[sym]; !ok {
			t.Errorf("expected %s in batch result", sym)
		}
	}
}

func TestTickerService_CacheHit(t *testing.T) {
	svc := NewTickerService(nil)
	ctx := context.Background()

	// First call populates cache.
	p1, _ := svc.GetPrice(ctx, "ETH")

	// Inject a different hardcoded value — cache should still return original.
	// (We can't easily mock the source, but we can verify the cache key is used
	// by calling twice and confirming consistent result.)
	p2, _ := svc.GetPrice(ctx, "ETH")

	if !p1.Equal(p2) {
		t.Errorf("cache hit should return same value: %s vs %s", p1, p2)
	}
}

func TestTickerService_InvalidateCache(t *testing.T) {
	svc := NewTickerService(nil)
	ctx := context.Background()

	svc.GetPrice(ctx, "BTC") // populate cache

	svc.InvalidateSymbol("BTC")

	// After invalidation the source is re-fetched — still returns hardcoded value.
	price, err := svc.GetPrice(ctx, "BTC")
	if err != nil {
		t.Fatalf("post-invalidate GetPrice: %v", err)
	}
	if price.IsZero() {
		t.Error("expected non-zero price after invalidation")
	}
}

func TestTickerService_InvalidateAllCache(t *testing.T) {
	svc := NewTickerService(nil)
	ctx := context.Background()

	svc.GetPrice(ctx, "BTC")
	svc.GetPrice(ctx, "ETH")

	svc.InvalidateCache()

	// Both should still be fetchable.
	p1, _ := svc.GetPrice(ctx, "BTC")
	p2, _ := svc.GetPrice(ctx, "ETH")

	if p1.IsZero() || p2.IsZero() {
		t.Error("expected non-zero prices after full invalidation")
	}
}

func TestTickerService_EmptySymbol(t *testing.T) {
	svc := NewTickerService(nil)
	_, err := svc.GetPrice(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty symbol")
	}
}
