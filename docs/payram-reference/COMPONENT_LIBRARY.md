# PayRam Component Library

Reusable UI primitives observed across the captured DOM. PayRam's frontend is
**Tailwind v4** with a thin layer of branded utility prefixes (`bg-pr-*`,
`text-pr-*`, `rounded-pr-*`, `shadow-pr-*`) that resolve to the tokens in
[`DESIGN_TOKENS.md`](./DESIGN_TOKENS.md). Component logic comes from a mix of
**Headless UI**, **react-responsive-modal**, **react-toastify**, and a couple
of in-house wrappers — there is no shadcn/ui dependency.

Snippets below are taken verbatim from the captured DOMs in `captures/`. Class
strings are long because Tailwind is unbundled in production; group them
mentally by purpose (layout / surface / typography / state).

---

## 1. App shell

The dashboard layout is a three-region grid: top bar, left sidebar, main pane.
Captured at [`captures/project-projectId-dashboard/dom.html`](captures/project-projectId-dashboard/dom.html).

### 1.1 Sidebar card

```html
<aside class="flex w-full h-full flex-col overflow-visible m-3 rounded-pr-2xl
              shadow-pr-xl bg-[var(--pr-sidebar-bg)] relative">
  ...
</aside>
```

- Purple, fully-rounded (`rounded-pr-2xl` ≈ 20px) **floating card** with
  margin on all sides — not flush against the viewport edge.
- `--pr-sidebar-bg` is a dedicated token (PayRam-purple in light mode,
  near-black in dark).
- Sections separated by all-caps gray section labels: `ASSETS`, `GENERAL`.

### 1.2 Top bar

A thin (`h-[55px] sm:h-[64px]`) glass header with the wordmark on the left, a
**Testnet Mode** chip (yellow pulse dot — see badge §3 below), the **All
Projects** project switcher, and a profile avatar dropdown on the right.

### 1.3 Main pane card

```html
<section class="w-full flex flex-col bg-pr-surface border border-pr-border
                shadow-pr-sm rounded-pr-lg">
  ...
</section>
```

The main content area uses many of these stacked white cards on a
`bg-pr-bg-subtle` page background.

---

## 2. Buttons

### 2.1 Primary CTA (lime / green)

The most distinctive PayRam element is its **neon-lime** primary button with
a black icon overlay. Used for "Create Payment Link", "Deploy Contract",
"Generate Address", "Confirm Sweep".

- Background: `--pr-accent` `#caff54`, hover `--pr-accent-hover` `#b8f038`.
- Text: black (`#000`).
- Radius: `rounded-pr-lg` (12px).
- Padding: `px-4 py-2` for default, `px-6 py-3` for hero CTAs.
- Includes a leading icon for almost every instance.

### 2.2 Secondary / outline

Purple-outline button used for "Cancel", "Back", and table row actions.

```
border border-pr-border bg-pr-surface hover:bg-pr-muted
text-pr-text rounded-pr-lg px-4 py-2 text-sm font-medium
```

### 2.3 Filter chip / tab pill

Used by the `payments/allPayments` and `activityLog` filter rows. Toggleable;
inactive state is muted gray, active state is filled brand purple.

```html
<button class="px-3 py-1.5 text-xs font-medium rounded-full border
               transition-all duration-200 bg-pr-surface-sunken
               text-pr-text-muted border-pr-border hover:bg-pr-muted">
  Filled
</button>
```

Captured states for "Filled / Partially Filled / Pending / Over Filled /
Cancelled" on the All Payments page — that's the canonical 5-state
[`PaymentStatus`](API_CONTRACTS.md) filter.

### 2.4 Destructive

Red text on hover-only red soft background. Used in row action menus
("Delete", "Revoke", "Ban").

```
text-pr-destructive hover:bg-pr-destructive-soft
rounded-pr-md px-3 py-1.5 text-sm
```

---

## 3. Status badges

PayRam ships a single badge component that resolves variants by status name.
Variants observed in the wild:

| Variant | Background | Text | Used by |
|---|---|---|---|
| `success` | `--pr-primary-soft` (`#01e46f1f`) | `--pr-primary` | "Filled", "Active" |
| `pending` | `--pr-pending-soft` | `--pr-pending` | "Pending", "Processing" |
| `info` | `--pr-info-soft` | `--pr-info-text` | "Partially Filled", "Draft" |
| `destructive` | `--pr-destructive-soft` | `--pr-destructive-text` | "Cancelled", "Failed", "Banned" |
| `muted` | `--pr-muted` | `--pr-muted-foreground` | "Closed", "Archived" |

A pulsing **status dot** is rendered as a sibling for live indicators
(Testnet Mode, websocket connected, sweep cycle running):

```html
<span class="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse"></span>
```

---

## 4. Tables

Long content tables are not present in the SSR'd HTML — every table on
`/payments/allPayments`, `/withdraw/user-payouts`, `/onramp-payments`,
`/settings/activityLog`, and `/settings/userManagement` is **client-side
rendered after a fetch**. The visible structure (from interactions captures)
is:

- **Sticky header row** with sort affordances (arrow icons on hover only).
- **Filter strip** above the header — chip group + search input + date range
  picker. The chip group reuses §2.3.
