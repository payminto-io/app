/**
 * POST /api/auth/refresh
 *
 * Reads the httpOnly refresh cookie, calls the Go backend
 * `POST /auth/refresh`, rotates the cookie with the new refresh token, and
 * returns the fresh access token to the browser. Used by the silent-refresh
 * path in lib/api/client.ts.
 */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { REFRESH_COOKIE, backendURL, refreshCookieOptions } from "../shared";

export const runtime = "nodejs";

interface BackendTokens {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

export async function POST() {
  const store = await cookies();
  const refreshToken = store.get(REFRESH_COOKIE)?.value;
  if (!refreshToken) {
    return NextResponse.json({ error: "no refresh token" }, { status: 401 });
  }

  const res = await fetch(`${backendURL()}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
    cache: "no-store",
  });

  if (!res.ok) {
    store.delete(REFRESH_COOKIE);
    const text = await res.text();
    return NextResponse.json(safeParse(text), { status: res.status });
  }

  const parsed = (await res.json().catch(() => null)) as
    | { tokens?: BackendTokens }
    | null;
  const tokens = parsed?.tokens;
  if (!tokens?.accessToken || !tokens.refreshToken) {
    store.delete(REFRESH_COOKIE);
    return NextResponse.json(
      { error: "malformed backend response" },
      { status: 502 }
    );
  }

  store.set(REFRESH_COOKIE, tokens.refreshToken, refreshCookieOptions());
  return NextResponse.json({ accessToken: tokens.accessToken });
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return { error: text || "backend error" };
  }
}
