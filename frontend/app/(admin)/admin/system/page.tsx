"use client";

import { useSystemInfo } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DetailItem, DetailList } from "@/components/detail-list";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { WorkersTable } from "./workers-table";

/** The API fills missing health fields with these; they are not values. */
const PLACEHOLDER = new Set(["unknown", "-", ""]);
const real = (v: string | undefined) => (v && !PLACEHOLDER.has(v) ? v : null);

export default function SystemPage() {
  const { data: info, error, refetch } = useSystemInfo();

  return (
    <div className="space-y-8">
      <PageHeader title="System" />

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : !info ? (
        <Skeleton className="h-[120px] rounded-md" />
      ) : (
        <Card>
          <CardContent>
            <DetailList className="grid-cols-2 sm:grid-cols-3 lg:grid-cols-4">
              <DetailItem label="Service">
                {real(info.overallStatus) ? <StatusBadge status={info.overallStatus} /> : null}
              </DetailItem>
              <DetailItem label="Database">
                {real(info.dbStatus) ? <StatusBadge status={info.dbStatus} /> : null}
              </DetailItem>
              <DetailItem label="Network">
                {real(info.mode) ? <span className="capitalize">{info.mode}</span> : null}
              </DetailItem>
              <DetailItem label="Version">
                {real(info.version) ? <span className="font-mono text-body-sm">{info.version}</span> : null}
              </DetailItem>
              <DetailItem label="Commit">
                {real(info.commit) ? <span className="font-mono text-body-sm">{info.commit.slice(0, 8)}</span> : null}
              </DetailItem>
              <DetailItem label="Built">
                {real(info.buildTime) ? <span className="num">{info.buildTime}</span> : null}
              </DetailItem>
              <DetailItem label="Uptime">{real(info.uptime) ? <span className="num">{info.uptime}</span> : null}</DetailItem>
            </DetailList>
          </CardContent>
        </Card>
      )}

      <WorkersTable />
    </div>
  );
}
