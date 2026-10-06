/**
 * Onramper (fiat→crypto) domain API.
 * Routes: /onramper/payments (merchant-facing list). Backend wraps
 * the list in { payments: [...] } so we unwrap before returning.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface OnramperSession {
  id: number;
  externalPlatformID: number;
  sessionID: string;
  fiatAmount: string; // decimal — string
  fiatCurrency: string;
  cryptoAmount?: string;
  cryptoCurrency: string;
  blockchainCode: string;
  state: "pending" | "processing" | "completed" | "failed" | "refunded";
  customerEmail?: string;
  createdAt: string;
  updatedAt: string;
}

export const onramperApi = {
  list: async (params?: {
    state?: string;
    limit?: number;
    offset?: number;
  }): Promise<OnramperSession[]> => {
    const query = new URLSearchParams();
    if (params?.state) query.set("state", params.state);
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.offset) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query}` : "";
    try {
      const res = await apiFetch<{ payments: OnramperSession[] }>(
        `/onramper/payments${suffix}`
      );
      return res.payments ?? [];
    } catch (err) {
      // Feature may be disabled server-side; show empty state instead of error.
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },
};
