"use client";

import Link from "next/link";
import { ExternalLink } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import type { Payment } from "@/lib/query/hooks/use-payments";

function timeAgo(dateStr: string): string {
  const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

interface RecentPaymentsTableProps {
  payments: Payment[] | undefined;
  isLoading: boolean;
}

export function RecentPaymentsTable({
  payments,
  isLoading,
}: RecentPaymentsTableProps) {
  return (
    <Card className="lg:col-span-2 border-border shadow-none">
      <CardHeader className="flex-row items-center justify-between pb-2">
        <CardTitle className="text-[15px] font-semibold">
          Recent Payments
        </CardTitle>
        <Link href="/dashboard/payments">
          <Button
            variant="ghost"
            size="sm"
            className="h-8 text-[var(--pm-primary)] hover:text-[var(--pm-primary)]"
          >
            View All
            <ExternalLink className="size-3" />
          </Button>
        </Link>
      </CardHeader>
      <CardContent className="px-0 pb-0">
        {isLoading ? (
          <div className="space-y-3 px-4 pb-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-10 w-full" />
            ))}
          </div>
        ) : payments && payments.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow className="border-border hover:bg-transparent">
                <TableHead className="pm-label pl-6">Reference</TableHead>
                <TableHead className="pm-label">Amount</TableHead>
                <TableHead className="pm-label">Customer</TableHead>
                <TableHead className="pm-label">Status</TableHead>
                <TableHead className="pm-label text-right pr-6">Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {payments.slice(0, 5).map((payment) => (
                <TableRow
                  key={payment.referenceID}
                  className="border-border/60 hover:bg-muted/30"
                >
                  <TableCell className="pl-6">
                    <Link
                      href={`/dashboard/payments/${payment.referenceID}`}
                      className="font-medium text-[var(--pm-primary)] hover:underline text-[13px]"
                    >
                      {payment.referenceID.slice(0, 12)}...
                    </Link>
                  </TableCell>
                  <TableCell className="font-semibold tabular-nums text-[13px]">
                    ${payment.amountInUSD}
                  </TableCell>
                  <TableCell className="text-[13px] text-muted-foreground truncate max-w-[160px]">
                    {payment.customerEmail ?? "\u2014"}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={payment.paymentState} />
                  </TableCell>
                  <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                    {timeAgo(payment.createdAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
            No payments found
          </div>
        )}
      </CardContent>
    </Card>
  );
}
