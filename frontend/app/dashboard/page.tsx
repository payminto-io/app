"use client";

import Link from "next/link";
import { Plus } from "lucide-react";
import { useAnalyticsRevenue, useAnalyticsSummary, useAnalyticsVolume } from "@/lib/query/hooks/use-analytics";
import { usePaymentsList } from "@/lib/query/hooks/use-payments";
import { PageHeader } from "@/components/page-header";
import { buttonVariants } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/states";
import { DashboardMetrics } from "./_components/dashboard-metrics";
import { DashboardCharts } from "./_components/dashboard-charts";
import { PaidByAsset } from "./_components/paid-by-asset";
import { RecentPaymentsTable } from "./_components/recent-payments-table";

export default function DashboardPage() {
  const summary = useAnalyticsSummary();
  const volume = useAnalyticsVolume("day");
  const revenue = useAnalyticsRevenue();
  const payments = usePaymentsList({ limit: 5 });

  return (
    <div className="space-y-8">
      <PageHeader title="Home">
        <Link href="/dashboard/payments/create" className={buttonVariants()}>
          <Plus />
          Create payment
        </Link>
      </PageHeader>

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

      {payments.error ? (
        <ErrorState message={payments.error.message} retry={payments.refetch} />
      ) : (
        <RecentPaymentsTable payments={payments.data?.payments} isLoading={!payments.data} />
      )}
    </div>
  );
}
