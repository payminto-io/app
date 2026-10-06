"use client";

import { ArrowDownToLine, Zap, Info } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState, EmptyState } from "@/components/ui/states";
import { useSweepStats } from "@/lib/query/hooks/use-merchant-misc";

export default function SweepsPage() {
  const { data: stats, isLoading, error, refetch } = useSweepStats();

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">SmartSweep</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Auto-consolidation from deposit wallets to cold storage
          </p>
        </div>
        <Button variant="outline" size="sm" className="h-9 rounded-lg" disabled>
          <Zap className="size-4" />
          Trigger Sweep
        </Button>
      </div>

      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
      ) : stats ? (
        <>
          {/* Summary stats */}
          <div className="grid gap-4 md:grid-cols-3">
            <Card className="border-border shadow-sm">
              <CardContent className="p-4">
                <span className="text-[12px] text-muted-foreground">Total Sweeps</span>
                <p className="text-[20px] font-bold tabular-nums mt-1">
                  {stats.TotalSweeps}
                </p>
              </CardContent>
            </Card>
            <Card className="border-border shadow-sm">
              <CardContent className="p-4">
                <span className="text-[12px] text-muted-foreground">Total Swept</span>
                <p className="text-[20px] font-bold tabular-nums mt-1">
                  {stats.TotalSwept !== "0" ? `$${stats.TotalSwept}` : "$0.00"}
                </p>
              </CardContent>
            </Card>
            <Card className="border-border shadow-sm">
              <CardContent className="p-4">
                <span className="text-[12px] text-muted-foreground">Total Gas Fees</span>
                <p className="text-[20px] font-bold tabular-nums mt-1">
                  {stats.TotalGas !== "0" ? `$${stats.TotalGas}` : "$0.00"}
                </p>
              </CardContent>
            </Card>
          </div>

          {stats.TotalSweeps === 0 ? (
            <EmptyState
              title="No sweep activity yet"
              description="Sweep transactions will appear here when deposits are processed and auto-consolidated to cold storage."
              icon={<ArrowDownToLine className="size-5" />}
            />
          ) : null}
        </>
      ) : null}

      {/* Info note */}
      <Card className="border-border/50 bg-muted/30 shadow-sm">
        <CardContent className="p-4">
          <div className="flex items-start gap-3">
            <Info className="size-4 text-muted-foreground shrink-0 mt-0.5" />
            <p className="text-[12px] text-muted-foreground">
              SmartSweep automatically consolidates deposits from individual
              deposit wallets to your configured cold storage address. Sweeps
              are triggered when deposit balances exceed the configured
              threshold.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
