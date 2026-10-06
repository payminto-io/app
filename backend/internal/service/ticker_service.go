package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/shopspring/decimal"
)

const tickerCacheNamespace = "ticker"

// geckoIDs maps Payminto currency symbols to CoinGecko coin IDs.
var geckoIDs = map[string]string{
	"BTC":   "bitcoin",
	"ETH":   "ethereum",
	"USDT":  "tether",
	"USDC":  "usd-coin",
	"TRX":   "tron",
	"MATIC": "matic-network",
	"POL":   "matic-network",
	"BNB":   "binancecoin",
}

// tickerCacheEntry is a single in-memory cached price.
type tickerCacheEntry struct {
	price     decimal.Decimal
	expiresAt time.Time
}

// hardcodedPrices is the resilience fallback used when there is no
// configuration override AND the live source (CoinGecko) is unavailable. It
// keeps checkout functional during a price-feed outage and keeps unit tests
// deterministic/offline.
var hardcodedPrices = map[string]decimal.Decimal{
	"BTC":   decimal.NewFromInt(50000),
	"ETH":   decimal.NewFromInt(3000),
	"USDT":  decimal.NewFromFloat(1.00),
	"USDC":  decimal.NewFromFloat(1.00),
	"TRX":   decimal.NewFromFloat(0.10),
	"MATIC": decimal.NewFromFloat(0.85),
	"BNB":   decimal.NewFromInt(400),
}

// TickerService provides current USD prices for crypto assets. Prices are
// cached in memory (TTL constants.TickerCacheTTL). Resolution order is:
// configuration override → live source (CoinGecko, when enabled) → hardcoded
// fallback. Overrides let operators pin a price; the fallback guarantees a
// usable price during a live-feed outage.
type TickerService struct {
	configSvc *ConfigurationService

	// liveFetch resolves a live USD price for a symbol. nil disables live
	// lookups (used in unit tests so the suite stays offline/deterministic).
	liveFetch func(ctx context.Context, symbol string) (decimal.Decimal, error)

	mu    sync.RWMutex
	cache map[string]*tickerCacheEntry
}

// TickerOption configures a TickerService at construction time.
type TickerOption func(*TickerService)

// WithCoinGecko enables live price lookups against the free CoinGecko API.
// Production wiring uses this; unit tests omit it so they never hit the network.
func WithCoinGecko() TickerOption {
	client := &http.Client{Timeout: constants.CoinGeckoHTTPTimeout}
	return func(s *TickerService) {
		s.liveFetch = func(ctx context.Context, symbol string) (decimal.Decimal, error) {
			return fetchCoinGeckoPrice(ctx, client, constants.CoinGeckoBaseURL, symbol)
		}
	}
}

// WithLiveFetcher injects a custom live-price function. Primarily for tests.
func WithLiveFetcher(fn func(ctx context.Context, symbol string) (decimal.Decimal, error)) TickerOption {
	return func(s *TickerService) { s.liveFetch = fn }
}

// NewTickerService constructs a TickerService. configSvc may be nil — in that
// case configuration overrides are skipped. Without options, live lookups are
// disabled and only overrides + the hardcoded fallback are used.
func NewTickerService(configSvc *ConfigurationService, opts ...TickerOption) *TickerService {
	s := &TickerService{
		configSvc: configSvc,
		cache:     make(map[string]*tickerCacheEntry),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// fetchCoinGeckoPrice queries CoinGecko's simple/price endpoint for one symbol
// and returns its USD price. Unknown symbols or HTTP/parse failures return an
// error so the caller can fall back to the hardcoded table.
func fetchCoinGeckoPrice(ctx context.Context, client *http.Client, baseURL, symbol string) (decimal.Decimal, error) {
	id, ok := geckoIDs[strings.ToUpper(symbol)]
	if !ok {
		return decimal.Zero, fmt.Errorf("ticker: no coingecko id for %q", symbol)
	}
	url := fmt.Sprintf("%s%s?ids=%s&vs_currencies=usd", baseURL, constants.CoinGeckoSimplePricePath, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return decimal.Zero, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("ticker: coingecko status %d", resp.StatusCode)
	}
	// Response shape: {"bitcoin":{"usd":50123.4}}
	var body map[string]map[string]json.Number
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, err
	}
	coin, ok := body[id]
	if !ok {
		return decimal.Zero, fmt.Errorf("ticker: coingecko missing %q", id)
	}
	usd, ok := coin["usd"]
	if !ok {
		return decimal.Zero, fmt.Errorf("ticker: coingecko missing usd for %q", id)
	}
	price, err := decimal.NewFromString(usd.String())
	if err != nil {
		return decimal.Zero, err
	}
	if price.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("ticker: coingecko non-positive price for %q", id)
	}
	return price, nil
}

