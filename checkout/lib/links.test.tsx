import { renderToStaticMarkup } from "react-dom/server";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import LinkPage from "../app/l/[shortCode]/page";
import { startPayment } from "../app/l/[shortCode]/actions";

vi.mock("next/navigation", () => ({
  redirect: (url: string) => { throw new Error(`redirect:${url}`); },
  notFound: () => { throw new Error("not-found"); },
}));

const shortCode = "AbCdEf123456";
const key = "18983afe-1169-4a26-b66b-5eb2233a4f52";
const link = {
  short_code: shortCode, available: true, unavailable_reason: null,
  merchant_name: "Merchant", title: "Order", description: null,
  amount_mode: "fixed", amount: "25", currency: "USD",
  methods: [{ method: "crypto", chain: "SOLANA", asset: "USDC", customer_total: "25" }],
};
const paymentRequests: string[] = [];

beforeEach(() => {
  paymentRequests.length = 0;
  vi.stubGlobal("fetch", vi.fn(async (_url: string, init?: RequestInit) => {
    if (init?.method !== "POST") return Response.json(link);
    paymentRequests.push(new Headers(init.headers).get("Idempotency-Key") ?? "");
    return Response.json({ payment_reference: "pl_issued-reference" });
  }));
});
afterEach(() => vi.unstubAllGlobals());

function payerForm(idempotencyKey: string) {
  const form = new FormData();
  form.set("short_code", shortCode);
  form.set("method", "crypto");
  form.set("chain", "SOLANA");
  form.set("asset", "USDC");
  form.set("idempotency_key", idempotencyKey);
  return form;
}
async function renderLink() {
  return renderToStaticMarkup(await LinkPage({ params: Promise.resolve({ shortCode }), searchParams: Promise.resolve({}) }));
}
function renderedKey(html: string) {
  const match = html.match(/name="idempotency_key" value="([^"]+)"/);
  expect(match, "the payer form must carry its checkout submission identity").not.toBeNull();
  return match![1];
}

describe("hosted link checkout submission identity", () => {
  it("passes one form's identity unchanged for two submissions", async () => {
    const form = payerForm(key);
    await expect(startPayment(form)).rejects.toThrow("redirect:/pay/pl_issued-reference");
    await expect(startPayment(form)).rejects.toThrow("redirect:/pay/pl_issued-reference");
    expect(paymentRequests).toEqual([key, key]);
  });

  it("uses the rendered form identity for a double submit, and a new identity for a new checkout", async () => {
    const firstKey = renderedKey(await renderLink());
    const sameForm = payerForm(firstKey);
    await expect(startPayment(sameForm)).rejects.toThrow("redirect:/pay/pl_issued-reference");
    await expect(startPayment(sameForm)).rejects.toThrow("redirect:/pay/pl_issued-reference");
    expect(paymentRequests).toEqual([firstKey, firstKey]);
    expect(renderedKey(await renderLink())).not.toBe(firstKey);
  });

  it.each(["", "bad-key", "\n18983afe-1169-4a26-b66b-5eb2233a4f52", "x".repeat(129)])("refuses malformed submission identity %j without opening a payment", async (invalidKey) => {
    await expect(startPayment(payerForm(invalidKey))).rejects.toThrow(`redirect:/l/${shortCode}?error=1`);
    expect(paymentRequests).toEqual([]);
  });
});
