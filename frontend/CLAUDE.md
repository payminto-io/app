# Payminto Frontend Dashboard — CLAUDE.md

## What This Is

The Payminto frontend is the merchant-facing web dashboard — the control panel where merchants manage payments, wallets, sweeps, webhooks, analytics, and API keys. It is a Next.js 16 App Router application with a dark-themed UI built on Tailwind CSS v4 and shadcn/ui. It runs on port 3000 and communicates exclusively with the Payminto Go backend at `NEXT_PUBLIC_API_URL` (default: `http://localhost:8080/api/v1`). This is entirely separate from `payminto/landing/` (the marketing site) — do not merge the two.

## Tech Stack

| Component | Version / Package |
|-----------|------------------|
| Framework | Next.js 16.2.2 (App Router) |
| Language | TypeScript 5 (strict mode) |
| UI Framework | React 19.2.4 |
| Styling | Tailwind CSS v4, `tw-animate-css` |
| Component library | shadcn/ui (new-york style), `@base-ui/react` |
| Class merging | `clsx` + `tailwind-merge` → `cn()` helper |
| Variant handling | `class-variance-authority` |
| Icons | `lucide-react` v1.7.0 |
| Data fetching | `@tanstack/react-query` v5.96.2 |
| Charts | `recharts` v3.8.1 |
| QR codes | `qrcode.react` v4.2.0 |
| Linting | ESLint 9 + `eslint-config-next` |
| Build tool | Turbopack (Next.js built-in) |

## Directory Layout

```
frontend/
├── app/                            # Next.js App Router root
│   ├── layout.tsx                  # Root layout — sets className="dark" on <html>, loads fonts, wraps QueryProvider
│   ├── page.tsx                    # Root route — redirects to /dashboard or login
│   ├── globals.css                 # Tailwind v4 @theme inline directives, CSS variables
│   ├── favicon.ico
│   └── dashboard/
│       ├── layout.tsx              # Dashboard shell — sidebar + topbar wrapper
│       ├── page.tsx                # /dashboard — overview / summary stats
│       ├── payments/
│       │   └── page.tsx            # Payment list, search, status filters
│       ├── wallets/
│       │   └── page.tsx            # Wallet addresses per chain
│       ├── sweeps/
│       │   └── page.tsx            # Sweep history and trigger
│       ├── webhooks/
│       │   └── page.tsx            # Webhook endpoint management
│       ├── analytics/
│       │   └── page.tsx            # Charts: volume, conversions
│       └── settings/
│           ├── page.tsx            # General settings
│           └── api-keys/
│               └── page.tsx        # API key creation and revocation
├── components/
│   ├── layout/
│   │   ├── sidebar.tsx             # Left nav — links, active state, collapse
│   │   └── topbar.tsx              # Top bar — breadcrumbs, user menu
│   └── ui/                         # shadcn/ui primitives (Button, Card, Table, Dialog, …)
├── lib/
│   ├── api.ts                      # Typed API client — all fetch calls to backend
│   ├── query-provider.tsx          # TanStack QueryClientProvider (client component)
│   └── utils.ts                    # cn() helper, misc utils
├── public/                         # Static assets
├── components.json                 # shadcn/ui config (style: new-york, baseColor: neutral)
├── next.config.ts
├── tsconfig.json                   # strict: true
└── package.json
```

## Common Commands

```bash
# Development server (Turbopack, hot reload)
npm run dev
# or from monorepo root:
docker compose -f docker-compose.dev.yml up frontend

# Production build
npm run build

# Start production server
npm start

# Lint
npm run lint

# Install new shadcn/ui component
npx shadcn@latest add <component-name>
# Examples:
npx shadcn@latest add dialog
npx shadcn@latest add data-table
```

## Code Conventions

### Server vs Client Components

Next.js App Router defaults to **Server Components**. Add `"use client"` only when you need:
- React hooks (`useState`, `useEffect`, `useQuery`, etc.)
- Browser-only APIs (`window`, `localStorage`, event handlers)
- Context providers

