"use client";

import { useWorkersStatus, useWorkerAction, type WorkerStatus } from "@/lib/query/hooks/use-admin";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { RowActions } from "@/components/row-actions";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

export function WorkersTable() {
  const { data, error, refetch } = useWorkersStatus();
  const action = useWorkerAction();

  function handleAction(name: string, act: "start" | "stop" | "restart") {
    action.mutate({ name, action: act });
  }

  const columns: DataTableColumn<WorkerStatus>[] = [
    { key: "name", header: "Worker", cell: (w) => <span className="font-mono text-label font-medium">{w.name}</span> },
    { key: "status", header: "Status", cell: (w) => <StatusBadge status={w.running ? "running" : "stopped"} /> },
    {
      key: "started",
      header: "Last started",
      className: "text-ink-soft",
      cell: (w) => <DateTime value={w.lastStartedAt} />,
    },
    {
      key: "error",
      header: "Last error",
      cell: (w) =>
        w.lastError ? (
          <span className="block max-w-xs truncate text-body-sm text-bad" title={w.lastError}>
            {w.lastError}
          </span>
        ) : null,
    },
    {
      key: "actions",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      className: "w-0",
      cell: (w) => (
        <RowActions label={`Actions for ${w.name}`}>
          {w.running ? (
            <DropdownMenuItem disabled={action.isPending} onClick={() => handleAction(w.name, "stop")}>
              Stop
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem disabled={action.isPending} onClick={() => handleAction(w.name, "start")}>
              Start
            </DropdownMenuItem>
          )}
          <DropdownMenuItem disabled={action.isPending} onClick={() => handleAction(w.name, "restart")}>
            Restart
          </DropdownMenuItem>
        </RowActions>
      ),
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
