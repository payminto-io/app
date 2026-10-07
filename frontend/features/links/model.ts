/**
 * Builder form state and its mapping to the link body (docs/API_SPECIFICATION.md 4.44).
 * Inputs hold strings while typing; conversion to the API's types happens once, here.
 * Business rules live on the server: this module only refuses what JSON cannot carry.
 */
import {
  IMMUTABLE_WHEN_PUBLISHED,
  type AmountMode,
  type FeeBearer,
  type FieldMode,
  type LinkInput,
  type LinkStatus,
  type MethodSpec,
  type PaymentLink,
  type QuestionType,
} from "@/lib/api/links";

export interface FieldForm {
  mode: FieldMode;
  prefill: string;
}

export interface LineItemForm {
  name: string;
  quantity: string;
  unit_price: string;
  tax_rate: string;
}

export interface QuestionForm {
  key: string;
  label: string;
  type: QuestionType;
  /** One option per line. */
  options: string;
  required: boolean;
  per_order: boolean;
}

export type SettlementMode = "default" | "fiat" | "crypto";

export interface LinkForm {
  title: string;
  description: string;
  amount_mode: AmountMode;
  amount: string;
  amount_min: string;
  amount_max: string;
  currency: string;
  reference_id: string;
  metadata: { key: string; value: string }[];
  category: string;
  customer: { name: FieldForm; email: FieldForm; phone: FieldForm };
  billing_required: boolean;
  shipping_required: boolean;
  multi_use: boolean;
  use_limit: string;
  questions: QuestionForm[];
  methods: MethodSpec[];
  capture_mode: "automatic" | "manual";
  three_ds_policy: "inherit" | "force";
  chain_tolerance_bps: string;
  quote_expiry_seconds: string;
  fee_bearer: FeeBearer;
  success_mode: "message" | "redirect";
  success_url: string;
  success_message: string;
  receipt_email: boolean;
  receipt_note: string;
  webhook_id: string;
  failure_retry: boolean;
  failure_message: string;
  settlement: { mode: SettlementMode; destination_id: string; chain: string; asset: string };
  hold_in_asset: boolean;
  settlement_timing: "cycle" | "immediate";
  /** `datetime-local` value in the viewer's zone, or empty. */
  expires_at: string;
  expires_after_payments: string;
  logo_url: string;
  accent_color: string;
  language: string;
  line_items: LineItemForm[];
}

/** Mirrors the server's `DefaultInput()` (backend/internal/links/port.go). */
export function defaultForm(): LinkForm {
  return {
    title: "",
    description: "",
    amount_mode: "fixed",
    amount: "",
    amount_min: "",
    amount_max: "",
    currency: "",
    reference_id: "",
    metadata: [],
    category: "",
    customer: {
      name: { mode: "optional", prefill: "" },
      email: { mode: "optional", prefill: "" },
      phone: { mode: "hidden", prefill: "" },
    },
    billing_required: false,
    shipping_required: false,
    multi_use: false,
    use_limit: "",
    questions: [],
    methods: [],
    capture_mode: "automatic",
    three_ds_policy: "inherit",
    chain_tolerance_bps: "0",
    quote_expiry_seconds: "900",
    fee_bearer: "merchant",
    success_mode: "message",
    success_url: "",
    success_message: "",
    receipt_email: false,
    receipt_note: "",
    webhook_id: "",
    failure_retry: true,
    failure_message: "",
    settlement: { mode: "default", destination_id: "", chain: "", asset: "" },
    hold_in_asset: false,
    settlement_timing: "cycle",
    expires_at: "",
    expires_after_payments: "",
    logo_url: "",
    accent_color: "",
    language: "en",
    line_items: [],
  };
}

export const emptyLineItem = (): LineItemForm => ({ name: "", quantity: "1", unit_price: "", tax_rate: "0" });
export const emptyQuestion = (): QuestionForm => ({
  key: "",
  label: "",
  type: "text",
  options: "",
  required: false,
  per_order: true,
});

const str = (v: string | null | undefined) => v ?? "";
const intStr = (v: number | null | undefined) => (v === null || v === undefined ? "" : String(v));

