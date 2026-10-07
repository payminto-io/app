"use client";

import { useState } from "react";
import { useMissedDeposits, type MissedDeposit } from "@/lib/query/hooks/use-admin";
import { chainName } from "@/lib/chains";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { RowActions } from "@/components/row-actions";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ResolveDialog } from "./resolve-dialog";
import { DismissDialog } from "./dismiss-dialog";

type StatusFilter = "pending" | "resolved" | "dismissed" | undefined;

function truncateAddr(s: string) {
  if (s.length <= 14) return s;
  return `${s.slice(0, 6)}...${s.slice(-6)}`;
}

export default function MissedDepositsPage() {
  const [statusFilter, setStatusFilter] = useState<StatusFilter>(undefined);
  const { data, error, refetch } = useMissedDeposits(statusFilter ? { status: statusFilter } : undefined);
  const [resolving, setResolving] = useState<MissedDeposit | null>(null);
  const [dismissing, setDismissing] = useState<MissedDeposit | null>(null);

  const columns: DataTableColumn<MissedDeposit>[] = [
    {
      key: "amount", stack: "lead",
      header: "Amount",
      align: "right",
      className: "w-0",
      cell: (d) => <CurrencyDisplay amount={d.amount} currency={d.currencyCode} size="sm" />,
    },
    { key: "chain", stack: "meta", header: "Chain", className: "text-ink-soft", cell: (d) => chainName(d.blockchainCode) },
    { key: "status", stack: "trail", header: "Status", cell: (d) => <StatusBadge status={d.status} /> },
    {
      key: "address", stack: "detail",
      header: "Address",
      cell: (d) => <CopyField value={d.address} display={truncateAddr(d.address)} boxed={false} />,
    },
    {
      key: "tx", stack: "detail",
      header: "Transaction",
      cell: (d) => <CopyField value={d.transactionHash} display={truncateAddr(d.transactionHash)} boxed={false} />,
    },
    {
      key: "created", stack: "meta",
      header: "Seen",
      align: "right",
      className: "text-ink-soft",
      cell: (d) => <DateTime value={d.createdAt} />,
    },
    {
      key: "actions", stack: "action",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      className: "w-0",
      cell: (d) =>
        d.status === "pending" ? (
          <RowActions label="Deposit actions">
            <DropdownMenuItem onClick={() => setResolving(d)}>Resolve</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onClick={() => setDismissing(d)}>
              Dismiss
            </DropdownMenuItem>
          </RowActions>
        ) : null,
    },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="Missed deposits" />

      <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
        <Tabs
          value={statusFilter ?? "all"}
          onValueChange={(v) => setStatusFilter(String(v) === "all" ? undefined : (String(v) as StatusFilter))}
        >
          <TabsList variant="line">
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="pending">Pending</TabsTrigger>
            <TabsTrigger value="resolved">Resolved</TabsTrigger>
            <TabsTrigger value="dismissed">Dismissed</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(d) => d.id}
          emptyTitle={statusFilter ? `No ${statusFilter} deposits.` : "No missed deposits."}
          emptyDescription={statusFilter ? undefined : "Deposits that match no payment land here."}
        />
      )}

      {resolving ? <ResolveDialog deposit={resolving} onClose={() => setResolving(null)} /> : null}
      {dismissing ? <DismissDialog deposit={dismissing} onClose={() => setDismissing(null)} /> : null}
    </div>
  );
}
