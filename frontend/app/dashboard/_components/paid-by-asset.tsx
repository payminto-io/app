"use client";

import { CurrencyDisplay } from "@/components/currency-display";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { RevenueBreakdown } from "@/lib/api/analytics";
import { paidTotalsByAsset } from "@/lib/asset-totals";
import { chainName } from "@/lib/chains";

interface PaidByAssetProps {
  revenue: RevenueBreakdown[] | undefined;
  isLoading: boolean;
}

/** Paid deposits per asset from /analytics/revenue (default window: last 30 days). */
export function PaidByAsset({ revenue, isLoading }: PaidByAssetProps) {
  const totals = revenue ? paidTotalsByAsset(revenue) : [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Paid by asset</CardTitle>
        <CardDescription>Deposits on paid payments created in the last 30 days</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-5 w-full rounded-xs" />
            ))}
          </div>
        ) : totals.length > 0 ? (
          <ul className="divide-y divide-line">
            {totals.map((t) => (
              <li key={`${t.currency}-${t.chain}`} className="flex items-baseline justify-between gap-4 py-2.5 first:pt-0 last:pb-0">
                <CurrencyDisplay amount={t.amount} currency={t.currency} className="min-w-0 truncate font-medium" />
                <span className="num shrink-0 text-body-sm text-ink-soft">
                  {chainName(t.chain)}, {t.payments.toLocaleString("en-US")} {t.payments === 1 ? "payment" : "payments"}
                </span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-body-sm text-ink-soft">Totals appear here after the first paid payment.</p>
        )}
      </CardContent>
    </Card>
  );
}
