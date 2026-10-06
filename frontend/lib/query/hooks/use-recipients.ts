"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  recipientsApi,
  type Recipient,
  type CreateRecipientInput,
  type UpdateRecipientInput,
} from "@/lib/api/recipients";
export type { Recipient, CreateRecipientInput, UpdateRecipientInput };
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useRecipientsList() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.recipients.list({ platformId: scope.platformId }),
    queryFn: () => recipientsApi.list(),
    enabled: scope.ready,
  });
}

export function useRecipient(id: number | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.recipients.detail({ platformId: scope.platformId }, id ?? 0),
    queryFn: () => recipientsApi.get(id as number),
    enabled: Boolean(id) && scope.ready,
  });
}

function invalidate(qc: ReturnType<typeof useQueryClient>, platformId: number) {
  qc.invalidateQueries({ queryKey: qk.recipients.all({ platformId }) });
}

export function useCreateRecipient() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: CreateRecipientInput) => recipientsApi.create(input),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}

export function useUpdateRecipient() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: UpdateRecipientInput }) =>
      recipientsApi.update(id, input),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}

export function useDeleteRecipient() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: number) => recipientsApi.remove(id),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}
