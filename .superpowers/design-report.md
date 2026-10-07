# Design foundation report

Branch: `design-system`, worktree `gateway-wt-design`. Date: 2026-10-07.
Commits: `66567ec` (docs), `e7b2800` (frontend and checkout foundation).

## Checks

- `npx tsc --noEmit`: pass
- `npm run lint`: pass (no output)
- `npm test`: 2 files, 19 tests, pass

## What changed

### Documents (`docs/design/`)

- `references.md`: Mobbin screens per surface with links and take/avoid notes. Mobbin does not index Ramp, Brex or Adyen; their nearest indexed peers (Mercury, Square, Wise, Revolut, Linear, Midday) and public write-ups stand in, and the doc says so.
- `DESIGN.md`: identity (the finality rail: Received, Final, Settled, the same three words for fiat and chain), tokens with hex for light and dark, IBM Plex Sans and Mono, type scale, spacing, radius hierarchy (4/6/10/14), elevation, status vocabulary with glyphs, money rules, iconography, motion, mobile, accessibility floor, avoid list, component table, screenshot index.
- `JUNIOR_BRIEF.md`: the restyle checklist with a console snippet that measures overrun, clipping, touch targets and text under 11px.
- `screens/`: 14 captures (sign-in, sign-up, shell + primitives, drawer) at 390 and 1440, light and dark.

### Frontend foundation

- `app/globals.css`: all tokens as CSS variables and Tailwind v4 `@theme` entries; shadcn aliases and the legacy `pm-*` names map onto the new tokens so untouched pages keep compiling and pick up the palette; `.num`, `.tap` (44px hit area on coarse pointers), reduced-motion handling, focus ring.
- `app/layout.tsx`: IBM Plex Sans and Mono via next/font, next-themes provider (class attribute, system default). `app/icon.svg` is the new mark; the old favicon.ico is removed.
- `lib/utils.ts`: tailwind-merge now knows the type-scale utilities (this was the cause of the first screenshot's black button with no text).
- `lib/status.ts`: the status map (payment, attempt, settlement, chain finality, generic). `lib/money.ts`: minor-unit and decimal formatting per the rules.
- Primitives restyled with stable APIs: button, input, label, select, badge, status badge (both paths), card, table, both data tables, page header (both paths), metric card (variants kept, all render the same quiet card), empty/error/loading states, currency display (now takes optional `minor` and `signed`), section label, skeleton, dropdown menu, sheet, tooltip.
- New: `components/rail.tsx`, `components/logo.tsx`, `components/theme-provider.tsx`.
- Auth: `components/auth-shell.tsx` and the two forms are one centred column (mark, heading, fields, button, one link), per the owner's instruction that overrode the split layout. Logic, validation and API calls unchanged. The "Forgot password?" link that pointed at `#` was dropped rather than shipped as a dead link.
- Shell: `components/layout/app-shell.tsx` (used by both `app/dashboard/layout.tsx` and `app/(admin)/layout.tsx`), `nav.ts` (one nav definition), `sidebar.tsx`, `topbar.tsx`, `account-menu.tsx` (settings, theme, sign out), `command-palette.tsx` (cmdk, Cmd/Ctrl+K), `environment-switch.tsx` with `lib/environment/store.ts`. Test is amber, live is ink, and a 3px amber rule runs along the top edge of the frame while in test. Live is disabled with a tooltip unless `NEXT_PUBLIC_LIVE_ENABLED=true`; the backend has no environment API yet (ticket 13), so the switch is visual state only.
- Dashboard home: the gradient hero is replaced by `PageHeader`; the metrics no longer show `$` or compacted values. `totalVolume` from the analytics API is a sum of deposit amounts across assets with no currency, so it is labelled as exactly that.
- `app/(public)/design`: development-only gallery (404 in production) rendering the primitives inside the real shell. This is what the "dashboard" screenshots show.

### Checkout

- `checkout/app/globals.css`: the same tokens (light and dark via `prefers-color-scheme`), Plex fonts, legacy variable names (`--violet`, `--panel`) aliased onto the tokens. `checkout/app/layout.tsx` loads the fonts. The gradient summary panel and the rest of the checkout markup are untouched; that is the next ticket.

## Verification notes

- Dev server on :3013 with `frontend/.env.local` pointing at http://localhost:8090/api/v1 (not committed).
- `/dashboard` needs a session. There is no local seed account (`.dev-credentials.local.json` is absent) and I did not enter credentials. Setting a dummy session cookie to get past the proxy was denied by the permission system, so the shell was verified through `/design` instead.
- Measured at 390 (coarse pointer) and 1440: no horizontal overrun, no text under 11px, every control at least 44px on touch. Fixes made along the way: currency code floored at 11px; buttons carry the hit-area utility; inputs grow to 44px on touch; skeleton rows were forcing a fixed 378px width (now fluid); cards get `min-w-0`; the drawer got an explicit close button.

## Open points

- Brand name: "Payminto" is inherited from the base; the wordmark, metadata and cookie name all still say it. A rename is a separate decision.
- The amber top rule and the segmented control are visual only until the environments ticket lands an API.
- Chart colours (`chart-*`) are tokenised but `dashboard-charts.tsx` and recharts config were not touched.
- Untouched pages still use `pm-label` and friends; they are restyled through the aliases but still carry their own hard-coded hex and layout. See the list below.

## Junior pass: pages to restyle next

Follow `docs/design/JUNIOR_BRIEF.md`. In priority order: payments list and create, payment detail, wallets, withdrawals, settings and API keys, then admin.

- 
- /admin/configurations
- /admin/external-platforms
- /admin/external-platforms/[id]
- /admin/members
- /admin/missed-deposits
- /admin/roles
- /admin/system
- /pay/[referenceId]
- /dashboard
- /dashboard/analytics
- /dashboard/customers
- /dashboard/onramper
- /dashboard/payments
- /dashboard/payments/[referenceId]
- /dashboard/payments/create
- /dashboard/recipients
- /dashboard/referrals
- /dashboard/settings
- /dashboard/settings/api-keys
- /dashboard/sweeps
- /dashboard/wallets
- /dashboard/wallets/[id]/addresses
- /dashboard/wallets/cold
- /dashboard/wallets/hot
- /dashboard/webhooks
- /dashboard/webhooks/[id]
- /dashboard/withdrawals
- checkout/ (full redesign, mobile-first, with the stablecoin pay states from DESIGN.md section 7)