// GetPrice returns the current USD price for the given symbol.
// Lookup order: (1) in-memory cache, (2) configuration override, (3) live
// CoinGecko price (when enabled), (4) hardcoded fallback.
func (s *TickerService) GetPrice(ctx context.Context, symbol string) (decimal.Decimal, error) {
	if symbol == "" {
		return decimal.Zero, fmt.Errorf("ticker: symbol is required")
	}

	// Check in-memory cache.
	if price, ok := s.cacheGet(symbol); ok {
		return price, nil
	}

	price, err := s.fetchFromSource(ctx, symbol)
	if err != nil {
		return decimal.Zero, err
	}

	s.cacheSet(symbol, price)
	return price, nil
}

// GetPrices returns current USD prices for all requested symbols. Symbols
// that cannot be resolved are omitted from the result map.
func (s *TickerService) GetPrices(ctx context.Context, symbols []string) (map[string]decimal.Decimal, error) {
	result := make(map[string]decimal.Decimal, len(symbols))
	for _, sym := range symbols {
		price, err := s.GetPrice(ctx, sym)
		if err != nil {
			// Skip unknown symbols instead of failing the batch.
			continue
		}
		result[sym] = price
	}
	return result, nil
}

// fetchFromSource resolves a price using the configured source priority:
//  1. Configuration override key "ticker.override.<SYMBOL>"
//  2. Live source (CoinGecko) when enabled
//  3. Hardcoded fallback table
func (s *TickerService) fetchFromSource(ctx context.Context, symbol string) (decimal.Decimal, error) {
	// 1. Configuration override (e.g. for testing or manual adjustment).
	if s.configSvc != nil {
		overrideKey := "ticker.override." + symbol
		val := s.configSvc.GetString(ctx, overrideKey, "")
		if val != "" {
			price, err := decimal.NewFromString(val)
			if err == nil {
				return price, nil
			}
		}
	}

	// 2. Live source (CoinGecko) when enabled. On any failure we fall through
	//    to the hardcoded table so a CoinGecko outage never blocks checkout.
	if s.liveFetch != nil {
		if price, err := s.liveFetch(ctx, symbol); err == nil {
			return price, nil
		}
	}

	// 3. Hardcoded fallback (dev/tests, and resilience when live source is down).
	if price, ok := hardcodedPrices[strings.ToUpper(symbol)]; ok {
		return price, nil
	}

	return decimal.Zero, fmt.Errorf("ticker: unknown symbol %q (no override, no live price, no fallback)", symbol)
}

// cacheGet returns a cached price and whether it was found and non-expired.
func (s *TickerService) cacheGet(symbol string) (decimal.Decimal, bool) {
	s.mu.RLock()
	entry, ok := s.cache[symbol]
	s.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return decimal.Zero, false
	}
	return entry.price, true
}

// cacheSet stores a price in the in-memory cache with the configured TTL.
func (s *TickerService) cacheSet(symbol string, price decimal.Decimal) {
	s.mu.Lock()
	s.cache[symbol] = &tickerCacheEntry{
		price:     price,
		expiresAt: time.Now().Add(constants.TickerCacheTTL),
	}
	s.mu.Unlock()
}

// InvalidateCache removes all cached prices, forcing a fresh fetch on next call.
// Useful for tests that inject configuration overrides mid-test.
func (s *TickerService) InvalidateCache() {
	s.mu.Lock()
	s.cache = make(map[string]*tickerCacheEntry)
	s.mu.Unlock()
}

// InvalidateSymbol removes a single symbol from the cache.
func (s *TickerService) InvalidateSymbol(symbol string) {
	s.mu.Lock()
	delete(s.cache, symbol)
	s.mu.Unlock()
}

// priceToFloat64 is a helper for JSON serialisation of decimal prices.
func priceToFloat64(d decimal.Decimal) float64 {
	f, _ := strconv.ParseFloat(d.String(), 64)
	return f
}
