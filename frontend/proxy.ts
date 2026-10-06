/**
 * Next.js 16 proxy — gates /dashboard/** and /admin/** behind the presence
 * of the httpOnly refresh cookie. Unauthenticated hits redirect to /signin
 * with a ?redirect=<path> query so the signin page can bounce back after
 * success.
 *
 * This is a presence check only — it does NOT validate the token against
 * the backend. Invalid tokens still get through the proxy and fail on the
 * first data fetch, which then clears the Zustand store and bounces via
 * client-side redirect. Checking the backend from here would make every
 * navigation pay a round-trip.
 */
import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const PROTECTED_PREFIXES = ["/dashboard", "/admin"];
const REFRESH_COOKIE = "payminto_rt";

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isProtected = PROTECTED_PREFIXES.some(
    (p) => pathname === p || pathname.startsWith(`${p}/`)
  );
  if (!isProtected) {
    return NextResponse.next();
  }

  const hasSession = Boolean(request.cookies.get(REFRESH_COOKIE)?.value);
  if (hasSession) {
    return NextResponse.next();
  }

  const url = request.nextUrl.clone();
  url.pathname = "/signin";
  url.searchParams.set("redirect", pathname);
  return NextResponse.redirect(url);
}

export const config = {
  matcher: ["/dashboard/:path*", "/admin/:path*"],
};
