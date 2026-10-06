package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/shopspring/decimal"
)

// TestTickerService_LiveFetcherUsed verifies that when a live fetcher is
// configured it takes priority over the hardcoded fallback.
func TestTickerService_LiveFetcherUsed(t *testing.T) {
	svc := NewTickerService(nil, WithLiveFetcher(func(_ context.Context, sym string) (decimal.Decimal, error) {
		if sym == "BTC" {
			return decimal.NewFromInt(67890), nil
		}
		return decimal.Zero, errors.New("unknown")
	}))

	price, err := svc.GetPrice(context.Background(), "BTC")
	if err != nil {
		t.Fatalf("GetPrice: %v", err)
	}
	if !price.Equal(decimal.NewFromInt(67890)) {
		t.Fatalf("expected live price 67890, got %s", price)
	}
}

// TestTickerService_LiveFailureFallsBack verifies that a live-source error
// transparently falls back to the hardcoded table — a CoinGecko outage must
// never block checkout.
func TestTickerService_LiveFailureFallsBack(t *testing.T) {
	svc := NewTickerService(nil, WithLiveFetcher(func(context.Context, string) (decimal.Decimal, error) {
		return decimal.Zero, errors.New("coingecko down")
	}))

	price, err := svc.GetPrice(context.Background(), "ETH")
	if err != nil {
		t.Fatalf("GetPrice should fall back, got error: %v", err)
	}
	if !price.Equal(hardcodedPrices["ETH"]) {
		t.Fatalf("expected fallback %s, got %s", hardcodedPrices["ETH"], price)
	}
}

// TestFetchCoinGeckoPrice parses a real CoinGecko-shaped response from a local
// test server, proving the HTTP/JSON path without hitting the network.
func TestFetchCoinGeckoPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ids") != "bitcoin" {
			t.Errorf("unexpected ids param: %s", r.URL.Query().Get("ids"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bitcoin":{"usd":68250.42}}`))
	}))
	defer srv.Close()

	price, err := fetchCoinGeckoPrice(context.Background(), srv.Client(), srv.URL, "BTC")
	if err != nil {
		t.Fatalf("fetchCoinGeckoPrice: %v", err)
	}
	if !price.Equal(decimal.RequireFromString("68250.42")) {
		t.Fatalf("expected 68250.42, got %s", price)
	}
}

// TestFetchCoinGeckoPrice_UnknownSymbol ensures unmapped symbols error out so
// the caller can fall back.
func TestFetchCoinGeckoPrice_UnknownSymbol(t *testing.T) {
	if _, err := fetchCoinGeckoPrice(context.Background(), http.DefaultClient, constants.CoinGeckoBaseURL, "DOGE"); err == nil {
		t.Fatal("expected error for unmapped symbol DOGE")
	}
}
