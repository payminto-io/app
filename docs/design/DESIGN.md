# Design system

Status: foundation shipped 2026-10-07 on branch `design-system`.
Scope: merchant dashboard (`frontend/`), admin area (`frontend/app/(admin)`), hosted checkout (`checkout/`).
References with Mobbin links: `references.md`. Checklist for restyling a page: `JUNIOR_BRIEF.md`.

## 1. Identity and point of view

We are the gateway where fiat and on-chain money are peers.
A card payment and a USDC transfer land as the same double-entry lines, carry the same fee snapshot, and settle under the same policy.
The interface has to make that true at a glance, so three things that other payment UIs hide are first-class visual concepts here:

1. **Ledger truth.** Every number on screen is a ledger line or a receipt. Amounts are always shown with their currency or asset, always in tabular figures, and a fee or conversion is a separate line, never folded into a total.
2. **Finality.** "Succeeded" is not one thing. A card is authorised, then captured; a chain transfer is seen, then confirmed, then final. We show the step the money is at, and we name the next step.
3. **Settlement.** Where the money ends up (bank, USDC on Solana, held in asset) is a merchant choice and it is visible on the payment, not buried in a report.

**The one memorable thing: the rail.**
Every payment carries a three-segment rail: `Received -> Final -> Settled`.
It is a 3px-tall track with three segments; segments fill as the money crosses each threshold.
It appears in the payments list (as a 36px micro-rail in the status cell), on the payment detail page (full width, with timestamps under each segment), and on the checkout's confirmed state (so the payer sees what the merchant sees).
Fiat and chain use the same three words.
That sameness is the product thesis drawn as a shape, and it is something Stripe, Adyen and every crypto checkout lack.
The brand mark is the rail: a rounded square containing three short bars.

**What makes us not-Stripe.**
Stripe's visual signature is a colour wash and a purple button on white.
Ours is restraint in colour and precision in numbers: ink on paper, one accent (Tide), amber only for "test", and every money value treated as data, not decoration.
We do not use a brand colour as a background anywhere in the product.

**Tone of copy.**
Sentences, not labels, wherever a human has to decide something.
"Received 40.00 of 50.00 USDC. Send the remaining 10.00 USDC to the same address." beats "Partially paid".
Sentence case everywhere, including table headers and nav groups.

## 2. Colour tokens

All values are hex. Tokens are defined in `frontend/app/globals.css` (and mirrored in `checkout/app/globals.css`).
Names are the CSS variables without the leading `--`.

### Neutrals and surfaces

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `canvas` | `#F4F5F7` | `#0F1115` | Page background behind cards |
| `surface` | `#FFFFFF` | `#171A20` | Cards, tables, sidebar, inputs |
| `surface-sunken` | `#EEF0F3` | `#121419` | Table header rows, code blocks, sunken panels |
| `surface-raised` | `#FFFFFF` | `#1D2129` | Popovers, dialogs, sheets |
| `line` | `#E1E4E9` | `#262B33` | Default border and table rules |
| `line-strong` | `#C8CED6` | `#4B5462` | Input borders, dividers that must read |
| `ink` | `#15181D` | `#EDEFF2` | Primary text, primary button background (light) |
| `ink-soft` | `#5B6472` | `#A7AFBA` | Secondary text, labels |
| `ink-faint` | `#8A93A1` | `#6F7886` | Placeholders, captions, disabled text |
| `ink-inverse` | `#FFFFFF` | `#0F1115` | Text on `ink` |

### Accent

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `tide` | `#0B7285` | `#3FB8CC` | Links, focus ring, active nav, selected state, primary chart series |
| `tide-strong` | `#095F6F` | `#5CC9DA` | Link hover |
| `tide-tint` | `#E3F3F6` | `#13343B` | Selected row, active nav background, info strip |

Tide is the only chromatic brand colour.
It is never a background for a page or a card, never a gradient, never applied to body text.

### Status

