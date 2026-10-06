"use client";

/**
 * Wallets hooks — HD wallet management (list + address pool) and cold
 * wallet configuration.
 *
 * Backed by GET /wallets, GET /wallets/:id/addresses and POST/GET
 * /wallets/cold on the Payminto backend.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  walletsApi,
  type AddressPoolItem,
  type BlockchainFamilyInfo,
  type ColdWallet,
  type ConfigureColdWalletInput,
  type HotWallet,
  type ListAddressesParams,
  type RegisterHotWalletInput,
  type Wallet,
} from "@/lib/api/wallets";
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export type {
  AddressPoolItem,
  BlockchainFamilyInfo,
  ColdWallet,
  ConfigureColdWalletInput,
  HotWallet,
  ListAddressesParams,
  RegisterHotWalletInput,
  Wallet,
};

/* ── HD wallet list ───────────────────────────────────── */

export function useWalletsList() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.wallets.list({ platformId: scope.platformId }),
    queryFn: () => walletsApi.list(),
    enabled: scope.ready,
  });
}

/* ── Wallet address pool ──────────────────────────────── */

export function useWalletAddresses(
  walletID: number | undefined,
  params?: ListAddressesParams
) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.wallets.addresses(
      { platformId: scope.platformId },
      walletID ?? 0,
      params as Record<string, unknown> | undefined
    ),
    queryFn: () => walletsApi.listAddresses(walletID as number, params),
    enabled: scope.ready && Boolean(walletID),
  });
}

/* ── Cold wallets (existing) ──────────────────────────── */

export function useColdWallets() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.wallets.cold({ platformId: scope.platformId }),
    queryFn: () => walletsApi.listCold(),
    enabled: scope.ready,
  });
}

export function useConfigureColdWallet() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: ConfigureColdWalletInput) =>
      walletsApi.configureCold(input),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: qk.wallets.cold({ platformId: scope.platformId }),
      });
    },
  });
}

/* ── Hot wallets ──────────────────────────────────────── */

export function useHotWallets() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.wallets.hot({ platformId: scope.platformId }),
    queryFn: () => walletsApi.listHot(),
    enabled: scope.ready,
  });
}

/**
 * Live native balance for a single hot wallet, fetched lazily per card so the
 * wallet list itself never blocks on RPC. Refreshes every 30s; tolerant of
 * transient RPC errors (returns the error without breaking the list).
 */
export function useHotWalletBalance(walletID: number, enabled = true) {
  return useQuery({
    queryKey: ["wallets", "hot", "balance", walletID],
    queryFn: () => walletsApi.hotBalance(walletID),
    enabled,
    staleTime: 15_000,
    refetchInterval: 30_000,
    retry: 1,
  });
}

export function useRegisterHotWallet() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: RegisterHotWalletInput) =>
      walletsApi.registerHot(input),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: qk.wallets.hot({ platformId: scope.platformId }),
      });
    },
  });
}
