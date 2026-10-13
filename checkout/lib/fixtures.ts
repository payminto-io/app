/**
 * Sample payments for /preview. Every value here is invented and the preview
 * says so on screen. Never import this from a live route.
 */

import type { ChainLeg, CheckoutPayment, ViewState } from "./model";
import { encodeMemo, TEMPO_CODE, TEMPO_TESTNET } from "./tempo";

const SOON = () => new Date(Date.now() + 14 * 60_000 + 32_000).toISOString();

const base: CheckoutPayment = {
  referenceId: "pay_7Hk2mQ9sLx4T",
  state: "open",
  merchant: { name: "Northwind Supply" },
  total: { amount: "128.00", code: "USD" },
  lineItems: [
    { name: "Annual plan", amount: { amount: "120.00", code: "USD" } },
    { name: "Priority support", quantity: 2, amount: { amount: "8.00", code: "USD" } },
  ],
  methods: {
    card: true,
    stablecoin: { asset: "USDC", networks: [{ code: "SOL", name: "Solana" }, { code: "BASE", name: "Base" }, { code: "ETH", name: "Ethereum" }] },
    tempo: { asset: TEMPO_TESTNET.token.symbol, network: { code: TEMPO_CODE, name: TEMPO_TESTNET.name } },
  },
  environment: "test",
};

const chain = {
  network: "SOL",
  networkName: "Solana",
  asset: "USDC",
  address: "7GhYz3kQmP2vXr9LwE5sNbT1cJfH8dUaK6pRoW4iVyMq",
  due: { amount: "128.00", code: "USDC" },
  quoteExpiresAt: SOON(),
};

const tempoChain: ChainLeg = {
  network: TEMPO_CODE,
  networkName: TEMPO_TESTNET.name,
  asset: TEMPO_TESTNET.token.symbol,
  address: "0x7a3f9c21b84e5d06c1f2a9e8d47b3c5f60e1d2a4",
  due: { amount: "128.00", code: TEMPO_TESTNET.token.symbol },
  quoteExpiresAt: SOON(),
  memo: { reference: "PM-1042", hex: encodeMemo("PM-1042") },
};

const tx = {
  txHash: "5KtP9qzX2mW8vL4nRcY7bD1hJ3fGsA6eTuN0oCiEkQx9rM2pZ8wB4yH7dF1gS3nL",
  explorerUrl: "https://explorer.solana.com/tx/5KtP9qzX2mW8vL4nRcY7bD1hJ3fGsA6eTuN0oCiEkQx9rM2pZ8wB4yH7dF1gS3nL",
};

export const FIXTURES: Record<ViewState | "single-amount" | "tempo" | "tempo-underpaid", { payment: CheckoutPayment; label: string; state?: ViewState }> = {
  choose: { label: "Choose method", payment: base },
  "single-amount": { label: "Choose method, single amount", state: "choose", payment: { ...base, lineItems: undefined, description: "Invoice 2026-0412" } },
  card: { label: "Card (hosted fields slot)", payment: base },
  awaiting: { label: "Awaiting transfer", payment: { ...base, chain } },
  tempo: { label: "Awaiting Tempo transfer", state: "awaiting", payment: { ...base, referenceId: "PM-1042", chain: tempoChain } },
  "tempo-underpaid": {
    label: "Tempo, under paid",
    state: "underpaid",
    payment: { ...base, referenceId: "PM-1042", state: "underpaid", chain: { ...tempoChain, received: { amount: "100.00", code: TEMPO_TESTNET.token.symbol } } },
  },
  confirming: {
    label: "Confirming",
    payment: { ...base, state: "confirming", chain: { ...chain, ...tx, received: { amount: "128.00", code: "USDC" }, confirmations: 12, requiredConfirmations: 32 } },
  },
  paid: {
    label: "Paid",
    payment: { ...base, state: "paid", paidAt: "2026-10-07T12:04:31Z", chain: { ...chain, ...tx, received: { amount: "128.00", code: "USDC" }, confirmations: 32, requiredConfirmations: 32 } },
  },
  underpaid: {
    label: "Under paid",
    payment: { ...base, state: "underpaid", chain: { ...chain, ...tx, received: { amount: "100.00", code: "USDC" } } },
  },
  overpaid: {
    label: "Over paid",
    payment: { ...base, state: "overpaid", paidAt: "2026-10-07T12:04:31Z", chain: { ...chain, ...tx, received: { amount: "140.00", code: "USDC" } } },
  },
  expired: { label: "Expired", payment: { ...base, state: "expired", chain: { ...chain, quoteExpiresAt: undefined }, retryAllowed: true } },
  cancelled: { label: "Cancelled", payment: { ...base, state: "cancelled" } },
  failed: { label: "Failed", payment: { ...base, state: "failed", failureReason: "The card issuer declined the payment. Nothing was charged.", retryAllowed: true, card: { brand: "Visa", last4: "4242" } } },
};

export const PAID_WITH_MESSAGE: CheckoutPayment = {
  ...FIXTURES.paid.payment,
  success: { message: "Thanks. Your licence key is on its way to the email on the order." },
};

export const PAID_WITHOUT_LINES: CheckoutPayment = {
  ...FIXTURES.paid.payment,
  lineItems: undefined,
};
