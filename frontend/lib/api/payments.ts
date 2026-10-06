/**
 * Payments domain API.
 *
 * DTO source of truth: payminto/backend/internal/api/dto/payment_dto.go
 * Routes: see payminto-backend-routes.md Section 1.
 */
import { apiFetch } from "./client";

/**
 * Payment response from the backend.
 * Backend DTO uses `paymentState` (json tag), states are UPPERCASE.
 * Fields like blockchainCode/currencyCode are not in the DTO yet.
 */
export interface Payment {
  referenceID: string;
  amountInUSD: string; // decimal — always string
  /** Backend returns this as `paymentState` with UPPERCASE values. */
  paymentState: string;
  customerEmail?: string;
  customerID?: string;
  invoiceID?: string;
  createdAt: string;
  expiresAt?: string;
}

export interface PaymentsListResponse {
  payments: Payment[];
  total: number;
}

export interface CreatePaymentInput {
  amountInUSD: string;
  blockchainCode?: string;
  currencyCode?: string;
  customerEmail?: string;
  customerID?: string;
  description?: string;
  successURL?: string;
  cancelURL?: string;
  expiryMinutes?: number;
}

export interface CreatePaymentResponse {
  reference_id: string;
  referenceID?: string; // alias — backend uses snake_case
  url: string;
  host: string;
  depositAddress?: string;
  blockchainCode?: string;
}

export const paymentsApi = {
  list: (params?: { state?: string; limit?: number; offset?: number }) => {
    const query = new URLSearchParams();
    if (params?.state) query.set("state", params.state);
    if (params?.limit) query.set("limit", String(params.limit));
    if (params?.offset) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query}` : "";
    return apiFetch<PaymentsListResponse>(`/payments${suffix}`);
  },

  get: (referenceID: string) =>
    apiFetch<Payment>(`/payment/reference/${encodeURIComponent(referenceID)}`),

  create: (input: CreatePaymentInput) =>
    apiFetch<CreatePaymentResponse>("/payment", { method: "POST", body: input }),
};