| Token | Light | Dark | Tint light | Tint dark | Meaning |
| --- | --- | --- | --- | --- | --- |
| `ok` | `#1A7F4B` | `#3DBB78` | `#E4F3EA` | `#12301F` | Final, settled, delivered, active |
| `wait` | `#B45309` | `#E8A33D` | `#FBEEDC` | `#3A2A10` | Pending, confirming, under paid, awaiting approval |
| `bad` | `#B42318` | `#F0665A` | `#FBE9E7` | `#3B1612` | Failed, cancelled, rejected, expired with loss |
| `note` | `#0B7285` | `#3FB8CC` | `#E3F3F6` | `#13343B` | Informational: open, created, over paid (money is safe) |
| `mute` | `#5B6472` | `#A7AFBA` | `#EEF0F3` | `#20252D` | Neutral: expired with no loss, inactive, dismissed |

### Environment

| Token | Light | Dark | Use |
| --- | --- | --- | --- |
| `env-test` | `#D97706` | `#F0B454` | Test environment band, segmented control fill, "Test" chip |
| `env-test-ink` | `#15181D` | `#0F1115` | Text on `env-test` |
| `env-live` | `#15181D` | `#EDEFF2` | Live environment segmented control fill |

Test is loud (amber), live is calm (ink).
The frame shows a 3px `env-test` rule along the very top edge of the viewport while in test; it disappears in live.

### Chart series

`chart-1` = `tide`, `chart-2` = `ink-soft`, `chart-3` = `ok`, `chart-4` = `wait`, `chart-5` = `#7C5CBF` (only for a fifth series; never for status).
Comparison series (previous period) is always `chart-2`.

## 3. Typography

**Family.** IBM Plex Sans for everything readable; IBM Plex Mono for identifiers, addresses, hashes, timers and API keys.
Chosen because Plex has real tabular figures, holds up at 12 to 13px in dense tables, and reads as infrastructure rather than marketing.
Inter is not used.
Loaded with `next/font/google` as `--font-sans` and `--font-mono`.

**Money.** Every amount uses `font-variant-numeric: tabular-nums` (utility `.num`).
Amount and currency are two spans: the amount in `ink`, the currency or asset code after it in `ink-soft` at 0.75em (never below 11px) with a 0.35em gap.
Never a currency symbol alone when an asset could be confused (`$` is never used for USDC).

**Scale** (px / line-height / weight):

| Token | Size | Line | Weight | Use |
| --- | --- | --- | --- | --- |
| `display` | 40 | 1.1 | 600 | Hero amount on checkout and payment detail |
| `h1` | 28 | 1.2 | 600 | Page title |
| `h2` | 20 | 1.3 | 600 | Section title |
| `h3` | 16 | 1.4 | 600 | Card title |
| `lead` | 16 | 1.5 | 400 | Auth product statement, intros |
| `body` | 14 | 1.5 | 400 | Default |
| `body-sm` | 13 | 1.45 | 400 | Tables, nav, dense lists |
| `label` | 12 | 1.35 | 500 | Form labels, table headers, metric labels |
| `caption` | 11 | 1.35 | 400 | Timestamps, helper text; never below 11 |

Letter spacing: -0.01em on body, -0.02em from 20px up, 0 on labels.
No uppercase tracking anywhere except the two-letter currency codes that are already uppercase.

## 4. Spacing

Base unit 4px. The working set is 4, 8, 12, 16, 20, 24, 32, 40, 48, 64.
Page gutter 24px (16px below 640px).
Card padding 20px (16px below 640px).
Table cell padding 12px horizontal, 10px vertical, row height 36px.
Form field vertical rhythm 16px between fields, 24px between groups.
Max content width 1280px for pages, 560px for forms, 360px for auth.

## 5. Radius hierarchy

| Token | Value | Use |
| --- | --- | --- |
| `radius-xs` | 4px | Badges, chips, checkboxes, tiny chrome |
| `radius-sm` | 6px | Buttons, inputs, selects, tabs |
| `radius-md` | 10px | Cards, tables, panels, popovers |
| `radius-lg` | 14px | Dialogs, sheets, the QR panel |
| `radius-full` | 9999px | Avatars, status dots, the rail segments |

