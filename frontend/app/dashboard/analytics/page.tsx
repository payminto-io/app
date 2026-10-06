"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  XAxis,
  YAxis,
  Line,
  LineChart,
} from "recharts";
import {
  useAnalyticsSummary,
  useAnalyticsVolume,
} from "@/lib/query/hooks/use-analytics";
import { ErrorState } from "@/components/ui/states";
import { MetricCard } from "@/components/metric-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

const volumeConfig: ChartConfig = {
  value: { label: "Volume", color: "var(--pm-primary)" },
};

const lineConfig: ChartConfig = {
  value: { label: "Payments", color: "var(--pm-blue)" },
};

export default function AnalyticsPage() {
  const { data, isLoading, error, refetch } = useAnalyticsSummary();
  const volume = useAnalyticsVolume("day");

  const chartData =
    volume.data?.map((b) => ({
      name: b.BucketLabel,
      value: Number(b.Volume),
      count: b.PaymentCount,
    })) ?? [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Analytics</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Payment volume and performance metrics
          </p>
        </div>
      </div>

      {/* Metric cards */}
      {isLoading ? (
        <div className="grid gap-3 grid-cols-1 sm:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-28 rounded-xl" />
          ))}
        </div>
      ) : error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : data ? (
        <div className="grid gap-3 grid-cols-1 sm:grid-cols-3">
          <MetricCard
            variant="primary"
            label="Total Volume"
            value={`$${data.totalVolume}`}
          />
          <MetricCard
            variant="lime"
            label="Filled Payments"
            value={String(data.filledPayments)}
          />
          <MetricCard
            variant="dark"
            label="Total Payments"
            value={String(data.totalPayments)}
          />
        </div>
      ) : null}

      {/* Charts */}
      <div className="grid gap-4 lg:grid-cols-2">
        {/* Bar chart */}
        <Card className="border-border shadow-sm">
          <CardHeader className="pb-2">
            <CardTitle className="text-[15px] font-semibold">
              Volume Over Time
            </CardTitle>
          </CardHeader>
          <CardContent>
            {volume.isLoading ? (
              <Skeleton className="h-[280px] w-full" />
            ) : chartData.length > 0 ? (
              <ChartContainer config={volumeConfig} className="h-[280px] w-full">
                <BarChart
                  data={chartData}
                  margin={{ top: 4, right: 4, bottom: 0, left: 0 }}
                >
                  <CartesianGrid
                    vertical={false}
                    strokeDasharray="3 3"
                    className="stroke-border/40"
                  />
                  <XAxis
                    dataKey="name"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    className="text-[11px]"
                  />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    tickFormatter={(v: number) =>
                      v >= 1000 ? `$${(v / 1000).toFixed(1)}k` : `$${v}`
                    }
                    className="text-[11px]"
                  />
                  <ChartTooltip
                    content={
                      <ChartTooltipContent
                        formatter={(value) => (
                          <span className="font-semibold">
                            ${Number(value).toLocaleString()}
                          </span>
                        )}
                      />
                    }
                  />
                  <Bar
                    dataKey="value"
                    fill="var(--pm-primary)"
                    radius={[4, 4, 0, 0]}
                    maxBarSize={32}
                  />
                </BarChart>
              </ChartContainer>
            ) : (
              <div className="flex h-[280px] items-center justify-center text-sm text-muted-foreground">
                No chart data available
              </div>
            )}
          </CardContent>
        </Card>

        {/* Line chart */}
        <Card className="border-border shadow-sm">
          <CardHeader className="pb-2">
            <CardTitle className="text-[15px] font-semibold">
              Payment Trend
            </CardTitle>
          </CardHeader>
          <CardContent>
            {volume.isLoading ? (
              <Skeleton className="h-[280px] w-full" />
            ) : chartData.length > 0 ? (
              <ChartContainer config={lineConfig} className="h-[280px] w-full">
                <LineChart
                  data={chartData}
                  margin={{ top: 4, right: 4, bottom: 0, left: 0 }}
                >
                  <CartesianGrid
                    vertical={false}
                    strokeDasharray="3 3"
                    className="stroke-border/40"
                  />
                  <XAxis
                    dataKey="name"
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    className="text-[11px]"
                  />
                  <YAxis
                    tickLine={false}
                    axisLine={false}
                    tickMargin={8}
                    className="text-[11px]"
                  />
                  <ChartTooltip
                    content={<ChartTooltipContent />}
                  />
                  <Line
                    type="monotone"
                    dataKey="value"
                    stroke="var(--pm-blue)"
                    strokeWidth={2}
                    dot={{ r: 3 }}
                  />
                </LineChart>
              </ChartContainer>
            ) : (
              <div className="flex h-[280px] items-center justify-center text-sm text-muted-foreground">
                No trend data available
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
