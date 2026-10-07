/**
 * Checkout view model. `fromApi` maps the public payment payload onto it and
 * fills only what the API sends; every other field stays undefined and the
 * UI omits it. Fields the API does not have yet are listed in
 * .superpowers/checkout-ui-report.md.
 */

import { compareDecimal, isStablecoin, subtractDecimal } from "./money";

export interface Money {
  amount: string;
  code: string;
}

export interface LineItem {
  name: string;
  quantity?: number;
  amount: Money;
}

export interface Merchant {
  name?: string;
  logoUrl?: string;
}

export interface NetworkOption {
  code: string;
  name: string;
}

export interface MethodOptions {
  card: boolean;
  stablecoin?: { asset: string; networks: NetworkOption[] };
}

export interface ChainLeg {
  network: string;
  networkName: string;
  asset: string;
  address: string;
  /** Exact amount the chain watcher compares against; USD-stable only. */
  due?: Money;
  quoteExpiresAt?: string;
  received?: Money;
  txHash?: string;
  explorerUrl?: string;
  confirmations?: number;
  requiredConfirmations?: number;
}

export interface CardLeg {
  last4?: string;
  brand?: string;
}

export type PaymentState =
  | "open"
  | "confirming"
  | "paid"
  | "underpaid"
  | "overpaid"
  | "expired"
  | "cancelled"
  | "failed";

export interface CheckoutPayment {
  referenceId: string;
  state: PaymentState;
  merchant: Merchant;
  total: Money;
  description?: string;
  lineItems?: LineItem[];
  expiresAt?: string;
  paidAt?: string;
  methods: MethodOptions;
  chain?: ChainLeg;
  card?: CardLeg;
  success?: { message?: string; redirectUrl?: string };
  failureReason?: string;
  retryAllowed?: boolean;
  settled?: boolean;
  environment?: "test" | "live";
}

export type ViewState =
  | "choose"
  | "card"
  | "awaiting"
  | "confirming"
  | "paid"
  | "underpaid"
  | "overpaid"
  | "expired"
  | "cancelled"
  | "failed";

/** Shape of GET /api/v1/public/payment/:reference_id. */
export interface ApiPayment {
  referenceID: string;
  amountInUSD: string;
  state: string;
  expiresAt?: string | null;
  depositAddress?: string;
  blockchainCode?: string;
  currencyCode?: string;
  merchantName?: string;
}

/** Shape of GET /api/v1/public/blockchain-currencies. */
export interface ApiMethod {
  id: number;
  blockchainCode: string;
  currencyCode: string;
  blockchain?: { code: string; name: string };
  currency?: { code: string; name: string };
}

export const NETWORK_NAMES: Record<string, string> = {
  SOL: "Solana",
  SOLANA: "Solana",
  ETH: "Ethereum",
  BASE: "Base",
  POLYGON: "Polygon",
  ARB: "Arbitrum",
  ARBITRUM: "Arbitrum",
  OP: "Optimism",
  TRX: "Tron",
  BTC: "Bitcoin",
};

export function networkName(code: string, fallback?: string): string {
  return NETWORK_NAMES[code.toUpperCase()] ?? fallback ?? code;
}

export function mapState(raw: string, expiresAt?: string | null, now = Date.now()): PaymentState {
  const state = raw.toUpperCase();
  if (state === "FILLED" || state === "CONFIRMED" || state === "CLOSED") return "paid";
  if (state === "PARTIALLY_FILLED") return "underpaid";
  if (state === "OVER_FILLED") return "overpaid";
  if (state === "CONFIRMING") return "confirming";
  if (state === "CANCELLED") return "cancelled";
  if (state === "FAILED") return "failed";
  if (state === "EXPIRED") return "expired";
  if (expiresAt && Date.parse(expiresAt) <= now) return "expired";
  return "open";
}

export function fromApi(payment: ApiPayment, methods: ApiMethod[], now = Date.now()): CheckoutPayment {
  const stable = methods.filter((m) => isStablecoin(m.currencyCode));
  const asset = stable[0]?.currencyCode.toUpperCase();
  const networks = stable
    .filter((m) => m.currencyCode.toUpperCase() === asset)
    .map((m) => ({ code: m.blockchainCode.toUpperCase(), name: networkName(m.blockchainCode, m.blockchain?.name) }));
  const expiresAt = payment.expiresAt ?? undefined;
  const model: CheckoutPayment = {
    referenceId: payment.referenceID,
    state: mapState(payment.state, expiresAt, now),
    merchant: { name: payment.merchantName },
    total: { amount: payment.amountInUSD, code: "USD" },
    expiresAt,
    methods: { card: false, stablecoin: asset ? { asset, networks } : undefined },
  };
  if (payment.depositAddress && payment.blockchainCode && payment.currencyCode) {
    const code = payment.currencyCode.toUpperCase();
    model.chain = {
      network: payment.blockchainCode.toUpperCase(),
      networkName: networkName(payment.blockchainCode),
      asset: code,
      address: payment.depositAddress,
      // The chain watcher compares the raw asset amount to amountInUSD
      // (repository FinalizeFromConfirmedDeposits), so for USD stablecoins
      // the exact due amount is that number.
      due: isStablecoin(code) ? { amount: payment.amountInUSD, code } : undefined,
      quoteExpiresAt: expiresAt,
    };
  }
  return model;
}

export function viewState(payment: CheckoutPayment, selectedMethod?: "card" | "stablecoin"): ViewState {
  switch (payment.state) {
    case "paid":
      return "paid";
    case "underpaid":
      return "underpaid";
    case "overpaid":
      return "overpaid";
    case "expired":
      return "expired";
    case "cancelled":
      return "cancelled";
    case "failed":
      return "failed";
    case "confirming":
      return "confirming";
    case "open":
      if (payment.chain) return "awaiting";
      if (selectedMethod === "card") return "card";
      return "choose";
  }
}

/** Remaining amount for an under paid chain leg, only when both numbers are held. */
export function remainingDue(chain?: ChainLeg): Money | undefined {
  if (!chain?.due || !chain.received || chain.received.code !== chain.due.code) return undefined;
  if (compareDecimal(chain.received.amount, chain.due.amount) >= 0) return undefined;
  return { amount: subtractDecimal(chain.due.amount, chain.received.amount), code: chain.due.code };
}

/** Overage for an over paid chain leg, only when both numbers are held. */
export function overage(chain?: ChainLeg): Money | undefined {
  if (!chain?.due || !chain.received || chain.received.code !== chain.due.code) return undefined;
  if (compareDecimal(chain.received.amount, chain.due.amount) <= 0) return undefined;
  return { amount: subtractDecimal(chain.received.amount, chain.due.amount), code: chain.due.code };
}

/** Rail segments filled: Received, Final, Settled. DESIGN.md section 7. */
export function railStep(payment: CheckoutPayment): 0 | 1 | 2 | 3 {
  if (payment.settled) return 3;
  if (payment.state === "paid" || payment.state === "overpaid") return 2;
  if (payment.state === "confirming" || payment.state === "underpaid") return 1;
  return 0;
}

export function secondsUntil(iso?: string, now = Date.now()): number | undefined {
  if (!iso) return undefined;
  const t = Date.parse(iso);
  if (!Number.isFinite(t)) return undefined;
  return Math.max(0, Math.floor((t - now) / 1000));
}

export function formatCountdown(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}
