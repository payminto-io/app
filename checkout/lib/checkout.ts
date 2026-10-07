import type { ApiPayment } from "./model";
import { isStablecoin } from "./money";

/** Estimate for non-stable assets from the ticker; labelled as an estimate wherever shown. */
export function estimatedAmount(amountUSD: string, symbol: string | undefined, prices: Record<string, string>): string | null {
  if (!symbol) return null;
  const amount = Number(amountUSD);
  const normalized = symbol.toUpperCase();
  const price = isStablecoin(normalized) ? 1 : Number(prices[normalized]);
  if (!Number.isFinite(amount) || amount <= 0 || !Number.isFinite(price) || price <= 0) return null;
  const precision = normalized === "BTC" ? 8 : normalized === "TRX" ? 6 : isStablecoin(normalized) ? 2 : 8;
  return (amount / price).toFixed(precision);
}

export function paymentURI(payment: Pick<ApiPayment, "depositAddress" | "blockchainCode" | "currencyCode">, amount: string | null): string {
  const address = payment.depositAddress ?? "";
  const chain = payment.blockchainCode?.toUpperCase();
  const symbol = payment.currencyCode?.toUpperCase();
  if (chain === "BTC" && amount) return `bitcoin:${address}?amount=${amount}`;
  if (chain === "ETH" && symbol === "ETH" && amount) return `ethereum:${address}?value=${amount}`;
  return address;
}

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
  }
}

export async function getJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  const body = (await response.json().catch(() => ({}))) as { error?: string };
  if (!response.ok) throw new ApiError(body.error || "Payment service unavailable", response.status);
  return body as T;
}
