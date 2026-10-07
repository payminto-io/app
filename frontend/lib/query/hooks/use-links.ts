"use client";

import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { linksApi, type LinkInput, type LinkListParams, type PaymentLink } from "@/lib/api/links";
import { feesApi, type FeePreviewRequest } from "@/lib/api/fees";
import { isApiError } from "@/lib/api/errors";
import { qk } from "@/lib/query/keys";
import type { MethodFee } from "@/features/links/preview";
import { usePlatformScope } from "./use-platform-scope";

export function useLinksList(params: LinkListParams) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.links.list({ platformId: scope.platformId }, { ...params }),
    queryFn: () => linksApi.list(params),
    enabled: scope.ready,
    placeholderData: (prev) => prev,
  });
}

export function useLink(id: string | undefined) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.links.detail({ platformId: scope.platformId }, id ?? ""),
    queryFn: () => linksApi.get(id as string),
    enabled: Boolean(id) && scope.ready,
  });
}

/** Writes the returned link into its detail cache and marks lists stale. */
function useLinkWrite<V>(fn: (v: V) => Promise<PaymentLink>) {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: fn,
    onSuccess: (link) => {
      qc.setQueryData(qk.links.detail({ platformId: scope.platformId }, link.id), link);
      qc.invalidateQueries({ queryKey: [...qk.links.all({ platformId: scope.platformId }), "list"] });
    },
  });
}

export const useCreateLink = () => useLinkWrite((input: Partial<LinkInput>) => linksApi.create(input));
export const useUpdateLink = () =>
  useLinkWrite(({ id, patch }: { id: string; patch: Partial<LinkInput> }) => linksApi.update(id, patch));
export const usePublishLink = () => useLinkWrite((id: string) => linksApi.publish(id));
export const usePauseLink = () => useLinkWrite((id: string) => linksApi.pause(id));
export const useArchiveLink = () => useLinkWrite((id: string) => linksApi.archive(id));
export const useDuplicateLink = () => useLinkWrite((id: string) => linksApi.duplicate(id));

export function useDeleteLink() {
  const qc = useQueryClient();
  const scope = usePlatformScope();
  return useMutation({
    mutationFn: (id: string) => linksApi.remove(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.links.all({ platformId: scope.platformId }) }),
  });
}

/**
 * One fee preview per method, in order. `plans` comes from `feeRequestFor`: a request to send,
 * or a state that needs no request. Refusals (404 no_fee_rule, 422 surcharge_forbidden,
 * fee_exceeds_amount) are states, not errors.
 */
export function useMethodFees(plans: ({ request: FeePreviewRequest } | { state: MethodFee })[]): MethodFee[] {
  const scope = usePlatformScope();
  const results = useQueries({
    queries: plans.map((p) => {
      const request = "request" in p ? p.request : null;
      return {
        queryKey: qk.fees.preview({ platformId: scope.platformId }, { ...(request ?? {}) }),
        queryFn: () => feesApi.preview(request as FeePreviewRequest),
        enabled: Boolean(request) && scope.ready,
        staleTime: 30_000,
        retry: false,
      };
    }),
  });
  return plans.map((p, i): MethodFee => {
    if ("state" in p) return p.state;
    const r = results[i];
    if (r.data) return { state: "ok", preview: r.data };
    if (r.error) {
      const e = r.error;
      if (isApiError(e) && (e.status === 404 || e.status === 422)) {
        const body = e.body as { code?: unknown } | undefined;
        return { state: "refused", code: typeof body?.code === "string" ? body.code : "", message: e.message };
      }
      return { state: "error", message: e instanceof Error ? e.message : String(e) };
    }
    return { state: "loading" };
  });
}
