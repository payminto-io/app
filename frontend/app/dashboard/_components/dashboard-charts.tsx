"use client";

import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import type { VolumeBucket } from "@/lib/api/analytics";

const countChartConfig: ChartConfig = {
  count: { label: "Payments", color: "var(--chart-1)" },
};

interface DashboardChartsProps {
  volume: VolumeBucket[] | undefined;
  isLoading: boolean;
}

/** Payments with a confirmed deposit per day. The buckets' `Volume` sums every asset, so it is not plotted. */
export function DashboardCharts({ volume, isLoading }: DashboardChartsProps) {
  const chartData = volume?.map((b) => ({ name: b.BucketLabel, count: b.PaymentCount })) ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Payments received</CardTitle>
        <CardDescription>Payments with a confirmed deposit, per day, last 30 days</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[240px] w-full rounded-sm" />
        ) : chartData.length > 0 ? (
          <ChartContainer config={countChartConfig} className="aspect-auto h-[240px] w-full">
            <BarChart data={chartData} margin={{ top: 4, right: 0, bottom: 0, left: 0 }}>
              <CartesianGrid vertical={false} stroke="var(--line)" />
              <XAxis dataKey="name" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
              <YAxis tickLine={false} axisLine={false} tickMargin={8} width={32} allowDecimals={false} />
              <ChartTooltip cursor={{ fill: "var(--surface-sunken)" }} content={<ChartTooltipContent />} />
              <Bar dataKey="count" fill="var(--color-count)" radius={[2, 2, 0, 0]} maxBarSize={28} />
            </BarChart>
          </ChartContainer>
        ) : (
          <div className="flex h-[240px] items-center justify-center rounded-sm border border-dashed border-line-strong text-body-sm text-ink-soft">
            Payments appear here after the first confirmed deposit.
          </div>
        )}
      </CardContent>
    </Card>
  );
}