The dashboard pages that show live data are currently client components because they use TanStack Query hooks. Layout files can remain Server Components if they don't use client features.

### Class Merging

Always use the `cn()` helper from `lib/utils.ts` when combining Tailwind classes — never concatenate strings:

```tsx
import { cn } from "@/lib/utils"

<div className={cn("base-classes", isActive && "active-class", className)} />
```

### Tailwind v4 — Important

Tailwind v4 **does not use `tailwind.config.js`**. All theme customisation lives in `app/globals.css` using `@theme inline {}`. Do NOT create a `tailwind.config.ts` or `tailwind.config.js` file.

**Critical pitfall:** Do not create self-referencing CSS variable loops in `@theme inline`. For example, this breaks:
```css
/* BAD — circular reference */
@theme inline {
  --font-sans: var(--font-sans);
}
```

The correct pattern is:
```css
@theme inline {
  --font-sans: ui-sans-serif, system-ui, sans-serif;
}
```

### Dark Theme

The root `<html>` element has `className="dark"` set in `app/layout.tsx`. All components must work correctly in dark mode. Use Tailwind's dark-mode semantic tokens (e.g., `bg-background`, `text-foreground`, `border-border`) rather than hardcoded colours wherever possible.

### Data Fetching with TanStack Query

Data fetching lives in client components using `useQuery` and `useMutation`. Wrap the result in a standard loading/error guard:

```tsx
"use client"
import { useQuery } from "@tanstack/react-query"
import { fetchPayments } from "@/lib/api"

export default function PaymentsPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["payments"],
    queryFn: fetchPayments,
  })

  if (isLoading) return <LoadingSpinner />
  if (error) return <ErrorMessage error={error} />
  // render data
}
```

### API Client

All backend calls go through `lib/api.ts`. Never call `fetch()` directly from a component — add a typed function to `lib/api.ts` instead:

```ts
// lib/api.ts pattern
const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1"

export async function fetchPayments(): Promise<Payment[]> {
  const res = await fetch(`${BASE_URL}/payments`, {
    headers: { "X-API-Key": getStoredApiKey() },
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}
```

Authentication: will use either a JWT cookie (session-based login) or `X-API-Key` header (API key auth). The pattern is consistent in `lib/api.ts`.

### TypeScript

Strict mode is enabled. Key rules:
- No `any` — use `unknown` and narrow with type guards
- All API response shapes should have explicit interfaces in `lib/api.ts`
- Prefer `interface` over `type` for object shapes

## Integration Points

| System | Direction | Details |
|--------|-----------|---------|
| Go backend | outbound | `NEXT_PUBLIC_API_URL` (default `http://localhost:8080/api/v1`) |
| QR code display | local | `qrcode.react` renders wallet address as QR |
| Charts | local | `recharts` renders analytics data |

The frontend has no direct database access, no blockchain access, and no secrets. Everything sensitive runs through the backend.

## Adding a New Page

1. Create `app/dashboard/<section>/page.tsx`
2. Add a link in `components/layout/sidebar.tsx`
3. If the page needs data, add fetch functions to `lib/api.ts`
4. Use `useQuery` in the page component to load and display data

## Key Files to Read First

1. `app/layout.tsx` — root layout, `QueryProvider` wrapping, dark-mode `className`
2. `app/dashboard/layout.tsx` — dashboard shell with sidebar + topbar
3. `app/dashboard/page.tsx` — overview page (example of a complete page with query + UI)
4. `lib/api.ts` — full API client with all endpoint functions
5. `lib/query-provider.tsx` — TanStack Query setup
6. `components/layout/sidebar.tsx` — nav structure and active-link logic
7. `app/globals.css` — Tailwind v4 `@theme` config and CSS variables
8. `components.json` — shadcn/ui config (needed before running `npx shadcn add`)
