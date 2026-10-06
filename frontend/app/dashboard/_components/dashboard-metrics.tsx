"use client";

import { MetricCard } from "@/components/metric-card";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { AnalyticsSummary } from "@/lib/api/analytics";

interface DashboardMetricsProps {
  data: AnalyticsSummary | undefined;
  isLoading: boolean;
}

function formatCompact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(2)}K`;
  return n.toFixed(2);
}

export function DashboardMetrics({ data, isLoading }: DashboardMetricsProps) {
  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {isLoading ? (
          <Skeleton className="h-36 rounded-xl sm:col-span-2" />
        ) : data ? (
          <div className="sm:col-span-2">
            <MetricCard
              variant="primary"
              label="Total Volume"
              value={`$${formatCompact(parseFloat(data.totalVolume || "0"))}`}
              sublabel={`${data.totalPayments.toLocaleString()} total payments`}
              className="h-full"
            />
          </div>
        ) : null}

        {isLoading ? (
          <Skeleton className="h-36 rounded-xl" />
        ) : data ? (
          <MetricCard
            variant="lime"
            label="Filled Payments"
            value={data.filledPayments.toLocaleString()}
            sublabel="Confirmed"
          />
        ) : null}

        {isLoading ? (
          <Skeleton className="h-36 rounded-xl" />
        ) : data ? (
          <MetricCard
            variant="dark"
            label="Total Payments"
            value={data.totalPayments.toLocaleString()}
            sublabel="All time"
          />
        ) : null}
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <SecondaryMetric
          label="Sweeps"
          value={data ? String(data.totalSweeps) : "\u2014"}
          loading={isLoading}
        />
        <SecondaryMetric
          label="Withdrawals"
          value={data ? String(data.totalWithdrawals) : "\u2014"}
          loading={isLoading}
        />
        <SecondaryMetric
          label="Active Webhooks"
          value={data ? String(data.activeWebhooks) : "\u2014"}
          loading={isLoading}
        />
      </div>
    </>
  );
}

function SecondaryMetric({
  label,
  value,
  sublabel,
  loading,
}: {
  label: string;
  value: string;
  sublabel?: string;
  loading: boolean;
}) {
  return (
    <Card className="border-border shadow-none">
      <CardContent className="p-5">
        <div className="pm-label">{label}</div>
        {loading ? (
          <Skeleton className="h-8 w-24 mt-3" />
        ) : (
          <div className="mt-3 text-[28px] font-bold tracking-tight tabular-nums">
            {value}
          </div>
        )}
        {sublabel && (
          <p className="mt-1 text-[12px] text-muted-foreground">{sublabel}</p>
        )}
      </CardContent>
    </Card>
  );
}
