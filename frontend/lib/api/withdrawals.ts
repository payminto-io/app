/**
 * Withdrawals domain API.
 * Routes: POST /withdrawal/merchant, POST /withdrawal/:id/otp/verify,
 * /withdrawal/:id/approve, /withdrawal/:id/cancel, GET /withdrawal/:id,
 * GET /withdrawal/merchant.
 */
import { apiFetch } from "./client";
import {
  normalizeCreatedWithdrawal,
  normalizeWithdrawalList,
  toCreateWithdrawalRequest,
  unwrapWithdrawal,
  type CreateWithdrawalInput,
  type Withdrawal,
} from "./withdrawal-contract";

export type {
  CreateWithdrawalInput,
  Withdrawal,
  WithdrawalState,
} from "./withdrawal-contract";

export interface WithdrawalsListResponse {
  withdrawals: Withdrawal[];
}

export const withdrawalsApi = {
  list: async () => {
    const response = await apiFetch<unknown>("/withdrawal/merchant");
    return normalizeWithdrawalList(response);
  },

  get: async (id: number) =>
    unwrapWithdrawal(await apiFetch<unknown>(`/withdrawal/${id}`)),

  create: async (input: CreateWithdrawalInput) =>
    normalizeCreatedWithdrawal(
      await apiFetch<unknown>("/withdrawal/merchant", {
        method: "POST",
        body: toCreateWithdrawalRequest(input),
      })
    ),

  verifyOTP: (id: number, code: string) =>
    apiFetch<{ message: string }>(`/withdrawal/${id}/otp/verify`, {
      method: "POST",
      body: { code },
    }),

  approve: (id: number) =>
    apiFetch<{ message: string }>(`/withdrawal/${id}/approve`, { method: "POST" }),

  cancel: (id: number) =>
    apiFetch<{ message: string }>(`/withdrawal/${id}/cancel`, { method: "POST" }),
};
