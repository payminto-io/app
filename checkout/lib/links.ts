// Payment links are served by the gateway's v2 public API. PAYMINTO_API_URL
// names the v1 base, so the v2 base is derived from it rather than configured
// twice and allowed to drift.
const API_BASE = (process.env.PAYMINTO_API_URL ?? "http://localhost:8080/api/v1").replace(/\/$/, "");
const V2_BASE = API_BASE.replace(/\/api\/v1$/, "/api/v2");

/** Builds a gateway URL for a public payment-link route. */
export function linksURL(path: string): string {
  if (!path.startsWith("/public/links")) {
    throw new Error("Only public payment-link routes may be proxied");
  }
  return `${V2_BASE}${path}`;
}

/** Short codes are unguessable and URL-safe; anything else never reaches the gateway. */
export function validShortCode(value: string): boolean {
  return /^[A-Za-z0-9_-]{6,64}$/.test(value);
}

export interface LinkMethod {
  method: string;
  chain: string | null;
  asset: string | null;
  customer_total: string | null;
}

export interface PublicLink {
  short_code: string;
  available: boolean;
  unavailable_reason: string | null;
  merchant_name: string | null;
  title: string;
  description: string | null;
  amount_mode: string;
  amount: string | null;
  currency: string;
  methods: LinkMethod[];
}

/** Reads one published link. Returns null when the gateway will not serve it. */
export async function loadPublicLink(shortCode: string): Promise<PublicLink | null> {
  const res = await fetch(linksURL(`/public/links/${encodeURIComponent(shortCode)}`), {
    cache: "no-store",
    headers: { Accept: "application/json" },
  });
  if (!res.ok) return null;
  return (await res.json()) as PublicLink;
}

/** Labels a method the way a payer reads it, not the way the API names it. */
export function methodLabel(m: LinkMethod): string {
  if (m.method === "crypto" && m.asset) {
    return m.chain ? `${m.asset} on ${titleCase(m.chain)}` : m.asset;
  }
  return titleCase(m.method);
}

function titleCase(v: string): string {
  return v.charAt(0).toUpperCase() + v.slice(1).toLowerCase();
}
