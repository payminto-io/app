"use client";

import { PageHeader } from "@/components/page-header";
import { EmptyState } from "@/components/ui/states";

/**
 * /analytics/sweeps is instance-wide (not scoped to this account) and sums
 * every asset and every chain's gas into unitless numbers, so nothing from it
 * is shown. Ticket 18 lists the fields a per-account report needs.
 */
export default function SweepsPage() {
  return (
    <div className="space-y-5">
      <PageHeader title="Sweeps" />
      <EmptyState
        title="Sweep reporting for this account is not available yet."
        description="Deposits are swept to the cold wallet once a hot wallet can pay the gas. Per-asset sweep totals will appear here when the API reports them for this account."
      />
    </div>
  );
}
