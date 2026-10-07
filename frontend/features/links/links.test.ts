import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api/errors";
import type { MethodPreview, PaymentLink } from "@/lib/api/links";
import {
  defaultForm,
  formFromLink,
  formToInput,
  formToPatch,
  hasContent,
  effectiveUseLimit,
  localErrors,
  methodKey,
  pricingFingerprint,
  type LinkForm,
} from "./model";
import { countByStep, errorAt, mapApiErrors, stepForField } from "./errors";
import { droppedByKey, savedPricing } from "./pricing";
import { feeView } from "./components/steps";

const form = (patch: Partial<LinkForm> = {}): LinkForm => ({ ...defaultForm(), ...patch });

function link(patch: Partial<PaymentLink> = {}): PaymentLink {
  return {
    ...formToInput(form({ title: "Workshop", currency: "USD", amount: "50.00", methods: [{ method: "card" }] })),
    id: "lnk_1",
    merchant_name: "Acme",
    fee_preview: null,
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

  it("flags values JSON cannot carry and blanks that would otherwise be invented", () => {
    const errs = localErrors(
      form({ amount: "ten", multi_use: true, use_limit: "1.5", amount_mode: "fixed", line_items: [{ name: "", quantity: "x", unit_price: "1", tax_rate: "0" }] })
    );
    expect(errs).toEqual({ amount: "Enter a number", use_limit: "Enter a whole number" });
    expect(localErrors(form({ amount_mode: "line_items", line_items: [{ name: "", quantity: "x", unit_price: "-1", tax_rate: "0" }] }))).toEqual({
      "line_items[0].quantity": "Enter a whole number",
      "line_items[0].unit_price": "Enter a number",
    });
    expect(localErrors(form({ amount_mode: "line_items", line_items: [{ name: "A", quantity: "", unit_price: "", tax_rate: "" }] }))).toEqual({
      "line_items[0].quantity": "Enter a quantity",
      "line_items[0].unit_price": "Enter a price",
      "line_items[0].tax_rate": "Enter a rate, 0 for none",
    });
    expect(localErrors(form({ chain_tolerance_bps: "", quote_expiry_seconds: " " }))).toEqual({
      chain_tolerance_bps: "Enter a value",
      quote_expiry_seconds: "Enter a value",
    });
    expect(localErrors(form({ amount: "" }))).toEqual({});
  });

  it("leaves immutable keys out of a published link's patch", () => {
    const f = form({ title: "T", amount: "5", currency: "USD", methods: [{ method: "card" }] });
    expect(formToPatch(f, "draft")).toHaveProperty("amount", "5");
    const p = formToPatch(f, "active");
    for (const k of ["amount_mode", "amount", "amount_min", "amount_max", "currency", "methods", "line_items", "fee_bearer"]) {
      expect(p).not.toHaveProperty(k);
    }
    expect(p).toHaveProperty("title", "T");
  });

  it("pricing fingerprint moves only with pricing fields", () => {
    const b = form({ amount: "5", currency: "USD" });
    expect(pricingFingerprint({ ...b, fee_bearer: "customer" })).not.toBe(pricingFingerprint(b));
    expect(pricingFingerprint({ ...b, methods: [{ method: "card" }] })).not.toBe(pricingFingerprint(b));
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

describe("server pricing matched by method", () => {
  const mp = (patch: Partial<MethodPreview>): MethodPreview => ({
    method: "card",
    chain: null,
    asset: null,
    connector: "stripe",
    rule_id: 4,
    rule_version: 2,
    fee_bearer: "merchant",
    fee_currency: "USD",
    amount: "50.00",
    fee: "1.75",
    tax: "0.00",
    customer_total: "50.00",
    merchant_net: "48.25",
    unavailable: null,
    ...patch,
  });

  it("keys methods by identity, not position", () => {
    expect(methodKey({ method: "crypto", chain: "sol", asset: "usdc" })).toBe("crypto:USDC@SOL");
    expect(methodKey({ method: "upi", chain: null, asset: null })).toBe("upi");
  });

  it("uses the saved link's fee preview only while the form prices the same", () => {
    const usdc = mp({ method: "crypto", chain: "SOL", asset: "USDC", fee_currency: "USDC", fee: null, rule_id: 7, rule_version: 1 });
    const l = link({ methods: [{ method: "card" }, { method: "crypto", chain: "SOL", asset: "USDC" }], fee_preview: [mp({}), usdc] });
    const f = formFromLink(l);
    const priced = savedPricing(l, f);
    expect(priced?.get("card")?.fee).toBe("1.75");
    expect(priced?.get("crypto:USDC@SOL")?.rule_id).toBe(7);
    // Reordering methods in the form changes no pairing; changing the amount drops the stale numbers.
    expect(savedPricing(l, { ...f, methods: [...f.methods].reverse() })).toBeNull();
    expect(savedPricing(l, { ...f, amount: "60.00" })).toBeNull();
    expect(savedPricing(l, { ...f, fee_bearer: "customer" })).toBeNull();
    expect(savedPricing(link({ fee_preview: null }), f)).toBeNull();
  });

  it("shows refusals from the checkout render first, then the saved pricing", () => {
    const l = link({
      methods: [{ method: "card" }, { method: "crypto", chain: "SOL", asset: "USDC" }],
      fee_preview: [mp({}), mp({ method: "crypto", chain: "SOL", asset: "USDC", fee_currency: "USDC", fee: null, customer_total: null, merchant_net: null })],
    });
    const f = formFromLink(l);
    const pricing = savedPricing(l, f);
    const dropped = droppedByKey([{ method: "crypto", chain: "SOL", asset: "USDC", code: "surcharge_needs_quote", message: "needs a quote" }]);
    expect(feeView({ method: "crypto", chain: "SOL", asset: "USDC" }, "USD", pricing, dropped)).toEqual({ kind: "refused", code: "surcharge_needs_quote", message: "needs a quote" });
    expect(feeView({ method: "crypto", chain: "SOL", asset: "USDC" }, "USD", pricing, droppedByKey([]))).toEqual({ kind: "pay_time" });
    expect(feeView({ method: "card" }, "USD", pricing, dropped)).toMatchObject({ kind: "priced" });
    expect(feeView({ method: "card" }, "USD", null, dropped)).toEqual({ kind: "pending" });
    const refused = savedPricing(link({ fee_preview: [mp({ unavailable: "method_no_fee_rule", fee: null })] }), formFromLink(link()));
    expect(feeView({ method: "card" }, "USD", refused, droppedByKey([]))).toMatchObject({ kind: "refused", code: "method_no_fee_rule" });
  });

  it("computes the use limit in one place", () => {
    expect(effectiveUseLimit({ multi_use: false, use_limit: 9, expires_after_payments: null })).toBe(1);
    expect(effectiveUseLimit({ multi_use: true, use_limit: 9, expires_after_payments: 4 })).toBe(4);
    expect(effectiveUseLimit({ multi_use: true, use_limit: null, expires_after_payments: null })).toBeNull();
  });
});
