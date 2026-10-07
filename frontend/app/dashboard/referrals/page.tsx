"use client";

import { useReferralOverview, useReferralCampaigns, type ReferralCampaign } from "@/lib/query/hooks/use-merchant-misc";
import { formatDecimal } from "@/lib/money";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

const campaignColumns: DataTableColumn<ReferralCampaign>[] = [
  { key: "name", header: "Name", className: "font-medium", cell: (r) => r.name },
  { key: "description", header: "Description", className: "text-ink-soft", cell: (r) => r.description ?? null },
  {
    key: "reward",
    header: "Reward",
    align: "right",
    cell: (r) => (r.rewardType === "percentage" ? `${r.rewardValue}%` : `${formatDecimal(r.rewardValue, "")} fixed`),
  },
  { key: "active", header: "Status", cell: (r) => <StatusBadge status={r.active ? "active" : "inactive"} /> },
  {
    key: "period",
    header: "Runs",
    className: "text-ink-soft",
    cell: (r) =>
      r.startsAt || r.endsAt ? (
        <span className="num">
          <DateTime value={r.startsAt} format="date" />
          {r.startsAt && r.endsAt ? " to " : r.endsAt ? "Until " : ""}
          <DateTime value={r.endsAt} format="date" />
        </span>
      ) : null,
  },
];

export default function ReferralsPage() {
  const overview = useReferralOverview();
  const campaigns = useReferralCampaigns();
  const ov = overview.data;

  return (
    <div className="space-y-8">
      <PageHeader title="Referrals" />

      {overview.error ? (
        <ErrorState message={overview.error.message} retry={overview.refetch} />
      ) : !ov ? (
        <Skeleton className="h-[120px] rounded-md" />
      ) : (
        <Card>
          <CardContent className="grid gap-6 sm:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)] sm:items-end">
            <div className="min-w-0 space-y-1.5">
              <div className="text-label font-medium text-ink-soft">Your code</div>
              {ov.code ? (
                <CopyField value={ov.code} />
              ) : (
                <p className="text-body-sm text-ink-soft">No code issued yet.</p>
              )}
            </div>
            <Stat label="Referred" value={ov.totalReferred.toLocaleString("en-US")} />
            <Stat label="Earned" value={formatDecimal(ov.paidRewards || "0", "")} />
          </CardContent>
        </Card>
      )}

      <section className="space-y-3">
        <h2 className="text-h3 font-semibold text-ink">Campaigns</h2>
        {campaigns.error ? (
          <ErrorState message={campaigns.error.message} retry={campaigns.refetch} />
        ) : (
          <DataTable
            columns={campaignColumns}
            rows={campaigns.data ?? []}
            loading={!campaigns.data}
            getRowId={(r) => r.id}
            emptyTitle="No campaigns running."
          />
        )}
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-1">
      <div className="text-label font-medium text-ink-soft">{label}</div>
      <div className="num text-h2 font-semibold text-ink">{value}</div>
    </div>
  );
}
