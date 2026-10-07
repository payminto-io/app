"use client";

import Link from "next/link";
import { MetricCard } from "@/components/metric-card";
import { Skeleton } from "@/components/ui/skeleton";
import type { AnalyticsSummary } from "@/lib/api/analytics";
import { formatDecimal } from "@/lib/money";

interface DashboardMetricsProps {
  data: AnalyticsSummary | undefined;
  isLoading: boolean;
}

/** All-time account totals from /analytics/summary. */
export function DashboardMetrics({ data, isLoading }: DashboardMetricsProps) {
  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-[108px] rounded-md last:col-span-2 sm:last:col-span-1" />
          ))}
        </div>
        <Skeleton className="h-[76px] rounded-md" />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        <MetricCard label="Paid payments" value={data.filledPayments.toLocaleString("en-US")} sublabel="All time" />
        <MetricCard label="All payments" value={data.totalPayments.toLocaleString("en-US")} sublabel="All time" />
        <MetricCard
          label="Deposit volume"
          value={formatDecimal(data.totalVolume || "0", "")}
          sublabel="All assets, summed"
          className="col-span-2 sm:col-span-1"
        />
      </div>

      <div className="grid grid-cols-3 divide-x divide-line rounded-md border border-line bg-surface">
        <Stat label="Sweeps" value={data.totalSweeps} href="/dashboard/sweeps" />
        <Stat label="Withdrawals" value={data.totalWithdrawals} href="/dashboard/withdrawals" />
        <Stat label="Active webhooks" value={data.activeWebhooks} href="/dashboard/webhooks" />
      </div>
    </div>
  );
}

function Stat({ label, value, href }: { label: string; value: number; href: string }) {
  return (
    <Link
      href={href}
      className="flex min-w-0 flex-col gap-1 px-5 py-4 transition-colors duration-120 first:rounded-l-md last:rounded-r-md hover:bg-surface-sunken/60 max-sm:px-3"
    >
      <span className="truncate text-label font-medium text-ink-soft">{label}</span>
      <span className="num text-h2 font-semibold text-ink">{value.toLocaleString("en-US")}</span>
    </Link>
  );
}
