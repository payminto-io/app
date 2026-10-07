export const APP_NAME = "Payminto"
export const APP_VERSION = "0.1.0"
export const BUILD_HASH = process.env.NEXT_PUBLIC_BUILD_HASH ?? "dev-local"
export const APP_ENVIRONMENT = process.env.NODE_ENV ?? "development"
export const APP_HOMEPAGE = "https://payminto.dev"
export const APP_GITHUB = "https://github.com/sagarjethi/payminto"
export const APP_DOCS = "/developers/documentation"

/** Backend REST base URL, e.g. http://localhost:8080/api/v1 */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1"

/** v2 routes (payment links) live beside v1 on the same host; see docs/API_SPECIFICATION.md 4.44. */
export const API_V2_BASE_URL =
  process.env.NEXT_PUBLIC_API_V2_URL ?? API_BASE_URL.replace(/\/api\/v1\/?$/, "/api/v2")

/**
 * Polling intervals (ms). With real-time SSE in place, the public checkout uses
 * a slow poll purely as a reconciliation fallback rather than the primary
 * update mechanism.
 */
export const INTERVALS = {
  /** Fallback reconciliation poll for the hosted checkout. */
  checkoutFallbackPoll: 20_000,
  /** Ticker price refresh on the checkout. */
  tickerRefresh: 60_000,
  /** Delay before attempting to reconnect a dropped SSE stream. */
  sseReconnect: 3_000,
} as const

/** Public (unauthenticated) checkout API paths. */
export const PUBLIC_PATHS = {
  paymentEvents: (referenceId: string) =>
    `/public/events/${encodeURIComponent(referenceId)}`,
} as const
