"use client";

import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useWorkersStatus, useWorkerAction } from "@/lib/query/hooks/use-admin";
import type { WorkerStatus } from "@/lib/query/hooks/use-admin";
import { Play, Square, RotateCcw, Cpu } from "lucide-react";
import { SectionLabel } from "@/components/section-label";

export function WorkersTable() {
  const { data, isLoading, error, refetch } = useWorkersStatus();
  const action = useWorkerAction();

  function handleAction(name: string, act: "start" | "stop" | "restart") {
    action.mutate({ name, action: act });
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-5 w-24" />
        <Card className="p-0">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
              <Skeleton className="size-2.5 rounded-full" />
              <Skeleton className="h-4 w-40" />
              <Skeleton className="ml-auto h-4 w-24" />
            </div>
          ))}
        </Card>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }

  return (
    <div className="space-y-3">
      <SectionLabel>Workers</SectionLabel>
      <p className="text-[11px] text-muted-foreground">
        Auto-refreshes every 10 seconds
      </p>

      {(data ?? []).length === 0 ? (
        <EmptyState
          title="No workers"
          description="No background workers found."
          icon={<Cpu className="size-5" />}
        />
      ) : (
        <Card className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pm-label">Worker</TableHead>
                <TableHead className="pm-label">Status</TableHead>
                <TableHead className="pm-label">Last Started</TableHead>
                <TableHead className="pm-label">Last Error</TableHead>
                <TableHead className="pm-label">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data ?? []).map((w: WorkerStatus) => (
                <TableRow key={w.name}>
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <span
                        className={`size-2 rounded-full ${
                          w.running
                            ? "bg-[oklch(0.72_0.22_140)]"
                            : "bg-[oklch(0.68_0.22_22)]"
                        }`}
                      />
                      <span className="text-[13px] font-medium">{w.name}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ${
                        w.running
                          ? "bg-[oklch(0.93_0.22_128_/_0.15)] text-[oklch(0.85_0.22_140)]"
                          : "bg-muted text-muted-foreground"
                      }`}
                    >
                      {w.running ? "Running" : "Stopped"}
                    </span>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-[12px] tabular-nums">
                    {w.lastStartedAt
                      ? new Date(w.lastStartedAt).toLocaleString()
                      : "--"}
                  </TableCell>
                  <TableCell>
                    {w.lastError ? (
                      <span className="max-w-xs truncate block text-destructive text-[11px]">
                        {w.lastError}
                      </span>
                    ) : (
                      <span className="text-[12px] text-muted-foreground">--</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {!w.running ? (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleAction(w.name, "start")}
                          disabled={action.isPending}
                          title="Start"
                        >
                          <Play className="size-3" />
                        </Button>
                      ) : (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleAction(w.name, "stop")}
                          disabled={action.isPending}
                          title="Stop"
                        >
                          <Square className="size-3" />
                        </Button>
                      )}
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleAction(w.name, "restart")}
                        disabled={action.isPending}
                        title="Restart"
                      >
                        <RotateCcw className="size-3" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </div>
  );
}
