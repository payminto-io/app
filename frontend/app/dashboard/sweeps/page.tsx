"use client";

import { useSweepStats } from "@/lib/query/hooks/use-merchant-misc";
import { formatDecimal } from "@/lib/money";
import { PageHeader } from "@/components/page-header";
import { MetricCard } from "@/components/metric-card";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";

/** Sweep totals from /analytics/sweeps. The API sums every sweep without a currency, so no unit is shown. */
export default function SweepsPage() {
  const { data: stats, error, refetch } = useSweepStats();

  return (
    <div className="space-y-5">
      <PageHeader title="Sweeps" />

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : !stats ? (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-[108px] rounded-md first:col-span-2 sm:first:col-span-1" />
          ))}
        </div>
      ) : stats.TotalSweeps === 0 ? (
        <EmptyState
          title="No sweeps yet."
          description="Deposits are swept to the cold wallet once a hot wallet can pay the gas."
        />
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <MetricCard
            label="Sweeps"
            value={stats.TotalSweeps.toLocaleString("en-US")}
            sublabel="All time"
            className="col-span-2 sm:col-span-1"
          />
          <MetricCard label="Swept" value={formatDecimal(stats.TotalSwept || "0", "")} sublabel="All assets, summed" />
          <MetricCard label="Gas paid" value={formatDecimal(stats.TotalGas || "0", "")} sublabel="All chains, summed" />
        </div>
      )}
    </div>
  );
}
