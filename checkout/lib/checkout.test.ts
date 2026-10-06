import { describe, expect, it, vi } from "vitest";
import { estimatedAmount, paymentURI, phaseFor, type Payment } from "./checkout";

const base: Payment = { referenceID: "ref_12345678", amountInUSD: "25", state: "OPEN" };

describe("checkout state", () => {
  it("distinguishes selection, awaiting, detected, and terminal states", () => {
    expect(phaseFor(base)).toBe("choose");
    expect(phaseFor({ ...base, depositAddress: "bc1qaddress" })).toBe("awaiting");
    expect(phaseFor({ ...base, state: "PARTIALLY_FILLED" })).toBe("detected");
    expect(phaseFor({ ...base, state: "FILLED" })).toBe("paid");
  });

  it("expires by authoritative timestamp", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-08-29T12:00:00Z"));
    expect(phaseFor({ ...base, expiresAt: "2026-08-29T11:59:59Z" })).toBe("expired");
    vi.useRealTimers();
  });
});

describe("payment presentation", () => {
  it("labels ticker conversion as an estimate and never returns invalid math", () => {
    expect(estimatedAmount("25", "BTC", { BTC: "50000" })).toBe("0.00050000");
    expect(estimatedAmount("25", "USDC", {})).toBe("25.00");
    expect(estimatedAmount("x", "BTC", { BTC: "50000" })).toBeNull();
  });

  it("encodes a BIP-21 URI for native bitcoin", () => {
    expect(paymentURI({ ...base, blockchainCode: "BTC", currencyCode: "BTC", depositAddress: "bc1qabc" }, "0.1"))
      .toBe("bitcoin:bc1qabc?amount=0.1");
  });
});
