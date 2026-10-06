/**
 * POST /api/auth/signout
 *
 * Reads the refresh cookie, fires a best-effort POST to the Go backend
 * `POST /auth/signout` to invalidate the refresh family, then clears the
 * cookie regardless of backend outcome. 204 on success.
 */
import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { REFRESH_COOKIE, backendURL } from "../shared";

export const runtime = "nodejs";

export async function POST() {
  const store = await cookies();
  const refreshToken = store.get(REFRESH_COOKIE)?.value;

  if (refreshToken) {
    // Best effort — if the backend is down we still clear the cookie.
    await fetch(`${backendURL()}/auth/signout`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
      cache: "no-store",
    }).catch(() => undefined);
  }

  store.delete(REFRESH_COOKIE);
  return new NextResponse(null, { status: 204 });
}
