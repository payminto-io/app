/**
 * Sample payment-link fixtures for /design/preview only. Every value is invented for layout
 * review; the page labels it as sample. A small in-memory stand-in for the v2 link routes
 * (including preview and options) so the builder can be driven without a backend. Not product logic:
 * the real rules live in backend/internal/links.
 */
import { defaultForm, effectiveUseLimit, formToInput, methodKey } from "@/features/links/model";
import type { DroppedMethod, LinkInput, LinkOptions, MethodPreview, MethodSpec, PaymentLink, RenderModel } from "@/lib/api/links";

type Reply = { status: number; body: unknown };

const NOW = Date.parse("2026-10-07T10:00:00Z");
const iso = (days: number) => new Date(NOW + days * 86_400_000).toISOString();
const BASE = "https://pay.example.com/l/";
const MERCHANT = "Sample Studio";
const cents = (v: string | number) => Math.round(Number(v) * 100);
const fx = (c: number) => (c / 100).toFixed(2);

const OPTIONS: LinkOptions = {
  environment: "test",
  currencies: ["EUR", "INR", "USD"],
  methods: [
    { method: "card", chain: null, asset: null, currencies: ["EUR", "INR", "USD"] },
    { method: "upi", chain: null, asset: null, currencies: ["INR"] },
    { method: "bank", chain: null, asset: null, currencies: ["USD"] },
    { method: "crypto", chain: "SOL", asset: "USDC", currencies: ["USD"] },
  ],
};

function lines(items: LinkInput["line_items"]) {
  const rows = items.map((li) => {
    const sub = li.quantity * cents(li.unit_price);
    const tax = Math.round((sub * Number(li.tax_rate)) / 100);
    return { row: { ...li, subtotal: fx(sub), tax: fx(tax), total: fx(sub + tax) }, sub, tax };
  });
  const sub = rows.reduce((a, r) => a + r.sub, 0);
  const tax = rows.reduce((a, r) => a + r.tax, 0);
  return { rows: rows.map((r) => r.row), subtotal: fx(sub), tax: fx(tax), total: fx(sub + tax) };
}

const totalOf = (i: LinkInput) =>
  i.amount_mode === "fixed" ? i.amount : i.amount_mode === "line_items" && i.line_items.length ? lines(i.line_items).total : null;

/** Sample rules: card 2.9% + 0.30, UPI free and never surcharged, bank none, USDC 1% in USDC. */
function price(i: LinkInput, m: MethodSpec): { code?: string; rule?: [number, number]; b?: { fee: string; tax: string; total: string; net: string } } {
  const offered = OPTIONS.methods.find((o) => methodKey(o) === methodKey(m));
  if (!offered || !offered.currencies.includes(i.currency)) return { code: "method_no_connector" };
  if (m.method === "bank") return { code: "method_no_fee_rule" };
  const feeCur = m.method === "crypto" ? m.asset : i.currency;
  const rule: [number, number] = m.method === "card" ? [3, 2] : m.method === "upi" ? [5, 1] : [7, 1];
  if (feeCur !== i.currency) return i.fee_bearer === "customer" ? { code: "surcharge_needs_quote" } : { rule };
  if (m.method === "upi" && i.fee_bearer === "customer") return { code: "surcharge_forbidden" };
  const total = totalOf(i);
  const amount = total ?? i.amount_min;
  if (!amount) return { rule };
  const a = cents(amount);
  const fee = m.method === "card" ? Math.round(a * 0.029) + 30 : 0;
  const tax = i.currency === "INR" ? Math.round(fee * 0.18) : 0;
  if (fee + tax > a) return total ? { code: "fee_exceeds_amount" } : { rule };
  const customer = i.fee_bearer === "customer";
  return { rule, b: { fee: fx(fee), tax: fx(tax), total: fx(customer ? a + fee + tax : a), net: fx(customer ? a : a - fee - tax) } };
}

