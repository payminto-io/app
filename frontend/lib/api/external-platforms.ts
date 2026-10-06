/**
 * External Platforms (admin) domain API.
 * Routes: /admin/external-platforms + /admin/external-platforms/:id/currencies.
 *
 * The backend wraps the currency list in { currencies: [...] }; we unwrap
 * to keep the hook consumers simple.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface ExternalPlatform {
  id: number;
  name: string;
  websiteURL?: string;
  brandColor?: string;
  logoURL?: string;
  active: boolean;
  apiKey?: string; // revealed once on create/regenerate
  createdAt: string;
  updatedAt: string;
}

export interface CreateExternalPlatformInput {
  name: string;
  websiteURL?: string;
  brandColor?: string;
  logoURL?: string;
}

export interface ExternalPlatformBlockchainCurrency {
  id: number;
  externalPlatformID: number;
  blockchainCode: string;
  currencyCode: string;
  active: boolean;
  autoApproveThreshold?: string; // decimals — string
  hourlyCap?: string;
  dailyCap?: string;
  minAmount?: string;
  maxAmount?: string;
}

export interface UpsertEPBCInput {
  blockchainCode: string;
  currencyCode: string;
  active: boolean;
  autoApproveThreshold?: string;
  hourlyCap?: string;
  dailyCap?: string;
  minAmount?: string;
  maxAmount?: string;
}

export const externalPlatformsApi = {
  list: async (): Promise<ExternalPlatform[]> => {
    try {
      // Backend returns { platforms: [...] } — unwrap for hook consumers.
      const res = await apiFetch<{ platforms: ExternalPlatform[] }>(
        "/admin/external-platforms"
      );
      return res.platforms ?? [];
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },
  get: (id: number) =>
    apiFetch<ExternalPlatform>(`/admin/external-platforms/${id}`),
  create: (input: CreateExternalPlatformInput) =>
    apiFetch<ExternalPlatform>("/admin/external-platforms", {
      method: "POST",
      body: input,
    }),
  regenerateAPIKey: (id: number) =>
    apiFetch<{ apiKey: string }>(
      `/admin/external-platforms/${id}/regenerate-key`,
      { method: "POST" }
    ),
  currencies: async (
    id: number
  ): Promise<ExternalPlatformBlockchainCurrency[]> => {
    try {
      const res = await apiFetch<{
        currencies: ExternalPlatformBlockchainCurrency[];
      }>(`/admin/external-platforms/${id}/currencies`);
      return res.currencies ?? [];
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },
  upsertCurrency: (id: number, input: UpsertEPBCInput) =>
    apiFetch<ExternalPlatformBlockchainCurrency>(
      `/admin/external-platforms/${id}/currencies`,
      { method: "POST", body: input }
    ),
};
