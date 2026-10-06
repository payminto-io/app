export interface Payment {
  referenceID: string;
  amountInUSD: string;
  state: string;
  expiresAt?: string;
  depositAddress?: string;
  blockchainCode?: string;
  currencyCode?: string;
  merchantName?: string;
}

export interface PaymentMethod {
  id: number;
  blockchainCode: string;
  currencyCode: string;
  standard: string;
  blockchain?: { code: string; name: string };
  currency?: { code: string; name: string; iconURL?: string };
}

export type CheckoutPhase = "loading" | "choose" | "awaiting" | "detected" | "paid" | "expired" | "cancelled" | "error";

export function phaseFor(payment: Payment): CheckoutPhase {
  const state = payment.state.toUpperCase();
  if (state === "FILLED" || state === "CONFIRMED") return "paid";
  if (state === "PARTIALLY_FILLED" || state === "OVER_FILLED") return "detected";
  if (state === "CANCELLED") return "cancelled";
  if (state === "EXPIRED" || (payment.expiresAt && Date.parse(payment.expiresAt) <= Date.now())) return "expired";
  return payment.depositAddress ? "awaiting" : "choose";
}

export function estimatedAmount(amountUSD: string, symbol: string | undefined, prices: Record<string, string>): string | null {
  if (!symbol) return null;
  const amount = Number(amountUSD);
  const normalized = symbol.toUpperCase();
  const stable = new Set(["USDC", "USDT", "DAI", "PYUSD", "USDP", "TUSD"]);
  const price = stable.has(normalized) ? 1 : Number(prices[normalized]);
  if (!Number.isFinite(amount) || amount <= 0 || !Number.isFinite(price) || price <= 0) return null;
  const precision = normalized === "BTC" ? 8 : normalized === "TRX" ? 6 : stable.has(normalized) ? 2 : 8;
  return (amount / price).toFixed(precision);
}

export function paymentURI(payment: Payment, amount: string | null): string {
  const address = payment.depositAddress ?? "";
  const chain = payment.blockchainCode?.toUpperCase();
  const symbol = payment.currencyCode?.toUpperCase();
  if (chain === "BTC" && amount) return `bitcoin:${address}?amount=${amount}`;
  if (chain === "ETH" && symbol === "ETH" && amount) return `ethereum:${address}?value=${amount}`;
  return address;
}

export async function getJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, { ...init, headers: { "Content-Type": "application/json", ...init?.headers } });
  const body = await response.json().catch(() => ({})) as { error?: string };
  if (!response.ok) throw new Error(body.error || "Payment service unavailable");
  return body as T;
}
