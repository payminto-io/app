"use client";

import { useAnalyticsRevenue, useAnalyticsSummary, useAnalyticsVolume } from "@/lib/query/hooks/use-analytics";
import { PageHeader } from "@/components/page-header";
import { ErrorState } from "@/components/ui/states";
import { DashboardMetrics } from "../_components/dashboard-metrics";
import { DashboardCharts } from "../_components/dashboard-charts";
import { PaidByAsset } from "../_components/paid-by-asset";

export default function AnalyticsPage() {
  const summary = useAnalyticsSummary();
  const volume = useAnalyticsVolume("day");
  const revenue = useAnalyticsRevenue();

  return (
    <div className="space-y-8">
      <PageHeader title="Analytics" />

      {summary.error ? (
        <ErrorState message={summary.error.message} retry={summary.refetch} />
      ) : (
        <DashboardMetrics data={summary.data} isLoading={!summary.data} />
      )}

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        {revenue.error ? (
          <ErrorState message={revenue.error.message} retry={revenue.refetch} />
        ) : (
          <PaidByAsset revenue={revenue.data} isLoading={!revenue.data} />
        )}
        {volume.error ? (
          <ErrorState message={volume.error.message} retry={volume.refetch} />
        ) : (
          <DashboardCharts volume={volume.data} isLoading={!volume.data} />
        )}
      </div>
    </div>
  );
}
