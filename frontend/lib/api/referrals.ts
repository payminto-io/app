/**
 * Referrals domain API.
 *
 * Backend exposes these referral endpoints (see internal/api/router.go):
 *   GET  /referrals/code              -> { referralCode }
 *   GET  /referrals/stats             -> { stats: { MemberID, TotalReferrals, TotalEarned, ConversionRate } }
 *   GET  /referrals/rewards           -> { rewards: [...] }
 *   GET  /admin/referrals/campaigns   -> { campaigns: [...] }  (system.admin only)
 *
 * The frontend historically called /referrals/overview and /referrals/campaigns
 * which never existed on the backend, surfacing as "Not Found" on the
 * merchant referrals page. We now compose an overview from code+stats and
 * gracefully fall back to empty data when the admin campaign list is
 * unavailable (missing perms or feature disabled).
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface ReferralOverview {
  code: string;
  totalReferred: number;
  pendingRewards: string;
  paidRewards: string;
}

export interface ReferralCampaign {
  id: number;
  name: string;
  description?: string;
  rewardType: "fixed" | "percentage";
  rewardValue: string;
  active: boolean;
  startsAt?: string;
  endsAt?: string;
}

/** Backend referral stats shape (Go exports PascalCase JSON). */
interface BackendReferralStats {
  MemberID: number;
  TotalReferrals: number;
  TotalEarned: number;
  ConversionRate: number;
}

export const referralsApi = {
  /**
   * Composite "overview" — there is no single backend endpoint for this, so we
   * fan out to code+stats. Any sub-call returning 404 degrades to an empty
   * placeholder so the page renders instead of crashing.
   */
  overview: async (): Promise<ReferralOverview> => {
    const fallback: ReferralOverview = {
      code: "",
      totalReferred: 0,
      pendingRewards: "0",
      paidRewards: "0",
    };

    const [codeResult, statsResult] = await Promise.allSettled([
      apiFetch<{ referralCode: string }>("/referrals/code"),
      apiFetch<{ stats: BackendReferralStats }>("/referrals/stats"),
    ]);

    const code =
      codeResult.status === "fulfilled" ? codeResult.value.referralCode : "";

    let totalReferred = 0;
    let paidRewards = "0";
    if (statsResult.status === "fulfilled") {
      totalReferred = statsResult.value.stats.TotalReferrals;
      paidRewards = String(statsResult.value.stats.TotalEarned ?? 0);
    } else if (
      !isApiError(statsResult.reason) ||
      !statsResult.reason.isNotFound
    ) {
      // Unexpected failure — surface it rather than silently empty.
      throw statsResult.reason;
    }

    return {
      ...fallback,
      code,
      totalReferred,
      paidRewards,
    };
  },

  /**
   * The merchant-facing "campaigns" view reads from the admin campaign list
   * because the backend does not currently expose a merchant-scoped
   * projection. If the caller lacks system.admin, we return [] so the page
   * shows an empty state instead of a 403/404 error.
   */
  campaigns: async (): Promise<ReferralCampaign[]> => {
    try {
      const res = await apiFetch<{ campaigns: ReferralCampaign[] }>(
        "/admin/referrals/campaigns"
      );
      return res.campaigns ?? [];
    } catch (err) {
      if (
        isApiError(err) &&
        (err.isNotFound || err.isForbidden || err.isUnauthorized)
      ) {
        return [];
      }
      throw err;
    }
  },
};
