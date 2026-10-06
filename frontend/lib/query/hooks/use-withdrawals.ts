"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  withdrawalsApi,
  type Withdrawal,
  type WithdrawalsListResponse,
  type WithdrawalState,
  type CreateWithdrawalInput,
} from "@/lib/api/withdrawals";
export type { Withdrawal, WithdrawalsListResponse, WithdrawalState, CreateWithdrawalInput };
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useWithdrawalsList() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.withdrawals.list({ platformId: scope.platformId }),
    queryFn: () => withdrawalsApi.list(),
    enabled: scope.ready,
  });
}

export function useWithdrawal(id: number | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.withdrawals.detail(
      { platformId: scope.platformId },
      id ?? 0
    ),
    queryFn: () => withdrawalsApi.get(id as number),
    enabled: Boolean(id) && scope.ready,
  });
}

function invalidateList(qc: ReturnType<typeof useQueryClient>, scope: { platformId: number }) {
  qc.invalidateQueries({ queryKey: qk.withdrawals.all(scope) });
}

export function useCreateWithdrawal() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: CreateWithdrawalInput) =>
      withdrawalsApi.create(input),
    onSuccess: () => invalidateList(qc, { platformId: scope.platformId }),
  });
}

export function useVerifyWithdrawalOTP(id: number) {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (code: string) => withdrawalsApi.verifyOTP(id, code),
    onSuccess: () => invalidateList(qc, { platformId: scope.platformId }),
  });
}

export function useApproveWithdrawal() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: number) => withdrawalsApi.approve(id),
    onSuccess: () => invalidateList(qc, { platformId: scope.platformId }),
  });
}

export function useCancelWithdrawal() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: number) => withdrawalsApi.cancel(id),
    onSuccess: () => invalidateList(qc, { platformId: scope.platformId }),
  });
}
