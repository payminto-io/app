import { describe, expect, it } from "vitest";
import { estimatedAmount, paymentURI } from "./checkout";
import { fromApi, mapState, overage, railStep, remainingDue, viewState, type ApiMethod, type ApiPayment } from "./model";
import { compareDecimal, formatAmount, subtractDecimal } from "./money";

const now = Date.parse("2026-10-07T12:00:00Z");
const base: ApiPayment = { referenceID: "ref_12345678", amountInUSD: "25", state: "OPEN", merchantName: "Northwind" };
const methods: ApiMethod[] = [
  { id: 1, blockchainCode: "SOL", currencyCode: "USDC", blockchain: { code: "SOL", name: "Solana" } },
  { id: 2, blockchainCode: "BASE", currencyCode: "USDC", blockchain: { code: "BASE", name: "Base" } },
  { id: 3, blockchainCode: "BTC", currencyCode: "BTC" },
];

describe("state mapping", () => {
  it("maps backend states", () => {
    expect(mapState("OPEN")).toBe("open");
    expect(mapState("FILLED")).toBe("paid");
    expect(mapState("PARTIALLY_FILLED")).toBe("underpaid");
    expect(mapState("OVER_FILLED")).toBe("overpaid");
    expect(mapState("CANCELLED")).toBe("cancelled");
  });

  it("expires by the authoritative timestamp", () => {
    expect(mapState("OPEN", "2026-10-07T11:59:59Z", now)).toBe("expired");
    expect(mapState("OPEN", "2026-10-07T12:00:01Z", now)).toBe("open");
  });

  it("derives choose, card, awaiting from the leg", () => {
    const open = fromApi(base, methods, now);
    expect(viewState(open)).toBe("choose");
    expect(viewState(open, "card")).toBe("card");
    const assigned = fromApi({ ...base, depositAddress: "9xQe", blockchainCode: "SOL", currencyCode: "USDC" }, methods, now);
    expect(viewState(assigned)).toBe("awaiting");
    expect(assigned.chain?.due).toEqual({ amount: "25", code: "USDC" });
    expect(assigned.chain?.networkName).toBe("Solana");
  });

  it("offers card only when the API says so and groups stablecoin networks", () => {
    const model = fromApi(base, methods, now);
    expect(model.methods.card).toBe(false);
    expect(model.methods.stablecoin).toEqual({ asset: "USDC", networks: [{ code: "SOL", name: "Solana" }, { code: "BASE", name: "Base" }] });
  });

  it("holds no due amount for a non-stable asset", () => {
    const btc = fromApi({ ...base, depositAddress: "bc1q", blockchainCode: "BTC", currencyCode: "BTC" }, methods, now);
    expect(btc.chain?.due).toBeUndefined();
  });
});

describe("under and over paid arithmetic", () => {
  const chain = { network: "SOL", networkName: "Solana", asset: "USDC", address: "x", due: { amount: "50", code: "USDC" } };
  it("computes the remainder only from held numbers", () => {
    expect(remainingDue({ ...chain, received: { amount: "40", code: "USDC" } })).toEqual({ amount: "10", code: "USDC" });
    expect(remainingDue(chain)).toBeUndefined();
    expect(remainingDue({ ...chain, received: { amount: "60", code: "USDC" } })).toBeUndefined();
  });
  it("computes the overage", () => {
    expect(overage({ ...chain, received: { amount: "60.5", code: "USDC" } })).toEqual({ amount: "10.5", code: "USDC" });
    expect(overage({ ...chain, received: { amount: "40", code: "USDC" } })).toBeUndefined();
  });
});

describe("rail", () => {
  it("fills by state", () => {
    const model = fromApi(base, methods, now);
    expect(railStep(model)).toBe(0);
    expect(railStep({ ...model, state: "confirming" })).toBe(1);
    expect(railStep({ ...model, state: "paid" })).toBe(2);
    expect(railStep({ ...model, state: "paid", settled: true })).toBe(3);
  });
});

describe("money", () => {
  it("formats per asset rules", () => {
    expect(formatAmount("1250.5", "USD")).toBe("1,250.50");
    expect(formatAmount("25", "USDC")).toBe("25.00");
    expect(formatAmount("0.000123", "USDC")).toBe("0.000123");
    expect(formatAmount("0.1234567891", "SOL")).toBe("0.123456789");
  });
  it("does exact decimal arithmetic", () => {
    expect(subtractDecimal("50", "40.25")).toBe("9.75");
    expect(compareDecimal("10.0", "10")).toBe(0);
  });
});

describe("legacy presentation helpers", () => {
  it("labels ticker conversion as an estimate and never returns invalid math", () => {
    expect(estimatedAmount("25", "BTC", { BTC: "50000" })).toBe("0.00050000");
    expect(estimatedAmount("25", "USDC", {})).toBe("25.00");
    expect(estimatedAmount("x", "BTC", { BTC: "50000" })).toBeNull();
  });
  it("encodes a BIP-21 URI for native bitcoin", () => {
    expect(paymentURI({ blockchainCode: "BTC", currencyCode: "BTC", depositAddress: "bc1qabc" }, "0.1")).toBe("bitcoin:bc1qabc?amount=0.1");
  });
});
