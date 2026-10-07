import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api/errors";
import type { PaymentLink } from "@/lib/api/links";
import {
  defaultForm,
  formFromLink,
  formToInput,
  formToPatch,
  hasContent,
  localErrors,
  pricingFingerprint,
  type LinkForm,
} from "./model";
import { countByStep, errorAt, mapApiErrors, stepForField } from "./errors";
import { buildRenderModel, feeRequestFor, type MethodFee } from "./preview";

const form = (patch: Partial<LinkForm> = {}): LinkForm => ({ ...defaultForm(), ...patch });

function link(patch: Partial<PaymentLink> = {}): PaymentLink {
  return {
    ...formToInput(form({ title: "Workshop", currency: "USD", amount: "50.00", methods: [{ method: "card" }] })),
    id: "lnk_1",
    status: "draft",
    environment: "test",
    short_code: null,
    url: null,
    total: "50.00",
    uses_count: 0,
    revision: 1,
    published_at: null,
    created_at: "2026-10-07T10:00:00Z",
    updated_at: "2026-10-07T10:00:00Z",
    ...patch,
  };
}

describe("form state", () => {
  it("defaults match the server's DefaultInput and have no content", () => {
    const i = formToInput(defaultForm());
    expect(i.amount_mode).toBe("fixed");
    expect(i.customer_field_policy).toEqual({ name: { mode: "optional" }, email: { mode: "optional" }, phone: { mode: "hidden" } });
    expect(i.quote_expiry_seconds).toBe(900);
    expect(i.fee_bearer).toBe("merchant");
    expect(i.failure_retry).toBe(true);
    expect(i.language).toBe("en");
    expect(hasContent(defaultForm())).toBe(false);
    expect(hasContent(form({ title: "x" }))).toBe(true);
  });

  it("sends only the amount fields of the chosen mode", () => {
    const base = { amount: "10", amount_min: "1", amount_max: "9", line_items: [{ name: "A", quantity: "2", unit_price: "3.5", tax_rate: "18" }] };
    expect(formToInput(form({ ...base, amount_mode: "fixed" }))).toMatchObject({ amount: "10", amount_min: null, amount_max: null, line_items: [] });
    expect(formToInput(form({ ...base, amount_mode: "customer" }))).toMatchObject({ amount: null, amount_min: "1", amount_max: "9", line_items: [] });
    expect(formToInput(form({ ...base, amount_mode: "line_items" }))).toMatchObject({
      amount: null,
      amount_min: null,
      line_items: [{ name: "A", quantity: 2, unit_price: "3.5", tax_rate: "18" }],
    });
  });

  it("normalises codes, metadata, questions, settlement and limits", () => {
    const i = formToInput(
      form({
        currency: "usd",
        metadata: [{ key: " order ", value: "42" }, { key: "", value: "" }],
        methods: [{ method: "crypto", chain: "sol", asset: "usdc" }, { method: "upi", chain: "x" }],
        questions: [
          { key: "size", label: "Size", type: "select", options: "S\n\n M ", required: true, per_order: true },
          { key: "note", label: "Note", type: "text", options: "ignored", required: false, per_order: false },
        ],
        settlement: { mode: "crypto", destination_id: " d1 ", chain: "sol", asset: "usdc" },
        use_limit: "5",
        expires_after_payments: "3",
        webhook_id: "7",
      })
    );
    expect(i.currency).toBe("USD");
    expect(i.metadata).toEqual({ order: "42" });
    expect(i.methods).toEqual([{ method: "crypto", chain: "SOL", asset: "USDC" }, { method: "upi" }]);
    expect(i.questions[0].options).toEqual(["S", "M"]);
    expect(i.questions[1]).not.toHaveProperty("options");
    expect(i.settlement_override).toEqual({ kind: "crypto", destination_id: "d1", chain: "SOL", asset: "USDC" });
    expect(i.use_limit).toBeNull();
    expect(i.expires_after_payments).toBeNull();
    expect(i.webhook_id).toBe(7);
    expect(formToInput(form({ multi_use: true, use_limit: "5" })).use_limit).toBe(5);
    expect(formToInput(form()).settlement_override).toBeNull();
  });

  it("round-trips a stored link", () => {
    const l = link({ customer_field_policy: { name: { mode: "required", prefill: "Ada" }, email: { mode: "optional" }, phone: { mode: "hidden" } } });
    const f = formFromLink(l);
    expect(f.customer.name).toEqual({ mode: "required", prefill: "Ada" });
    const back = formToInput(f);
    const input = Object.fromEntries(Object.keys(back).map((k) => [k, l[k as keyof PaymentLink]]));
    expect(back).toEqual(input);
  });

  it("flags only values JSON cannot carry", () => {
    const errs = localErrors(
      form({ amount: "ten", use_limit: "1.5", amount_mode: "fixed", line_items: [{ name: "", quantity: "x", unit_price: "1", tax_rate: "0" }] })
    );
    expect(errs).toEqual({ amount: "Enter a number", use_limit: "Enter a whole number" });
    expect(localErrors(form({ amount_mode: "line_items", line_items: [{ name: "", quantity: "x", unit_price: "-1", tax_rate: "0" }] }))).toEqual({
      "line_items[0].quantity": "Enter a whole number",
      "line_items[0].unit_price": "Enter a number",
    });
    expect(localErrors(form({ amount: "" }))).toEqual({});
  });

  it("leaves immutable keys out of a published link's patch", () => {
    const f = form({ title: "T", amount: "5", currency: "USD", methods: [{ method: "card" }] });
    expect(formToPatch(f, "draft")).toHaveProperty("amount", "5");
    const p = formToPatch(f, "active");
    for (const k of ["amount_mode", "amount", "amount_min", "amount_max", "currency", "methods", "line_items"]) {
      expect(p).not.toHaveProperty(k);
    }
    expect(p).toHaveProperty("title", "T");
  });

  it("pricing fingerprint moves only with pricing fields", () => {
    const a = form({ amount_mode: "line_items", line_items: [{ name: "A", quantity: "1", unit_price: "2", tax_rate: "0" }] });
    expect(pricingFingerprint({ ...a, title: "other" })).toBe(pricingFingerprint(a));
    expect(pricingFingerprint({ ...a, line_items: [{ ...a.line_items[0], quantity: "2" }] })).not.toBe(pricingFingerprint(a));
  });
});

