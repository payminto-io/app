/**
 * Missed deposits (admin) domain API.
 *
 * Backend routes (gated by system.admin):
 *   GET  /admin/missed-deposits                 -> { missedDeposits: [...] }
 *   GET  /admin/missed-deposits/:id             -> MissedDeposit
 *   POST /admin/missed-deposits/:id/resolve     body: { action, reason }
 *
 * The backend combines "resolve" and "dismiss" behind the single /resolve
 * endpoint — the `action` payload field selects the transition. The
 * frontend historically called a non-existent /dismiss route which surfaced
 * as "Not Found"; we now route dismiss through /resolve with action="dismiss".
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface MissedDeposit {
  id: number;
  blockchainCode: string;
  currencyCode: string;
  address: string;
  amount: string;
  transactionHash: string;
  status: "pending" | "resolved" | "dismissed";
  reason?: string;
  resolvedAt?: string;
  resolvedBy?: number;
  createdAt: string;
}

interface ListResponse {
  missedDeposits: MissedDeposit[];
}

export const missedDepositsApi = {
  list: async (params?: {
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<MissedDeposit[]> => {
    const query = new URLSearchParams();
    if (params?.status) query.set("status", params.status);
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.offset) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query}` : "";
    try {
      const res = await apiFetch<ListResponse>(
        `/admin/missed-deposits${suffix}`
      );
      return res.missedDeposits ?? [];
    } catch (err) {
      if (
        isApiError(err) &&
        (err.isNotFound || err.isForbidden || err.isUnauthorized)
      ) {
        return [];
      }
      throw err;
    }
  },

  resolve: (id: number, paymentReferenceID?: string) =>
    apiFetch<{ status: string }>(`/admin/missed-deposits/${id}/resolve`, {
      method: "POST",
      body: { action: "resolve", reason: paymentReferenceID ?? "" },
    }),

  /** Backend routes dismiss through the same /resolve endpoint. */
  dismiss: (id: number, reason?: string) =>
    apiFetch<{ status: string }>(`/admin/missed-deposits/${id}/resolve`, {
      method: "POST",
      body: { action: "dismiss", reason: reason ?? "" },
    }),
};
