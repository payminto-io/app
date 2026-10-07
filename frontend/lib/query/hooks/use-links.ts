"use client";

import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { linksApi, type LinkInput, type LinkListParams, type PaymentLink } from "@/lib/api/links";
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useLinksList(params: LinkListParams) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.links.list({ platformId: scope.platformId }, { ...params }),
    queryFn: () => linksApi.list(params),
    enabled: scope.ready,
    placeholderData: keepPreviousData,
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

/** Methods and currencies the form may offer in this environment. */
export function useLinkOptions() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.links.options({ platformId: scope.platformId }),
    queryFn: () => linksApi.options(),
    enabled: scope.ready,
    staleTime: 60_000,
  });
}

/**
 * The server's render model for `input` (null while the form cannot be sent). One request per settled
 * body; the previous complete response stays on screen, flagged `isPlaceholderData`, until the new one lands.
 */
export function useLinkPreview(input: LinkInput | null, linkId: string | null) {
  const scope = usePlatformScope();
  const body = input ? JSON.stringify(input) : "";
  return useQuery({
    queryKey: qk.links.preview({ platformId: scope.platformId }, body, linkId ?? ""),
    queryFn: () => linksApi.preview(input as LinkInput, linkId ?? undefined),
    enabled: scope.ready && input !== null,
    placeholderData: keepPreviousData,
    staleTime: 15_000,
    retry: false,
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
