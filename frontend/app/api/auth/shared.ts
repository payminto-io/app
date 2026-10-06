/**
 * Shared helpers for every auth route handler. Kept tiny on purpose —
 * cookie name, cookie options, backend URL resolver.
 */
export const REFRESH_COOKIE = "payminto_rt";

/**
 * Resolves the Go backend base URL. Uses `BACKEND_API_URL` for server-side
 * (route handlers) and falls back to `NEXT_PUBLIC_API_URL` so local dev can
 * use one value for both layers.
 */
export function backendURL(): string {
  return (
    process.env.BACKEND_API_URL ??
    process.env.NEXT_PUBLIC_API_URL ??
    "http://localhost:8080/api/v1"
  );
}

export interface RefreshCookieOptions {
  httpOnly: true;
  sameSite: "lax";
  secure: boolean;
  path: "/";
  maxAge: number;
}

/**
 * Options applied to the refresh-token cookie on signin + refresh.
 * Secure is disabled in dev so localhost can read the cookie; every other
 * environment forces it on.
 */
/**
 * Decode the payload of a JWT without verification (the backend already
 * verified it). Returns the parsed claims object or null on failure.
 */
export function decodeJWTPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const json = Buffer.from(payload, "base64").toString("utf-8");
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function refreshCookieOptions(): RefreshCookieOptions {
  return {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24 * 7, // 7 days — matches backend refresh TTL
  };
}