Nothing else.
A badge is not a pill; a card is not a dialog.

## 6. Elevation

| Level | Treatment | Use |
| --- | --- | --- |
| 0 | `line` border, no shadow | Cards, tables, the sidebar |
| 1 | `line` border + `0 1px 2px rgba(21,24,29,.04)` | Inputs on hover, metric cards |
| 2 | `line` border + `0 4px 16px rgba(21,24,29,.10)` | Popovers, menus, command palette |
| 3 | `line` border + `0 16px 48px rgba(21,24,29,.18)` | Dialogs, sheets |

Dark mode uses the same shadows at 2x alpha and a 1px `line-strong` border, because shadows vanish on dark surfaces.
No glass, no blur, no inner highlight.

## 7. Status vocabulary

One badge component (`StatusBadge`) maps every backend status string to a label, a tone and a glyph.
A badge is `radius-xs`, 20px tall, 12px text, with a 12px glyph on the left; colour is never the only carrier (the glyph and the word carry it too).

### Payment (intent)

| Backend | Label | Tone | Glyph |
| --- | --- | --- | --- |
| `created` | Created | note | circle |
| `open` | Awaiting payment | wait | clock |
| `confirming` | Confirming | wait | clock |
| `partially_filled` | Under paid | wait | clock |
| `filled`, `confirmed`, `closed` | Paid | ok | check |
| `over_filled` | Over paid | note | arrow-up |
| `refunded` | Refunded | mute | undo |
| `cancelled` | Cancelled | bad | x |
| `expired` | Expired | mute | minus |
| `failed` | Failed | bad | x |

### Attempt (per connector or chain try)

`initiated` Initiated / note, `authorized` Authorised / wait, `captured` Captured / ok, `declined` Declined / bad, `errored` Error / bad.

### Settlement

`pending_approval` Awaiting approval / wait, `approved` Approved / note, `initiated` Initiated / note, `sent` Sent / wait, `processed`, `completed` Settled / ok, `failed` Failed / bad.

### Chain finality

| Label | Tone | Rule |
| --- | --- | --- |
| Seen | note | Transaction observed in mempool or as unconfirmed |
| Confirming (n of N) | wait | Confirmations below the asset's threshold; always show the count |
| Final | ok | Threshold reached; on Solana this is `finalized` commitment |
| Reorged | bad | Previously seen transaction dropped |

The rail segments map to: Received = Seen or Authorised; Final = Final or Captured; Settled = Settlement `completed`.

## 8. Money formatting rules

1. Amounts arrive as minor units (`bigint`) plus a currency or asset code. The formatter is the only place that divides.
2. Fiat: two decimals always (`1,250.00 USD`). Group thousands with a comma; locale switching is a later ticket.
3. Stablecoins: show up to 6 decimals, trim trailing zeros but keep at least 2 (`1,250.00 USDC`, `0.000123 USDC`).
4. Native assets: up to 9 decimals for SOL, 8 for BTC, 18 trimmed to 8 for ETH; same trailing-zero rule.
5. The code follows the amount, never a symbol, never a prefix. `USD`, `USDC`, `SOL`.
6. Signed amounts carry the sign as a character (`-12.00 USD`, `+12.00 USD`) and colour only in ledger views (in = `ok`, out = `ink`, never red for out).
7. Fiat equivalent of a crypto amount is shown only when a recorded rate exists for that ledger line; it is `caption` size, `ink-soft`, prefixed with `~` (`~1,249.80 USD`).
8. A missing amount renders as an empty cell, not `0.00` and not a dash.
9. Fees and conversions are their own lines. A total is the last line and is the only bold one.
10. Never round for display in a way that changes what the ledger holds; if space is short, truncate with the full value in a tooltip.

## 9. Iconography

Lucide, 16px in UI, 14px inside badges and 20px in empty states, stroke 1.75.
Icons sit on the left of text, never on the right, except the external-link glyph after "View on explorer".
Asset marks (USDC, SOL, Visa) are 20px circles with the asset's own colour; they are the only coloured icons.
No emoji anywhere.

