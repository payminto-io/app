"use client";

import { useState } from "react";
import { useOnramperList } from "@/lib/query/hooks/use-merchant-misc";
import { LoadingRows, EmptyState, ErrorState } from "@/components/ui/states";
import { PageHeader } from "@/components/ui/page-header";
import { DataTable, type Column } from "@/components/ui/data-table";
import { StatusBadge } from "@/components/ui/status-badge";
import type { OnramperSession } from "@/lib/query/hooks/use-merchant-misc";

const TABS = ["all", "pending", "processing", "completed", "failed"] as const;

const columns: Column<OnramperSession>[] = [
  {
    key: "session",
    header: "Session ID",
    cell: (r) => (
      <span className="font-mono text-xs">{r.sessionID.slice(0, 12)}...</span>
    ),
  },
  {
    key: "fiat",
    header: "Fiat Amount",
    cell: (r) => `${r.fiatAmount} ${r.fiatCurrency}`,
  },
  {
    key: "crypto",
    header: "Crypto Amount",
    cell: (r) =>
      r.cryptoAmount
        ? `${r.cryptoAmount} ${r.cryptoCurrency}`
        : "-",
  },
  {
    key: "status",
    header: "Status",
    cell: (r) => <StatusBadge status={r.state} />,
  },
  {
    key: "created",
    header: "Created",
    cell: (r) => new Date(r.createdAt).toLocaleDateString(),
  },
];

export default function OnramperPage() {
  const [tab, setTab] = useState<string>("all");
  const filters = { state: tab === "all" ? undefined : tab };
  const { data, isLoading, error, refetch } = useOnramperList(filters);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Card Payments (Onramper)"
        description="Fiat-to-crypto sessions via Onramper integration."
      />

      <div className="flex gap-1 rounded-lg border border-border bg-muted/30 p-1">
        {TABS.map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium capitalize transition-colors ${
              tab === t
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {isLoading ? <LoadingRows /> : null}
      {error ? <ErrorState message={error.message} retry={refetch} /> : null}
      {data ? (
        <DataTable
          columns={columns}
          rows={data}
          keyOf={(r) => r.id}
          empty={
            <EmptyState
              title="No onramper sessions"
              description="Card payment sessions will appear here."
            />
          }
        />
      ) : null}
    </div>
  );
}
