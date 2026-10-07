"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { CurrencyDisplay } from "@/components/currency-display";
import { DateTime } from "@/components/date-time";
import { Rail } from "@/components/rail";
import { buttonVariants } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { paymentRail } from "@/lib/status";
import type { Payment } from "@/lib/query/hooks/use-payments";

const COLUMNS: DataTableColumn<Payment>[] = [
  {
    key: "amount", stack: "lead",
    header: "Amount",
    align: "right",
    className: "w-0",
    cell: (p) => (
      <Link
        href={`/dashboard/payments/${p.referenceID}`}
        onClick={(e) => e.stopPropagation()}
        className="tap rounded-xs"
      >
        <CurrencyDisplay amount={p.amountInUSD} currency="USD" size="sm" />
      </Link>
    ),
  },
  {
    key: "status", stack: "trail",
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
    key: "customer", stack: "meta",
    header: "Customer",
    className: "text-ink-soft max-w-[220px] truncate",
    cell: (p) => p.customerEmail ?? null,
  },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (p) => <DateTime value={p.createdAt} />,
  },
];

interface RecentPaymentsTableProps {
  payments: Payment[] | undefined;
  isLoading: boolean;
}

export function RecentPaymentsTable({ payments, isLoading }: RecentPaymentsTableProps) {
  const router = useRouter();
  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-h3 font-semibold text-ink">Recent payments</h2>
        <Link href="/dashboard/payments" className={buttonVariants({ variant: "link", size: "sm" })}>
          View all
        </Link>
      </div>
      <DataTable
        columns={COLUMNS}
        rows={payments?.slice(0, 5) ?? []}
        loading={isLoading}
        getRowId={(p) => p.referenceID}
        onRowClick={(p) => router.push(`/dashboard/payments/${p.referenceID}`)}
        footer={null}
        emptyTitle="No payments yet."
        emptyDescription="Payments appear here as soon as a link is created."
        emptyAction={
          <Link href="/dashboard/payments/create" className={buttonVariants({ size: "sm" })}>
            Create payment
          </Link>
        }
      />
    </section>
  );
}