## 10. Motion

Motion is a response to a user action or a change in money state, never ambient.

- Hover and focus: 120ms, `ease-out`, colour and border only. No translate on hover.
- Press: 1px translate-y on `:active` for buttons.
- Popovers, menus, sheets: 140ms fade + 4px slide from the anchor side.
- Dialogs: 160ms fade + 0.98 scale.
- The rail: a segment fills over 400ms `ease-in-out` when a state changes while the page is open.
- Confirming states: a 3-dot pulse at 1.2s, the only looping animation in the product.
- Skeletons: 1.6s shimmer, no more than one shimmer per card.
- `prefers-reduced-motion`: all of the above become instant except the 3-dot pulse, which becomes a static glyph.

## 11. Mobile rules

Checkout is designed at 390px first and widened; the dashboard is designed at 1440px and narrowed to 390px.

- Checkout at 390px: single column, summary collapsed into a header row (merchant, amount, chevron), pay button pinned above the safe area, 16px gutters, 48px inputs, 16px input text (prevents iOS zoom).
- Dashboard below 1024px (the tablet breakpoint, Tailwind `lg`): sidebar becomes a sheet from the left; the top bar keeps the environment control, the account menu and the menu button.
- Tables below 1024px become stacked rows, one card-width row per record: the amount or name and the status on line one, reference, customer and date on line two, other fields as labelled rows under it.
  `DataTable` does this for every table; a column picks its place with `stack` (`lead`, `trail`, `meta`, `detail`, `action`, `hidden`).
- Tables at 1024px and up that are wider than their card scroll horizontally inside it; the first column is sticky and gains a 1px rule once the table is scrolled.
- Nothing is hover-only. Row actions are a kebab, not a hover reveal.
- No fixed element may sit inside an ancestor with `transform`, `filter` or `backdrop-filter`.

## 12. Accessibility floor

- Text contrast AA: 4.5:1 for body, 3:1 for 20px+ and for UI borders. Every token pair in section 2 was chosen to pass on its intended surface.
- Focus: 2px `tide` ring with 2px offset on every interactive element, visible on keyboard focus (`:focus-visible`) only.
- Touch targets: 44px minimum on coarse pointers (`.tap` utility grows the hit area with padding and gives it back with negative margin); 32px on fine pointers.
- Every status has a glyph and a word; colour is never the only signal.
- Form fields have visible labels; placeholders are examples, not labels.
- Live regions for checkout state changes (`aria-live="polite"` on the state sentence).
- Dark mode respects `prefers-color-scheme` and can be overridden; both schemes ship together.

## 13. Avoid list (AI-slop tells)

If a page has any of these, it is not done:

- cream + terracotta, or any "warm beige" canvas
- acid green on black
- gradient washes, colour-wash hero panels, aurora blobs
- identical rounded cards with soft grey shadows in a grid
- all-caps eyebrows with tracking
- middle-dot meta strings (`Networks • Encrypted • Live`)
- arrows appended to links (`View all ->`)
- glassmorphism, backdrop blur, inner highlights
- purple-blue gradients, purple primary buttons
- emoji
- a "secure" padlock sentence that asserts something the code does not do
- decorative illustrations in empty states
- fake numbers, deltas with no prior period, "Multi-chain" style capability chips

## 14. Components

Shipped in this pass (`frontend/components`):

