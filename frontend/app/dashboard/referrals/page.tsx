"use client";

import {
  useReferralOverview,
  useReferralCampaigns,
} from "@/lib/query/hooks/use-merchant-misc";
import { LoadingRows, EmptyState, ErrorState } from "@/components/ui/states";
import { PageHeader } from "@/components/ui/page-header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { DataTable, type Column } from "@/components/ui/data-table";
import { StatusBadge } from "@/components/ui/status-badge";
import type { ReferralCampaign } from "@/lib/query/hooks/use-merchant-misc";

const campaignColumns: Column<ReferralCampaign>[] = [
  { key: "name", header: "Name", cell: (r) => r.name },
  {
    key: "description",
    header: "Description",
    cell: (r) => r.description ?? "-",
  },
  {
    key: "reward",
    header: "Reward",
    cell: (r) =>
      r.rewardType === "percentage"
        ? `${r.rewardValue}%`
        : `$${r.rewardValue}`,
  },
  {
    key: "active",
    header: "Status",
    cell: (r) => (
      <StatusBadge status={r.active ? "active" : "inactive"} />
    ),
  },
  {
    key: "period",
    header: "Period",
    cell: (r) => {
      const start = r.startsAt
        ? new Date(r.startsAt).toLocaleDateString()
        : "-";
      const end = r.endsAt ? new Date(r.endsAt).toLocaleDateString() : "-";
      return `${start} - ${end}`;
    },
  },
];

export default function ReferralsPage() {
  const overview = useReferralOverview();
  const campaigns = useReferralCampaigns();

  if (overview.isLoading) return <LoadingRows rows={4} />;
  if (overview.error) {
    return (
      <ErrorState
        message={overview.error.message}
        retry={overview.refetch}
      />
    );
  }

  const ov = overview.data;

  return (
    <div className="space-y-6">
      <PageHeader title="Referrals" />

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard title="Referral Code" value={ov?.code ?? "-"} />
        <MetricCard
          title="Total Referred"
          value={String(ov?.totalReferred ?? 0)}
        />
        <MetricCard
          title="Pending Rewards"
          value={`$${ov?.pendingRewards ?? "0"}`}
        />
        <MetricCard
          title="Paid Rewards"
          value={`$${ov?.paidRewards ?? "0"}`}
        />
      </div>

      <section className="space-y-4">
        <h2 className="text-lg font-medium">Campaigns</h2>
        {campaigns.isLoading ? <LoadingRows rows={3} /> : null}
        {campaigns.error ? (
          <ErrorState
            message={campaigns.error.message}
            retry={campaigns.refetch}
          />
        ) : null}
        {campaigns.data ? (
          <DataTable
            columns={campaignColumns}
            rows={campaigns.data}
            keyOf={(r) => r.id}
            empty={<EmptyState title="No campaigns yet" />}
          />
        ) : null}
      </section>
    </div>
  );
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm text-muted-foreground">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold tracking-tight">{value}</p>
      </CardContent>
    </Card>
  );
}