- **Striped rows** alternating `--app-table-striped-color` (`#6a0dad05`),
  hover `--app-table-hover-color` (`#6a0dad0f`).
- **Row actions** in a kebab dropdown (Headless UI `Menu`).
- **Empty state**: a centered illustration + heading + secondary CTA, on a
  `--color-cream` (`#f6f6ef`) background card.
- **Pagination footer**: `Previous` / `Next` outline buttons + page size
  selector + `Showing N of M` count.

---

## 5. Forms

PayRam uses uncontrolled inputs in custom React wrappers. Field anatomy:

```html
<label class="block text-sm font-medium text-pr-text mb-1.5">Email</label>
<input type="email" name="email"
       placeholder="Email address"
       class="w-full rounded-pr-md border border-pr-border bg-pr-surface
              px-3 py-2 text-sm placeholder:text-pr-text-muted
              focus:border-pr-primary focus:ring-2 focus:ring-pr-primary-soft
              focus:outline-none" />
<p class="text-xs text-pr-destructive-text mt-1">{{validation message}}</p>
```

Every captured form is **two-column on desktop**, single-column under `sm:`.
Inline validation appears under the field (no toast for field-level errors).

Specialized inputs in use:

- **Password strength meter** (`/signup`, `/createPassword`) — four
  side-by-side pill segments; lights up green as the password matches each
  rule (uppercase / lowercase / number / special).
- **Currency amount input** (`/payments/createPaymentLink`) — leading
  fiat-symbol prefix and trailing currency selector dropdown.
- **Network selector** (`/manageWallet/*`) — chain logo grid (BTC, ETH, BASE,
  POLYGON, TRX) with the selected chain highlighted lime.
- **CSV uploader** (`/withdraw/address-book`, `/withdraw/user-payouts`) —
  drop zone + "Browse files" button + parsed-row preview table.
- **Address input** with built-in QR scanner button (camera icon).

---

## 6. Modals

PayRam uses **`react-responsive-modal`** for every modal. Confirmed by the
presence of the matching CSS bundle (`d750f7b35207409c.css`) and the
`ReactModalPortal` div in every captured DOM.

- Centered overlay with `--pr-glass-bg` backdrop and 12px blur.
- Modal card uses `rounded-pr-2xl` (20px), `shadow-pr-xl`, max-width
  `~520px` for forms and `~720px` for the payment-detail drawer.
- Close button is a `Heroicons/x-mark` in the top-right corner.
- Footer is a right-aligned action row: secondary on the left, primary CTA
  on the right.

Modals seen during the crawl (button → label):

- "Generate Address" → wallet QR + copy
- "Deploy Sweep Contract" → multi-step wizard
- "Test Webhook" → payload preview + send
- "Invite User" → email + role selector
- "Reveal API Key" → reveal-once flow with copy + download
- "Confirm Sweep" → cycle-summary recap

---

## 7. Toasts

`react-toastify` (CSS bundle `bdc8353fadd67def.css`). Top-right placement,
auto-dismiss 4s, 4 variants matching `--toastify-color-{success,info,warning,error}`.

PayRam **does not** use sonner. Our Payminto clone does, which is fine —
the UX is interchangeable. We just need to mirror the dismissal duration and
the success-variant icon style.

---

## 8. Charts

`@nivo/*` (line, bar, pie). Used on `/dashboard`, `/project/all/dashboard`,
`/project/all/growth/analytics`, `/onramp-payments`. Tooltips are themed via
the same `--pr-*` token set; the legends sit below the chart, never to the
side.

---

## 9. Web3 wallet connect

`@rainbow-me/rainbowkit` modal (CSS bundle `715be398208dca58.css`). Triggered
from the deploy-contract and gas-fee-wallet pages when a user has to sign a
transaction with their browser wallet. Uses RainbowKit's built-in chrome —
PayRam does not restyle it.

---

## 10. Component-name → Payminto-file map

For Phase 5 (gap closure), here's the cross-walk between PayRam patterns and
the existing Payminto primitives:

| PayRam | Payminto file |
|---|---|
| Status badge | `payminto/frontend/src/components/status-badge.tsx` |
| Currency display | `payminto/frontend/src/components/currency-display.tsx` |
| Blockchain icon | `payminto/frontend/src/components/blockchain-icon.tsx` |
| Page header | `payminto/frontend/src/components/page-header.tsx` |
| Section label | `payminto/frontend/src/components/section-label.tsx` |
| Metric / KPI tile | `payminto/frontend/src/components/metric-card.tsx` |
| Sidebar shell | `payminto/frontend/src/components/sidebar.tsx` |
| Modal | shadcn `dialog` (already installed) |
| Toast | sonner (already installed) |
| Chart | recharts (Payminto chose recharts over nivo — fine) |

New primitives we'll need (do not exist yet in Payminto):

- **Filter chip group** (§2.3)
- **Multi-step wizard shell** (used by deposit-wallet, deploy-contract, signup, sweep)
- **CSV uploader with preview**
- **Reveal-once secret** (API keys, recovery phrases)
- **Address input with QR scanner**
- **Network/chain selector grid**