describe("API error mapping", () => {
  it("routes every errors[] entry to its field and step", () => {
    const err = new ApiError(422, "is required", {
      error: "is required",
      code: "title_required",
      field: "title",
      errors: [
        { code: "title_required", field: "title", message: "is required" },
        { code: "line_item_invalid", field: "line_items[1].quantity", message: "must be 1-100000" },
        { code: "method_no_connector", field: "methods[0]", message: "no connector takes card for USD in test" },
        { code: "customer_field_policy_invalid", field: "customer_field_policy.phone.prefill", message: "must be E.164" },
      ],
    });
    const m = mapApiErrors(err);
    expect(m.fields).toEqual({
      title: "Is required",
      "line_items[1].quantity": "Must be 1-100000",
      "methods[0]": "No connector takes card for USD in test",
      "customer_field_policy.phone.prefill": "Must be E.164",
    });
    expect(m.general).toEqual([]);
    expect(countByStep(m.fields)).toEqual({ item: 2, payment: 1, customer: 1 });
  });

  it("uses code/field when there is a single error, and keeps the first message per field", () => {
    const one = mapApiErrors(new ApiError(422, "must be #RRGGBB", { error: "must be #RRGGBB", code: "accent_color_invalid", field: "accent_color" }));
    expect(one.fields).toEqual({ accent_color: "Must be #RRGGBB" });
    const two = mapApiErrors(
      new ApiError(422, "x", {
        error: "a",
        code: "c",
        errors: [
          { code: "a", field: "amount_min", message: "first" },
          { code: "b", field: "amount_min", message: "second" },
        ],
      })
    );
    expect(two.fields.amount_min).toBe("First");
  });

  it("keeps fieldless and unknown-field errors general", () => {
    const conflict = mapApiErrors(new ApiError(409, "a live link cannot change its amount", { error: "a live link cannot change its amount", code: "link_published_immutable" }));
    expect(conflict.fields).toEqual({});
    expect(conflict.general).toEqual([{ code: "link_published_immutable", message: "A live link cannot change its amount" }]);
    expect(mapApiErrors(new ApiError(400, "body", { error: "x", code: "invalid_json", field: "nope" })).general[0].code).toBe("invalid_json");
    expect(mapApiErrors(new Error("offline")).general[0].message).toBe("offline");
  });

  it("finds steps and nested errors", () => {
    expect(stepForField("settlement_override.destination_id")).toBe("settlement");
    expect(stepForField("questions[2].key")).toBe("customer");
    expect(stepForField("unknown")).toBeNull();
    expect(errorAt({ "methods[1].chain": "x" }, "methods[1]", true)).toBe("x");
    expect(errorAt({ "methods[1].chain": "x" }, "methods[1]")).toBeUndefined();
    expect(errorAt({ "methods[10]": "y" }, "methods[1]", true)).toBeUndefined();
  });
});

