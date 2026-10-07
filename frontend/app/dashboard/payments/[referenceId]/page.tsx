"use client";

import { useParams } from "next/navigation";
import { ExternalLink } from "lucide-react";
import { usePayment } from "@/lib/query/hooks/use-payments";
import { checkoutURL } from "@/lib/checkout-url";
import { paymentRail } from "@/lib/status";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { DateTime } from "@/components/date-time";
import { DetailItem, DetailList } from "@/components/detail-list";
import { Rail } from "@/components/rail";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

const CRUMBS = [{ label: "Payments", href: "/dashboard/payments" }];

export default function PaymentDetailPage() {
  const params = useParams<{ referenceId: string }>();
  const { data, error, refetch } = usePayment(params.referenceId);
  const paymentLink = checkoutURL(params.referenceId);

  if (error) return <ErrorState message={error.message} retry={refetch} />;

  if (!data) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-10 w-56" />
        <Skeleton className="h-24 w-full rounded-md" />
        <Skeleton className="h-48 w-full rounded-md" />
      </div>
    );
  }


  const state = data.paymentState;
  const rail = paymentRail(state);

  return (
    <div className="space-y-6">
      <PageHeader
       
        breadcrumbs={[...CRUMBS, { label: data.referenceID }]}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <CurrencyDisplay amount={data.amountInUSD} currency="USD" size="display" />
            <StatusBadge status={state} />
          </span>
        }
        description={data.customerEmail ? `From ${data.customerEmail}` : undefined}
      >
        <a
          href={paymentLink}
          target="_blank"
          rel="noopener noreferrer"
          className={buttonVariants({ variant: "outline" })}
        >
          Open checkout
          <ExternalLink />
        </a>
      </PageHeader>

      <Card>
        <CardContent>
          <Rail step={rail.step} failed={rail.failed} size="lg" />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Details</CardTitle>
        </CardHeader>
        <CardContent>
          <DetailList columns={2}>
            <DetailItem label="Payment link" className="sm:col-span-2">
              <CopyField value={paymentLink} />
            </DetailItem>
            <DetailItem label="Reference">
              <CopyField value={data.referenceID} boxed={false} />
            </DetailItem>
            <DetailItem label="Customer">{data.customerEmail}</DetailItem>
            <DetailItem label="Invoice">
              {data.invoiceID ? <span className="font-mono text-body-sm">{data.invoiceID}</span> : null}
            </DetailItem>
            <DetailItem label="Created">
              <DateTime value={data.createdAt} />
            </DetailItem>
            <DetailItem label={state === "EXPIRED" ? "Expired" : "Expires"}>
              {data.expiresAt ? <DateTime value={data.expiresAt} /> : null}
            </DetailItem>
          </DetailList>
        </CardContent>
      </Card>
    </div>
  );
}
