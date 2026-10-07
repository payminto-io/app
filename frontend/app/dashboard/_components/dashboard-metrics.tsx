"use client";

import { MetricCard } from "@/components/metric-card";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { AnalyticsSummary } from "@/lib/api/analytics";
import { formatDecimal } from "@/lib/money";

interface DashboardMetricsProps {
  data: AnalyticsSummary | undefined;
  isLoading: boolean;
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
              label="Deposit volume"
              value={formatDecimal(data.totalVolume || "0", "")}
              sublabel="Sum of filled deposits across all assets, as the API reports it"
              className="h-full"
            />
          </div>
        ) : null}

        {isLoading ? (
          <Skeleton className="h-36 rounded-xl" />
        ) : data ? (
          <MetricCard
            variant="lime"
            label="Paid payments"
            value={data.filledPayments.toLocaleString()}
            sublabel="Filled"
          />
        ) : null}

        {isLoading ? (
          <Skeleton className="h-36 rounded-xl" />
        ) : data ? (
          <MetricCard
            variant="dark"
            label="All payments"
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
          label="Active webhooks"
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
    <Card size="sm">
      <CardContent>
        <div className="text-label font-medium text-ink-soft">{label}</div>
        {loading ? (
          <Skeleton className="mt-2 h-8 w-24" />
        ) : (
          <div className="num mt-2 text-h1 font-semibold text-ink">{value}</div>
        )}
        {sublabel && (
          <p className="mt-1 text-caption text-ink-soft">{sublabel}</p>
        )}
      </CardContent>
    </Card>
  );
}
