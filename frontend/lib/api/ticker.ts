/**
 * Ticker domain API — public, no auth.
 *
 * Backend route: GET /public/ticker?symbols=ETH,BTC,USDC
 * Returns: { prices: { ETH: "3000.00", BTC: "60000.00", ... } }
 *
 * The backend ticker endpoint requires a non-empty `symbols` query param; it
 * returns 400 when called without it. The frontend previously expected a
 * shape-less "all prices" GET, which surfaced as "Not Found" / "Bad Request"
 * on the wallets page. We now always pass a default symbol set for the
 * unfiltered call.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface TickerEntry {
  symbol: string;
  priceUSD: string;
  /** @deprecated backend does not provide 24h change yet. */
  change24h?: string;
  /** @deprecated backend does not expose a per-symbol updatedAt yet. */
  updatedAt?: string;
}

export interface PublicTickerResponse {
  prices: Record<string, string>;
}

/** Symbols we show tiles for on the wallets page by default. */
const DEFAULT_SYMBOLS = ["BTC", "ETH", "TRX", "MATIC", "USDT", "USDC"];

function toEntries(prices: Record<string, string> | undefined): TickerEntry[] {
  if (!prices) return [];
  return Object.entries(prices).map(([symbol, priceUSD]) => ({
    symbol,
    priceUSD,
  }));
}

export const tickerApi = {
  /**
   * Unfiltered list used by the wallets/dashboard tiles. Always asks the
   * backend for a canonical default symbol set.
   */
  all: async (): Promise<TickerEntry[]> => {
    const query = encodeURIComponent(DEFAULT_SYMBOLS.join(","));
    try {
      const res = await apiFetch<PublicTickerResponse>(
        `/public/ticker?symbols=${query}`
      );
      return toEntries(res.prices);
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },

  /** Typed price lookup for one or more symbols. */
  prices: (symbols: string[]) => {
    const query = encodeURIComponent(
      symbols.filter(Boolean).join(",") || DEFAULT_SYMBOLS.join(",")
    );
    return apiFetch<PublicTickerResponse>(`/public/ticker?symbols=${query}`);
  },
};
