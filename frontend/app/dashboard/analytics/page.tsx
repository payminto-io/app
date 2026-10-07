"use client";

import { Line, LineChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { useAnalyticsSummary, useAnalyticsVolume } from "@/lib/query/hooks/use-analytics";
import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { DashboardMetrics } from "../_components/dashboard-metrics";
import { DashboardCharts } from "../_components/dashboard-charts";

const countConfig: ChartConfig = {
  count: { label: "Payments", color: "var(--chart-1)" },
};

export default function AnalyticsPage() {
  const summary = useAnalyticsSummary();
  const volume = useAnalyticsVolume("day");

  const countData = volume.data?.map((b) => ({ name: b.BucketLabel, count: b.PaymentCount })) ?? [];

  return (
    <div className="space-y-8">
      <PageHeader title="Analytics" />

      {summary.error ? (
        <ErrorState message={summary.error.message} retry={summary.refetch} />
      ) : (
        <DashboardMetrics data={summary.data} isLoading={!summary.data} />
      )}

      {volume.error ? (
        <ErrorState message={volume.error.message} retry={volume.refetch} />
      ) : (
        <div className="grid gap-6 xl:grid-cols-2">
          <DashboardCharts volume={volume.data} isLoading={!volume.data} />
          <Card>
            <CardHeader>
              <CardTitle>Payments</CardTitle>
              <CardDescription>Per day</CardDescription>
            </CardHeader>
            <CardContent>
              {!volume.data ? (
                <Skeleton className="h-[240px] w-full rounded-sm" />
              ) : countData.length > 0 ? (
                <ChartContainer config={countConfig} className="aspect-auto h-[240px] w-full">
                  <LineChart data={countData} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
                    <CartesianGrid vertical={false} stroke="var(--line)" />
                    <XAxis dataKey="name" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
                    <YAxis tickLine={false} axisLine={false} tickMargin={8} width={32} allowDecimals={false} />
                    <ChartTooltip cursor={{ stroke: "var(--line-strong)" }} content={<ChartTooltipContent />} />
                    <Line type="linear" dataKey="count" stroke="var(--color-count)" strokeWidth={1.75} dot={false} />
                  </LineChart>
                </ChartContainer>
              ) : (
                <div className="flex h-[240px] items-center justify-center rounded-sm border border-dashed border-line-strong text-body-sm text-ink-soft">
                  Payments appear here after the first one is created.
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </div>
  );
}
