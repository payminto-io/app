/**
 * Sample payment-link fixtures for /design/preview only. Every value is invented for layout
 * review; the page labels it as sample. A tiny in-memory stand-in for the v2 routes and the
 * fee preview so the builder can be driven end to end without a backend. Not product logic.
 */
import { defaultForm, formToInput } from "@/features/links/model";
import type { LinkInput, PaymentLink } from "@/lib/api/links";

type Reply = { status: number; body: unknown };

const NOW = Date.parse("2026-10-07T10:00:00Z");
const iso = (days: number) => new Date(NOW + days * 86_400_000).toISOString();
const BASE = "https://pay.example.com/l/";

function sumItems(items: LinkInput["line_items"]): string | null {
  if (items.length === 0) return null;
  let cents = 0;
  for (const li of items) {
    const sub = Math.round(li.quantity * Number(li.unit_price) * 100);
    cents += sub + Math.round((sub * Number(li.tax_rate)) / 100);
  }
  return (cents / 100).toFixed(2);
}

function make(id: string, input: Partial<LinkInput>, extra: Partial<PaymentLink>): PaymentLink {
  const full = { ...formToInput(defaultForm()), ...input };
  const code = extra.status && extra.status !== "draft" ? `smpl${id.slice(-8)}` : null;
  return {
    ...full,
    id,
    status: "draft",
    environment: "test",
    short_code: code,
    url: code ? BASE + code : null,
    total: full.amount_mode === "fixed" ? full.amount : full.amount_mode === "line_items" ? sumItems(full.line_items) : null,
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
      methods: [{ method: "card" }, { method: "bank" }],
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
      methods: [{ method: "upi" }, { method: "card" }],
      questions: [{ key: "tshirt", label: "T-shirt size", type: "select", options: ["S", "M", "L"], required: true, per_order: true }],
    },
    { created_at: iso(-1) }
  ),
  make(
    "sample_lnk_00springsale",
    { title: "Spring sale deposit", amount: "25.00", currency: "EUR", methods: [{ method: "card" }] },
    { status: "paused", uses_count: 7, published_at: iso(-30), created_at: iso(-31) }
  ),
  make(
    "sample_lnk_0000webinar",
    { title: "Webinar replay", amount: "9.00", currency: "USD", methods: [{ method: "card" }] },
    { status: "archived", uses_count: 41, published_at: iso(-60), created_at: iso(-61) }
  ),
];

let store: PaymentLink[] | null = null;
const db = () => (store ??= SEED.map((l) => structuredClone(l)));
let seq = 1;

const err = (status: number, code: string, message: string, field?: string): Reply => ({
  status,
  body: { error: message, code, ...(field ? { field } : {}) },
});

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
    if (m.method === "bank") list.push({ code: "method_no_fee_rule", field: `methods[${n}]`, message: `no active fee rule for bank in ${l.currency}` });
  });
  return list.length ? errs(list) : null;
}

function withTotal(l: PaymentLink): PaymentLink {
  return { ...l, total: l.amount_mode === "fixed" ? l.amount : l.amount_mode === "line_items" ? sumItems(l.line_items) : null };
}

/** Sample fee rules: card 2.9% + 0.30, UPI free and never surcharged, bank none, crypto 1%. */
function feePreview(b: { amount: string; currency: string; method: string; fee_bearer?: string }): Reply {
  const amount = Number(b.amount);
  if (b.method === "bank") return err(404, "no_fee_rule", "no active fee rule");
  if (b.method === "upi" && b.fee_bearer === "customer") return err(422, "surcharge_forbidden", "upi does not allow a customer surcharge");
  const rule = { card: { id: 3, v: 2, pct: 2.9, flat: 0.3 }, upi: { id: 5, v: 1, pct: 0, flat: 0 }, crypto: { id: 7, v: 1, pct: 1, flat: 0 } }[b.method as "card" | "upi" | "crypto"];
  if (!rule) return err(404, "no_fee_rule", "no active fee rule");
  const fee = Math.round((amount * rule.pct + rule.flat * 100)) / 100;
  const tax = b.currency === "INR" ? Math.round(fee * 18) / 100 : 0;
  const customer = b.fee_bearer === "customer";
  const fx = (n: number) => n.toFixed(2);
  return {
    status: 200,
    body: {
      rule_id: rule.id,
      version: rule.v,
      currency: b.currency,
      fee_bearer: customer ? "customer" : "merchant",
      amount: fx(amount),
      fee: fx(fee),
      tax: fx(tax),
      customer_total: fx(customer ? amount + fee + tax : amount),
      merchant_net: fx(customer ? amount : amount - fee - tax),
    },
  };
}

const CURRENCIES = {
  currencies: [
    { id: 1, blockchainCode: "SOL", currencyCode: "USDC", standard: "SPL", address: "" },
    { id: 2, blockchainCode: "BASE", currencyCode: "USDC", standard: "ERC20", address: "" },
    { id: 3, blockchainCode: "ETH", currencyCode: "USDT", standard: "ERC20", address: "" },
  ],
};

/** v1 paths this module answers; everything else falls through to fixtures.ts. */
export function resolveLinksV1(path: string, method: string, body: string | undefined): Reply | undefined {
  if (method === "POST" && path === "/fees/preview") return feePreview(JSON.parse(body ?? "{}"));
  if (method === "GET" && path === "/public/blockchain-currencies") return { status: 200, body: CURRENCIES };
  return undefined;
}

export function resolveLinksV2(pathWithQuery: string, method: string, body: string | undefined, empty: boolean): Reply | undefined {
  const [path, query = ""] = pathWithQuery.split("?");
  const q = new URLSearchParams(query);
  const rows = db();
  if (path === "/links" && method === "GET") {
    if (empty) return { status: 200, body: { links: [], total: 0 } };
    const status = q.get("status");
    const list = rows.filter((l) => !status || l.status === status).sort((a, b) => b.created_at.localeCompare(a.created_at));
    return { status: 200, body: { links: list, total: list.length } };
  }
  if (path === "/links" && method === "POST") {
    const input = { ...formToInput(defaultForm()), ...JSON.parse(body ?? "{}") } as LinkInput;
    const bad = saveChecks(input);
    if (bad) return bad;
    const l = withTotal(make(`sample_lnk_new${String(seq++).padStart(9, "0")}`, input, { created_at: new Date().toISOString() }));
    rows.push(l);
    return { status: 201, body: l };
  }
  const m = path.match(/^\/links\/([^/]+)(\/[a-z]+)?$/);
  if (!m) return undefined;
  const idx = rows.findIndex((l) => l.id === decodeURIComponent(m[1]));
  if (idx < 0) return err(404, "link_not_found", "link not found");
  const l = rows[idx];
  const action = m[2];
  const put = (next: PaymentLink, status = 200): Reply => {
    rows[idx] = next;
    return { status, body: next };
  };
  if (!action && method === "GET") return { status: 200, body: l };
  if (!action && method === "PATCH") {
    if (l.status === "archived") return err(409, "link_not_editable", "an archived link cannot be edited");
    const next = withTotal({ ...l, ...JSON.parse(body ?? "{}"), revision: l.revision + 1, updated_at: new Date().toISOString() });
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
    const copy = withTotal({ ...structuredClone(l), id: `sample_lnk_dup${String(seq++).padStart(9, "0")}`, status: "draft", short_code: null, url: null, uses_count: 0, published_at: null, created_at: new Date().toISOString() });
    rows.push(copy);
    return { status: 201, body: copy };
  }
  return undefined;
}
