import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import type { RenderModel } from "@/lib/api/links";
import { defaultForm, type LinkForm } from "./model";
import { PaymentStep } from "./components/steps";
import { CheckoutPreview } from "./components/checkout-preview";
import { splitLinkUrl } from "./components/link-share";

const form = (patch: Partial<LinkForm> = {}): LinkForm => ({ ...defaultForm(), ...patch });

function model(patch: Partial<RenderModel> = {}): RenderModel {
  return {
    short_code: "",
    url: "",
    available: true,
    unavailable_reason: null,
    merchant_name: "Acme",
    title: "Tip jar",
    description: null,
    amount_mode: "fixed",
    amount: "40",
    amount_min: null,
    amount_max: null,
    currency: "USD",
    line_items: [],
    subtotal: null,
    tax_total: null,
    customer_fields: { name: { mode: "optional", prefill: null }, email: { mode: "optional", prefill: null }, phone: { mode: "hidden", prefill: null } },
    billing_required: false,
    shipping_required: false,
    questions: [],
    methods: [{ method: "card", chain: null, asset: null, fee: null, tax: null, customer_total: null }],
    fee_bearer: "merchant",
    chain_tolerance_bps: 0,
    quote_expiry_seconds: 900,
    success_mode: "message",
    success_message: null,
    failure_retry: true,
    failure_message: null,
    receipt_email: false,
    expires_at: null,
    branding: { logo_url: null, accent_color: null, language: "en" },
    ...patch,
  };
}

describe("live link payment step", () => {
  it("shows the fee bearer as read-only text, not a control", () => {
    render(
      <PaymentStep
        form={form({ currency: "USD", amount: "40", fee_bearer: "customer", methods: [{ method: "card" }] })}
        edit={vi.fn()}
        err={() => undefined}
        locked
        options={undefined}
        pricing={null}
        dropped={new Map()}
      />
    );
    expect(screen.queryByRole("radiogroup", { name: "Fees paid by" })).toBeNull();
    expect(screen.getByText("Fees paid by").nextElementSibling).toHaveTextContent("Customer");
    expect(screen.queryByRole("checkbox")).toBeNull();
  });

  it("offers the fee bearer as a choice on a draft", () => {
    render(<PaymentStep form={form()} edit={vi.fn()} err={() => undefined} locked={false} options={undefined} pricing={null} dropped={new Map()} />);
    expect(screen.getByRole("radiogroup", { name: "Fees paid by" })).toBeInTheDocument();
  });
});

describe("checkout preview", () => {
  it("shows no total for a customer-entered amount", () => {
    render(<CheckoutPreview state="ready" model={model({ amount_mode: "customer", amount: null, amount_min: "10", amount_max: "100" })} />);
    expect(screen.queryByText("Total due")).toBeNull();
    expect(screen.getByText("10.00 USD to 100.00 USD")).toBeInTheDocument();
    expect(screen.getByText("Pay").textContent).toBe("Pay");
  });

  it("never shows an amount without its currency", () => {
    render(<CheckoutPreview state="ready" model={model({ currency: "" })} />);
    expect(screen.getByText("Add a currency")).toBeInTheDocument();
    expect(screen.queryByText("40.00")).toBeNull();
  });

  it("shows server line totals once, with the surcharge for the chosen method", () => {
    render(
      <CheckoutPreview
        state="ready"
        model={model({
          amount_mode: "line_items",
          amount: "11.00",
          subtotal: "10.00",
          tax_total: "1.00",
          line_items: [{ name: "Mug", quantity: 2, unit_price: "5", tax_rate: "10", subtotal: "10.00", tax: "1.00", total: "11.00" }],
          fee_bearer: "customer",
          methods: [{ method: "card", chain: null, asset: null, fee: "0.62", tax: "0", customer_total: "11.62" }],
        })}
      />
    );
    const due = screen.getByText("Total due").closest("div") as HTMLElement;
    expect(within(due).getByText("11.62")).toBeInTheDocument();
    expect(screen.getAllByText("Total due")).toHaveLength(1);
  });

  it("keeps numbers off screen while paused on a local error", () => {
    render(<CheckoutPreview state="paused" model={model()} />);
    expect(screen.queryByText("40.00")).toBeNull();
  });
});

describe("short link", () => {
  it("keeps the short code whole and lets the host truncate", () => {
    expect(splitLinkUrl("https://pay.example.com/l/smplworkshop")).toEqual({ head: "pay.example.com/l/", code: "smplworkshop" });
  });
});
