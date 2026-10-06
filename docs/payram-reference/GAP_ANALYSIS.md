# Payminto ↔ PayRam Gap Analysis

> Definitive plan-of-attack for Phase 5 (gap closure). One row per PayRam route,
> grounded in `INVENTORY.md` (75 routes), `SCREENS/*.md`, `FLOWS/*.md`, and
> `API_CONTRACTS.md` (62 endpoints) on the PayRam side, and a fresh audit of
> the **real** Payminto frontend at
> `.worktrees/frontend/payminto/frontend/src/app/**` (35 pages, ~10.5k LOC) on
> the Payminto side.
>
> **This document was rewritten 2026-04-07 against the correct frontend.** A
> previous pass mistakenly compared against the stale 9-page skeleton at
> `payminto/frontend/`. The real clone is far more complete than that.

---

## 1. Executive Summary

### Route counts

| Side | Count |
|---|---|
| PayRam routes (canonical, from `INVENTORY.md`) | **75** |
| Payminto routes (current, real clone) | **35** |
| Gap (routes still to add) | **~36** |

The real Payminto frontend lives at
`.worktrees/frontend/payminto/frontend/src/app/`. It already has the
`(auth)` / `(dashboard)` route group split, the `/project/[projectId]/...`
namespace, the 6 shared components (`metric-card`, `page-header`,
`section-label`, `status-badge`, `currency-display`, `blockchain-icon`), the
`lib/{api, mock-data, types, providers, utils, formatters}.ts` data layer, and
the royal-blue Outfit design system. Most existing pages are 200–580 LOC and
hand-roll real tables, charts, modals, and filters off mock data.

### Status totals (vs the 73 in-scope PayRam routes; 2 are N/A builtins)

| Status | Count | Notes |
|---|---|---|
| **Match** (true UI parity, only backend wiring left) | **22** | login, signup, allPayments, paymentDetail, dashboard, customers, growth/analytics, growth/campaigns, promoter detail, deposit-wallet, hot-wallet, sweepContract, address-book, user-payouts, refunds, branding, integrations, userManagement, activityLog, emailConfig, paymentChannels, settings/account/[projectId] |
| **Partial** (route exists, but missing sub-features, empty states, or sub-routes) | **10** | root, manageWallet/wallets, sweepIn, withdraw/referral-payouts, settings/withdrawal, settings/roleManagement, project/[id]/payments/createPaymentLink, project/[id]/payments/invoice, project/[id]/accounts/revenue, project/[id]/accounts/userBalance |
| **Missing** (no route in clone) | **36** | mostly P1/P2 sub-routes, the 5-way wallet split's missing legs, sweep/withdraw asset-scoped pages, settings stubs, public `/pay` `/success`, auth password flows |
| **Skipped / N/A** (P3) | **5** | dead PayRam endpoints (`settings/updater`, `adminControl`, `api`, `policyManagement`, `testTokens`) |
| Builtin (Next.js) | **2** | `_not-found`, `_global-error` |

### Priority distribution (from PayRam side; unchanged)

