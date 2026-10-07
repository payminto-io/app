/**
 * React Query key factory.
 *
 * Every key is tenant-scoped by `platformId` (merchant data) or `memberId`
 * (dashboard-user data). Tenant switch remounts the QueryClientProvider —
 * see lib/query/provider.tsx. Never write a query key inline in a hook or
 * page; always import from here.
 *
 * See FRONTEND_CODING_STANDARDS §4.
 */

export type PlatformScope = { platformId: number };
export type MemberScope = { memberId: number };

const base = (scope: { platformId?: number; memberId?: number }) =>
  [scope.platformId ?? scope.memberId ?? "anon"] as const;

export const qk = {
  auth: {
    me: () => ["auth", "me"] as const,
  },
  payments: {
    all: (s: PlatformScope) => [...base(s), "payments"] as const,
    list: (s: PlatformScope, filters?: Record<string, unknown>) =>
      [...base(s), "payments", "list", filters ?? {}] as const,
    detail: (s: PlatformScope, ref: string) =>
      [...base(s), "payments", "detail", ref] as const,
  },
  links: {
    all: (s: PlatformScope) => [...base(s), "links"] as const,
    list: (s: PlatformScope, filters?: Record<string, unknown>) =>
      [...base(s), "links", "list", filters ?? {}] as const,
    detail: (s: PlatformScope, id: string) =>
      [...base(s), "links", "detail", id] as const,
  },
  fees: {
    preview: (s: PlatformScope, body: Record<string, unknown>) =>
      [...base(s), "fees", "preview", body] as const,
  },
  depositAddresses: {
    list: (s: PlatformScope, ref: string) =>
      [...base(s), "deposit-addresses", ref] as const,
  },
  withdrawals: {
    all: (s: PlatformScope) => [...base(s), "withdrawals"] as const,
    list: (s: PlatformScope, filters?: Record<string, unknown>) =>
      [...base(s), "withdrawals", "list", filters ?? {}] as const,
    detail: (s: PlatformScope, id: number) =>
      [...base(s), "withdrawals", "detail", id] as const,
  },
  customers: {
    all: (s: PlatformScope) => [...base(s), "customers"] as const,
    list: (s: PlatformScope, filters?: Record<string, unknown>) =>
      [...base(s), "customers", "list", filters ?? {}] as const,
  },
  recipients: {
    all: (s: PlatformScope) => [...base(s), "recipients"] as const,
    list: (s: PlatformScope) => [...base(s), "recipients", "list"] as const,
    detail: (s: PlatformScope, id: number) =>
      [...base(s), "recipients", "detail", id] as const,
  },
  webhooks: {
    all: (s: PlatformScope) => [...base(s), "webhooks"] as const,
    list: (s: PlatformScope) => [...base(s), "webhooks", "list"] as const,
    detail: (s: PlatformScope, id: number) =>
      [...base(s), "webhooks", "detail", id] as const,
    deliveries: (s: PlatformScope, id: number) =>
      [...base(s), "webhooks", "detail", id, "deliveries"] as const,
  },
  analytics: {
    summary: (s: PlatformScope, range: string) =>
      [...base(s), "analytics", "summary", range] as const,
    groups: (s: PlatformScope) => [...base(s), "analytics", "groups"] as const,
    data: (s: PlatformScope, body: Record<string, unknown>) =>
      [...base(s), "analytics", "data", body] as const,
  },
  ticker: {
    all: () => ["ticker"] as const,
  },
  wallets: {
    all: (s: PlatformScope) => [...base(s), "wallets"] as const,
    list: (s: PlatformScope) => [...base(s), "wallets", "list"] as const,
    addresses: (
      s: PlatformScope,
      walletID: number,
      params?: Record<string, unknown>
    ) =>
      [...base(s), "wallets", "addresses", walletID, params ?? {}] as const,
    currencies: (s: PlatformScope) =>
      [...base(s), "wallets", "currencies"] as const,
    cold: (s: PlatformScope) => [...base(s), "wallets", "cold"] as const,
    hot: (s: PlatformScope) => [...base(s), "wallets", "hot"] as const,
  },
  sweeps: {
    stats: (s: PlatformScope) =>
      [...base(s), "sweeps", "stats"] as const,
  },
  referrals: {
    overview: (s: PlatformScope) => [...base(s), "referrals", "overview"] as const,
    campaigns: (s: PlatformScope) => [...base(s), "referrals", "campaigns"] as const,
  },
  apiKeys: {
    list: (s: PlatformScope) => [...base(s), "api-keys", "list"] as const,
  },
  onramper: {
    list: (s: PlatformScope, filters?: Record<string, unknown>) =>
      [...base(s), "onramper", "list", filters ?? {}] as const,
  },
  admin: {
    externalPlatforms: {
      list: (s: MemberScope) => [...base(s), "admin", "external-platforms"] as const,
      detail: (s: MemberScope, id: number) => [...base(s), "admin", "external-platforms", id] as const,
      currencies: (s: MemberScope, id: number) =>
        [...base(s), "admin", "external-platforms", id, "currencies"] as const,
    },
    members: {
      list: (s: MemberScope) => [...base(s), "admin", "members"] as const,
      detail: (s: MemberScope, id: number) => [...base(s), "admin", "members", id] as const,
    },
    roles: {
      list: (s: MemberScope) => [...base(s), "admin", "roles"] as const,
    },
    permissions: {
      list: (s: MemberScope) => [...base(s), "admin", "permissions"] as const,
    },
    configurations: {
      list: (s: MemberScope) => [...base(s), "admin", "configurations"] as const,
    },
    system: {
      info: (s: MemberScope) => [...base(s), "admin", "system", "info"] as const,
      workers: (s: MemberScope) => [...base(s), "admin", "system", "workers"] as const,
    },
    missedDeposits: {
      list: (s: MemberScope, filters?: Record<string, unknown>) =>
        [...base(s), "admin", "missed-deposits", filters ?? {}] as const,
    },
    referralCampaigns: {
      list: (s: MemberScope) => [...base(s), "admin", "referral-campaigns"] as const,
    },
  },
  public: {
    payment: (ref: string) => ["public", "payment", ref] as const,
    blockchainCurrencies: () => ["public", "blockchain-currencies"] as const,
  },
};
