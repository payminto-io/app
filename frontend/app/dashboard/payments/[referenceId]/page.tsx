"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ChevronRight, ExternalLink } from "lucide-react";
import { usePayment } from "@/lib/query/hooks/use-payments";
import { ErrorState } from "@/components/ui/states";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyButton } from "@/components/copy-button";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { checkoutURL } from "@/lib/checkout-url";

const STATUS_STEPS = ["OPEN", "PARTIALLY_FILLED", "FILLED"];

function usePaymentLink(referenceId: string): string {
  return checkoutURL(referenceId);
}

export default function PaymentDetailPage() {
  const params = useParams<{ referenceId: string }>();
  const { data, isLoading, error, refetch } = usePayment(params.referenceId);
  const paymentLink = usePaymentLink(params.referenceId);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-6 w-64" />
        <div className="grid gap-6 lg:grid-cols-3">
          <Skeleton className="h-80 lg:col-span-2 rounded-xl" />
          <Skeleton className="h-80 rounded-xl" />
        </div>
      </div>
    );
  }

  if (error) return <ErrorState message={error.message} retry={refetch} />;
  if (!data) return <ErrorState message="Payment not found" />;

  const state = data.paymentState;

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1.5 text-[13px] text-muted-foreground">
        <Link href="/dashboard" className="hover:text-foreground transition-colors">
          Dashboard
        </Link>
        <ChevronRight className="size-3.5" />
        <Link href="/dashboard/payments" className="hover:text-foreground transition-colors">
          Payments
        </Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground font-medium">
          {data.referenceID.slice(0, 12)}...
        </span>
      </nav>

      {/* Payment Link Card */}
      <Card className="border-[var(--pm-primary)]/20 bg-[var(--pm-primary)]/5 shadow-sm">
        <CardContent className="py-4 px-6">
          <div className="flex flex-col sm:flex-row sm:items-center gap-3">
            <div className="flex-1 min-w-0">
              <div className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-1">
                Payment Link
              </div>
              <code className="block truncate text-[13px] font-mono text-foreground/80">
                {paymentLink}
              </code>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <CopyButton
                value={paymentLink}
                label="Copy Link"
                size="sm"
                variant="outline"
              />
              <a
                href={paymentLink}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(buttonVariants({ size: "sm", variant: "default" }), "gap-1.5")}
              >
                <ExternalLink className="size-3.5" />
                Open Payment Link
              </a>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Status timeline */}
      <StatusTimeline currentStatus={state} steps={STATUS_STEPS} />

      {/* Payment info card */}
      <Card className="border-border shadow-sm">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-[15px] font-semibold">
              Payment Details
            </CardTitle>
            <StatusBadge status={state} />
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <DetailRow label="Reference ID" value={data.referenceID}>
              <CopyButton value={data.referenceID} size="sm" variant="ghost" label="" />
            </DetailRow>
            <DetailRow label="Amount (USD)" value={`$${data.amountInUSD}`} />
            <DetailRow label="Status">
              <StatusBadge status={state} />
            </DetailRow>
            <DetailRow label="Customer Email" value={data.customerEmail ?? "\u2014"} />
            <DetailRow label="Invoice ID" value={data.invoiceID ?? "\u2014"} />
            <DetailRow
              label="Created At"
              value={new Date(data.createdAt).toLocaleString()}
            />
            <DetailRow
              label="Expires At"
              value={data.expiresAt ? new Date(data.expiresAt).toLocaleString() : "\u2014"}
            />
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function DetailRow({
  label,
  value,
  children,
}: {
  label: string;
  value?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="pm-label">{label}</span>
      <div className="flex items-center gap-2 text-[13px]">
        {value && <span className="break-all">{value}</span>}
        {children}
      </div>
    </div>
  );
}

function StatusTimeline({
  currentStatus,
  steps,
}: {
  currentStatus: string;
  steps: string[];
}) {
  const currentIdx = steps.indexOf(currentStatus);
  const isFailed = currentStatus === "EXPIRED" || currentStatus === "CANCELLED";

  return (
    <Card className="border-border shadow-sm">
      <CardContent className="py-4 px-6">
        <div className="flex items-center justify-between">
          {steps.map((step, i) => {
            const done = currentIdx >= 0 && i <= currentIdx;
            const active = i === currentIdx;
            return (
              <div key={step} className="flex items-center gap-0 flex-1 last:flex-none">
                <div className="flex flex-col items-center gap-1">
                  <div
                    className={cn(
                      "size-3 rounded-full border-2 transition-all",
                      done && !isFailed && "border-[var(--pm-primary)] bg-[var(--pm-primary)]",
                      active && isFailed && "border-red-400 bg-red-400",
                      !done && !active && "border-border bg-transparent"
                    )}
                  />
                  <span
                    className={cn(
                      "text-[10px]",
                      active ? "text-foreground font-medium" : "text-muted-foreground"
                    )}
                  >
                    {step.toLowerCase().replace(/_/g, " ")}
                  </span>
                </div>
                {i < steps.length - 1 && (
                  <div
                    className={cn(
                      "flex-1 h-0.5 mx-2",
                      done && currentIdx > i && !isFailed
                        ? "bg-[var(--pm-primary)]"
                        : "bg-border"
                    )}
                  />
                )}
              </div>
            );
          })}
        </div>
        {isFailed && (
          <p className="text-[12px] text-red-400 mt-2 text-center">
            Payment {currentStatus.toLowerCase()}
          </p>
        )}
      </CardContent>
    </Card>
  );
}
