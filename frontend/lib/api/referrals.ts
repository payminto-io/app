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

/**
 * `totalReferred` is null when /referrals/stats is unavailable, so the page can
 * tell "no stats" from a real zero. Earnings are not exposed: TotalEarned sums
 * rewards across currencies and has no unit.
 */
export interface ReferralOverview {
  code: string;
  totalReferred: number | null;
}

export interface ReferralCampaign {
  id: number;
  name: string;
  description?: string;
  rewardType?: string;
  rewardValue?: string;
  /** The reward's currency; a fixed reward is in this unit. */
  currencyCode?: string;
  status: string;
  startsAt?: string;
  endsAt?: string;
}

/** Matches models.ReferralCampaign (camelCase JSON tags). */
interface BackendReferralCampaign {
  id: number;
  name: string;
  description?: string;
  rewardType?: string;
  rewardValue?: string | number;
  currencyCode?: string;
  status: string;
  startDate?: string;
  endDate?: string;
}

export function normalizeReferralCampaign(c: BackendReferralCampaign): ReferralCampaign {
  return {
    id: c.id,
    name: c.name,
    description: c.description || undefined,
    rewardType: c.rewardType || undefined,
    rewardValue: c.rewardValue === undefined || c.rewardValue === null ? undefined : String(c.rewardValue),
    currencyCode: c.currencyCode || undefined,
    status: c.status,
    startsAt: c.startDate,
    endsAt: c.endDate,
  };
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
   * fan out to code+stats. A 404 leaves that part empty (null), never zero.
   */
  overview: async (): Promise<ReferralOverview> => {
    const [codeResult, statsResult] = await Promise.allSettled([
      apiFetch<{ referralCode: string }>("/referrals/code"),
      apiFetch<{ stats: BackendReferralStats }>("/referrals/stats"),
    ]);

    const code =
      codeResult.status === "fulfilled" ? codeResult.value.referralCode : "";

    let totalReferred: number | null = null;
    if (statsResult.status === "fulfilled") {
      totalReferred = statsResult.value.stats.TotalReferrals;
    } else if (
      !isApiError(statsResult.reason) ||
      !statsResult.reason.isNotFound
    ) {
      // Unexpected failure: surface it rather than render an empty card.
      throw statsResult.reason;
    }

    return { code, totalReferred };
  },

  /**
   * The merchant-facing "campaigns" view reads from the admin campaign list
   * because the backend does not currently expose a merchant-scoped
   * projection. If the caller lacks system.admin, we return [] so the page
   * shows an empty state instead of a 403/404 error.
   */
  campaigns: async (): Promise<ReferralCampaign[]> => {
    try {
      const res = await apiFetch<{ campaigns: BackendReferralCampaign[] | null }>(
        "/admin/referrals/campaigns"
      );
      return (res.campaigns ?? []).map(normalizeReferralCampaign);
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
