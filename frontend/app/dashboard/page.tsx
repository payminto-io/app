"use client";

import Link from "next/link";
import { Plus, FileText } from "lucide-react";
import {
  useAnalyticsSummary,
  useAnalyticsVolume,
} from "@/lib/query/hooks/use-analytics";
import { usePaymentsList } from "@/lib/query/hooks/use-payments";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import { DashboardMetrics } from "./_components/dashboard-metrics";
import { DashboardCharts } from "./_components/dashboard-charts";
import { RecentPaymentsTable } from "./_components/recent-payments-table";

export default function DashboardPage() {
  const summary = useAnalyticsSummary();
  const volume = useAnalyticsVolume("day");
  const payments = usePaymentsList({ limit: 5 });

  if (summary.error) {
    return <ErrorState message={summary.error.message} retry={summary.refetch} />;
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Home"
        description="Payments, settlement and integrations for this account."
      >
        <Link href="/dashboard/payments">
          <Button variant="outline">
            <FileText />
            Payments
          </Button>
        </Link>
        <Link href="/dashboard/payments/create">
          <Button>
            <Plus />
            Create payment link
          </Button>
        </Link>
      </PageHeader>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-h2 font-semibold text-ink">Account activity</h2>
          <span className="text-caption text-ink-soft">All time</span>
        </div>
      <DashboardMetrics data={summary.data} isLoading={summary.isLoading} />
      </div>

      <div className="flex items-center justify-between gap-4 pt-1">
        <div>
          <h2 className="text-h2 font-semibold text-ink">Volume</h2>
          <p className="mt-0.5 text-body-sm text-ink-soft">Deposit volume per day</p>
        </div>
      </div>

      {/* Charts */}
      <DashboardCharts
        volume={volume.data}
        isLoading={volume.isLoading}
      />

      {/* Recent payments */}
      <div className="grid gap-4 lg:grid-cols-3">
        <RecentPaymentsTable
          payments={payments.data?.payments}
          isLoading={payments.isLoading}
        />
      </div>
    </div>
  );
}
