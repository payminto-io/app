import type { RevenueBreakdown } from "@/lib/api/analytics";

/** One asset on one chain. Amounts of different assets are never added together. */
export interface AssetTotal {
  currency: string;
  chain: string;
  amount: string;
  payments: number;
}

/** Payment states whose deposits count as paid (backend `PaymentState*`). */
const PAID_STATES = new Set(["FILLED", "OVER_FILLED"]);

/** Exact decimal-string addition; the backend sends `decimal.Decimal` as a string. */
export function addDecimal(a: string, b: string): string {
  const [ai = "0", af = ""] = a.trim().split(".");
  const [bi = "0", bf = ""] = b.trim().split(".");
  const scale = Math.max(af.length, bf.length);
  const sum = BigInt(ai + af.padEnd(scale, "0")) + BigInt(bi + bf.padEnd(scale, "0"));
  const neg = sum < BigInt(0);
  const digits = (neg ? -sum : sum).toString().padStart(scale + 1, "0");
  const int = digits.slice(0, digits.length - scale);
  const frac = scale ? digits.slice(-scale).replace(/0+$/, "") : "";
  return `${neg ? "-" : ""}${int}${frac ? `.${frac}` : ""}`;
}

/**
 * Paid deposits per asset from /analytics/revenue rows, which arrive per
 * chain, currency and payment state. Rows whose asset the API could not name
 * are left out rather than shown under a guessed unit.
 */
export function paidTotalsByAsset(rows: RevenueBreakdown[]): AssetTotal[] {
  const byAsset = new Map<string, AssetTotal>();
  for (const r of rows) {
    if (!PAID_STATES.has(r.State.toUpperCase())) continue;
    if (!r.CurrencyCode || r.CurrencyCode === "unknown" || !r.BlockchainCode || r.BlockchainCode === "unknown") continue;
    const key = `${r.CurrencyCode}|${r.BlockchainCode}`;
    const prev = byAsset.get(key);
    byAsset.set(key, {
      currency: r.CurrencyCode,
      chain: r.BlockchainCode,
      amount: prev ? addDecimal(prev.amount, String(r.TotalAmount)) : String(r.TotalAmount),
      payments: (prev?.payments ?? 0) + r.Count,
    });
  }
  return [...byAsset.values()]
    .filter((t) => /[1-9]/.test(t.amount))
    .sort((x, y) => x.currency.localeCompare(y.currency) || x.chain.localeCompare(y.chain));
}
