import { describe, expect, it } from "vitest";

import { addDecimal, paidTotalsByAsset } from "./asset-totals";

const row = (CurrencyCode: string, BlockchainCode: string, State: string, TotalAmount: string, Count: number) => ({
  CurrencyCode,
  BlockchainCode,
  State,
  TotalAmount,
  Count,
});

describe("addDecimal", () => {
  it("adds without float error", () => {
    expect(addDecimal("0.1", "0.2")).toBe("0.3");
    expect(addDecimal("1250", "0.000001")).toBe("1250.000001");
    expect(addDecimal("999.99", "0.01")).toBe("1000");
  });
});

describe("paidTotalsByAsset", () => {
  it("sums paid states within one asset and never across assets", () => {
    const totals = paidTotalsByAsset([
      row("USDC", "ETH", "FILLED", "100.5", 2),
      row("USDC", "ETH", "OVER_FILLED", "20.25", 1),
      row("USDC", "ETH", "OPEN", "5", 1),
      row("USDC", "SOL", "FILLED", "40", 1),
      row("BTC", "BTC", "FILLED", "0.0005", 1),
    ]);
    expect(totals).toEqual([
      { currency: "BTC", chain: "BTC", amount: "0.0005", payments: 1 },
      { currency: "USDC", chain: "ETH", amount: "120.75", payments: 3 },
      { currency: "USDC", chain: "SOL", amount: "40", payments: 1 },
    ]);
  });

  it("drops rows the API could not attribute to an asset, and zero totals", () => {
    expect(
      paidTotalsByAsset([row("unknown", "unknown", "FILLED", "10", 1), row("USDT", "TRON", "FILLED", "0", 1)])
    ).toEqual([]);
  });
});
