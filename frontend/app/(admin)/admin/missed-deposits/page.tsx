"use client";

import { useState } from "react";
import { AlertTriangle } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/status-badge";
import { CopyButton } from "@/components/copy-button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useMissedDeposits } from "@/lib/query/hooks/use-admin";
import type { MissedDeposit } from "@/lib/query/hooks/use-admin";
import { ResolveDialog } from "./resolve-dialog";
import { DismissDialog } from "./dismiss-dialog";

type StatusFilter = "pending" | "resolved" | "dismissed" | undefined;

function truncateAddr(s: string) {
  if (s.length <= 14) return s;
  return `${s.slice(0, 6)}...${s.slice(-6)}`;
}

const CHAIN_COLORS: Record<string, string> = {
  BTC: "bg-orange-500/10 text-orange-400",
  ETH: "bg-blue-500/10 text-blue-400",
  BASE: "bg-blue-400/10 text-blue-300",
  TRX: "bg-red-500/10 text-red-400",
  POLYGON: "bg-purple-500/10 text-purple-400",
};

export default function MissedDepositsPage() {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(undefined);
  const { data, isLoading, error, refetch } = useMissedDeposits(
    statusFilter ? { status: statusFilter } : undefined
  );
  const [resolving, setResolving] = useState<MissedDeposit | null>(null);
  const [dismissing, setDismissing] = useState<MissedDeposit | null>(null);

  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }

  function handleTabChange(value: string | number) {
    const v = String(value);
    setStatusFilter(v === "all" ? undefined : (v as StatusFilter));
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Missed Deposits"
        description="Deposits that could not be matched to a payment"
        icon={<AlertTriangle className="size-5" />}
      />

      <Tabs defaultValue="all" onValueChange={handleTabChange}>
        <TabsList variant="line">
          <TabsTrigger value="all">All</TabsTrigger>
          <TabsTrigger value="pending">Pending</TabsTrigger>
          <TabsTrigger value="resolved">Resolved</TabsTrigger>
          <TabsTrigger value="dismissed">Dismissed</TabsTrigger>
        </TabsList>

        <TabsContent value={statusFilter ?? "all"}>
          {isLoading ? (
            <Card className="mt-4 p-0">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
                  <Skeleton className="h-4 w-12" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="ml-auto h-5 w-16 rounded-full" />
                </div>
              ))}
            </Card>
          ) : (data ?? []).length === 0 ? (
            <div className="mt-4">
              <EmptyState
                title="No missed deposits"
                description="No missed deposits match the current filter."
                icon={<AlertTriangle className="size-5" />}
              />
            </div>
          ) : (
            <Card className="mt-4 p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="pm-label">Chain</TableHead>
                    <TableHead className="pm-label">Currency</TableHead>
                    <TableHead className="pm-label">Address</TableHead>
                    <TableHead className="pm-label tabular-nums">Amount</TableHead>
                    <TableHead className="pm-label">TX Hash</TableHead>
                    <TableHead className="pm-label">Status</TableHead>
                    <TableHead className="pm-label">Created</TableHead>
                    <TableHead className="w-24" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(data ?? []).map((d: MissedDeposit) => {
                    const chainColor =
                      CHAIN_COLORS[d.blockchainCode.toUpperCase()] ??
                      "bg-muted text-muted-foreground";
                    return (
                      <TableRow key={d.id}>
                        <TableCell>
                          <span
                            className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold ${chainColor}`}
                          >
                            {d.blockchainCode}
                          </span>
                        </TableCell>
                        <TableCell className="font-mono text-[12px]">
                          {d.currencyCode}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            <span
                              title={d.address}
                              className="font-mono text-[11px]"
                            >
                              {truncateAddr(d.address)}
                            </span>
                            <CopyButton
                              value={d.address}
                              size="sm"
                              variant="ghost"
                              label=""
                              className="size-6 p-0"
                            />
                          </div>
                        </TableCell>
                        <TableCell className="tabular-nums text-[13px] font-medium">
                          {d.amount}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            <span
                              title={d.transactionHash}
                              className="font-mono text-[11px]"
                            >
                              {truncateAddr(d.transactionHash)}
                            </span>
                            <CopyButton
                              value={d.transactionHash}
                              size="sm"
                              variant="ghost"
                              label=""
                              className="size-6 p-0"
                            />
                          </div>
                        </TableCell>
                        <TableCell>
                          <StatusBadge status={d.status} />
                        </TableCell>
                        <TableCell className="text-muted-foreground text-[12px] tabular-nums">
                          {new Date(d.createdAt).toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                          })}
                        </TableCell>
                        <TableCell>
                          {d.status === "pending" ? (
                            <div className="flex gap-1">
                              <Button
                                variant="outline"
                                size="sm"
                                onClick={() => setResolving(d)}
                              >
                                Resolve
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setDismissing(d)}
                              >
                                Dismiss
                              </Button>
                            </div>
                          ) : null}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </Card>
          )}
        </TabsContent>
      </Tabs>

      {resolving ? (
        <ResolveDialog
          deposit={resolving}
          onClose={() => setResolving(null)}
        />
      ) : null}
      {dismissing ? (
        <DismissDialog
          deposit={dismissing}
          onClose={() => setDismissing(null)}
        />
      ) : null}
    </div>
  );
}
