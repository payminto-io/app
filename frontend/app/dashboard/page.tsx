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
    <div className="space-y-7">
      <section className="relative overflow-hidden rounded-2xl bg-[linear-gradient(112deg,#c91e1e_0%,#e22323_42%,#f06b28_72%,#e96991_100%)] px-6 py-7 text-white shadow-[0_12px_30px_rgba(165,28,36,.14)] sm:px-8">
        <div className="pointer-events-none absolute -right-20 -top-24 size-64 rounded-full border border-white/15" />
        <div className="pointer-events-none absolute -right-5 -top-14 size-40 rounded-full border border-white/15" />
        <div className="relative flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-[10px] font-bold uppercase tracking-[.16em] text-white/70">
              Merchant overview
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-[-.045em] text-white sm:text-[38px]">
              Payments at a glance
            </h1>
            <p className="mt-2 max-w-xl text-sm leading-6 text-white/75">
              Monitor incoming payments, settlement activity, and integrations
              from one workspace.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Link href="/dashboard/payments/create">
              <Button
                size="lg"
                className="h-10 rounded-xl bg-[#17152f] px-4 text-white shadow-lg hover:bg-black"
              >
                <Plus className="size-4" />
                Create payment
              </Button>
            </Link>
            <Link href="/dashboard/payments">
              <Button
                variant="outline"
                size="lg"
                className="h-10 rounded-xl border-white/35 bg-white/10 px-4 text-white backdrop-blur hover:bg-white hover:text-[#17152f]"
              >
                <FileText className="size-4" />
                View reports
              </Button>
            </Link>
          </div>
        </div>
      </section>

      <div>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <p className="pm-label">Performance</p>
            <h2 className="mt-1 text-lg font-semibold tracking-[-.025em]">
              Account activity
            </h2>
          </div>
          <span className="rounded-full border border-border bg-white px-3 py-1.5 text-[10px] font-semibold text-muted-foreground">
            All time
          </span>
        </div>
      <DashboardMetrics data={summary.data} isLoading={summary.isLoading} />
      </div>

      <div className="flex items-center justify-between gap-4 pt-1">
        <div>
          <p className="pm-label">Analytics</p>
          <h2 className="mt-1 text-lg font-semibold tracking-[-.025em] text-foreground">
            Transaction Summary
          </h2>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Payment activity overview
          </p>
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
