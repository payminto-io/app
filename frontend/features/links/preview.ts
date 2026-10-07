/**
 * Builds the checkout preview from form state in the public render-model shape
 * (backend/internal/links/render.go), so the builder shows what checkout will.
 * Fees come only from `POST /fees/preview`; line totals only from the server's saved `total`.
 */
import type { FeePreview, FeePreviewRequest } from "@/lib/api/fees";
import type { LinkStatus, MethodSpec, PaymentLink, RenderField, RenderMethod, RenderModel } from "@/lib/api/links";
import { formToInput, isDecimal, type FieldForm, type LinkForm } from "./model";

export type MethodFee =
  | { state: "loading" }
  | { state: "ok"; preview: FeePreview }
  /** No amount to price yet (empty, unsaved, or customer-entered without a minimum). */
  | { state: "no_amount" }
  /** A crypto asset priced against a different link currency: priced at pay time with a quote. */
  | { state: "at_pay_time" }
  | { state: "refused"; code: string; message: string }
  | { state: "error"; message: string };

/** Codes after which the server leaves a method out of the render model. */
const DROPS_METHOD = new Set(["no_fee_rule", "surcharge_forbidden"]);

export function feeCurrency(m: MethodSpec, linkCurrency: string): string {
  return m.method === "crypto" ? (m.asset ?? "").toUpperCase() : linkCurrency;
}

/**
 * The amount the server would price this link on: the fixed amount, the saved line-item
 * total, or the customer minimum (backend `representative`). Null when there is none yet.
 */
export function pricingAmount(form: LinkForm, serverTotal: string | null): string | null {
  switch (form.amount_mode) {
    case "fixed":
      return isDecimal(form.amount) && Number(form.amount) > 0 ? form.amount.trim() : null;
    case "line_items":
      return serverTotal;
    case "customer":
      return isDecimal(form.amount_min) && Number(form.amount_min) > 0 ? form.amount_min.trim() : null;
  }
}

/** The fee preview request for one method, or the state to show when there is nothing to ask. */
export function feeRequestFor(
  form: LinkForm,
  m: MethodSpec,
  serverTotal: string | null
): { request: FeePreviewRequest } | { state: MethodFee } {
  const currency = form.currency.trim().toUpperCase();
  if (!currency) return { state: { state: "no_amount" } };
  if (m.method === "crypto" && (!m.asset || !m.chain)) return { state: { state: "no_amount" } };
  const fc = feeCurrency(m, currency);
  if (fc !== currency) return { state: { state: "at_pay_time" } };
  const amount = pricingAmount(form, serverTotal);
  if (!amount) return { state: { state: "no_amount" } };
  return { request: { amount, currency, method: m.method, fee_bearer: form.fee_bearer } };
}

function renderField(f: FieldForm): RenderField {
  const prefill = f.prefill.trim();
  return { mode: f.mode, prefill: f.mode === "hidden" || !prefill ? null : prefill };
}

const orNull = (s: string) => (s.trim() === "" ? null : s);

function unavailable(status: LinkStatus | null, input: ReturnType<typeof formToInput>, uses: number, now: Date) {
  if (status === "paused") return "paused" as const;
  if (input.expires_at && new Date(input.expires_at) <= now) return "expired" as const;
  const limit = !input.multi_use
    ? 1
    : [input.use_limit, input.expires_after_payments].filter((v): v is number => v !== null).sort((a, b) => a - b)[0];
  if (limit !== undefined && uses >= limit && status !== null && status !== "draft") return "use_limit_reached" as const;
  return null;
}

export interface PreviewContext {
  link: PaymentLink | null;
  /** The server's `total`, only while it matches the form's pricing fields. */
  serverTotal: string | null;
  /** One entry per `form.methods`, same order. */
  fees: MethodFee[];
  merchantName: string | null;
  now?: Date;
}

export function buildRenderModel(form: LinkForm, ctx: PreviewContext): RenderModel {
  const input = formToInput(form);
  const now = ctx.now ?? new Date();
  const methods: RenderMethod[] = [];
  input.methods.forEach((m, i) => {
    const fee = ctx.fees[i];
    if (fee?.state === "refused" && DROPS_METHOD.has(fee.code)) return;
    const surcharge = input.fee_bearer === "customer" && fee?.state === "ok" ? fee.preview : null;
    methods.push({
      method: m.method,
      chain: m.chain ? m.chain : null,
      asset: m.asset ? m.asset : null,
      fee: surcharge?.fee ?? null,
      tax: surcharge?.tax ?? null,
      customer_total: surcharge?.customer_total ?? null,
    });
  });

  let reason: RenderModel["unavailable_reason"] = unavailable(ctx.link?.status ?? null, input, ctx.link?.uses_count ?? 0, now);
  if (!reason && methods.length === 0) reason = "no_methods_available";

  const isItems = input.amount_mode === "line_items";
  return {
    short_code: ctx.link?.short_code ?? "",
    url: ctx.link?.url ?? "",
    available: reason === null,
    unavailable_reason: reason,
    merchant_name: ctx.merchantName,
    title: input.title,
    description: orNull(input.description),
    amount_mode: input.amount_mode,
    amount: input.amount_mode === "fixed" ? input.amount : isItems ? ctx.serverTotal : null,
    amount_min: input.amount_min,
    amount_max: input.amount_max,
    currency: input.currency,
    line_items: isItems
      ? input.line_items.map((li) => ({ ...li, subtotal: null, tax: null, total: null }))
      : [],
    subtotal: null,
    tax_total: null,
    customer_fields: {
      name: renderField(form.customer.name),
      email: renderField(form.customer.email),
      phone: renderField(form.customer.phone),
    },
    billing_required: input.billing_required,
    shipping_required: input.shipping_required,
    questions: input.questions.map((q) => ({ ...q, options: q.options ?? [] })),
    methods,
    fee_bearer: input.fee_bearer,
    chain_tolerance_bps: input.chain_tolerance_bps,
    quote_expiry_seconds: input.quote_expiry_seconds,
    success_mode: input.success_mode,
    success_message: orNull(input.success_message),
    failure_retry: input.failure_retry,
    failure_message: orNull(input.failure_message),
    receipt_email: input.receipt_email,
    expires_at: input.expires_at,
    branding: {
      logo_url: orNull(input.logo_url),
      accent_color: /^#[0-9a-fA-F]{6}$/.test(input.accent_color) ? input.accent_color : null,
      language: input.language,
    },
  };
}
