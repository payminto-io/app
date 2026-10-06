"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  paymentsApi,
  type Payment,
  type PaymentsListResponse,
  type CreatePaymentInput,
} from "@/lib/api/payments";
export type { Payment, PaymentsListResponse, CreatePaymentInput };
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function usePaymentsList(filters?: {
  state?: string;
  limit?: number;
  offset?: number;
}) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.payments.list({ platformId: scope.platformId }, filters),
    queryFn: () => paymentsApi.list(filters),
    enabled: scope.ready,
  });
}

export function usePayment(referenceID: string | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.payments.detail(
      { platformId: scope.platformId },
      referenceID ?? ""
    ),
    queryFn: () => paymentsApi.get(referenceID as string),
    enabled: Boolean(referenceID) && scope.ready,
    refetchInterval: 5000,
  });
}

export function useCreatePayment() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: CreatePaymentInput) => paymentsApi.create(input),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: qk.payments.all({ platformId: scope.platformId }),
      });
    },
  });
}
