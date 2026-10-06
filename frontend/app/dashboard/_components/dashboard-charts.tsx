"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  XAxis,
  YAxis,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import type { VolumeBucket } from "@/lib/api/analytics";

const volumeChartConfig: ChartConfig = {
  value: { label: "Volume", color: "var(--pm-primary)" },
};

interface DashboardChartsProps {
  volume: VolumeBucket[] | undefined;
  isLoading: boolean;
}

export function DashboardCharts({ volume, isLoading }: DashboardChartsProps) {
  const chartData =
    volume?.map((b) => ({
      name: b.BucketLabel,
      value: Number(b.Volume),
      count: b.PaymentCount,
    })) ?? [];

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {/* Volume bar chart */}
      <Card className="border-border shadow-none">
        <CardHeader className="pb-2">
          <CardTitle className="text-[15px] font-semibold">
            Volume Over Time
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[260px] w-full" />
          ) : chartData.length > 0 ? (
            <ChartContainer
              config={volumeChartConfig}
              className="h-[260px] w-full"
            >
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
            <div className="flex h-[260px] items-center justify-center text-sm text-muted-foreground">
              No volume data yet
            </div>
          )}
        </CardContent>
      </Card>

      {/* Payment distribution / summary */}
      <Card className="border-border shadow-none">
        <CardHeader className="pb-2">
          <CardTitle className="text-[15px] font-semibold">
            Payment Summary
          </CardTitle>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Skeleton className="h-[260px] w-full" />
          ) : chartData.length > 0 ? (
            <div className="space-y-3 pt-2">
              {chartData.map((entry) => {
                const max = Math.max(...chartData.map((d) => d.value), 1);
                const pct = Math.round((entry.value / max) * 100);
                return (
                  <div key={entry.name} className="space-y-1.5">
                    <div className="flex items-center justify-between text-[13px]">
                      <span className="text-muted-foreground">
                        {entry.name}
                        <span className="ml-1.5 text-[11px]">
                          ({entry.count} txn{entry.count !== 1 ? "s" : ""})
                        </span>
                      </span>
                      <span className="font-semibold tabular-nums">
                        ${entry.value.toLocaleString()}
                      </span>
                    </div>
                    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-[var(--pm-primary)] transition-all"
                        style={{ width: `${pct}%` }}
                      />
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="flex h-[260px] items-center justify-center text-sm text-muted-foreground">
              No data available
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
