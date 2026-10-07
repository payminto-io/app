"use client";

import { PageHeader } from "@/components/page-header";
import { EmptyState } from "@/components/ui/states";

/** Sweeps batch addresses across accounts, so there is no per-account sweep report to show. */
export default function SweepsPage() {
  return (
    <div className="space-y-5">
      <PageHeader title="Sweeps" />
      <EmptyState
        title="No sweep report for this account yet."
      />
    </div>
  );
}
