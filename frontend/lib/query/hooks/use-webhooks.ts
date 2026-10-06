"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  webhooksApi,
  type CreatedWebhook,
  type RedactedWebhook,
  type WebhookDelivery,
  type CreateWebhookInput,
  type UpdateWebhookInput,
} from "@/lib/api/webhooks";
export type {
  CreatedWebhook,
  RedactedWebhook,
  WebhookDelivery,
  CreateWebhookInput,
  UpdateWebhookInput,
};
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useWebhooksList() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.webhooks.list({ platformId: scope.platformId }),
    queryFn: () => webhooksApi.list(),
    enabled: scope.ready,
  });
}

export function useWebhook(id: number | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.webhooks.detail({ platformId: scope.platformId }, id ?? 0),
    queryFn: () => webhooksApi.get(id as number),
    enabled: Boolean(id) && scope.ready,
  });
}

export function useWebhookDeliveries(id: number | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.webhooks.deliveries(
      { platformId: scope.platformId },
      id ?? 0
    ),
    queryFn: () => webhooksApi.deliveries(id as number),
    enabled: Boolean(id) && scope.ready,
  });
}

function invalidate(qc: ReturnType<typeof useQueryClient>, platformId: number) {
  qc.invalidateQueries({ queryKey: qk.webhooks.all({ platformId }) });
}

export function useCreateWebhook() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (input: CreateWebhookInput) => webhooksApi.create(input),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}

export function useUpdateWebhook() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: UpdateWebhookInput }) =>
      webhooksApi.update(id, input),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}

export function useDeleteWebhook() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: number) => webhooksApi.remove(id),
    onSuccess: () => invalidate(qc, scope.platformId),
  });
}
