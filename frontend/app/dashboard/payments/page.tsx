"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import { usePaymentsList, type Payment } from "@/lib/query/hooks/use-payments";
import { paymentRail } from "@/lib/status";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { CurrencyDisplay } from "@/components/currency-display";
import { DateTime } from "@/components/date-time";
import { Pagination } from "@/components/pagination";
import { Rail } from "@/components/rail";
import { SearchInput } from "@/components/search-input";
import { buttonVariants } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const TABS = [
  { value: "all", label: "All" },
  { value: "OPEN", label: "Awaiting payment" },
  { value: "PARTIALLY_FILLED", label: "Under paid" },
  { value: "FILLED", label: "Paid" },
  { value: "EXPIRED", label: "Expired" },
  { value: "CANCELLED", label: "Cancelled" },
] as const;

const PAGE_SIZE = 20;

const COLUMNS: DataTableColumn<Payment>[] = [
  {
    key: "amount",
    header: "Amount",
    align: "right",
    className: "w-0",
    cell: (p) => <CurrencyDisplay amount={p.amountInUSD} currency="USD" size="sm" />,
  },
  {
    key: "status",
    header: "Status",
    cell: (p) => {
      const rail = paymentRail(p.paymentState);
      return (
        <span className="flex items-center gap-2.5">
          <Rail step={rail.step} failed={rail.failed} />
          <StatusBadge status={p.paymentState} />
        </span>
      );
    },
  },
  {
    key: "reference",
    header: "Reference",
    cell: (p) => (
      <Link
        href={`/dashboard/payments/${p.referenceID}`}
        onClick={(e) => e.stopPropagation()}
        className="tap rounded-xs font-mono text-label text-ink hover:text-tide"
        title={p.referenceID}
      >
        {p.referenceID.length > 18 ? `${p.referenceID.slice(0, 18)}...` : p.referenceID}
      </Link>
    ),
  },
  {
    key: "customer",
    header: "Customer",
    className: "text-ink-soft",
    cell: (p) => p.customerEmail ?? null,
  },
  {
    key: "created",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (p) => <DateTime value={p.createdAt} />,
  },
];

export default function PaymentsPage() {
  const router = useRouter();
  const [tab, setTab] = useState("all");
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const { data, error, refetch } = usePaymentsList({
    state: tab === "all" ? undefined : tab,
    limit: PAGE_SIZE,
    offset,
  });

  const q = search.trim().toLowerCase();
  const rows =
    data?.payments.filter(
      (p) =>
        !q ||
        p.referenceID.toLowerCase().includes(q) ||
        p.customerEmail?.toLowerCase().includes(q)
    ) ?? [];
  const searching = q.length > 0;

  return (
    <div className="space-y-5">
      <PageHeader title="Payments">
        <Link href="/dashboard/payments/create" className={buttonVariants()}>
          <Plus />
          Create payment
        </Link>
      </PageHeader>

      <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div className="-mx-4 min-w-0 overflow-x-auto px-4 sm:mx-0 sm:px-0">
          <Tabs
            value={tab}
            onValueChange={(v) => {
              setTab(v);
              setOffset(0);
            }}
          >
            <TabsList variant="line">
              {TABS.map((t) => (
                <TabsTrigger key={t.value} value={t.value}>
                  {t.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
        <SearchInput
          aria-label="Search this page by reference or email"
          placeholder="Reference or email"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={rows}
          loading={!data}
          getRowId={(p) => p.referenceID}
          onRowClick={(p) => router.push(`/dashboard/payments/${p.referenceID}`)}
          total={searching ? undefined : data?.total}
          footer={
            data && !searching && data.total > PAGE_SIZE ? (
              <Pagination offset={offset} limit={PAGE_SIZE} total={data.total} onOffsetChange={setOffset} />
            ) : undefined
          }
          emptyTitle={
            searching
              ? "No payments on this page match."
              : tab === "all"
                ? "No payments yet."
                : "No payments in this state."
          }
          emptyDescription={
            !searching && tab === "all" ? "Payments appear here as soon as a link is created." : undefined
          }
          emptyAction={
            !searching && tab === "all" ? (
              <Link href="/dashboard/payments/create" className={buttonVariants({ size: "sm" })}>
                Create payment
              </Link>
            ) : undefined
          }
        />
      )}
    </div>
  );
}
