import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { fromApi, type ApiPayment } from "../lib/model";
import { encodeMemo } from "../lib/tempo";
import { CheckoutView } from "./checkout-view";
import { Paid, Underpaid } from "./status";

const payload: ApiPayment = {
  referenceID: "e257a2ed-c690-4a6e-87f6-78912ff6b340",
  memo: "e257a2edc6904a6e87f678912ff6b340",
  amountInUSD: "1250.000001",
  state: "PARTIALLY_FILLED",
  depositAddress: "0x7a3f9c21b84e5d06c1f2a9e8d47b3c5f60e1d2a4",
  blockchainCode: "TEMPO",
  currencyCode: "PATHUSD",
};
const methods = [
  { id: 10, blockchainCode: "TEMPO", currencyCode: "ALPHAUSD" },
  { id: 9, blockchainCode: "TEMPO", currencyCode: "PATHUSD" },
];

describe("Tempo payer API contract", () => {
  it("uses the compact UUID memo supplied by the backend", () => {
    expect(fromApi(payload, methods).chain?.memo?.hex).toBe(encodeMemo(payload.memo!));
  });

  it("prefers pathUSD independently of API currency ordering", () => {
    expect(fromApi(payload, methods).methods.tempo?.asset).toBe("pathUSD");
  });

  it("supports an issued AlphaUSD leg without labeling it as pathUSD", () => {
    const model = fromApi({ ...payload, state: "OPEN", currencyCode: "ALPHAUSD" }, methods);
    expect(model.chain?.due).toEqual({ amount: payload.amountInUSD, code: "AlphaUSD" });
    const html = renderToStaticMarkup(<CheckoutView payment={model} handlers={{ onMethod() {}, onNetwork() {}, onContinue() {}, onBack() {} }} />);
    expect(html).toContain("AlphaUSD");
    expect(html).not.toContain("pathUSD");
    expect(html).not.toContain(">Change<");
  });

  it("does not offer unknown Tempo tokens", () => {
    expect(fromApi(payload, [{ id: 1, blockchainCode: "TEMPO", currencyCode: "UNKNOWN" }]).methods.tempo).toBeUndefined();
  });

  it.each(["TEMPO", "SOL"])("does not show the full invoice amount as a %s top-up when received amount is absent", (blockchainCode) => {
    const model = fromApi({ ...payload, blockchainCode, currencyCode: blockchainCode === "TEMPO" ? "PATHUSD" : "USDC" }, methods);
    const html = renderToStaticMarkup(<Underpaid payment={model} />);
    expect(html).not.toContain('aria-label="Copy amount"');
    expect(html).toContain("remaining amount is unavailable");
    expect(html).toContain("before sending more");
  });

  it("shows only the known remainder when received amount is authoritative", () => {
    const model = fromApi(payload, methods);
    model.chain!.received = { amount: "1250", code: "pathUSD" };
    const html = renderToStaticMarkup(<Underpaid payment={model} />);
    expect(html).toContain('aria-label="Copy amount"');
    expect(html).toContain("0.000001");
    const copyAmount = html.match(/<button[^>]*aria-label="Copy amount"[\s\S]*?<\/button>/)?.[0];
    expect(copyAmount).toContain("0.000001");
    expect(copyAmount).not.toContain("1,250.000001");
  });

  it("labels invoice amounts as due when a paid API response contains no received amount", () => {
    const html = renderToStaticMarkup(<Paid payment={fromApi({ ...payload, state: "FILLED" }, methods)} />);
    expect(html).toContain("The payment is final.");
    expect(html).toContain("Amount due");
    expect(html).not.toContain(">Paid</span>");
  });
});
