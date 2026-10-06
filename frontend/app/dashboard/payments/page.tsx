"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Search, Plus } from "lucide-react";
import { usePaymentsList } from "@/lib/query/hooks/use-payments";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Payment } from "@/lib/query/hooks/use-payments";

const TABS = [
  { value: "all", label: "All" },
  { value: "OPEN", label: "Open" },
  { value: "PARTIALLY_FILLED", label: "Partial" },
  { value: "FILLED", label: "Filled" },
  { value: "EXPIRED", label: "Expired" },
  { value: "CANCELLED", label: "Cancelled" },
] as const;

const PAGE_SIZE = 20;

export default function PaymentsPage() {
  const router = useRouter();
  const [tab, setTab] = useState("all");
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const filters = {
    state: tab === "all" ? undefined : tab,
    limit: PAGE_SIZE,
    offset,
  };
  const { data, isLoading, error, refetch } = usePaymentsList(filters);

  const filtered = data?.payments.filter((p) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      p.referenceID.toLowerCase().includes(q) ||
      p.customerEmail?.toLowerCase().includes(q)
    );
  });

  return (
    <div className="space-y-6">
      {/* Metric strip */}
      <MetricStrip data={data} isLoading={isLoading} />

      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <h1 className="text-xl font-bold tracking-tight">Payments</h1>
        <Link href="/dashboard/payments/create">
          <Button className="h-9 rounded-lg bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
            <Plus className="size-4" />
            Create Payment
          </Button>
        </Link>
      </div>

      {/* Search + Tabs */}
      <div className="space-y-3">
        <div className="relative max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search by reference, email, chain..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-background pl-10 pr-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>

        <Tabs
          value={tab}
          onValueChange={(v) => {
            setTab(v);
            setOffset(0);
          }}
        >
          <TabsList className="h-auto flex-wrap">
            {TABS.map((t) => (
              <TabsTrigger key={t.value} value={t.value} className="text-xs">
                {t.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {/* Error state */}
      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {/* Table */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      ) : filtered && filtered.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">Reference ID</TableHead>
                  <TableHead className="pm-label">Amount</TableHead>
                  <TableHead className="pm-label">Status</TableHead>
                  <TableHead className="pm-label">Customer</TableHead>
                  <TableHead className="pm-label text-right pr-6">Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((p) => (
                  <TableRow
                    key={p.referenceID}
                    className="border-border/60 hover:bg-muted/30 cursor-pointer"
                    onClick={() =>
                      router.push(`/dashboard/payments/${p.referenceID}`)
                    }
                  >
                    <TableCell className="pl-6">
                      <Link
                        href={`/dashboard/payments/${p.referenceID}`}
                        className="font-medium text-[var(--pm-primary)] hover:underline text-[13px]"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {p.referenceID.slice(0, 16)}...
                      </Link>
                    </TableCell>
                    <TableCell className="font-semibold tabular-nums text-[13px]">
                      ${p.amountInUSD}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={p.paymentState} />
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">
                      {p.customerEmail ?? "\u2014"}
                    </TableCell>
                    <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                      {new Date(p.createdAt).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-border shadow-sm">
          <CardContent className="flex h-40 items-center justify-center">
            <div className="text-center">
              <p className="text-sm text-muted-foreground">No payments found</p>
              <p className="text-xs text-muted-foreground/60 mt-1">
                Create your first payment to get started.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Pagination */}
      {data && data.total > PAGE_SIZE ? (
        <div className="flex items-center justify-between">
          <p className="text-[13px] text-muted-foreground tabular-nums">
            Showing {offset + 1}&ndash;
            {Math.min(offset + PAGE_SIZE, data.total)} of {data.total}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + PAGE_SIZE >= data.total}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function MetricStrip({
  data,
  isLoading,
}: {
  data: { payments: Payment[]; total: number } | undefined;
  isLoading: boolean;
}) {
  if (isLoading) {
    return (
      <div className="grid gap-3 grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-20 rounded-xl" />
        ))}
      </div>
    );
  }

  const payments = data?.payments ?? [];
  const total = data?.total ?? 0;
  const settled = payments.filter(
    (p) => p.paymentState === "FILLED"
  ).length;
  const pending = payments.filter(
    (p) => p.paymentState === "OPEN" || p.paymentState === "PARTIALLY_FILLED"
  ).length;
  const expired = payments.filter((p) => p.paymentState === "EXPIRED").length;

  const cards = [
    { label: "Total Payments", value: String(total) },
    { label: "Settled", value: String(settled) },
    { label: "Pending", value: String(pending) },
    { label: "Expired", value: String(expired) },
  ];

  return (
    <div className="grid gap-3 grid-cols-2 lg:grid-cols-4">
      {cards.map((c) => (
        <Card key={c.label} className="border-border shadow-sm">
          <CardContent className="p-4">
            <div className="pm-label">{c.label}</div>
            <div className="mt-2 text-2xl font-bold tabular-nums">{c.value}</div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