function feePreview(l: PaymentLink): MethodPreview[] {
  if (!l.currency) return [];
  return l.methods.map((m) => {
    const p = price(l, m);
    return {
      method: m.method,
      chain: m.chain ?? null,
      asset: m.asset ?? null,
      connector: p.code ? null : "sample",
      rule_id: p.rule?.[0] ?? null,
      rule_version: p.rule?.[1] ?? null,
      fee_bearer: l.fee_bearer,
      fee_currency: m.method === "crypto" ? (m.asset ?? "") : l.currency,
      amount: p.b ? (totalOf(l) ?? l.amount_min) : null,
      fee: p.b?.fee ?? null,
      tax: p.b?.tax ?? null,
      customer_total: p.b?.total ?? null,
      merchant_net: p.b?.net ?? null,
      unavailable: p.code ?? null,
    };
  });
}

function render(i: LinkInput, l: Pick<PaymentLink, "status" | "uses_count" | "short_code" | "url"> | null): { model: RenderModel; dropped_methods: DroppedMethod[] } {
  const total = totalOf(i);
  const dropped: DroppedMethod[] = [];
  const methods: RenderModel["methods"] = [];
  for (const m of i.methods) {
    const p = price(i, m);
    if (p.code) {
      dropped.push({ method: m.method, chain: m.chain ?? null, asset: m.asset ?? null, code: p.code, message: p.code.replace(/_/g, " ") });
      continue;
    }
    const s = total && i.fee_bearer === "customer" ? p.b : undefined;
    methods.push({ method: m.method, chain: m.chain ?? null, asset: m.asset ?? null, fee: s?.fee ?? null, tax: s?.tax ?? null, customer_total: s?.total ?? null });
  }
  const limit = effectiveUseLimit(i);
  let reason: RenderModel["unavailable_reason"] = null;
  if (l?.status === "paused") reason = "paused";
  else if (i.expires_at && Date.parse(i.expires_at) <= Date.now()) reason = "expired";
  else if (limit !== null && (l?.uses_count ?? 0) >= limit) reason = "use_limit_reached";
  else if (methods.length === 0) reason = "no_methods_available";
  const items = i.amount_mode === "line_items" ? lines(i.line_items) : null;
  const field = (f: LinkInput["customer_field_policy"]["name"]) => ({ mode: f.mode, prefill: f.mode === "hidden" ? null : (f.prefill ?? null) });
  return {
    dropped_methods: dropped,
    model: {
      short_code: l?.short_code ?? "",
      url: l?.url ?? "",
      available: reason === null,
      unavailable_reason: reason,
      merchant_name: MERCHANT,
      title: i.title,
      description: i.description || null,
      amount_mode: i.amount_mode,
      amount: total,
      amount_min: i.amount_min,
      amount_max: i.amount_max,
      currency: i.currency,
      line_items: items?.rows ?? [],
      subtotal: items?.subtotal ?? null,
      tax_total: items?.tax ?? null,
      customer_fields: { name: field(i.customer_field_policy.name), email: field(i.customer_field_policy.email), phone: field(i.customer_field_policy.phone) },
      billing_required: i.billing_required,
      shipping_required: i.shipping_required,
      questions: i.questions.map((q) => ({ ...q, options: q.options ?? [] })),
      methods,
      fee_bearer: i.fee_bearer,
      chain_tolerance_bps: i.chain_tolerance_bps,
      quote_expiry_seconds: i.quote_expiry_seconds,
      success_mode: i.success_mode,
      success_message: i.success_message || null,
      failure_retry: i.failure_retry,
      failure_message: i.failure_message || null,
      receipt_email: i.receipt_email,
      expires_at: i.expires_at,
      branding: { logo_url: i.logo_url || null, accent_color: i.accent_color || null, language: i.language },
    },
  };
}

function make(id: string, input: Partial<LinkInput>, extra: Partial<PaymentLink>): PaymentLink {
  const full = { ...formToInput(defaultForm()), ...input };
  const code = extra.status && extra.status !== "draft" ? `smpl${id.slice(-8)}` : null;
  return {
    ...full,
    id,
    merchant_name: MERCHANT,
    fee_preview: null,
    status: "draft",
    environment: "test",
    short_code: code,
    url: code ? BASE + code : null,
    total: totalOf(full),
    uses_count: 0,
    revision: 1,
    published_at: null,
    created_at: iso(-3),
    updated_at: iso(-1),
    ...extra,
  };
}

