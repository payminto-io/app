"use client";

import { Server } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { MetricCard } from "@/components/metric-card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { useSystemInfo } from "@/lib/query/hooks/use-admin";
import { WorkersTable } from "./workers-table";

export default function SystemPage() {
  const { data: info, isLoading, error, refetch } = useSystemInfo();

  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="System"
        description="System information and worker management"
        icon={<Server className="size-5" />}
      />

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="rounded-xl border border-border bg-card p-5 space-y-3">
              <Skeleton className="h-3 w-16" />
              <Skeleton className="h-7 w-24" />
            </div>
          ))}
        </div>
      ) : info ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <MetricCard
            label="Version"
            value={info.version}
            variant="dark"
          />
          <MetricCard
            label="Commit"
            value={info.commit.slice(0, 8)}
            variant="dark"
          />
          <MetricCard
            label="Build Time"
            value={info.buildTime}
            variant="dark"
          />
          <MetricCard
            label="Uptime"
            value={info.uptime}
            variant="dark"
          />
          <MetricCard
            label="Mode"
            value={info.mode}
            variant="primary"
          />
        </div>
      ) : null}

      <WorkersTable />
    </div>
  );
}
