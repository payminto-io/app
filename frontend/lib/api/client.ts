/**
 * Typed fetch wrapper for the Payminto backend.
 *
 * Responsibilities:
 *   1. Resolve the base URL from `NEXT_PUBLIC_API_URL`.
 *   2. Attach `Authorization: Bearer <accessToken>` when the in-memory store
 *      has one (dashboard calls only — the Next.js route handlers running on
 *      the server talk to the backend directly without this wrapper).
 *   3. Parse JSON, throw `ApiError` on any non-2xx.
 *   4. On a 401, attempt ONE silent refresh via `/api/auth/refresh` and retry
 *      the original request once. Second 401 is surfaced and the Zustand
 *      store is cleared so the proxy.ts gate redirects to /signin on the
 *      next navigation.
 *
 * NOT responsible for:
 *   - UI concerns (loading spinners, toasts) — that lives in components.
 *   - Business logic — that lives in the per-domain modules under lib/api/*.
 *
 * See ADR-0008 for the JWT-or-API-key auth model this client supports.
 */
import { ApiError } from "./errors";
import { authStore } from "@/lib/auth/store";
import { API_BASE_URL } from "@/lib/constants";

const BASE_URL = API_BASE_URL;

interface ApiFetchInit extends Omit<RequestInit, "body"> {
  /** Request body; will be JSON-stringified automatically. */
  body?: unknown;
  /** Skip the 401 auto-refresh retry (used by the refresh call itself). */
  skipRefresh?: boolean;
}

/**
 * Single entry point for every call to the Payminto backend from the browser.
 * Server-side route handlers proxy to the backend directly using the same
 * BASE_URL but do not use this function.
 */
export async function apiFetch<T>(
  path: string,
  init: ApiFetchInit = {}
): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type") && init.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }
  const accessToken = authStore.getState().accessToken;
  if (accessToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const body =
    init.body === undefined
      ? undefined
      : typeof init.body === "string"
      ? init.body
      : JSON.stringify(init.body);

  let response: Response;
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      ...init,
      body,
      headers,
    });
  } catch {
    // Browsers collapse DNS, offline, connection-refused, and CORS failures
    // into an unhelpful TypeError("Failed to fetch"). Keep that implementation
    // detail out of page-level error states and give operators a useful next
    // action instead.
    throw new ApiError(
      0,
      "Payment service is temporarily unreachable. Confirm the backend is running, then try again."
    );
  }

  if (response.status === 401 && !init.skipRefresh) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      return apiFetch<T>(path, { ...init, skipRefresh: true });
    }
    authStore.getState().clear();
  }

  if (!response.ok) {
    const parsed = await parseErrorBody(response);
    // For 404s coming from the backend we rewrite the raw "Not Found" /
    // "404 page not found" text to a friendlier message so that an error
    // boundary (which shows `error.message` verbatim) does not render the
    // literal string "Not Found" — which looks indistinguishable from a
    // missing-route page to the user.
    const message =
      response.status === 404 &&
      (!parsed.message ||
        parsed.message === "Not Found" ||
        parsed.message === "404 page not found")
        ? "This data is not available yet."
        : parsed.message;
    throw new ApiError(response.status, message, parsed.body);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

/**
 * Calls the Next.js route handler at /api/auth/refresh which in turn calls
 * the Go backend with the httpOnly refresh cookie. On success the new access
 * token lands in the Zustand store via the route handler's response body.
 */
async function tryRefresh(): Promise<boolean> {
  try {
    const res = await fetch("/api/auth/refresh", {
      method: "POST",
      credentials: "include",
    });
    if (!res.ok) return false;
    const data = (await res.json()) as { accessToken?: string };
    if (!data.accessToken) return false;
    authStore.getState().setAccessToken(data.accessToken);
    return true;
  } catch {
    return false;
  }
}

async function parseErrorBody(
  response: Response
): Promise<{ message: string; body: unknown }> {
  try {
    const body = await response.json();
    const message =
      (typeof body === "object" &&
        body !== null &&
        "error" in body &&
        typeof (body as { error: unknown }).error === "string" &&
        (body as { error: string }).error) ||
      response.statusText;
    return { message, body };
  } catch {
    return { message: response.statusText, body: undefined };
  }
}
