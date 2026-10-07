/**
 * Fee preview (v1). Contract: backend/internal/fees/README.md "HTTP".
 * The browser never computes a fee; it shows what this endpoint returns.
 */
import { apiFetch } from "./client";
import type { FeeBearer, PayMethod } from "./links";

export interface FeePreviewRequest {
  amount: string;
  currency: string;
  method: PayMethod;
  fee_bearer?: FeeBearer;
}

export interface FeePreview {
  rule_id: number;
  version: number;
  currency: string;
  fee_bearer: FeeBearer;
  amount: string;
  fee: string;
  tax: string;
  customer_total: string;
  merchant_net: string;
}

export const feesApi = {
  preview: (body: FeePreviewRequest) =>
    apiFetch<FeePreview>("/fees/preview", { method: "POST", body }),
};
