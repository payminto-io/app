"use client";

import Link from "next/link";
import { MetricCard } from "@/components/metric-card";
import { Skeleton } from "@/components/ui/skeleton";
import type { AnalyticsSummary } from "@/lib/api/analytics";

interface DashboardMetricsProps {
  data: AnalyticsSummary | undefined;
  isLoading: boolean;
}

/**
 * All-time account counts from /analytics/summary. `TotalVolume` sums every
 * asset into one unitless number, so it is not shown; money is per asset in `PaidByAsset`.
 */
export function DashboardMetrics({ data, isLoading }: DashboardMetricsProps) {
  if (isLoading || !data) {
    return (
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-[108px] rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <MetricCard label="Paid payments" value={data.filledPayments.toLocaleString("en-US")} sublabel="All time" />
      <MetricCard label="All payments" value={data.totalPayments.toLocaleString("en-US")} sublabel="All time" />
      <LinkMetric label="Withdrawals" value={data.totalWithdrawals} href="/dashboard/withdrawals" />
      <LinkMetric label="Active webhooks" value={data.activeWebhooks} href="/dashboard/webhooks" />
    </div>
  );
}

function LinkMetric({ label, value, href }: { label: string; value: number; href: string }) {
  return (
    <Link
      href={href}
      className="flex min-w-0 flex-col gap-2 rounded-md border border-line bg-surface p-5 transition-colors duration-120 hover:bg-surface-sunken/60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide max-sm:p-4"
    >
      <span className="truncate text-label font-medium text-ink-soft">{label}</span>
      <span className="num text-h1 font-semibold text-ink">{value.toLocaleString("en-US")}</span>
      <span className="text-caption text-ink-soft">All time</span>
    </Link>
  );
}
