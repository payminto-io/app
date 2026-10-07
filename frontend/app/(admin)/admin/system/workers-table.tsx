"use client";

import { useWorkersStatus, type WorkerStatus } from "@/lib/query/hooks/use-admin";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

export function WorkersTable() {
  const { data, error, refetch } = useWorkersStatus();
  const columns: DataTableColumn<WorkerStatus>[] = [
    { key: "name", stack: "lead", header: "Worker", cell: (w) => <span className="font-mono text-label font-medium">{w.name}</span> },
    { key: "status", stack: "trail", header: "Status", cell: (w) => <StatusBadge status={w.running ? "running" : "stopped"} /> },
    {
      key: "started", stack: "meta",
      header: "Last started",
      className: "text-ink-soft",
      cell: (w) => <DateTime value={w.lastStartedAt} />,
    },
    {
      key: "error", stack: "detail",
      header: "Last error",
      cell: (w) =>
        w.lastError ? (
          <span className="block max-w-xs truncate text-body-sm text-bad" title={w.lastError}>
            {w.lastError}
          </span>
        ) : null,
    },
  ];

  return (
    <section className="space-y-3">
      <div className="flex items-baseline justify-between gap-3">
        <h2 className="text-h3 font-semibold text-ink">Workers</h2>
        <span className="text-caption text-ink-soft">Refreshes every 10 seconds</span>
      </div>
      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(w) => w.name}
          emptyTitle="No background workers reported."
        />
      )}
    </section>
  );
}
