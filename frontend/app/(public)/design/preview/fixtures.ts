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

const fam = (id: number, name: string, code: string, path: string) => ({
  id, name, code, family: code, path, supportsHDWallet: true, supportsSCWallet: false,
});
const wallets = [
  { id: 1, name: "Sample EVM deposits", kind: "hd", status: "active", blockchainFamilyID: 1, blockchainFamily: fam(1, "Ethereum family", "evm", "m/44'/60'/0'/0"), addressCount: 1000, createdAt: iso(-40 * DAY) },
  { id: 2, name: "Sample Bitcoin deposits", kind: "hd", status: "active", blockchainFamilyID: 2, blockchainFamily: fam(2, "Bitcoin family", "btc", "m/84'/0'/0'/0"), addressCount: 500, createdAt: iso(-40 * DAY) },
  { id: 3, name: "Sample Tron deposits", kind: "hd", status: "inactive", blockchainFamilyID: 3, blockchainFamily: fam(3, "Tron family", "trx", "m/44'/195'/0'/0"), addressCount: 250, createdAt: iso(-12 * DAY) },
];
const hex = (n: number) => n.toString(16).padStart(4, "0");
const addresses = Array.from({ length: 20 }, (_, i) => ({
  id: i + 1,
  address: `0x5a3c${hex(i * 977)}e1b09f7d2c4a8b6e0f13579bdf2468ace0${hex(i)}`,
  pathIndex: i,
  status: (["used", "available", "available", "locked"] as const)[i % 4],
  walletID: 1,
  blockchainFamilyID: 1,
  createdAt: iso(-40 * DAY + i * 3600_000),
}));
const hotWallets = [
  { id: 11, name: "Sample EVM gas", kind: "hot", status: "active", blockchainFamilyID: 1, blockchainFamilyCode: "evm", address: "0x9f2b7c41d0e8a3f6b5c2d1e0f9a8b7c6d5e4f3a2", createdAt: iso(-20 * DAY) },
  { id: 12, name: "Sample Tron gas", kind: "hot", status: "active", blockchainFamilyID: 3, blockchainFamilyCode: "trx", address: "TJsampleXq7wR2nB8vK3mD5pL9hF4cA6zE1y", createdAt: iso(-9 * DAY) },
];
const coldWallets = [
  { blockchainCode: "ETH", address: "0x1c4e8a7b2d9f3e6a5b0c8d7e6f5a4b3c2d1e0f9a", name: "Sample treasury" },
  { blockchainCode: "BTC", address: "bc1qsampl3xk7w9r2n8v3m5p0l9h4c6a2z8e1y7d5f", name: "Sample BTC vault" },
];

const WD_STATES = ["processed", "pending-approval", "sent", "pending-otp", "failed", "processed", "cancelled"];
const withdrawals = WD_STATES.map((state, i) => ({
  id: 4100 + i,
  externalPlatformID: 1,
  memberID: 1,
  toAddress: i % 3 === 1 ? "TJsampleRcpt7wR2nB8vK3mD5pL9hF4cA6zE" : `0x7e${hex(i * 313)}c3b1a9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3`,
  blockchainCode: i % 3 === 1 ? "TRON" : "ETH",
  currencyCode: i % 3 === 1 ? "USDT" : i % 2 ? "ETH" : "USDC",
  amount: ["2500", "180.5", "0.75", "1200", "40", "999.99", "15"][i],
  state,
  ...(state === "processed" || state === "sent" ? { transactionHash: `0x${hex(i * 911)}ab12cd34ef56ab78cd90ef12ab34cd56ef78ab90cd12ef34ab56cd78ef${hex(i)}` } : {}),
  createdAt: iso(-i * DAY * 1.5),
  updatedAt: iso(-i * DAY),
}));

type Route = { method: string; pattern: RegExp; body: (m: RegExpMatchArray, q: URLSearchParams) => unknown; empty?: unknown };

const ROUTES: Route[] = [
  { method: "GET", pattern: /^\/withdrawal\/merchant$/, body: () => ({ withdrawals }), empty: { withdrawals: [] } },
  { method: "GET", pattern: /^\/wallets$/, body: () => ({ wallets }), empty: { wallets: [] } },
  {
    method: "GET",
    pattern: /^\/wallets\/(\d+)\/addresses$/,
    body: (_m, q) => {
      const st = q.get("status");
      const rows = st ? addresses.filter((a) => a.status === st) : addresses;
      return { addresses: rows, total: st ? rows.length : 1000 };
    },
    empty: { addresses: [], total: 0 },
  },
  { method: "GET", pattern: /^\/wallets\/hot$/, body: () => ({ hotWallets }), empty: { hotWallets: [] } },
  {
    method: "GET",
    pattern: /^\/wallets\/hot\/(\d+)\/balance$/,
    body: (m) =>
      m[1] === "11"
        ? { address: hotWallets[0].address, chain: "ethereum", symbol: "ETH", balance: "0.004210" }
        : { address: hotWallets[1].address, chain: "tron", symbol: "TRX", balance: "312.5" },
  },
  { method: "GET", pattern: /^\/wallets\/cold$/, body: () => ({ coldWallets }), empty: { coldWallets: [] } },
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
