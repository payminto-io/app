import { describe, expect, it } from "vitest";
import { fromApi, viewState, type ApiMethod, type ApiPayment } from "./model";
import { encodeMemo, MemoError, toBaseUnits, tryEncodeMemo } from "./tempo";

describe("encodeMemo", () => {
  it("left aligns the UTF-8 bytes and pads with zeros to 32 bytes", () => {
    const hex = encodeMemo("PM-1042");
    expect(hex).toBe("0x504d2d3130343200000000000000000000000000000000000000000000000000");
    expect(hex).toHaveLength(2 + 64);
  });

  it("encodes multi-byte characters as UTF-8", () => {
    expect(encodeMemo("é")).toBe(`0xc3a9${"0".repeat(60)}`);
  });

  it("accepts exactly 32 bytes", () => {
    const ref = "A".repeat(32);
    expect(encodeMemo(ref)).toBe(`0x${"41".repeat(32)}`);
  });

  it("rejects more than 32 bytes, counting bytes not characters", () => {
    expect(() => encodeMemo("A".repeat(33))).toThrow(MemoError);
    expect(() => encodeMemo("é".repeat(17))).toThrow(MemoError);
    expect(tryEncodeMemo("A".repeat(33))).toBeUndefined();
  });

  it("rejects an empty reference", () => {
    expect(() => encodeMemo("")).toThrow(MemoError);
    expect(tryEncodeMemo(undefined)).toBeUndefined();
  });
});

describe("toBaseUnits", () => {
  it("scales exactly to 6 decimals", () => {
    expect(toBaseUnits("128", 6)).toBe(BigInt(128_000_000));
    expect(toBaseUnits("128.5", 6)).toBe(BigInt(128_500_000));
    expect(toBaseUnits("0.000001", 6)).toBe(BigInt(1));
  });
  it("refuses precision the token cannot hold", () => {
    expect(() => toBaseUnits("0.0000001", 6)).toThrow();
    expect(() => toBaseUnits("-1", 6)).toThrow();
  });
});

describe("Tempo in the checkout model", () => {
  const now = Date.parse("2026-10-07T12:00:00Z");
  const base: ApiPayment = { referenceID: "ref_12345678", amountInUSD: "25", state: "OPEN" };
  const methods: ApiMethod[] = [
    { id: 1, blockchainCode: "SOL", currencyCode: "USDC" },
    { id: 9, blockchainCode: "TEMPO", currencyCode: "PATHUSD" },
  ];

  it("offers Tempo as its own method, not as a stablecoin network", () => {
    const model = fromApi(base, methods, now);
    expect(model.methods.stablecoin).toEqual({ asset: "USDC", networks: [{ code: "SOL", name: "Solana" }] });
    expect(model.methods.tempo).toEqual({ asset: "pathUSD", network: { code: "TEMPO", name: "Tempo testnet" } });
  });

  it("builds a Tempo leg with the exact amount and the memo", () => {
    const model = fromApi({ ...base, depositAddress: "0xabc", blockchainCode: "TEMPO", currencyCode: "PATHUSD", memo: "PM-1042" }, methods, now);
    expect(viewState(model)).toBe("awaiting");
    expect(model.chain?.due).toEqual({ amount: "25", code: "pathUSD" });
    expect(model.chain?.memo).toEqual({ reference: "PM-1042", hex: encodeMemo("PM-1042") });
  });

  it("falls back to the payment reference for the memo", () => {
    const model = fromApi({ ...base, depositAddress: "0xabc", blockchainCode: "TEMPO", currencyCode: "PATHUSD" }, methods, now);
    expect(model.chain?.memo?.reference).toBe("ref_12345678");
  });

  it("holds no memo for other chains", () => {
    const model = fromApi({ ...base, depositAddress: "9xQe", blockchainCode: "SOL", currencyCode: "USDC" }, methods, now);
    expect(model.chain?.memo).toBeUndefined();
  });
});
