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

const webhooks = [
  { id: 7, externalPlatformID: 1, url: "https://sample-shop.example.com/hooks/payments", events: ["payment.filled", "payment.confirmed", "payment.expired", "withdrawal.sent"], active: true, createdAt: iso(-25 * DAY), updatedAt: iso(-1 * DAY) },
  { id: 8, externalPlatformID: 1, url: "https://staging.sample-shop.example.com/webhooks", events: ["payment.pending"], active: false, createdAt: iso(-8 * DAY), updatedAt: iso(-8 * DAY) },
];
const deliveries = Array.from({ length: 8 }, (_, i) => ({
  id: 900 + i,
  webhookID: 7,
  event: ["payment.filled", "payment.confirmed", "payment.expired", "withdrawal.sent"][i % 4],
  status: (["delivered", "delivered", "failed", "retrying", "delivered", "pending", "delivered", "delivered"] as const)[i],
  ...(i === 5 ? {} : { statusCode: i === 2 ? 500 : i === 3 ? 502 : 200 }),
  attempts: i === 2 ? 5 : i === 3 ? 2 : 1,
  createdAt: iso(-i * 3 * 3600_000),
}));

type Route = { method: string; pattern: RegExp; body: (m: RegExpMatchArray, q: URLSearchParams) => unknown; empty?: unknown };

const ROUTES: Route[] = [
  { method: "GET", pattern: /^\/referrals\/code$/, body: () => ({ referralCode: "SAMPLE-7KQ2" }) },
  {
    method: "GET",
    pattern: /^\/referrals\/stats$/,
    body: () => ({ stats: { MemberID: 1, TotalReferrals: 14, TotalEarned: 312.4, ConversionRate: 0.21 } }),
  },
  {
    method: "GET",
    pattern: /^\/admin\/referrals\/campaigns$/,
    body: () => ({
      campaigns: [
        { id: 1, name: "Sample launch offer", description: "First 90 days", rewardType: "percentage", rewardValue: "10", active: true, startsAt: iso(-30 * DAY), endsAt: iso(60 * DAY) },
        { id: 2, name: "Sample partner bonus", rewardType: "fixed", rewardValue: "25", active: false, startsAt: iso(-120 * DAY), endsAt: iso(-40 * DAY) },
      ],
    }),
    empty: { campaigns: [] },
  },
  {
    method: "GET",
    pattern: /^\/onramper\/payments$/,
    body: (_m, q) => {
      const rows = (["completed", "processing", "pending", "failed", "completed", "refunded"] as const).map((state, i) => ({
        id: 300 + i,
        externalPlatformID: 1,
        sessionID: `sample_onr_${hex(i * 4099)}9f2c7a1e`,
        fiatAmount: ["150", "75.5", "1200", "20", "300", "45"][i],
        fiatCurrency: i % 2 ? "EUR" : "USD",
        ...(state === "completed" || state === "refunded" ? { cryptoAmount: ["148.21", "", "", "", "296.4", "44.1"][i] } : {}),
        cryptoCurrency: "USDC",
        blockchainCode: "BASE",
        state,
        ...(i % 3 ? { customerEmail: `buyer${i}@example.com` } : {}),
        createdAt: iso(-i * DAY * 0.8),
        updatedAt: iso(-i * DAY * 0.5),
      }));
      const st = q.get("state");
      return { payments: st ? rows.filter((r) => r.state === st) : rows };
    },
    empty: { payments: [] },
  },
  { method: "GET", pattern: /^\/webhooks$/, body: () => ({ webhooks }), empty: { webhooks: [] } },
  { method: "GET", pattern: /^\/webhooks\/(\d+)$/, body: (m) => ({ webhook: webhooks.find((w) => String(w.id) === m[1]) ?? webhooks[0] }) },
  { method: "GET", pattern: /^\/webhooks\/(\d+)\/deliveries$/, body: () => ({ deliveries }), empty: { deliveries: [] } },
  {
    method: "GET",
    pattern: /^\/analytics\/sweeps$/,
    body: () => ({ sweeps: { TotalSweeps: 12, TotalSwept: "14820.5", TotalGas: "0.084213" } }),
    empty: { sweeps: { TotalSweeps: 0, TotalSwept: "0", TotalGas: "0" } },
  },
  {
    method: "GET",
    pattern: /^\/recipients$/,
    body: () => ({
      recipients: [
        { id: 1, externalPlatformID: 1, memberID: 1, name: "Sample supplier", email: "supplier@example.com", blockchainCode: "ETH", currencyCode: "USDC", address: "0x3b7d2e9f1a8c4b6d0e5f7a9c1b3d5e7f9a1c3e5b", createdAt: iso(-20 * DAY), updatedAt: iso(-2 * DAY) },
        { id: 2, externalPlatformID: 1, memberID: 1, name: "Sample contractor", blockchainCode: "TRON", currencyCode: "USDT", address: "TSampleContractorAddr9xK3mD5pL9hF4cA6z", createdAt: iso(-11 * DAY), updatedAt: iso(-11 * DAY) },
        { id: 3, externalPlatformID: 1, memberID: 1, name: "Sample treasury", blockchainCode: "BTC", currencyCode: "BTC", address: "bc1qsampletreasury0x7w9r2n8v3m5p0l9h4c6a2", createdAt: iso(-4 * DAY), updatedAt: iso(-4 * DAY) },
      ],
    }),
    empty: { recipients: [] },
  },
  {
    method: "POST",
    pattern: /^\/api-keys$/,
    body: () => ({ id: 9, name: "Sample", prefix: "pk_test_new0", active: true, status: "active", externalPlatformID: 1, createdAt: iso(0), rawKey: "pk_test_SAMPLE_not_a_real_key_0000000000000000" }),
  },
  {
    method: "GET",
    pattern: /^\/api-keys$/,
    body: () => ({
      apiKeys: [
        { id: 1, name: "Sample production server", prefix: "pk_live_s4mp", active: true, status: "active", externalPlatformID: 1, lastUsedAt: iso(-3600_000), createdAt: iso(-30 * DAY) },
        { id: 2, name: "Sample staging", prefix: "pk_test_9x2q", active: true, status: "active", externalPlatformID: 1, createdAt: iso(-6 * DAY) },
        { id: 3, name: "Sample old key", prefix: "pk_test_a1b2", active: false, status: "inactive", externalPlatformID: 1, lastUsedAt: iso(-50 * DAY), createdAt: iso(-90 * DAY) },
      ],
    }),
    empty: { apiKeys: [] },
  },
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
