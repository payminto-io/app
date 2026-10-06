"use client";

import { useQuery } from "@tanstack/react-query";
import {
  analyticsApi,
  type VolumeBucket,
  type RevenueBreakdown,
} from "@/lib/api/analytics";
import { qk } from "@/lib/query/keys";
import { usePlatformScope } from "./use-platform-scope";

export function useAnalyticsSummary() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.analytics.summary({ platformId: scope.platformId }, "all"),
    queryFn: () => analyticsApi.summary(),
    enabled: scope.ready,
    staleTime: 60_000,
  });
}

export function useAnalyticsVolume(
  interval: "day" | "week" | "month" = "day"
) {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.analytics.groups({ platformId: scope.platformId }),
    queryFn: () => analyticsApi.volume(interval),
    enabled: scope.ready,
    staleTime: 60_000,
  });
}

export function useAnalyticsRevenue() {
  const scope = usePlatformScope();
  return useQuery({
    queryKey: qk.analytics.data({ platformId: scope.platformId }, {}),
    queryFn: () => analyticsApi.revenue(),
    enabled: scope.ready,
    staleTime: 60_000,
  });
}

export type { VolumeBucket, RevenueBreakdown };
