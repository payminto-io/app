// Package constants is the single source of truth for cross-cutting tunable
// values used across the Payminto backend: cache TTLs, worker intervals,
// external endpoints, real-time streaming parameters, and observability names.
//
// Domain enumerations that belong to a specific model (e.g. payment states)
// live with that model. This package holds values that would otherwise be
// hardcoded in multiple unrelated places, so they can be reviewed and tuned in
// one location.
package constants

import "time"

// ── External services ───────────────────────────────────────────────────────

const (
	// CoinGeckoBaseURL is the base URL of the free CoinGecko v3 API.
	CoinGeckoBaseURL = "https://api.coingecko.com/api/v3"
	// CoinGeckoSimplePricePath is the simple-price endpoint path.
	CoinGeckoSimplePricePath = "/simple/price"
	// CoinGeckoHTTPTimeout bounds a single CoinGecko request.
	CoinGeckoHTTPTimeout = 8 * time.Second
)

// ── Ticker cache ─────────────────────────────────────────────────────────────

const (
	// TickerCacheTTL is how long a fetched price is served from memory before
	// a refresh is attempted.
	TickerCacheTTL = 5 * time.Minute
)

// ── Worker intervals ─────────────────────────────────────────────────────────

const (
	// DepositConfirmPollInterval is how often the deposit processor checks
	// CONFIRMING deposits for sufficient confirmations.
	DepositConfirmPollInterval = 10 * time.Second
)
