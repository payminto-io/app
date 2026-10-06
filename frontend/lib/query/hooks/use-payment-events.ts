"use client";

import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { qk } from "@/lib/query/keys";
import {
  subscribeToPaymentEvents,
  type PaymentEvent,
} from "@/lib/realtime/payment-events";

/**
 * Subscribes to the real-time payment stream for a checkout and pushes incoming
 * state changes into the TanStack Query cache so the existing `usePublicPayment`
 * query (and any component reading it) updates instantly — no manual refetch.
 *
 * The slow reconciliation poll on `usePublicPayment` remains as a safety net for
 * dropped connections or missed frames; SSE is the primary, instant path.
 *
 * @returns `connected` — whether the SSE stream is currently open.
 */
export function usePaymentEvents(referenceId: string | undefined): {
  connected: boolean;
} {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    if (!referenceId) return;

    const dispose = subscribeToPaymentEvents(referenceId, {
      onOpen: () => setConnected(true),
      onError: () => setConnected(false),
      onEvent: (event: PaymentEvent) => {
        queryClient.setQueryData(
          qk.public.payment(referenceId),
          (prev: Record<string, unknown> | undefined) =>
            prev ? { ...prev, state: event.state } : prev
        );
        // Pull the full enriched projection (deposit address, etc.) once.
        queryClient.invalidateQueries({
          queryKey: qk.public.payment(referenceId),
        });
      },
    });

    return () => {
      dispose();
      setConnected(false);
    };
  }, [referenceId, queryClient]);

  return { connected };
}