| Component | File | Notes |
| --- | --- | --- |
| Button | `ui/button.tsx` | `default` = ink, `outline`, `secondary`, `ghost`, `destructive`, `link`; sizes `xs` `sm` `default` `lg`; 36px default height |
| Input, Label | `ui/input.tsx`, `ui/label.tsx` | 36px, `radius-sm`, `line-strong` border, `tide` focus ring |
| Select | `ui/select.tsx` | Same chrome as Input |
| Badge, StatusBadge | `ui/badge.tsx`, `ui/status-badge.tsx` (and `status-badge.tsx`) | Section 7 map, glyphs |
| Card | `ui/card.tsx` | Level 0, `radius-md`, 20px padding |
| Table, DataTable | `ui/table.tsx`, `ui/data-table.tsx`, `data-table.tsx` | 36px rows, sentence-case 12px headers, sunken header |
| PageHeader | `ui/page-header.tsx`, `page-header.tsx` | h1 + description + actions, breadcrumb above |
| MetricCard | `metric-card.tsx` | Label, tabular value, caption; `variant` prop kept for API stability, all variants now render the same quiet card |
| EmptyState | `empty-state.tsx`, `ui/states.tsx` | Dashed panel, glyph, sentence, one action |
| CurrencyDisplay | `currency-display.tsx` | Section 8 rules |
| Rail | `rail.tsx` | The three-segment finality rail |
| EnvironmentSwitch | `layout/environment-switch.tsx` | Segmented Test / Live |
| AppShell, Sidebar, Topbar, AccountMenu, MobileNav | `layout/*` | Dashboard and admin share one shell |
| AuthShell | `auth-shell.tsx` | One centred column: mark, heading, fields, button, one link. No product copy (owner decision 2026-10-07) |
| Logo | `logo.tsx` | The rail mark and wordmark |

Added in the page pass (`frontend/components`):

| Component | File | Notes |
| --- | --- | --- |
| Pagination | `pagination.tsx` | "21-40 of 134" and two icon buttons; sits in `DataTable`'s `footer` |
| SearchInput | `search-input.tsx` | Input with a search glyph; needs an `aria-label` |
| DetailList, DetailItem | `detail-list.tsx` | Label/value rows; an empty value omits the row |
| DateTime | `date-time.tsx` | Tabular timestamp with the ISO value on hover; renders nothing when missing |
| CopyField | `copy-field.tsx` | Mono identifier with copy; `boxed` for a hero value, inline in tables |
| RowActions | `row-actions.tsx` | The row kebab |
| TableEmpty | `table-empty.tsx` | Empty list with the real header, two ghost rows, one sentence and one action; used by `DataTable` |
| RouteTabs | `route-tabs.tsx` | Underline tabs that link sibling pages (wallets, settings) |
| Notice | `notice.tsx` | One-line strip for a condition the page cannot hide |
| SettingsSection | `settings-section.tsx` | Title left, controls in a card right |
| EventList | `event-list.tsx` | Webhook event names as mono chips |

## 15. Screens

Captured with Chrome DevTools at 390 and 1440px, light and dark, from the local dev server on :3013.
Files live in `docs/design/screens/`.

| Screen | Light 1440 | Dark 1440 | Light 390 | Dark 390 |
| --- | --- | --- | --- | --- |
| Sign in | `signin-1440-light.png` | `signin-1440-dark.png` | `signin-390-light.png` | `signin-390-dark.png` |
| Sign up | `signup-1440-light.png` | `signup-1440-dark.png` | `signup-390-light.png` | `signup-390-dark.png` |
| Shell + primitives (`/design`) | `dashboard-1440-light.png` | `dashboard-1440-dark.png` | `dashboard-390-light.png` | `dashboard-390-dark.png` |
| Mobile drawer | - | - | `dashboard-390-drawer-light.png` | `dashboard-390-drawer-dark.png` |

The dashboard captures are of `/design`, a development-only route (`frontend/app/(public)/design`) that renders the primitives inside the real `AppShell`; `/dashboard` itself needs a session and no local seed account existed when these were taken.
The values on that page are labelled samples, not data.

Page captures from the page pass live in `docs/design/screens/pages/` as `<page>-<width>-<scheme>.png`.
They are taken from `/design/preview/*`, a development-only route (404 in production) that renders the real page components inside the real shell.
It answers API calls with labelled sample fixtures from `frontend/app/(public)/design/preview/fixtures.ts` and intercepts nothing outside that prefix.
Append `?state=empty` or `?state=error` to see the empty and error states.