| Priority | Count | Definition |
|---|---|---|
| **P0** (core flow, must ship) | **24** | login, dashboard, allPayments, createPaymentLink, pay, success, deposit-wallet, hot-wallet, sweepContract, sweepIn, apiKeys, webhook, userManagement, branding, integrations, withdrawal, etc. — most already Match |
| **P1** (important) | **22** | invoice, missed-payments, gasFeeWallet (×3), sweepIn/{btc,eth,usdc,usdt}, withdraw/* (×4), customers, analytics, campaigns, account, activityLog, roleManagement, etc. — many already Match |
| **P2** (nice-to-have) | **14** | settings/{checkoutAppearance, paymentChannels, paymentMethods, paymentsApp, mobileApp, walletManagement, emailConfig}, accounts/{revenue, userBalance}, onramp-payments, growth/promoter detail, addresses, about, merchantMock |
| **P3** (skip / dead code) | **5** | settings/updater, adminControl, api, policyManagement, testTokens |

### Cross-cutting state (what already exists, what's still missing)

These are the architectural bets that the previous rewrite called out as
"BLOCKING decisions". The real clone has already made several of them:

1. **Route namespace** — `/project/[projectId]/...` already in place under
   `src/app/(dashboard)/project/[projectId]/...`. ✅ Resolved.
2. **Auth model** — only `login` (116 LOC) and `signup` (265 LOC) UI exist.
   No `middleware.ts`, no cookie issuance, no logout, no `forgotPassword`/
   `createPassword`/`setPassword`. ❌ Still missing.
3. **Live updates** — TanStack Query is wired in `lib/providers.tsx` but no
   `useLiveQuery` polling abstraction yet, and no websocket client. ❌
   Still missing.
4. **Wallet 5-way split** — clone has `manageWallet/{wallets, deposit-wallet,
   hot-wallet, sweepContract}` (4 of 5). `gasFeeWallet` is missing entirely;
   `wallets/{add, cold, details/[id]}`, `hot-wallet/[id]`,
   `sweepContract/deploy` are missing sub-routes. ⚠️ Partial.
5. **Settings** — clone has 11 settings routes. Missing the index landing,
   `settings/account` (non-project), `roleManagement/role/[id]`,
   `walletManagement`, `settings/webhook`. The 5 P3 stubs intentionally skipped.
6. **Shared layout primitives** — already extracted: `metric-card`,
   `page-header`, `section-label`, `status-badge`, `currency-display`,
   `blockchain-icon`. ✅ Use as the kernel for §5.
7. **Mock data layer** — `lib/api.ts`, `lib/mock-data.ts`, `lib/types.ts`,
   `lib/formatters.ts` all exist. ✅ Resolved.
8. **Auth/dashboard route groups** — `(auth)` and `(dashboard)` exist. ✅
   Resolved.

### Effort bands per domain (residual gap only)

| Domain | Routes still to build | Effort | Why |
|---|---|---|---|
| Auth & infra | 6 | **M** | password flows, logout, middleware, root redirect |
| Payments | 4 | **M** | invoice (deepen), missed-payments, public `/pay`, `/success` |
| Wallets (5-way split residue) | 8 | **L** | `gasFeeWallet` ×3, wallets/{add, cold, details}, hot-wallet/[id], sweepContract/deploy |
| Sweep + Withdraw | 5 | **M** | sweepIn/{btc,eth,usdc,usdt}, address-book/[id] |
| Developers | 1 | **S** | documentation embed |
| Settings | 5 | **M** | index, account, role/[id], walletManagement, webhook (+ deepen withdrawal/roleManagement) |
| Growth/Customers/Accounts | 2 | **S** | revenue, userBalance — deepen from empty-state |
| Onramp + Public + Misc | 5 | **S** | onramp, merchantMock, about, addresses, /payments redirect |
| **Total residual** | **~36** | | down from 66 in the bad pass |

---

## 2. Architectural Decisions (most already resolved by the real clone)

| # | Decision | Status |
|---|---|---|
| 2.1 | `/project/[projectId]/...` namespace | ✅ Resolved — already in place |
| 2.2 | Auth model (cookie vs localStorage) | ✅ **Decided 2026-04-07** — httpOnly+Secure+SameSite cookie for the dashboard via `middleware.ts`, PLUS a separate user-revealable API token surfaced in `settings/api` for MCP/CLI/SDK consumers. Cookie carries the session, token carries programmatic access. |
| 2.3 | Live updates (websocket vs polling) | ✅ **Decided 2026-04-07** — WebSocket parity from day one. Open one WS per tab via `/api/v1/websocket-token/create` on the dashboard layout mount, fan events into the TanStack Query cache via `queryClient.setQueryData`. Build a `useLiveQuery` wrapper so pages don't speak WS directly. |
| 2.4 | Wallet 5-way split | ⚠️ Partial — 4 of 5 top-level routes; sub-routes missing |
| 2.5 | Settings — build all 22 vs prune | ⚠️ Partial — 11 built; recommend skipping P3 (5), building P0/P1 (~5 more) |

**Recommendation:** keep §2.2/§2.3 from the previous pass — httpOnly cookie +
`middleware.ts` shim, and `useLiveQuery` polling abstraction with a 5s
`refetchInterval`. Both are still open and still gate the Phase 5A entries.

---

## 3. Gap Table (canonical)

Sorted by domain. Paths in the "Payminto file" column are relative to
`.worktrees/frontend/payminto/frontend/`.

| PayRam Route | Payminto file | Status | Pri | Gaps (short) | Spec Link |
|---|---|---|---|---|---|
| **Auth & infra** | | | | | |
| `/login` | `src/app/(auth)/login/page.tsx` (116) | Match | P0 | wire to backend; cookie issuance; remember-me persistence | `SCREENS/login.md` |
| `/signup` | `src/app/(auth)/signup/page.tsx` (265) | Match | P1 | wire to backend; mock currently inline | `SCREENS/signup.md` |
| `/forgotPassword` | none | Missing | P1 | email form, mock send, success state | `SCREENS/forgotPassword.md` |
| `/createPassword` | none | Missing | P1 | password + confirm, strength meter, token from `?token=` | `SCREENS/createPassword.md` |
| `/setPassword` | none | Missing | P1 | same shape as `createPassword`, different copy | `SCREENS/setPassword.md` |
| `/` | `src/app/page.tsx` (5) | Partial | P0 | auth-aware redirect: cookie → `/project/all/dashboard` else `/login`; implement in `middleware.ts` | `SCREENS/root.md` |
| logout | none | Missing | P0 | `app/api/auth/logout/route.ts` clears cookie + redirects; sidebar footer button | `FLOWS/auth.md` |
| **Payments** | | | | | |
| `/project/[id]/dashboard` | `src/app/(dashboard)/project/[projectId]/dashboard/page.tsx` (581) | Match | P0 | wire to backend, swap mock for live query | `SCREENS/project-projectId-dashboard.md` |
| `/project/[id]/payments/allPayments` | `src/app/(dashboard)/project/[projectId]/payments/allPayments/page.tsx` (449) | Match | P0 | wire `useLiveQuery`; CSV export still TODO | `SCREENS/project-projectId-payments-allPayments.md` |
| `/project/[id]/payments/[paymentId]` (drawer/detail) | `src/app/(dashboard)/project/[projectId]/payments/[paymentId]/page.tsx` (396) | Match | P0 | wire to backend; refund link CTA into `withdraw/refunds` | (covered in `allPayments` + `pay` specs) |
| `/project/[id]/payments/createPaymentLink` | `src/app/(dashboard)/project/[projectId]/payments/createPaymentLink/page.tsx` (223) | Partial | P0 | step wizard exists; missing success modal w/ QR + share, expiry picker, success/cancel URL fields | `SCREENS/project-projectId-payments-createPaymentLink.md` + `FLOWS/create-payment-link.md` |
| `/project/[id]/payments/invoice` | `src/app/(dashboard)/project/[projectId]/payments/invoice/page.tsx` (60) | Partial | P1 | currently empty-state; build invoice list + create-invoice modal | `SCREENS/project-projectId-payments-invoice.md` |
| `/project/[id]/payments/missed-payments` | none | Missing | P1 | table of expired/cancelled with reason + retry action | `SCREENS/project-projectId-payments-missed-payments.md` |
| `/payments` (public reroute) | `src/app/payments/page.tsx` (712) | Match | P2 | already a substantial marketing/landing page; verify it's the right intent vs PayRam's stub | `SCREENS/payments.md` |
| `/pay` (customer checkout) | none | Missing | P0 | public route, reads `?paymentReference`, QR + countdown + status poll → `/success` | `SCREENS/pay.md` + `FLOWS/create-payment-link.md` |
| `/success` | none | Missing | P0 | thank-you screen, transaction summary | `SCREENS/success.md` |
| **Wallets (5-way split)** | | | | | |
| `/manageWallet` | none | Missing | P1 | landing card grid linking to the 5 sub-sections | `SCREENS/manageWallet.md` |
| `/manageWallet/wallets` | `src/app/(dashboard)/manageWallet/wallets/page.tsx` (214) | Partial | P0 | add per-card action menu, refresh, click-through to details/[id] | `SCREENS/manageWallet-wallets.md` |
| `/manageWallet/wallets/add` | none | Missing | P0 | chain picker + address input + label | `SCREENS/manageWallet-wallets-add.md` |
| `/manageWallet/wallets/cold` | none | Missing | P1 | read-only cold list with threshold indicators | `SCREENS/manageWallet-wallets-cold.md` |
| `/manageWallet/wallets/details/[id]` | none | Missing | P0 | wallet detail w/ Transactions / Sweep Config / Settings tabs | `SCREENS/manageWallet-wallets-details-id.md` |
| `/manageWallet/deposit-wallet` | `src/app/(dashboard)/manageWallet/deposit-wallet/page.tsx` (415) | Match | P0 | wire to backend | `SCREENS/manageWallet-deposit-wallet.md` + `FLOWS/deposit-wallet-setup.md` |
| `/manageWallet/hot-wallet` | `src/app/(dashboard)/manageWallet/hot-wallet/page.tsx` (376) | Match | P0 | wire to backend | `SCREENS/manageWallet-hot-wallet.md` |
| `/manageWallet/hot-wallet/[id]` | none | Missing | P0 | detail page, balance, tx list, send modal | `SCREENS/manageWallet-hot-wallet-id.md` |
| `/manageWallet/sweepContract` | `src/app/(dashboard)/manageWallet/sweepContract/page.tsx` (382) | Match | P0 | wire to backend | `SCREENS/manageWallet-sweepContract.md` + `FLOWS/sweep-cycle.md` |
| `/manageWallet/sweepContract/deploy` | none | Missing | P0 | deploy wizard: chain → factory → params → tx confirm → status | `SCREENS/manageWallet-sweepContract-deploy.md` |
| `/manageWallet/gasFeeWallet` | none | Missing | P1 | gas tank list per chain | `SCREENS/manageWallet-gasFeeWallet.md` |
| `/manageWallet/gasFeeWallet/add` | none | Missing | P1 | chain + generate/import private key + label | `SCREENS/manageWallet-gasFeeWallet-add.md` |
| `/manageWallet/gasFeeWallet/edit` | none | Missing | P1 | edit label, top-up, delete | `SCREENS/manageWallet-gasFeeWallet-edit.md` |
| **Sweep & Withdraw** | | | | | |
| `/sweepIn` | `src/app/(dashboard)/sweepIn/page.tsx` (301) | Partial | P0 | has Funds + History tabs; missing 3-step consolidation wizard, asset drilldown links | `SCREENS/sweepIn.md` + `FLOWS/sweep-cycle.md` |
| `/sweepIn/btc` | none | Missing | P1 | btc-scoped consolidation page | `SCREENS/sweepIn-btc.md` |
| `/sweepIn/eth` | none | Missing | P1 | eth-scoped page | `SCREENS/sweepIn-eth.md` |
| `/sweepIn/usdc` | none | Missing | P1 | usdc-scoped page | `SCREENS/sweepIn-usdc.md` |
| `/sweepIn/usdt` | none | Missing | P1 | usdt-scoped page | `SCREENS/sweepIn-usdt.md` |
| `/withdraw/user-payouts` | `src/app/(dashboard)/withdraw/user-payouts/page.tsx` (347) | Match | P0 | wire to backend; CSV upload modal still TODO | `SCREENS/withdraw-user-payouts.md` + `FLOWS/withdraw-payouts.md` |
| `/withdraw/referral-payouts` | `src/app/(dashboard)/withdraw/referral-payouts/page.tsx` (148) | Partial | P1 | thinner than PayRam; add status filter chips + batch select | `SCREENS/withdraw-referral-payouts.md` |
| `/withdraw/address-book` | `src/app/(dashboard)/withdraw/address-book/page.tsx` (386) | Match | P0 | wire to backend; CSV upload still TODO | `SCREENS/withdraw-address-book.md` |
| `/withdraw/address-book/[id]` | none | Missing | P1 | detail page, edit/delete, payout history for this address | `SCREENS/withdraw-address-book-id.md` |
| `/withdraw/refunds` | `src/app/(dashboard)/withdraw/refunds/page.tsx` (219) | Match | P1 | wire to backend; deep-link from payment detail | `SCREENS/withdraw-refunds.md` |
| **Developers** | | | | | |
| `/developers/apiKeys` | `src/app/(dashboard)/developers/apiKeys/page.tsx` (418) | Match | P0 | wire to backend; reveal-once flow already present | `SCREENS/developers-apiKeys.md` |
| `/developers/webhook` | `src/app/(dashboard)/developers/webhook/page.tsx` (469) | Match | P0 | wire to backend; test-delivery modal already present | `SCREENS/developers-webhook.md` + `FLOWS/webhook-config.md` |
| `/developers/documentation` | none | Missing | P1 | embed or link to API docs | `SCREENS/developers-documentation.md` |
| **Settings** | | | | | |
| `/settings` (index) | none | Missing | P0 | settings landing card grid linking to all sub-pages | `SCREENS/settings.md` |
| `/settings/account` (global) | none | Missing | P1 | profile, email, password change, 2FA toggle | `SCREENS/settings-account.md` |
| `/settings/account/[projectId]` | `src/app/(dashboard)/settings/account/[projectId]/page.tsx` (568) | Match | P1 | wire to backend | `SCREENS/settings-account-projectId.md` |
| `/settings/activityLog` | `src/app/(dashboard)/settings/activityLog/page.tsx` (463) | Match | P1 | wire to backend | `SCREENS/settings-activityLog.md` |
| `/settings/branding` | `src/app/(dashboard)/settings/branding/page.tsx` (258) | Match | P0 | wire to backend; live `/pay` preview iframe optional | `SCREENS/settings-branding.md` |
| `/settings/integrations` | `src/app/(dashboard)/settings/integrations/page.tsx` (309) | Match | P0 | wire to backend; latency test still mock | `SCREENS/settings-integrations.md` |
| `/settings/userManagement` | `src/app/(dashboard)/settings/userManagement/page.tsx` (533) | Match | P0 | wire to backend | `SCREENS/settings-userManagement.md` + `FLOWS/user-management.md` |
| `/settings/roleManagement` | `src/app/(dashboard)/settings/roleManagement/page.tsx` (54) | Partial | P1 | currently empty-state; build role list + permission count | `SCREENS/settings-roleManagement.md` |
| `/settings/roleManagement/role/[id]` | none | Missing | P1 | permission matrix (resource × action) | `SCREENS/settings-roleManagement-role-roleId.md` |
| `/settings/walletManagement` | none | Missing | P1 | global cold thresholds, sweep cadence | `SCREENS/settings-walletManagement.md` |
| `/settings/withdrawal` | `src/app/(dashboard)/settings/withdrawal/page.tsx` (126) | Partial | P0 | thin shell; build cold-wallet validation + per-network thresholds + daily limit | `SCREENS/settings-withdrawal.md` |
| `/settings/webhook` (global) | none | Missing | P1 | global webhook secret + signature scheme | `SCREENS/settings-webhook.md` |
| `/settings/emailConfig` | `src/app/(dashboard)/settings/emailConfig/page.tsx` (290) | Match | P1 | wire to backend; test-send button | `SCREENS/settings-emailConfig.md` |
| `/settings/paymentChannels` | `src/app/(dashboard)/settings/paymentChannels/page.tsx` (400) | Match | P2 | wire to backend | `SCREENS/settings-paymentChannels.md` |
| `/settings/checkoutAppearance` | none | Missing | P2 | theme + logo + custom CSS | `SCREENS/settings-checkoutAppearance.md` |
| `/settings/paymentMethods` | none | Missing | P2 | per-method toggles | `SCREENS/settings-paymentMethods.md` |
| `/settings/paymentsApp` | none | Missing | P2 | mobile payments app config | `SCREENS/settings-paymentsApp.md` |
| `/settings/mobileApp` | none | Missing | P2 | mobile download links + config | `SCREENS/settings-mobileApp.md` |
| `/settings/{updater, adminControl, api, policyManagement, testTokens}` | none | Skipped | P3 | dead/empty in PayRam — see §8 | — |
| **Growth + Customers + Accounts** | | | | | |
| `/project/[id]/growth/analytics` | `src/app/(dashboard)/project/[projectId]/growth/analytics/page.tsx` (266) | Match | P1 | wire to backend; recharts already present | `SCREENS/project-projectId-growth-analytics.md` |
| `/project/[id]/growth/analytics/promoter/[id]` | `src/app/(dashboard)/project/[projectId]/growth/analytics/promoter/[promoterId]/page.tsx` (167) | Match | P2 | wire to backend | `SCREENS/project-projectId-growth-analytics-promoter-promoterId.md` |
| `/project/[id]/growth/campaigns` | `src/app/(dashboard)/project/[projectId]/growth/campaigns/page.tsx` (368) | Match | P1 | wire to backend | `SCREENS/project-projectId-growth-campaigns.md` |
| `/project/[id]/customers` | `src/app/(dashboard)/project/[projectId]/customers/page.tsx` (202) | Match | P1 | wire to backend; payment-history drawer | `SCREENS/project-projectId-customers.md` |
| `/project/[id]/accounts/revenue` | `src/app/(dashboard)/project/[projectId]/accounts/revenue/page.tsx` (34) | Partial | P2 | currently empty-state; build card-onramp revenue table | `SCREENS/project-projectId-accounts-revenue.md` |
| `/project/[id]/accounts/userBalance` | `src/app/(dashboard)/project/[projectId]/accounts/userBalance/page.tsx` (33) | Partial | P2 | currently empty-state; build per-user ledger | `SCREENS/project-projectId-accounts-userBalance.md` |
| **Onramp + Public + Misc** | | | | | |
| `/onramp-payments` | none | Missing | P2 | card-to-crypto onramp table | `SCREENS/onramp-payments.md` |
| `/merchantMock` | none | Missing | P2 | demo merchant playground | `SCREENS/merchantMock.md` |
| `/about` | none | Missing | P2 | version info, license, links | `SCREENS/about.md` |
| `/addresses` | none | Missing | P2 | public address lookup | `SCREENS/addresses.md` |
| `/referral/[token]` | none | N/A | P3 | referral landing — out of dashboard scope | `SCREENS/referral-token.md` |
| `/_not-found` | builtin | N/A | — | Next.js builtin | — |

---

## 4. Gaps by Domain (residual only)

> Format note: only Missing / Partial entries are detailed below. Match rows
> are tracked in §3 and need only backend wiring (handled later, not Phase 5).

### 4.1 Auth & infra (residual: 6)

- **`/forgotPassword`** — Missing, P1. Email form, mock send, success state.
  Spec: `SCREENS/forgotPassword.md`. Depends on §2.2.
- **`/createPassword`** — Missing, P1. Password + confirm, strength meter,
  `?token=` query. New component: `PasswordStrengthMeter`.
- **`/setPassword`** — Missing, P1. Same shape, different copy.
- **`/`** — Partial. `src/app/page.tsx` is 5 lines. Add `middleware.ts`
  redirect: cookie → `/project/all/dashboard` else `/login`. Depends on §2.2.
- **logout** — Missing, P0. `app/api/auth/logout/route.ts` clears cookie +
  redirects; sidebar footer button calls it.
- **`middleware.ts`** — Missing infra. Auth guard reading `payminto_session`
  cookie.

### 4.2 Payments (residual: 4)

- **`/project/[id]/payments/createPaymentLink`** — Partial (223 LOC). Wizard
  shell exists. Add: success modal w/ QR + copy + share + "View payment",
  expiry picker, success/cancel URL fields, real submit handler.
- **`/project/[id]/payments/invoice`** — Partial (60 LOC, empty-state). Build
  invoice list + create-invoice modal w/ line items.
- **`/project/[id]/payments/missed-payments`** — Missing, P1. Table of
  expired/cancelled w/ reason + retry action.
- **`/pay`** — Missing, P0. Public route, no auth, `?paymentReference=`, QR
  + countdown + 5s status poll → `/success`. New layout: `PublicLayout`.
- **`/success`** — Missing, P0. Thank-you screen + transaction summary.

### 4.3 Wallets (5-way split residual: 8)

- **`/manageWallet`** — Missing, P1. Landing card grid.
- **`/manageWallet/wallets`** — Partial (214 LOC). Add per-card action menu
  (View / Refresh / Send / Receive), refresh button, click-through to details.
- **`/manageWallet/wallets/add`** — Missing, P0. Chain picker + address input.
- **`/manageWallet/wallets/cold`** — Missing, P1. Read-only cold list.
- **`/manageWallet/wallets/details/[id]`** — Missing, P0. Detail page w/ tabs.
- **`/manageWallet/hot-wallet/[id]`** — Missing, P0. Detail page, send modal.
- **`/manageWallet/sweepContract/deploy`** — Missing, P0. Deploy wizard.
- **`/manageWallet/gasFeeWallet`** + **`/add`** + **`/edit`** — all Missing,
  P1. Gas tank CRUD.

### 4.4 Sweep & Withdraw (residual: 5)

- **`/sweepIn`** — Partial (301 LOC). Add: 3-step consolidation wizard,
  countdown timer, asset drilldown links.
- **`/sweepIn/{btc, eth, usdc, usdt}`** — all Missing, P1. Asset-scoped pages.
- **`/withdraw/referral-payouts`** — Partial (148 LOC). Add status filter
  chips + batch select.
- **`/withdraw/address-book/[id]`** — Missing, P1. Detail / edit / payout
  history.

### 4.5 Developers (residual: 1)

- **`/developers/documentation`** — Missing, P1. Embed or link to API docs.

### 4.6 Settings (residual: 5 build + 2 deepen)

- **`/settings`** index — Missing, P0. Card grid landing.
- **`/settings/account`** (global) — Missing, P1. Profile + password + 2FA.
- **`/settings/roleManagement`** — Partial (54 LOC, empty-state). Build role
  list + permission count + create button.
- **`/settings/roleManagement/role/[id]`** — Missing, P1. Permission matrix.
- **`/settings/walletManagement`** — Missing, P1. Global cold thresholds.
- **`/settings/withdrawal`** — Partial (126 LOC). Build cold-wallet
  validation + per-network thresholds + daily limit.
- **`/settings/webhook`** (global) — Missing, P1. Global secret + signature.
- P2 stubs (`checkoutAppearance`, `paymentMethods`, `paymentsApp`,
  `mobileApp`) — placeholder shells in Phase 5D.
- P3 stubs (`updater`, `adminControl`, `api`, `policyManagement`,
  `testTokens`) — skipped, see §8.

### 4.7 Growth + Customers + Accounts (residual: 2)

- **`/project/[id]/accounts/revenue`** — Partial (34 LOC, empty-state). Build
  card-onramp revenue table.
- **`/project/[id]/accounts/userBalance`** — Partial (33 LOC, empty-state).
  Build per-user ledger.

### 4.8 Onramp + Public + Misc (residual: 5)

- **`/onramp-payments`** — Missing, P2. Card-to-crypto onramp table.
- **`/merchantMock`** — Missing, P2. Demo merchant playground.
- **`/about`** — Missing, P2. Version info.
- **`/addresses`** — Missing, P2. Public address lookup.
- **`/payments`** — already exists at `src/app/payments/page.tsx` (712 LOC).
  Verify intent vs PayRam stub; otherwise leave as Match.

---

## 5. Shared Components to Extract (NEW only)

The 6 already-shipped shared components (`metric-card`, `page-header`,
`section-label`, `status-badge`, `currency-display`, `blockchain-icon`) at
`.worktrees/frontend/payminto/frontend/src/components/` are **not listed
here** — they exist. Build these NEW ones in
`src/components/shared/` once the rule-of-three is met.

| Component | Purpose | Used by | Pri |
|---|---|---|---|
| `DataTable` | sort + filter + pagination + row actions + batch + empty state | allPayments (already hand-rolls), customers, address-book, payouts, refunds, invoices, missed-payments, activityLog, userManagement, contracts, transactions | P0 — biggest single ROI; many existing pages reinvent this |
| `EmptyState` | icon + title + body + CTA | invoice, revenue, userBalance, roleManagement (all currently inline) | P0 |
| `Wizard` | multi-step form shell with progress | createPaymentLink (deepen), deposit-wallet (already inline), sweepContract/deploy, sweepIn, webhook create | P0 |
| `ChainTabs` | tab strip per chain (BTC/ETH/Base/Polygon/Tron) | deposit-wallet, hot-wallet, sweepIn, integrations | P0 |
| `CopyToClipboard` | copy + toast | apiKeys, paymentLink success, address-book, /pay | P0 |
| `ApiKeyReveal` | reveal-once secret with copy | developers/apiKeys (already inline), webhook secret | P0 |
| `QRDisplay` | QR for address or payment URL | createPaymentLink, deposit-wallet, /pay | P0 |
| `ProjectSwitcher` | topbar dropdown for project selection | dashboard layout | P0 |
| `ConfirmDialog` | yes/no destructive action confirm | apiKeys revoke, wallet delete, role delete | P0 |
| `LiveBadge` | green dot indicating websocket connected | dashboard, allPayments, sweepIn | P1 |
| `CountdownTimer` | mm:ss countdown for sweep cycle / payment expiry | sweepIn, /pay, createPaymentLink | P1 |
| `DateRangePicker` | calendar range with presets | allPayments, analytics, activityLog | P1 |
| `FilterChipGroup` | toggleable chip row | allPayments, sweepIn, activityLog, payouts, referral-payouts | P0 |
| `CSVUploadModal` | drag-drop, parse, preview, errors, submit | address-book, user-payouts | P1 |
| `ColorPicker` | with preset palette | branding | P1 |
| `PayloadViewer` | JSON tree viewer | webhook test, activityLog | P1 |
| `PermissionMatrix` | resource × action checkbox grid | roleManagement/role/[id] | P1 |
| `PasswordStrengthMeter` | bar + label for password strength | createPassword, setPassword | P1 |
| `AuthLayout` | centered card layout | login + signup already share, extract | P0 |
| `PublicLayout` | minimal layout for /pay /success /addresses | public pages | P0 |
| `BalanceRefreshButton` | spinner + click to refetch | wallets, hot-wallet, sweepIn | P1 |
| `RPCRow` + `LatencyBadge` | per-chain RPC config row | settings/integrations (already inline) | P2 |

---

## 6. Backend / Mock Data Additions Needed

The data layer at `.worktrees/frontend/payminto/frontend/src/lib/` already has
`api.ts`, `mock-data.ts`, `types.ts`, `providers.tsx`, `utils.ts`, and
`formatters.ts`. Residual additions:

- **`src/lib/use-live-query.ts`** (NEW) — `useQuery` wrapper with
  `refetchInterval: 5000` and a `live: true` flag. Phase 5 polling
  abstraction (§2.3) — swap to websockets later without touching pages.
- **`middleware.ts`** (NEW, project root) — auth guard reading
  `payminto_session` cookie. Redirect unauthed → `/login`. See §2.2.
- **`src/app/api/auth/logout/route.ts`** (NEW) — clear cookie + redirect.
- Extend `lib/types.ts` for the missing domains: `GasFeeWallet`,
  `PermissionMatrix`, `Refund`, `MissedPayment`, `OnrampPayment`,
  `WalletDetail`, `HotWalletDetail`, `SweepDeployTx`.
- Extend `lib/mock-data.ts` accordingly.

---

## 7. Phased Implementation Order

Phase 5-pre is now mostly done by the worktree clone. Residual phases:

### Phase 5-pre — Architectural setup (residual, ~½ day)

1. Resolve §2.2 (auth model — cookie + middleware) and §2.3 (live query
   abstraction).
2. Add `middleware.ts` + `app/api/auth/logout/route.ts`.
3. Add `lib/use-live-query.ts`.
4. Extract Phase-1 NEW shared components from §5: `DataTable`, `EmptyState`,
   `Wizard`, `ChainTabs`, `CopyToClipboard`, `QRDisplay`, `ConfirmDialog`,
   `ProjectSwitcher`, `AuthLayout`, `PublicLayout`, `FilterChipGroup`,
   `ApiKeyReveal`.

### Phase 5A — Auth password flows + public checkout + core wallet sub-routes (P0/P1, ~3 days)

**9 entries:**
1. `forgotPassword`
2. `createPassword`
3. `setPassword`
4. root redirect (`/`) + `middleware.ts`
5. logout
6. `/pay` (public)
7. `/success` (public)
8. `manageWallet/wallets/add`
9. `manageWallet/wallets/details/[id]`

Plus deepen: `createPaymentLink` success modal; `manageWallet/wallets`
action menu.

### Phase 5B — Sweep + Withdraw + Wallet split residue (P0/P1, ~3 days)

**10 entries:**
10. `manageWallet` index
11. `manageWallet/wallets/cold`
12. `manageWallet/hot-wallet/[id]`
13. `manageWallet/sweepContract/deploy`
14. `manageWallet/gasFeeWallet`
15. `manageWallet/gasFeeWallet/add`
16. `manageWallet/gasFeeWallet/edit`
17. `sweepIn/btc`, `sweepIn/eth`, `sweepIn/usdc`, `sweepIn/usdt` (4 routes,
    1 entry by template)
18. `withdraw/address-book/[id]`
19. Deepen `sweepIn` (wizard + countdown); deepen `withdraw/referral-payouts`
    (filters + batch).

### Phase 5C — Settings residue + Developers docs (P0/P1, ~2 days)

**8 entries:**
20. `developers/documentation`
21. `settings` index
22. `settings/account` (global)
23. `settings/roleManagement` deepen
24. `settings/roleManagement/role/[id]`
25. `settings/walletManagement`
26. `settings/withdrawal` deepen
27. `settings/webhook` (global)

### Phase 5D — Accounts deepen + onramp + misc + P2 placeholders (P1/P2, ~2 days)

**12 entries:**
28. `project/[id]/payments/invoice` (deepen)
29. `project/[id]/payments/missed-payments`
30. `project/[id]/accounts/revenue` (deepen)
31. `project/[id]/accounts/userBalance` (deepen)
32. `onramp-payments`
33. `merchantMock`
34. `about`
35. `addresses`
36. `settings/checkoutAppearance`
37. `settings/paymentMethods`
38. `settings/paymentsApp`
39. `settings/mobileApp`

### Phase 5E — Skipped (§8)

P3 routes intentionally not built.

---

## 8. Out of Scope

Unchanged from the previous pass — these are dead/empty in PayRam:

| Route | Why |
|---|---|
| `/settings/updater` | All 4 `/system/updater/*` endpoints 404 in PayRam captures |
| `/settings/adminControl` | Empty placeholder in PayRam |
| `/settings/api` | Functional duplicate of `/developers/apiKeys` |
| `/settings/policyManagement` | Empty placeholder |
| `/settings/testTokens` | Empty placeholder |
| `/referral/[token]` | Out of dashboard scope — belongs on `payminto/landing/` |
| `/_not-found`, `/_global-error` | Next.js builtins |

If any of these become user-priority later, promote from §8 → §4 with a real
spec.

---

## Appendix: Source documents

- `INVENTORY.md` — 75 PayRam routes
- `SCREENS/*.md` — 75 per-screen specs
- `FLOWS/*.md` — 7 end-to-end flows
- `API_CONTRACTS.md` — 62 endpoints
- Real Payminto frontend: `.worktrees/frontend/payminto/frontend/src/app/**`
  (35 pages) and `.worktrees/frontend/payminto/frontend/AI_CONTEXT.md`
