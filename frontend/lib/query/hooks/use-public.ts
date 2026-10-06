"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  publicApi,
  type PublicPayment,
  type BlockchainCurrencyOption,
  type AssignDepositAddressResponse,
} from "@/lib/api/public";
import { tickerApi, type PublicTickerResponse } from "@/lib/api/ticker";
import { qk } from "@/lib/query/keys";
import { INTERVALS } from "@/lib/constants";

export type {
  PublicPayment,
  BlockchainCurrencyOption,
  AssignDepositAddressResponse,
  PublicTickerResponse,
};

export function usePublicPayment(referenceId: string | undefined) {
  return useQuery({
    queryKey: qk.public.payment(referenceId ?? ""),
    queryFn: () => publicApi.payment(referenceId as string),
    enabled: Boolean(referenceId),
    // Real-time SSE (see usePaymentEvents) drives instant updates; this slow
    // poll is only a reconciliation fallback for dropped connections.
    refetchInterval: INTERVALS.checkoutFallbackPoll,
  });
}

export function useBlockchainCurrencies() {
  return useQuery({
    queryKey: qk.public.blockchainCurrencies(),
    queryFn: () => publicApi.blockchainCurrencies(),
    staleTime: 60_000,
  });
}

/**
 * Fetches spot USD prices for one or more symbols from the public ticker.
 * Used by the checkout page to convert `amountInUSD` into the exact crypto
 * amount the customer must send. Stable over ~60s; no auth required.
 */
export function usePublicTicker(symbols: string[]) {
  const key = symbols.filter(Boolean).map((s) => s.toUpperCase()).sort();
  return useQuery({
    queryKey: ["public", "ticker", key],
    queryFn: () => tickerApi.prices(key),
    enabled: key.length > 0,
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function useAssignDepositAddress(referenceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      blockchainCode,
      currencyCode,
    }: {
      blockchainCode: string;
      currencyCode?: string;
    }) => publicApi.assignDepositAddress(referenceId, blockchainCode, currencyCode),
    onSuccess: () => {
      // Refetch the payment so it picks up the new deposit address.
      queryClient.invalidateQueries({
        queryKey: qk.public.payment(referenceId),
      });
    },
  });
}
