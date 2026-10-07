"use client";

import { useQuery } from "@tanstack/react-query";
import { attestationsApi } from "@/lib/api/attestations";
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

/** Attestation module status; `data === null` means the module is off. */
export function useAttestationStatus() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.attestations.status({ platformId: scope.platformId }),
    queryFn: () => attestationsApi.status(),
    enabled: scope.ready,
    refetchInterval: 30_000,
  });
}
