/**
 * Wallets domain API.
 *
 * DTO source of truth: payminto/backend/internal/api/dto/wallet_dto.go
 * Routes: see payminto-backend-routes.md — /wallets, /wallets/:id/addresses,
 * /wallets/cold, etc.
 */
import { apiFetch } from "./client";

/* ── Cold wallet config (existing) ──────────────────────── */

export interface ColdWallet {
  blockchainCode: string;
  address: string;
  name: string;
}

export interface ConfigureColdWalletInput {
  blockchainCode: string;
  address: string;
  name: string;
}

export interface ConfigureColdWalletResponse {
  configured: boolean;
  blockchainCode: string;
  address: string;
}

/* ── Wallet management (HD wallets + address pools) ─────── */

export interface BlockchainFamilyInfo {
  id: number;
  name: string;
  code: string;
  family: string;
  path: string;
  supportsHDWallet: boolean;
  supportsSCWallet: boolean;
}

export interface Wallet {
  id: number;
  name: string;
  kind: string;
  status: string;
  blockchainFamilyID: number;
  blockchainFamily?: BlockchainFamilyInfo;
  addressCount: number;
  createdAt: string;
}

export interface AddressPoolItem {
  id: number;
  address: string;
  pathIndex: number;
  status: "available" | "used" | "locked";
  walletID: number;
  blockchainFamilyID: number;
  createdAt: string;
}

export interface ListAddressesParams {
  status?: string;
  limit?: number;
  offset?: number;
}

/* ── Hot wallets (gas-fee wallets) ──────────────────────── */

/**
 * Hot wallet as returned by GET /wallets/hot. The private key is never
 * returned by the backend — it only lives in the SecretsVault. The public
 * address is pulled from a configuration row keyed by wallet id.
 */
export interface HotWallet {
  id: number;
  name: string;
  kind: string;
  status: string;
  blockchainFamilyID: number;
  blockchainFamily?: BlockchainFamilyInfo;
  blockchainFamilyCode?: string;
  address?: string;
  createdAt: string;
}

export interface RegisterHotWalletInput {
  blockchainFamilyCode: string;
  privateKey: string;
  name: string;
  address: string;
}

export interface RegisterHotWalletResponse {
  id: number;
  kind: string;
  address: string;
  blockchainFamilyCode: string;
}

export const walletsApi = {
  /* Cold wallets ──────────── */
  listCold: () =>
    apiFetch<{ coldWallets: ColdWallet[] }>("/wallets/cold"),

  configureCold: (input: ConfigureColdWalletInput) =>
    apiFetch<ConfigureColdWalletResponse>("/wallets/cold", {
      method: "POST",
      body: input,
    }),

  /* HD wallet management ─── */
  list: () => apiFetch<{ wallets: Wallet[] }>("/wallets"),

  listAddresses: (walletID: number, params?: ListAddressesParams) => {
    const query = new URLSearchParams();
    if (params?.status) query.set("status", params.status);
    if (params?.limit !== undefined) query.set("limit", String(params.limit));
    if (params?.offset !== undefined) query.set("offset", String(params.offset));
    const suffix = query.toString() ? `?${query.toString()}` : "";
    return apiFetch<{ addresses: AddressPoolItem[]; total: number }>(
      `/wallets/${walletID}/addresses${suffix}`
    );
  },

  /* Hot wallets ───────────── */
  listHot: () => apiFetch<{ hotWallets: HotWallet[] }>("/wallets/hot"),

  registerHot: (input: RegisterHotWalletInput) =>
    apiFetch<RegisterHotWalletResponse>("/wallets/hot", {
      method: "POST",
      body: input,
    }),

  /** Live on-chain native balance for a hot wallet, fetched lazily per card. */
  hotBalance: (walletID: number) =>
    apiFetch<HotWalletBalance>(`/wallets/hot/${walletID}/balance`),
};

/** Live native balance of a hot wallet's address. */
export interface HotWalletBalance {
  address: string;
  chain: string;
  symbol: string;
  balance: string; // decimal string in whole coins
}
