/**
 * POST /api/auth/signup
 *
 * Proxies to Go backend POST /auth/signup. On success stores the refresh
 * token in an httpOnly cookie (same as signin) and returns accessToken + member.
 */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { REFRESH_COOKIE, backendURL, refreshCookieOptions } from "../shared";

export const runtime = "nodejs";

interface BackendSignupResponse {
  member: {
    id: number;
    email: string;
    name: string;
    memberType?: string;
    externalPlatformID?: number;
  };
  tokens: {
    accessToken: string;
    refreshToken: string;
    expiresIn: number;
  };
}

export async function POST(request: Request) {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return NextResponse.json({ error: "invalid json" }, { status: 400 });
  }

  const res = await fetch(`${backendURL()}/auth/signup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });

  const text = await res.text();
  if (!res.ok) {
    return NextResponse.json(safeParse(text), { status: res.status });
  }

  const parsed = safeParse(text) as BackendSignupResponse | null;
  if (!parsed?.tokens?.refreshToken || !parsed.tokens.accessToken) {
    return NextResponse.json(
      { error: "malformed backend response" },
      { status: 502 }
    );
  }

  const store = await cookies();
  store.set(REFRESH_COOKIE, parsed.tokens.refreshToken, refreshCookieOptions());

  return NextResponse.json({
    accessToken: parsed.tokens.accessToken,
    member: parsed.member,
  });
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return { error: text || "backend error" };
  }
}