describe("preview mapping", () => {
  const preview = (fee: string) => ({
    rule_id: 4,
    version: 2,
    currency: "USD",
    fee_bearer: "customer" as const,
    amount: "50.00",
    fee,
    tax: "0.27",
    customer_total: "51.72",
    merchant_net: "50.00",
  });

  it("asks the fee API with the amount the server would price on", () => {
    const f = form({ currency: "usd", amount: "50.00", fee_bearer: "customer" });
    expect(feeRequestFor(f, { method: "card" }, null)).toEqual({ request: { amount: "50.00", currency: "USD", method: "card", fee_bearer: "customer" } });
    expect(feeRequestFor(form({ currency: "USD", amount_mode: "line_items" }), { method: "card" }, "12.36")).toMatchObject({ request: { amount: "12.36" } });
    expect(feeRequestFor(form({ currency: "USD", amount_mode: "line_items" }), { method: "card" }, null)).toEqual({ state: { state: "no_amount" } });
    expect(feeRequestFor(form({ currency: "USD", amount_mode: "customer", amount_min: "5" }), { method: "upi" }, null)).toMatchObject({ request: { amount: "5" } });
    expect(feeRequestFor(form({ currency: "USD", amount_mode: "customer" }), { method: "upi" }, null)).toEqual({ state: { state: "no_amount" } });
    expect(feeRequestFor(f, { method: "crypto", chain: "SOL", asset: "USDC" }, null)).toEqual({ state: { state: "at_pay_time" } });
    expect(feeRequestFor(form({ currency: "USDC", amount: "5" }), { method: "crypto", chain: "SOL", asset: "usdc" }, null)).toMatchObject({ request: { currency: "USDC", method: "crypto" } });
  });

  it("carries a surcharge only for a customer-borne fee and drops methods the server would drop", () => {
    const f = form({
      title: "Workshop",
      currency: "USD",
      amount: "50.00",
      fee_bearer: "customer",
      methods: [{ method: "card" }, { method: "upi" }, { method: "crypto", chain: "SOL", asset: "USDC" }, { method: "bank" }],
    });
    const fees: MethodFee[] = [
      { state: "ok", preview: preview("1.45") },
      { state: "refused", code: "surcharge_forbidden", message: "" },
      { state: "at_pay_time" },
      { state: "refused", code: "no_fee_rule", message: "" },
    ];
    const m = buildRenderModel(f, { link: null, serverTotal: null, fees, merchantName: null });
    expect(m.methods).toEqual([
      { method: "card", chain: null, asset: null, fee: "1.45", tax: "0.27", customer_total: "51.72" },
      { method: "crypto", chain: "SOL", asset: "USDC", fee: null, tax: null, customer_total: null },
    ]);
    expect(m.amount).toBe("50.00");
    expect(m.available).toBe(true);

    const merchant = buildRenderModel({ ...f, fee_bearer: "merchant" }, { link: null, serverTotal: null, fees, merchantName: null });
    expect(merchant.methods[0].fee).toBeNull();
  });

  it("matches the public model's privacy rules and availability", () => {
    const f = form({
      title: "T",
      currency: "USD",
      amount: "1",
      customer: { name: { mode: "hidden", prefill: "Ada" }, email: { mode: "required", prefill: "a@b.co" }, phone: { mode: "optional", prefill: "" } },
      accent_color: "#12345",
    });
    const m = buildRenderModel(f, { link: null, serverTotal: null, fees: [], merchantName: "Acme" });
    expect(m.customer_fields.name).toEqual({ mode: "hidden", prefill: null });
    expect(m.customer_fields.email).toEqual({ mode: "required", prefill: "a@b.co" });
    expect(m.customer_fields.phone.prefill).toBeNull();
    expect(m.branding.accent_color).toBeNull();
    expect(m.unavailable_reason).toBe("no_methods_available");
    expect(Object.keys(m)).not.toContain("reference_id");
    expect(Object.keys(m)).not.toContain("metadata");
    expect(Object.keys(m)).not.toContain("success_url");

    const withCard = { ...f, methods: [{ method: "card" as const }] };
    const fee: MethodFee[] = [{ state: "loading" }];
    expect(buildRenderModel(withCard, { link: link({ status: "paused" }), serverTotal: null, fees: fee, merchantName: null }).unavailable_reason).toBe("paused");
    expect(
      buildRenderModel(withCard, { link: link({ status: "active", uses_count: 1 }), serverTotal: null, fees: fee, merchantName: null }).unavailable_reason
    ).toBe("use_limit_reached");
    expect(
      buildRenderModel({ ...withCard, expires_at: "2026-10-01T10:00" }, { link: null, serverTotal: null, fees: fee, merchantName: null, now: new Date("2026-10-07T00:00:00Z") })
        .unavailable_reason
    ).toBe("expired");
  });

  it("takes the line-item total from the server, never sums in the browser", () => {
    const f = form({ currency: "USD", amount_mode: "line_items", line_items: [{ name: "A", quantity: "2", unit_price: "5", tax_rate: "10" }] });
    const pending = buildRenderModel(f, { link: null, serverTotal: null, fees: [], merchantName: null });
    expect(pending.amount).toBeNull();
    expect(pending.line_items[0]).toEqual({ name: "A", quantity: 2, unit_price: "5", tax_rate: "10", subtotal: null, tax: null, total: null });
    expect(buildRenderModel(f, { link: null, serverTotal: "11.00", fees: [], merchantName: null }).amount).toBe("11.00");
  });
});