const SEED: PaymentLink[] = [
  make(
    "sample_lnk_0000workshop",
    {
      title: "Design workshop seat",
      description: "Two-hour hands-on session. Laptop required.",
      amount: "120.00",
      currency: "USD",
      methods: [{ method: "card" }, { method: "crypto", chain: "SOL", asset: "USDC" }],
      multi_use: true,
      use_limit: 40,
      customer_field_policy: { name: { mode: "required" }, email: { mode: "required" }, phone: { mode: "hidden" } },
      success_message: "You're in. Details are on their way.",
      receipt_email: true,
      accent_color: "#0B7285",
    },
    { status: "active", uses_count: 12, published_at: iso(-2), created_at: iso(-5) }
  ),
  make(
    "sample_lnk_00consulting",
    {
      title: "Consulting hour",
      amount_mode: "customer",
      amount: null,
      amount_min: "50.00",
      amount_max: "500.00",
      currency: "USD",
      methods: [{ method: "card" }],
      multi_use: true,
      fee_bearer: "customer",
    },
    { status: "active", uses_count: 3, published_at: iso(-6), created_at: iso(-8) }
  ),
  make(
    "sample_lnk_00000bundle",
    {
      title: "Conference bundle",
      amount_mode: "line_items",
      amount: null,
      currency: "INR",
      line_items: [
        { name: "Conference pass", quantity: 1, unit_price: "4999.00", tax_rate: "18" },
        { name: "Workshop add-on", quantity: 2, unit_price: "1500.00", tax_rate: "18" },
      ],
      methods: [{ method: "card" }, { method: "upi" }],
      questions: [{ key: "tshirt", label: "T-shirt size", type: "select", options: ["S", "M", "L"], required: true, per_order: true }],
    },
    { created_at: iso(-1) }
  ),
  make(
    "sample_lnk_00springsale",
    { title: "Spring sale deposit", amount: "25.00", currency: "EUR", methods: [{ method: "card" }], multi_use: true, use_limit: 100 },
    { status: "paused", uses_count: 7, published_at: iso(-30), created_at: iso(-31) }
  ),
  make(
    "sample_lnk_0000webinar",
    { title: "Webinar replay", amount: "9.00", currency: "USD", methods: [{ method: "card" }], multi_use: true },
    { status: "archived", uses_count: 41, published_at: iso(-60), created_at: iso(-61) }
  ),
];

let store: PaymentLink[] | null = null;
const db = () => (store ??= SEED.map((l) => structuredClone(l)));
let seq = 1;

const err = (status: number, code: string, message: string, field?: string): Reply => ({ status, body: { error: message, code, ...(field ? { field } : {}) } });

function errs(list: { code: string; field?: string; message: string }[]): Reply {
  return { status: 422, body: { error: list[0].message, code: list[0].code, field: list[0].field, errors: list.length > 1 ? list : undefined } };
}

