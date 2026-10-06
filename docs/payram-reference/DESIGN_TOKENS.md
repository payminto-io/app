# PayRam Design Tokens

Extracted from the production CSS bundles in `_raw/chunks/css/*.css`. PayRam
uses a hand-rolled token layer (`--pr-*`) on top of Tailwind v4's generated
`--color-*` palette, plus a small handful of legacy `--app-*` variables left
over from an earlier brand refresh. The brand identity is **PayRam purple +
neon-lime accent + emerald primary** — deliberately distinct from Payminto's
royal-blue identity.

## 1. Brand palette

| Token | Value | Notes |
|---|---|---|
| `--pr-brand-purple` | `#6a0dad` | Primary brand mark — used in the wordmark, hero gradients, sidebar accents |
| `--pr-brand-neon` / `--pr-accent` | `#caff54` | Neon-lime CTA accent (buttons, hover halos, KPI highlights) |
| `--pr-accent-hover` | `#b8f038` (light) / `#d4ff70` (dark) | Hover state for the lime accent |
| `--pr-brand-green` / `--pr-primary` | `#01e46f` (light) / `#09984e` (dark) | "Pay successful" / positive-state green |
| `--pr-brand-green-dark` | `#09984e` | Pressed/active variant |
| `--app-color-primary` | `#6b41eb` | Legacy brand purple (still used in `--app-button-*` set) |
| `--app-progress-fill-color` | `#caff54` | Same neon-lime, used by the legacy progress component |

> **For Payminto:** do not adopt this palette. Our identity is royal blue
> (`#1e40af` family) + slate neutrals. The structural tokens (radii, spacing,
> shadows) below are reusable; the colors are not.

## 2. Semantic colors

PayRam ships **light + dark** variants for every semantic token. Both values
are in the bundled CSS — the active set is gated on `[data-theme="dark"]`.

| Token | Light | Dark |
|---|---|---|
| `--pr-bg` | `#f1f5f9` | `#0f172a` |
| `--pr-bg-subtle` | `#f8fafc` | `#1e293b` |
| `--pr-muted` | `#f1f5f9` | `#1e293b` |
| `--pr-muted-foreground` | `#64748b` | `#94a3b8` |
| `--pr-border` | `#e2e8f0` | `#334155` |
| `--pr-border-subtle` | `#f1f5f9` | `#1e293b` |
| `--pr-border-strong` | `#cbd5e1` | `#475569` |
| `--pr-destructive` | `#ef4444` | `#ef4444` |
| `--pr-destructive-text` | `#991b1b` | `#fca5a5` |
| `--pr-destructive-soft` | `#ef44441a` | `#ef44441a` |
| `--pr-info` | `#3b82f6` | `#3b82f6` |
| `--pr-info-text` | `#1e40af` | `#93c5fd` |
| `--pr-info-soft` | `#3b82f61a` | `#3b82f61a` |
| `--pr-pending` | `#f59e0b` | `#f59e0b` |
| `--pr-pending-soft` | `#f59e0b1a` | `#f59e0b1a` |
| `--pr-alert-critical-bg` | `#fef2f2` | `#270e0e` |
| `--pr-alert-warning-bg` | `#fffbeb` | `#271709` |

The 6-state status palette (success/pending/info/warning/critical/destructive)
is what `StatusBadge`, toast notifications, and the timeline pill on the
payment-detail screen all consume. Payminto's `status-badge.tsx` already maps
1:1 onto these — only the underlying colors change.

## 3. Glassmorphism

PayRam leans on a subtle glass treatment for the top bar and the project
switcher dropdown:

| Token | Light | Dark |
|---|---|---|
| `--pr-glass-bg` | `#ffffffb3` | `#1e293bb8` |
| `--pr-glass-border` | `#ffffff73` | `#ffffff1a` |
| `--pr-glass-blur` | `12px` | `12px` |
| `--pr-glass-shadow` | `0 1px 3px #0f172a0a, 0 4px 12px #0f172a08` | `0 1px 3px #00000026, 0 4px 12px #0000001a` |

## 4. Typography

| Token | Value |
|---|---|
| `--pr-font-sans` | `"Poppins", system-ui, -apple-system, sans-serif` |
| `--pr-font-mono` | `"JetBrains Mono", ui-monospace, monospace` |
| `--pr-font-normal` | `400` |
| `--pr-font-medium` | `500` |
| `--pr-font-semibold` | `600` |
| `--pr-font-bold` | `700` |
| `--pr-leading-tight` | `1.25` |
| `--pr-leading-normal` | `1.5` |
| `--pr-leading-relaxed` | `1.625` |

PayRam loads **Poppins** as its single brand font (no display/serif pairing).
JetBrains Mono is reserved for addresses, hashes, and code snippets in the
developer pages.

## 5. Radii

| Token | Value | Used by |
|---|---|---|
| `--pr-radius-md` | `8px` | inputs, badges, small buttons |
| `--pr-radius-lg` | `12px` | cards, larger buttons, dropdowns |
| `--pr-radius-2xl` | `20px` | hero modals, KPI tiles, the wallet drawer |
| `--pr-radius-full` | `9999px` | pills, avatar wrapper, the lime CTA button |

## 6. Easing curves

| Token | Value | Used by |
|---|---|---|
| `--pr-ease-out` | `cubic-bezier(0, 0, .2, 1)` | most enter transitions |
| `--pr-ease-out-cubic` | `cubic-bezier(.215, .61, .355, 1)` | sidebar collapse, drawer slide |
| `--pr-ease-in-out` | `cubic-bezier(.645, .045, .355, 1)` | hover state cross-fades |
| `--pr-ease-spring` | `cubic-bezier(.175, .885, .32, 1.275)` | the success-modal pop |

## 7. Tailwind v4 layer

PayRam uses Tailwind v4's automatic `--color-{name}-{shade}` set in `oklch()`
form alongside the `--pr-*` tokens. A few notable Tailwind escape-hatches the
SPA reaches for directly (rather than through `--pr-*`):

- `--color-custom-purple: #6a0dad` (alias for the brand mark; used by SVG fills)
- `--color-cream: #f6f6ef` (the empty-state background card)
- `--color-bg-green: #01e46f` and `--color-primary-green: #09984e`
- `--color-white-12: #ffffff1f` (the divider line in the dark sidebar)
- `--color-white-60: #fff9` (faded labels in the top bar)

## 8. Source files

All values above were grepped out of:

- `_raw/chunks/css/97a2c5b708ab828a.css` (131 KB — main app CSS)
- `_raw/chunks/css/efc99a7c0fddca4b.css` (32 KB — Tailwind v4 generated)
- `_raw/chunks/css/715be398208dca58.css` (14 KB — RainbowKit / wagmi modal)
- `_raw/chunks/css/bdc8353fadd67def.css` (8 KB — react-toastify)
- `_raw/chunks/css/d750f7b35207409c.css` (1 KB — react-responsive-modal)

Re-extract any time with:

```bash
cat payminto/docs/payram-reference/_raw/chunks/css/*.css | \
  tr '}' '\n' | grep -oE -- '--pr-[a-zA-Z0-9-]+:[^;]+' | sort -u
```
