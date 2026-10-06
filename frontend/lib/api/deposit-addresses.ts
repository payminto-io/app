/**
 * Deposit addresses domain API.
 * Routes: POST/GET /deposit-address/reference/:reference_id
 */
import { apiFetch } from "./client";

export interface DepositAddress {
  id: number;
  address: string;
  blockchainCode: string;
  currencyCode: string;
  assignedAt: string;
  paymentReferenceID: string;
}

export const depositAddressesApi = {
  list: (referenceID: string) =>
    apiFetch<DepositAddress[]>(
      `/deposit-address/reference/${encodeURIComponent(referenceID)}`
    ),

  assign: (
    referenceID: string,
    input: { blockchainCode: string; currencyCode: string }
  ) =>
    apiFetch<DepositAddress>(
      `/deposit-address/reference/${encodeURIComponent(referenceID)}`,
      { method: "POST", body: input }
    ),
};
