/**
 * POST /api/auth/signin
 *
 * Proxies the browser credentials to the Go backend at
 * `${BACKEND_URL}/auth/signin`. On success:
 *   - Stores the long-lived refreshToken in an httpOnly cookie so React
 *     code can never read it (XSS-safe).
 *   - Returns { accessToken, member } to the browser so the Zustand store
 *     can hold the short-lived access token in memory.
 *
 * Runtime pinned to nodejs — we need access to cookies() and the Node fetch
 * with credentials forwarding.
 */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { REFRESH_COOKIE, backendURL, refreshCookieOptions, decodeJWTPayload } from "../shared";

export const runtime = "nodejs";

interface BackendTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

interface BackendSigninResponse {
  member: {
    id: number;
    email: string;
    name: string;
    memberType?: string;
    externalPlatformID?: number;
  };
  tokens: BackendTokens;
}

export async function POST(request: Request) {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "invalid json" }, { status: 400 });
  }

  const res = await fetch(`${backendURL()}/auth/signin`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });

  const text = await res.text();
  if (!res.ok) {
    return NextResponse.json(safeParse(text), { status: res.status });
  }

  const parsed = safeParse(text) as BackendSigninResponse | null;
  if (!parsed?.tokens?.refreshToken || !parsed.tokens.accessToken) {
    return NextResponse.json(
      { error: "malformed backend response" },
      { status: 502 }
    );
  }

  const store = await cookies();
  store.set(REFRESH_COOKIE, parsed.tokens.refreshToken, refreshCookieOptions());

  // Extract externalPlatformID from JWT claims and merge into member object.
  // The Member model doesn't carry this field, but the JWT does.
  const claims = decodeJWTPayload(parsed.tokens.accessToken);
  const member = {
    ...parsed.member,
    externalPlatformID:
      (claims?.externalPlatformID as number | undefined) ??
      parsed.member.externalPlatformID ??
      0,
  };

  return NextResponse.json({
    accessToken: parsed.tokens.accessToken,
    member,
  });
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return { error: text || "backend error" };
  }
}
