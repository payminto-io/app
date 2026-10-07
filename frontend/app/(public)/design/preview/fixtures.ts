/**
 * Sample fixtures for /design/preview only. Every value is invented for
 * layout review and labelled as sample on screen; none of it is product data.
 */

const DAY = 86_400_000;
const NOW = Date.parse("2026-10-07T10:00:00Z");
const iso = (offsetMs: number) => new Date(NOW + offsetMs).toISOString();

const PAYMENT_STATES = ["FILLED", "OPEN", "PARTIALLY_FILLED", "FILLED", "EXPIRED", "FILLED", "CANCELLED", "OPEN", "FILLED", "FILLED"];
const AMOUNTS = ["1250.00", "40.00", "19.99", "320.50", "75.00", "2400.00", "12.00", "99.00", "640.25", "5.00"];

const payments = Array.from({ length: 20 }, (_, i) => PAYMENT_STATES[i % 10]).map((state, i) => ({
  referenceID: `sample_ref_${String(i + 1).padStart(2, "0")}_8f3a1c9d2e`,
  amountInUSD: AMOUNTS[i % 10],
  paymentState: state,
  customerEmail: i % 3 === 2 ? undefined : `customer${i + 1}@example.com`,
  customerID: `sample_cus_${i + 1}`,
  invoiceID: i % 2 === 0 ? `INV-SAMPLE-${1000 + i}` : undefined,
  createdAt: iso(-i * DAY * 0.7),
  expiresAt: iso(-i * DAY * 0.7 + DAY),
}));

const customers = Array.from({ length: 6 }, (_, i) => ({
  id: i + 1,
  name: ["Ada Sample", "Ben Sample", "Chen Sample", "Dee Sample", "Eli Sample", "Fay Sample"][i],
  email: `customer${i + 1}@example.com`,
  customerID: `sample_cus_${i + 1}`,
  state: "active",
  memberType: "customer",
  createdAt: iso(-i * DAY * 3),
  updatedAt: iso(-i * DAY),
}));

const volume = Array.from({ length: 14 }, (_, i) => {
  const d = new Date(NOW - (13 - i) * DAY);
  const v = [420, 980, 610, 0, 1520, 2210, 880, 1340, 760, 1990, 2480, 1120, 1730, 2050][i];
  return {
    Bucket: d.toISOString().slice(0, 10),
    BucketLabel: d.toLocaleDateString("en-US", { month: "short", day: "numeric" }),
    Volume: `${v}.50`,
    PaymentCount: Math.round(v / 90),
  };
});

type Route = { method: string; pattern: RegExp; body: (m: RegExpMatchArray, q: URLSearchParams) => unknown; empty?: unknown };

const ROUTES: Route[] = [
  {
    method: "GET",
    pattern: /^\/analytics\/summary$/,
    body: () => ({
      summary: { TotalPayments: 134, FilledPayments: 97, TotalVolume: "18210.75", TotalSweeps: 12, TotalWithdrawals: 8, ActiveWebhooks: 2 },
    }),
    empty: {
      summary: { TotalPayments: 0, FilledPayments: 0, TotalVolume: "0", TotalSweeps: 0, TotalWithdrawals: 0, ActiveWebhooks: 0 },
    },
  },
  {
    method: "GET",
    pattern: /^\/analytics\/volume$/,
    body: () => ({ volume }),
    empty: { volume: [] },
  },
  {
    method: "GET",
    pattern: /^\/payments$/,
    body: (_m, q) => {
      const state = q.get("state");
      const rows = state ? payments.filter((p) => p.paymentState === state) : payments;
      return { payments: rows, total: state ? rows.length : 134 };
    },
    empty: { payments: [], total: 0 },
  },
  {
    method: "GET",
    pattern: /^\/payment\/reference\/(.+)$/,
    body: (m) => payments.find((p) => p.referenceID === decodeURIComponent(m[1])) ?? payments[2],
  },
  {
    method: "POST",
    pattern: /^\/payment$/,
    body: () => ({ reference_id: "sample_ref_new_4b7e2a", url: "", host: "" }),
  },
  {
    method: "GET",
    pattern: /^\/customers$/,
    body: (_m, q) => {
      const s = (q.get("search") ?? "").toLowerCase();
      const rows = customers.filter((c) => !s || c.email.includes(s) || c.name.toLowerCase().includes(s));
      return { customers: rows, total: rows.length };
    },
    empty: { customers: [], total: 0 },
  },
];

export function resolveFixture(pathWithQuery: string, method: string, empty: boolean): unknown {
  const [path, query = ""] = pathWithQuery.split("?");
  const q = new URLSearchParams(query);
  for (const r of ROUTES) {
    if (r.method !== method) continue;
    const m = path.match(r.pattern);
    if (!m) continue;
    if (empty && r.empty !== undefined) return r.empty;
    return r.body(m, q);
  }
  return undefined;
}
