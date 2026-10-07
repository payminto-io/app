"use client";

import { useState } from "react";
import { useOnramperList, type OnramperSession } from "@/lib/query/hooks/use-merchant-misc";
import { PageHeader } from "@/components/page-header";
import { CurrencyDisplay } from "@/components/currency-display";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const TABS = [
  { value: "all", label: "All" },
  { value: "pending", label: "Pending" },
  { value: "processing", label: "Processing" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
] as const;

const columns: DataTableColumn<OnramperSession>[] = [
  {
    key: "fiat",
    header: "Paid",
    align: "right",
    className: "w-0",
    cell: (r) => <CurrencyDisplay amount={r.fiatAmount} currency={r.fiatCurrency} size="sm" />,
  },
  {
    key: "crypto",
    header: "Received",
    align: "right",
    className: "w-0",
    cell: (r) => (r.cryptoAmount ? <CurrencyDisplay amount={r.cryptoAmount} currency={r.cryptoCurrency} size="sm" /> : null),
  },
  { key: "status", header: "Status", cell: (r) => <StatusBadge status={r.state} /> },
  { key: "customer", header: "Customer", className: "text-ink-soft", cell: (r) => r.customerEmail ?? null },
  {
    key: "session",
    header: "Session",
    cell: (r) => (
      <span className="font-mono text-label text-ink-soft" title={r.sessionID}>
        {r.sessionID.length > 14 ? `${r.sessionID.slice(0, 14)}...` : r.sessionID}
      </span>
    ),
  },
  {
    key: "created",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (r) => <DateTime value={r.createdAt} />,
  },
];

export default function OnramperPage() {
  const [tab, setTab] = useState<string>("all");
  const { data, error, refetch } = useOnramperList({ state: tab === "all" ? undefined : tab });

  return (
    <div className="space-y-5">
      <PageHeader title="Card payments" />

      <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
        <Tabs value={tab} onValueChange={(v) => setTab(v)}>
          <TabsList variant="line">
            {TABS.map((t) => (
              <TabsTrigger key={t.value} value={t.value}>
                {t.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(r) => r.id}
          emptyTitle={tab === "all" ? "No card payments yet." : "No card payments in this state."}
          emptyDescription={tab === "all" ? "Card-to-crypto sessions from Onramper appear here." : undefined}
        />
      )}
    </div>
  );
}
