"use client";

/**
 * Merchant-side misc hooks: api keys, referrals, onramper, ticker.
 * Bundled together because each domain has only a handful of calls.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiKeysApi, type APIKey, type CreateAPIKeyInput } from "@/lib/api/api-keys";
import { referralsApi, type ReferralOverview, type ReferralCampaign } from "@/lib/api/referrals";
import { onramperApi, type OnramperSession } from "@/lib/api/onramper";
import { tickerApi, type TickerEntry } from "@/lib/api/ticker";
import {
  externalPlatformsApi,
  type ExternalPlatformBlockchainCurrency,
} from "@/lib/api/external-platforms";
export type { APIKey, CreateAPIKeyInput, ReferralOverview, ReferralCampaign, OnramperSession, TickerEntry, ExternalPlatformBlockchainCurrency };
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

// API keys
export function useApiKeys() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.apiKeys.list({ platformId: scope.platformId }),
    queryFn: () => apiKeysApi.list(),
    enabled: scope.ready,
  });
}
export function useCreateApiKey() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: CreateAPIKeyInput) => apiKeysApi.create(input),
    onSuccess: () =>
      qc.invalidateQueries({
        queryKey: qk.apiKeys.list({ platformId: scope.platformId }),
      }),
  });
}
export function useRevokeApiKey() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: number) => apiKeysApi.revoke(id),
    onSuccess: () =>
      qc.invalidateQueries({
        queryKey: qk.apiKeys.list({ platformId: scope.platformId }),
      }),
  });
}

// Referrals
export function useReferralOverview() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.referrals.overview({ platformId: scope.platformId }),
    queryFn: () => referralsApi.overview(),
    enabled: scope.ready,
  });
}
export function useReferralCampaigns() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.referrals.campaigns({ platformId: scope.platformId }),
    queryFn: () => referralsApi.campaigns(),
    enabled: scope.ready,
  });
}

// Onramper
export function useOnramperList(filters?: {
  state?: string;
  limit?: number;
  offset?: number;
}) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.onramper.list({ platformId: scope.platformId }, filters),
    queryFn: () => onramperApi.list(filters),
    enabled: scope.ready,
  });
}

// Ticker — public
export function useTicker() {
  return useQuery({
    queryKey: qk.ticker.all(),
    queryFn: () => tickerApi.all(),
    staleTime: 60_000,
  });
}

// Wallet currencies — merchant's enabled blockchain currencies
export function useWalletCurrencies() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.wallets.currencies({ platformId: scope.platformId }),
    queryFn: () => externalPlatformsApi.currencies(scope.platformId),
    enabled: scope.ready,
    staleTime: 5 * 60_000,
  });
}

// Sweep stats — analytics sweep summary
