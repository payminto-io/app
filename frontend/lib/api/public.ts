/**
 * Public (unauthenticated) domain API for the customer checkout widget.
 * Routes: /public/payment/:reference_id (Phase K).
 */
import { apiFetch } from "./client";

export interface PublicPayment {
  referenceID: string;
  amountInUSD: string;
  blockchainCode: string;
  currencyCode: string;
  state: string;
  depositAddress?: string;
  expiresAt?: string;
  description?: string;
  merchantName?: string;
}

export interface BlockchainCurrencyOption {
  id: number;
  blockchainCode: string;
  currencyCode: string;
  standard: string;
  address: string;
  blockchain?: { code: string; name: string };
  currency?: { code: string; name: string; iconURL?: string };
}

export interface AssignDepositAddressResponse {
  address: string;
  blockchainCode: string;
  currencyCode: string;
}

export const publicApi = {
  payment: (referenceID: string) =>
    apiFetch<PublicPayment>(
      `/public/payment/${encodeURIComponent(referenceID)}`
    ),

  assignDepositAddress: (
    referenceId: string,
    blockchainCode: string,
    currencyCode?: string
  ) =>
    apiFetch<AssignDepositAddressResponse>(
      `/public/deposit-address/reference/${encodeURIComponent(referenceId)}`,
      { method: "POST", body: { blockchainCode, currencyCode } }
    ),

  blockchainCurrencies: () =>
    apiFetch<{ currencies: BlockchainCurrencyOption[] }>(
      "/public/blockchain-currencies"
    ),
};