/** ISO instant to a `datetime-local` value in the viewer's zone. */
export function toLocalInput(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function formFromLink(l: PaymentLink): LinkForm {
  const field = (r: { mode: FieldMode; prefill?: string }): FieldForm => ({ mode: r.mode, prefill: str(r.prefill) });
  const o = l.settlement_override;
  return {
    title: l.title,
    description: l.description,
    amount_mode: l.amount_mode,
    amount: l.amount_mode === "fixed" ? str(l.amount) : "",
    amount_min: str(l.amount_min),
    amount_max: str(l.amount_max),
    currency: l.currency,
    reference_id: l.reference_id,
    metadata: Object.entries(l.metadata ?? {}).map(([key, value]) => ({ key, value })),
    category: l.category,
    customer: {
      name: field(l.customer_field_policy.name),
      email: field(l.customer_field_policy.email),
      phone: field(l.customer_field_policy.phone),
    },
    billing_required: l.billing_required,
    shipping_required: l.shipping_required,
    multi_use: l.multi_use,
    use_limit: intStr(l.use_limit),
    questions: (l.questions ?? []).map((q) => ({
      key: q.key,
      label: q.label,
      type: q.type,
      options: (q.options ?? []).join("\n"),
      required: q.required,
      per_order: q.per_order,
    })),
    methods: (l.methods ?? []).map((m) => ({ ...m })),
    capture_mode: l.capture_mode,
    three_ds_policy: l.three_ds_policy,
    chain_tolerance_bps: intStr(l.chain_tolerance_bps),
    quote_expiry_seconds: intStr(l.quote_expiry_seconds),
    fee_bearer: l.fee_bearer,
    success_mode: l.success_mode,
    success_url: l.success_url,
    success_message: l.success_message,
    receipt_email: l.receipt_email,
    receipt_note: l.receipt_note,
    webhook_id: intStr(l.webhook_id),
    failure_retry: l.failure_retry,
    failure_message: l.failure_message,
    settlement: o
      ? { mode: o.kind, destination_id: o.destination_id, chain: str(o.chain), asset: str(o.asset) }
      : { mode: "default", destination_id: "", chain: "", asset: "" },
    hold_in_asset: l.hold_in_asset,
    settlement_timing: l.settlement_timing,
    expires_at: toLocalInput(l.expires_at),
    expires_after_payments: intStr(l.expires_after_payments),
    logo_url: l.logo_url,
    accent_color: l.accent_color,
    language: l.language,
    line_items: (l.line_items ?? []).map((li) => ({
      name: li.name,
      quantity: String(li.quantity),
      unit_price: li.unit_price,
      tax_rate: li.tax_rate,
    })),
  };
}

const DECIMAL = /^\d{1,20}(\.\d{1,18})?$/;
const INTEGER = /^\d{1,9}$/;

/** True when `v` is a plain non-negative decimal the API can parse. */
export const isDecimal = (v: string) => DECIMAL.test(v.trim());

export type LocalErrors = Record<string, string>;

/**
 * Errors the browser must catch because the JSON body could not carry the value at all
 * (a word where a number goes). Everything else is the server's to judge.
 */
export function localErrors(f: LinkForm): LocalErrors {
  const out: LocalErrors = {};
  const dec = (path: string, v: string, required?: string) => {
    if (v.trim() === "") {
      if (required) out[path] = required;
    } else if (!isDecimal(v)) out[path] = LOCAL.number;
  };
  const int = (path: string, v: string, required?: string) => {
    if (v.trim() === "") {
      if (required) out[path] = required;
    } else if (!INTEGER.test(v.trim())) out[path] = LOCAL.whole;
  };
  if (f.amount_mode === "fixed") dec("amount", f.amount);
  if (f.amount_mode === "customer") {
    dec("amount_min", f.amount_min);
    dec("amount_max", f.amount_max);
  }
  if (f.amount_mode === "line_items") {
    f.line_items.forEach((li, i) => {
      int(`line_items[${i}].quantity`, li.quantity, LOCAL.quantity);
      dec(`line_items[${i}].unit_price`, li.unit_price, LOCAL.price);
      dec(`line_items[${i}].tax_rate`, li.tax_rate, LOCAL.rate);
    });
  }
  if (f.multi_use) {
    int("use_limit", f.use_limit);
    int("expires_after_payments", f.expires_after_payments);
  }
  int("chain_tolerance_bps", f.chain_tolerance_bps, LOCAL.required);
  int("quote_expiry_seconds", f.quote_expiry_seconds, LOCAL.required);
  if (f.expires_at && Number.isNaN(new Date(f.expires_at).getTime())) out.expires_at = LOCAL.date;
  return out;
}

/** Messages for values the JSON body cannot carry; everything else is the server's to judge. */
export const LOCAL = {
  number: "Enter a number",
  whole: "Enter a whole number",
  quantity: "Enter a quantity",
  price: "Enter a price",
  rate: "Enter a rate, 0 for none",
  required: "Enter a value",
  date: "Enter a date and time",
} as const;

const decOrNull = (v: string) => (v.trim() === "" ? null : v.trim());
const intOrNull = (v: string) => (v.trim() === "" ? null : Number.parseInt(v.trim(), 10));

function fieldRule(f: FieldForm) {
  const prefill = f.prefill.trim();
  return prefill ? { mode: f.mode, prefill } : { mode: f.mode };
}

/** The full link body for create and draft saves. Call only when `localErrors` is empty. */
export function formToInput(f: LinkForm): LinkInput {
  const mode = f.amount_mode;
  const metadata: Record<string, string> = {};
  for (const row of f.metadata) {
    if (row.key.trim() === "" && row.value.trim() === "") continue;
    metadata[row.key.trim()] = row.value;
  }
  const s = f.settlement;
  return {
    title: f.title.trim(),
    description: f.description,
    amount_mode: mode,
    amount: mode === "fixed" ? decOrNull(f.amount) : null,
    amount_min: mode === "customer" ? decOrNull(f.amount_min) : null,
    amount_max: mode === "customer" ? decOrNull(f.amount_max) : null,
    currency: f.currency.trim().toUpperCase(),
    reference_id: f.reference_id.trim(),
    metadata,
    category: f.category.trim(),
    customer_field_policy: {
      name: fieldRule(f.customer.name),
      email: fieldRule(f.customer.email),
      phone: fieldRule(f.customer.phone),
    },
    billing_required: f.billing_required,
    shipping_required: f.shipping_required,
    multi_use: f.multi_use,
    use_limit: f.multi_use ? intOrNull(f.use_limit) : null,
    methods: f.methods.map((m) =>
      m.method === "crypto"
        ? { method: "crypto", chain: (m.chain ?? "").toUpperCase(), asset: (m.asset ?? "").toUpperCase() }
        : { method: m.method }
    ),
    capture_mode: f.capture_mode,
    three_ds_policy: f.three_ds_policy,
    chain_tolerance_bps: Number.parseInt(f.chain_tolerance_bps.trim(), 10),
    quote_expiry_seconds: Number.parseInt(f.quote_expiry_seconds.trim(), 10),
    fee_bearer: f.fee_bearer,
    success_mode: f.success_mode,
    success_url: f.success_mode === "redirect" ? f.success_url.trim() : "",
    success_message: f.success_message,
    receipt_email: f.receipt_email,
    receipt_note: f.receipt_note,
    webhook_id: intOrNull(f.webhook_id),
    failure_retry: f.failure_retry,
    failure_message: f.failure_message,
    settlement_override:
      s.mode === "default"
        ? null
        : s.mode === "crypto"
          ? { kind: "crypto", destination_id: s.destination_id.trim(), chain: s.chain.trim().toUpperCase(), asset: s.asset.trim().toUpperCase() }
          : { kind: "fiat", destination_id: s.destination_id.trim() },
    hold_in_asset: f.hold_in_asset,
    settlement_timing: f.settlement_timing,
    expires_at: f.expires_at ? new Date(f.expires_at).toISOString() : null,
    expires_after_payments: f.multi_use ? intOrNull(f.expires_after_payments) : null,
    logo_url: f.logo_url.trim(),
    accent_color: f.accent_color.trim(),
    language: f.language.trim(),
    line_items:
      mode === "line_items"
        ? f.line_items.map((li) => ({
            name: li.name.trim(),
            quantity: Number.parseInt(li.quantity.trim(), 10),
            unit_price: li.unit_price.trim(),
            tax_rate: li.tax_rate.trim(),
          }))
        : [],
    questions: f.questions.map((q) => {
      const base = { key: q.key.trim(), label: q.label.trim(), type: q.type, required: q.required, per_order: q.per_order };
      if (q.type !== "select") return base;
      return { ...base, options: q.options.split("\n").map((o) => o.trim()).filter(Boolean) };
    }),
  };
}

export const isPublished = (s: LinkStatus) => s === "active" || s === "paused";

/** The PATCH body: everything for a draft; a published link leaves out the keys it may not change. */
export function formToPatch(f: LinkForm, status: LinkStatus): Partial<LinkInput> {
  const input: Partial<LinkInput> = formToInput(f);
  if (isPublished(status)) {
    for (const k of IMMUTABLE_WHEN_PUBLISHED) delete input[k];
  }
  return input;
}

/** Stable fingerprint of what a save would send; equal fingerprints mean nothing to save. */
export const fingerprint = (f: LinkForm) => JSON.stringify(formToInput(f));

/** True when the form has something a merchant typed, so a new draft is worth creating. */
export function hasContent(f: LinkForm): boolean {
  return fingerprint(f) !== fingerprint(defaultForm());
}

/** Everything the server prices a method on; when it matches the saved link, the link's `fee_preview` applies. */
export function pricingFingerprint(f: LinkForm): string {
  const i = formToInput(f);
  return JSON.stringify([i.amount_mode, i.amount, i.amount_min, i.currency, i.fee_bearer, i.methods, i.line_items]);
}

/** Stable identity of a method across lists: fees and refusals are matched by it, never by position. */
export function methodKey(m: { method: string; chain?: string | null; asset?: string | null }): string {
  return m.method === "crypto" ? `crypto:${(m.asset ?? "").toUpperCase()}@${(m.chain ?? "").toUpperCase()}` : m.method;
}

/** The most payments a link may take (backend `EffectiveUseLimit`); null is unlimited. */
export function effectiveUseLimit(l: Pick<LinkInput, "multi_use" | "use_limit" | "expires_after_payments">): number | null {
  if (!l.multi_use) return 1;
  const caps = [l.use_limit, l.expires_after_payments].filter((v): v is number => v !== null);
  return caps.length ? Math.min(...caps) : null;
}
