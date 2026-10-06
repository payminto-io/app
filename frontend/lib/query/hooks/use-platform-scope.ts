"use client";

import { useAuth } from "@/lib/auth/store";

/**
 * Returns a stable `{ platformId }` scope derived from the current
 * auth member. When the user is not yet loaded, platformId is 0 — the
 * query keys still work but the queries are skipped via `enabled`.
 */
export function usePlatformScope(): { platformId: number; ready: boolean } {
  const member = useAuth((s) => s.member);
  return {
    platformId: member?.externalPlatformID ?? 0,
    ready: Boolean(member?.externalPlatformID),
  };
}
