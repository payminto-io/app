/**
 * GET /api/auth/me
 *
 * Server-side relay: reads the httpOnly refresh cookie, exchanges it for
 * a fresh access token, rotates the cookie with the new refresh token,
 * fetches the member profile, and returns { accessToken, member }.
 *
 * The accessToken is included in the response so the QueryProvider can
 * populate the Zustand store on bootstrap without a separate /refresh call.
 */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { REFRESH_COOKIE, backendURL, refreshCookieOptions, decodeJWTPayload } from "../shared";

export const runtime = "nodejs";

interface BackendTokens {
  accessToken: string;
  refreshToken: string;
}

export async function GET() {
  const store = await cookies();
  const refreshToken = store.get(REFRESH_COOKIE)?.value;
  if (!refreshToken) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }

  const tokenRes = await fetch(`${backendURL()}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
    cache: "no-store",
  });
  if (!tokenRes.ok) {
    store.delete(REFRESH_COOKIE);
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }

  const parsed = (await tokenRes.json().catch(() => null)) as
    | { tokens?: BackendTokens }
    | null;
  const tokens = parsed?.tokens;
  if (!tokens?.accessToken || !tokens.refreshToken) {
    store.delete(REFRESH_COOKIE);
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }

  // Rotate the cookie so the old refresh token is never re-used.
  store.set(REFRESH_COOKIE, tokens.refreshToken, refreshCookieOptions());

  const meRes = await fetch(`${backendURL()}/members/me`, {
    method: "GET",
    headers: { Authorization: `Bearer ${tokens.accessToken}` },
    cache: "no-store",
  });
  if (!meRes.ok) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 });
  }

  const rawMember = await meRes.json();

  // Extract externalPlatformID from JWT claims and merge into member object.
  // The Member model doesn't carry this field, but the JWT does.
  const claims = decodeJWTPayload(tokens.accessToken);
  const member = {
    ...rawMember,
    externalPlatformID:
      (claims?.externalPlatformID as number | undefined) ??
      rawMember.externalPlatformID ??
      0,
  };

  return NextResponse.json({ accessToken: tokens.accessToken, member });
}
