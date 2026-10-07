# UI pages report

Branch: `ui-pages`, worktree `gateway-wt-ui-pages`. Date: 2026-10-07.
Scope: every page in `frontend/` listed in the task, restyled onto the design system in `docs/design/DESIGN.md`, presentation only.

## Status

All listed pages are done.
`/pay/[referenceId]` is a redirect to the checkout app; its unused legacy view was removed rather than restyled (see below).

## Checks

- `npx tsc --noEmit`: pass
- `npm run lint`: pass, no warnings
- `npm test`: 2 files, 19 tests, pass
- Measurements (brief snippet, extended so a `.tap` element's `::after` hit area counts) on 26 pages at 1440 and 390 (touch emulated), light and dark: 104 of 104 pass with no horizontal overrun, no clipped content, no touch target under 44px and no text under 11px.
  Console errors other than the expected `/api/auth/me` 401 (no session) were treated as failures; there were none.

## Commits

| Commit | Group |
| --- | --- |
| `4e5eb83` | Payments list, create, detail; shared primitives; `/design/preview` |
| `7aab5b8` | Dashboard home |
| `89851c2` | Wallets (deposit, hot, cold, address pool); Dialog, FormField, chain mark |
| `a099a58` | Withdrawals |
| `7e4cb37` | Settings and API keys |
| `4d20015` | Customers, sweeps, recipients |
| `585bbd2` | Webhooks and endpoint detail |
| `2495ff3` | Analytics, referrals, card payments (onramper) |
| `29a4ce5` | Admin: configuration, system, members, roles, missed deposits, projects |
| `6619a42` | `/pay`: unused legacy checkout view removed |
| `7e14b2e` | Error boundary, loading skeleton, leftovers, DESIGN.md notes |

## How pages were verified without a session

No backend was running on the configured API URL and there is no seed account, so no credentials were entered anywhere.
`frontend/app/(public)/design/preview/*` is a development-only route (404 in production) that renders the real page components inside the real `AppShell`.
It installs a sample in-memory session and answers API calls under that path prefix with fixtures from `preview/fixtures.ts`; every fixture value is labelled sample and the page shows a "Sample data" strip.
Nothing outside `/design/preview` is intercepted, and no fixture data is on a real route.
`?state=empty` and `?state=error` render the empty and error states.
Captures were taken with Playwright (scripted, same checks as the brief) plus Chrome DevTools MCP for interactive states (create-payment success, hot-wallet dialog, API key reveal).

## Screenshots

All in `docs/design/screens/pages/`, named `<page>-<width>-<scheme>.png` (1440 and 390, light and dark).

| Page | Files | Extra states |
| --- | --- | --- |
| /dashboard | `home-*` | `home-empty-*` |
| /dashboard/payments | `payments-*` | `payments-empty-*` |
| /dashboard/payments/create | `payment-create-*` | `payment-create-success-1440-light.png` |
| /dashboard/payments/[referenceId] | `payment-detail-*` | |
| /dashboard/wallets | `wallets-*` | |
| /dashboard/wallets/hot | `wallets-hot-*` | `wallets-hot-empty-*`, `wallets-hot-dialog-1440-light.png` |
| /dashboard/wallets/cold | `wallets-cold-*` | |
| /dashboard/wallets/[id]/addresses | `wallet-addresses-*` | |
| /dashboard/withdrawals | `withdrawals-*` | `withdrawals-empty-*`, `withdrawals-error-*` |
| /dashboard/settings | `settings-*` | |
| /dashboard/settings/api-keys | `settings-api-keys-*` | `settings-api-keys-reveal-1440-light.png` |
| /dashboard/customers | `customers-*` | |
| /dashboard/sweeps | `sweeps-*` | `sweeps-empty-*` |
| /dashboard/recipients | `recipients-*` | |
| /dashboard/webhooks | `webhooks-*` | `webhooks-empty-*` |
| /dashboard/webhooks/[id] | `webhook-detail-*` | |
| /dashboard/analytics | `analytics-*` | |
| /dashboard/referrals | `referrals-*` | |
| /dashboard/onramper | `onramper-*` | |
| /admin/configurations | `admin-configurations-*` | |
| /admin/system | `admin-system-*` | |
| /admin/members | `admin-members-*` | |
| /admin/roles | `admin-roles-*` | |
| /admin/missed-deposits | `admin-missed-deposits-*` | |
| /admin/external-platforms | `admin-projects-*` | |
| /admin/external-platforms/[id] | `admin-project-detail-*` | |

## Shared work

New primitives in `frontend/components/` (each used on 3 or more pages): `Pagination`, `SearchInput`, `DetailList`/`DetailItem`, `DateTime`, `CopyField`, `RowActions`, `TableEmpty`, `RouteTabs`, `Notice`, `SettingsSection`, `EventList`.
`lib/chains.ts` holds chain display names; `lib/status.ts` gained `paymentRail()` and keys for address pool, worker, health, delivery and OTP states.
Primitives moved onto tokens with their APIs kept: `ui/tabs`, `ui/dialog` (backdrop blur removed), `ui/textarea`, `ui/form-field`, `ui/page-header` (breadcrumbs are links, title accepts a node, one height with or without actions), `components/data-table` (empty state keeps the table shape, `total` and `footer` props), `api-key-reveal`, `blockchain-icon` (neutral monogram, no palette colours), `copy-button` (accessible name when icon-only).

## Numbers removed or relabelled because the API does not back them

- Payments list: the metric strip counted settled, pending and expired on the current page of 20 only and showed it as account totals. Removed; the table footer shows the server total.
- Home, analytics, sweeps: volume, swept and gas totals are sums across assets with no currency (`SUM(total_amount)` in `analytics_repo_impl.go`). The `$` prefixes and dollar axis are gone; values are labelled "All assets, summed".
- Analytics: the "Payment trend" chart plotted volume under a payments title; it now plots `PaymentCount`.
- Referrals: "Pending rewards" was always the client fallback `"0"`; removed. "Earned" has no unit, so no `$`. Campaign fixed rewards no longer claim dollars.
- System: the API layer fills missing health fields with `"unknown"` and `"-"`; the page omits those instead of showing them.
- Recipients: an unknown chain used to fall back to the Ethereum icon; it now shows the chain code.
- Payment rail: Received and Final are derived from the intent state; Settled is never filled because the payment API carries no settlement record.

## Removed controls that did nothing

Disabled "Enable 2FA" card (settings), disabled "Trigger sweep" (sweeps), disabled "Remove" on cold wallets, disabled wallet "Manage" kebab, the explainer banners on wallets, hot and cold pages (one `Notice` line kept on hot wallets; its "encrypted at rest" claim was checked against the vault service).

## Left for follow-up

- `/pay/[referenceId]` only redirects to the checkout app; `checkout-view.tsx` was unused and was deleted. The real checkout lives in `checkout/`, owned by another agent.
- Worker start/stop/restart in admin system call API functions that always throw a 501 ("not available in this backend build"); the menu items are kept as before but are effectively dead until the backend lands them.
- Referrals `totalReferred` falls back to 0 in the API layer when stats return 404, which the page cannot tell apart from a real 0. Fixing it needs a change in `lib/api/referrals.ts` (data layer, out of scope here).
- Hot wallet "Low" badge compares the live balance against hard-coded per-coin floors in the page; the floors are a product rule that should come from configuration.
- Shell (not in scope, owned by the foundation): on full-page captures the sidebar background stops at the first viewport; in dark mode the outline button border is faint against the canvas; DESIGN.md asks for a sticky first column in horizontally scrolling tables, which `DataTable` does not do yet.
- `components/wizard.tsx` and `components/ui/data-table.tsx` (the `Column`/`keyOf` variant) have no consumers left; they still carry legacy tokens.
- Mobbin references were checked for lists and payment detail (Stripe transactions, payment detail, customers); the builder, settings and empty-state references were followed from `references.md` notes.
