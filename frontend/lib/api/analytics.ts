/**
 * Analytics domain API.
 *
 * Backend endpoints:
 *   GET /analytics/summary          → { summary: DashboardSummary }
 *   GET /analytics/volume           → { volume: VolumeBucket[] }
 *   GET /analytics/revenue          → { revenue: RevenueBreakdown[] }
 *   GET /analytics/customers/top    → { customers: CustomerSummary[] }
 *   GET /analytics/withdrawals      → { withdrawals: WithdrawalStats }
 */
import { apiFetch } from "./client";

/* ---------- Backend response shapes (match Go JSON) ---------- */

/** Matches repository.DashboardSummary — Go uses PascalCase JSON by default. */
export interface BackendDashboardSummary {
  TotalPayments: number;
  FilledPayments: number;
  TotalVolume: string;
  TotalWithdrawals: number;
  ActiveWebhooks: number;
}

/** Normalised summary used by the dashboard UI. */
export interface AnalyticsSummary {
  totalVolume: string;
  totalPayments: number;
  filledPayments: number;
  totalWithdrawals: number;
  activeWebhooks: number;
}

/** Matches repository.VolumeBucket. */
export interface VolumeBucket {
  Bucket: string;
  BucketLabel: string;
  Volume: string;
  PaymentCount: number;
}

/** Matches repository.RevenueBreakdown. */
export interface RevenueBreakdown {
  BlockchainCode: string;
  CurrencyCode: string;
  State: string;
  TotalAmount: string;
  Count: number;
}

/* ---------- API functions ---------- */

export const analyticsApi = {
  /** Dashboard home tile summary. */
  summary: async (): Promise<AnalyticsSummary> => {
    const raw = await apiFetch<{ summary: BackendDashboardSummary }>(
      "/analytics/summary"
    );
    const s = raw.summary;
    return {
      totalVolume: s.TotalVolume,
      totalPayments: s.TotalPayments,
      filledPayments: s.FilledPayments,
      totalWithdrawals: s.TotalWithdrawals,
      activeWebhooks: s.ActiveWebhooks,
    };
  },

  /** Volume over time for charts. */
  volume: (
    interval: "day" | "week" | "month" = "day",
    start?: string,
    end?: string
  ): Promise<VolumeBucket[]> => {
    const params = new URLSearchParams({ interval });
    if (start) params.set("start", start);
    if (end) params.set("end", end);
    return apiFetch<{ volume: VolumeBucket[] }>(
      `/analytics/volume?${params}`
    ).then((r) => r.volume ?? []);
  },

  /** Revenue breakdown by chain/currency/state. */
  revenue: (start?: string, end?: string): Promise<RevenueBreakdown[]> => {
    const params = new URLSearchParams();
    if (start) params.set("start", start);
    if (end) params.set("end", end);
    const qs = params.toString();
    return apiFetch<{ revenue: RevenueBreakdown[] }>(
      `/analytics/revenue${qs ? `?${qs}` : ""}`
    ).then((r) => r.revenue ?? []);
  },
};