function saveChecks(i: LinkInput): Reply | null {
  const list: { code: string; field?: string; message: string }[] = [];
  if (i.title.length > 200) list.push({ code: "title_too_long", field: "title", message: "at most 200 characters" });
  if (i.accent_color && !/^#[0-9a-fA-F]{6}$/.test(i.accent_color)) list.push({ code: "accent_color_invalid", field: "accent_color", message: "must be #RRGGBB" });
  if (i.logo_url && !i.logo_url.startsWith("https://")) list.push({ code: "logo_url_invalid", field: "logo_url", message: "must be an absolute https URL" });
  i.line_items.forEach((li, n) => {
    if (!li.name) list.push({ code: "line_item_invalid", field: `line_items[${n}].name`, message: "is required, at most 200 characters" });
    else if (li.quantity < 1) list.push({ code: "line_item_invalid", field: `line_items[${n}].quantity`, message: "must be 1-100000" });
  });
  return list.length ? errs(list) : null;
}

function publishChecks(l: PaymentLink): Reply | null {
  const list: { code: string; field?: string; message: string }[] = [];
  if (!l.title) list.push({ code: "title_required", field: "title", message: "is required" });
  if (!l.currency) list.push({ code: "currency_required", field: "currency", message: "is required" });
  if (l.amount_mode === "fixed" && !l.amount) list.push({ code: "amount_required", field: "amount", message: "is required for a fixed amount" });
  if (l.amount_mode === "line_items" && l.line_items.length === 0) list.push({ code: "line_items_required", field: "line_items", message: "add at least one line item" });
  if (l.methods.length === 0) list.push({ code: "methods_required", field: "methods", message: "enable at least one method" });
  l.methods.forEach((m, n) => {
    const code = price(l, m).code;
    if (code) list.push({ code, field: `methods[${n}]`, message: code.replace(/_/g, " ") });
  });
  return list.length ? errs(list) : null;
}

const detail = (l: PaymentLink): PaymentLink => ({ ...l, total: totalOf(l), fee_preview: feePreview(l) });

export function resolveLinksV2(pathWithQuery: string, method: string, body: string | undefined, empty: boolean): Reply | undefined {
  const [path, query = ""] = pathWithQuery.split("?");
  const q = new URLSearchParams(query);
  const rows = db();
  if (path === "/links" && method === "GET") {
    if (empty) return { status: 200, body: { links: [], total: 0 } };
    const status = q.get("status");
    const list = rows.filter((l) => !status || l.status === status).sort((a, b) => b.created_at.localeCompare(a.created_at));
    return { status: 200, body: { links: list.map((l) => ({ ...l, fee_preview: null })), total: list.length } };
  }
  if (path === "/links/options" && method === "GET") return { status: 200, body: empty ? { ...OPTIONS, currencies: [], methods: [] } : OPTIONS };
  if (path === "/links/preview" && method === "POST") {
    const input = { ...formToInput(defaultForm()), ...JSON.parse(body ?? "{}") } as LinkInput;
    const bad = saveChecks(input);
    if (bad) return bad;
    const link = rows.find((l) => l.id === q.get("link_id")) ?? null;
    return { status: 200, body: render(input, link) };
  }
  if (path === "/links" && method === "POST") {
    const input = { ...formToInput(defaultForm()), ...JSON.parse(body ?? "{}") } as LinkInput;
    const bad = saveChecks(input);
    if (bad) return bad;
    const l = make(`sample_lnk_new${String(seq++).padStart(9, "0")}`, input, { created_at: new Date().toISOString() });
    rows.push(l);
    return { status: 201, body: detail(l) };
  }
  const m = path.match(/^\/links\/([^/]+)(\/[a-z]+)?$/);
  if (!m) return undefined;
  const idx = rows.findIndex((l) => l.id === decodeURIComponent(m[1]));
  if (idx < 0) return err(404, "link_not_found", "link not found");
  const l = rows[idx];
  const action = m[2];
  const put = (next: PaymentLink, status = 200): Reply => {
    rows[idx] = next;
    return { status, body: detail(next) };
  };
  if (!action && method === "GET") return { status: 200, body: detail(l) };
  if (!action && method === "PATCH") {
    if (l.status === "archived") return err(409, "link_not_editable", "an archived link cannot be edited");
    const next = { ...l, ...JSON.parse(body ?? "{}"), revision: l.revision + 1, updated_at: new Date().toISOString() };
    return saveChecks(next) ?? put(next);
  }
  if (!action && method === "DELETE") {
    if (l.status !== "draft") return err(409, "link_not_deletable", "only drafts can be deleted");
    rows.splice(idx, 1);
    return { status: 204, body: null };
  }
  if (action === "/publish" && method === "POST") {
    const bad = publishChecks(l);
    if (bad) return bad;
    const code = l.short_code ?? `smpl${l.id.slice(-8)}`;
    return put({ ...l, status: "active", short_code: code, url: BASE + code, published_at: l.published_at ?? new Date().toISOString() });
  }
  if (action === "/pause" && method === "POST") return put({ ...l, status: "paused" });
  if (action === "/archive" && method === "POST") return put({ ...l, status: "archived" });
  if (action === "/duplicate" && method === "POST") {
    const copy: PaymentLink = { ...structuredClone(l), id: `sample_lnk_dup${String(seq++).padStart(9, "0")}`, status: "draft", short_code: null, url: null, uses_count: 0, published_at: null, created_at: new Date().toISOString() };
    rows.push(copy);
    return { status: 201, body: detail(copy) };
  }
  return undefined;
}
