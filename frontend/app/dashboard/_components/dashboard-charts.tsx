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
import { formatDecimal } from "@/lib/money";

const volumeChartConfig: ChartConfig = {
  value: { label: "Volume", color: "var(--chart-1)" },
};

const compact = new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 });

interface DashboardChartsProps {
  volume: VolumeBucket[] | undefined;
  isLoading: boolean;
}

/** Daily deposit volume. The API sums all assets without a currency, so no unit is shown. */
export function DashboardCharts({ volume, isLoading }: DashboardChartsProps) {
  const chartData =
    volume?.map((b) => ({
      name: b.BucketLabel,
      raw: b.Volume,
      value: Number(b.Volume),
      count: b.PaymentCount,
    })) ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Deposit volume</CardTitle>
        <CardDescription>Per day, all assets summed</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-[240px] w-full rounded-sm" />
        ) : chartData.length > 0 ? (
          <ChartContainer config={volumeChartConfig} className="aspect-auto h-[240px] w-full">
            <BarChart data={chartData} margin={{ top: 4, right: 0, bottom: 0, left: 0 }}>
              <CartesianGrid vertical={false} stroke="var(--line)" />
              <XAxis dataKey="name" tickLine={false} axisLine={false} tickMargin={8} minTickGap={16} />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                width={44}
                tickFormatter={(v: number) => compact.format(v)}
              />
              <ChartTooltip
                cursor={{ fill: "var(--surface-sunken)" }}
                content={
                  <ChartTooltipContent
                    hideIndicator
                    formatter={(_value, _name, item) => (
                      <span className="num flex flex-col">
                        <span className="font-medium text-ink">{formatDecimal(String(item.payload.raw), "")}</span>
                        <span className="text-ink-soft">
                          {item.payload.count} {item.payload.count === 1 ? "payment" : "payments"}
                        </span>
                      </span>
                    )}
                  />
                }
              />
              <Bar dataKey="value" fill="var(--color-value)" radius={[2, 2, 0, 0]} maxBarSize={28} />
            </BarChart>
          </ChartContainer>
        ) : (
          <div className="flex h-[240px] items-center justify-center rounded-sm border border-dashed border-line-strong text-body-sm text-ink-soft">
            Volume appears after the first paid payment.
          </div>
        )}
      </CardContent>
    </Card>
  );
}
