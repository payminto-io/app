# Payminto Landing Page — CLAUDE.md

## What This Is

The Payminto landing page is the public-facing marketing site for the product. It is a separate Next.js 16 application, entirely independent from the merchant dashboard (`payminto/frontend/`). The landing page is a single-page app that explains what Payminto is, how to self-host it, what features it has, and which cryptocurrencies are supported. It has no backend calls — it is pure static content rendered on the server. The brand identity is a dark theme (`#0a0a0b`) with a violet primary (`#7c5cff`), cyan glow accent (`#22d3ee`), and amber highlight (`#fbbf24`). Imagery is generated via Replicate (`npm run generate`) into `public/generated/`.

Do not conflate this with the merchant dashboard in `payminto/frontend/`. If you are building payment management UI, you are in the wrong folder.

## Tech Stack

| Component | Version / Package |
|-----------|------------------|
| Framework | Next.js 16.2.2 (App Router) |
| Language | TypeScript 5 |
| UI Framework | React 19.2.4 |
| Styling | Tailwind CSS v4 |
| Build tool | Turbopack (Next.js built-in) |
| Linting | ESLint 9 + `eslint-config-next` |

No component library (no shadcn/ui), no data fetching, no API client. Minimal dependencies by design — the marketing site must be fast.

## Directory Layout

```
landing/
├── app/
│   ├── layout.tsx         # Root layout — dark <html> class, font, metadata
│   ├── page.tsx           # Single-page marketing site (all sections inline)
│   ├── globals.css        # Tailwind v4 base + @theme inline
│   └── favicon.ico
├── public/                # Static assets (logo, og-image, etc.)
├── next.config.ts
├── tsconfig.json
└── package.json
```

## Page Sections (in order in `app/page.tsx`)

The entire page lives in `app/page.tsx` as a `"use client"` component with all sections inline:

1. **Navbar** — logo, product/agents/pricing/docs links, GitHub + Self-host CTA
2. **Hero** — tagline "The payment processor you actually own.", hero-orb image, 3-stat strip
3. **Trust strip** — press marks row
4. **Card-to-Crypto** — live badge + generated `card-to-crypto.png` illustration
5. **Setup** — one-line install command + animated fake terminal output
6. **Features grid** — 6 cards using generated `icon-*.png` SDXL icons (no emoji)
7. **Flow diagram** — generated `flow-diagram.png` + 3-step caption
8. **AI Agents** — old-way vs Payminto comparison + MCP code snippet
9. **Dashboard showcase** — generated `dashboard-mockup.png` + 4 captions
10. **Mobile app** — generated `mobile-app.png` + 3 features
11. **Supported chains** — BTC/ETH/USDT/USDC/TRX/BASE chips
12. **Testimonial** — pull quote card
13. **FAQ** — accordion
14. **CTA banner** — full-bleed gradient "Stop renting your payment stack"
15. **Footer** — 4-column layout

## Common Commands

```bash
# Development server
npm run dev           # starts on http://localhost:3001 (or 3000 if frontend is not running)

# Production build
npm run build

# Start production server
npm start

# Lint
npm run lint
```

## Code Conventions

### Everything in one file by design

The entire site is intentionally in `app/page.tsx`. This makes it easy to see, edit, and deploy without hunting across many files. Only extract components when a section becomes large enough to warrant its own file (>150 lines or reused in multiple places).

### Tailwind v4

Tailwind v4 does NOT use `tailwind.config.js`. All custom tokens live in `app/globals.css` under `@theme inline {}`. Do not create a `tailwind.config.ts`. The green brand colour `#01e46f` should be defined as a CSS variable in `globals.css`:

```css
@theme inline {
  --color-brand: #7c5cff;       /* electric violet */
  --color-brand-2: #22d3ee;     /* cyan glow */
  --color-warm: #fbbf24;        /* amber */
  --color-bg: #0a0a0b;
  --color-surface: #111114;
}
```

Then use it in Tailwind as `text-brand`, `bg-brand`, `border-brand`. For gradients, use the `.brand-gradient` utility defined in `globals.css`.

### Generated assets

Hero/mockup/icon imagery is generated via Replicate. Run `npm run generate` (set `REPLICATE_API_TOKEN` first — see `.env.local.example`). Output lands in `public/generated/`. Add new asset specs to `scripts/generate-assets.mjs`.

### Dark theme

Background is `bg-gray-950` (very dark gray, almost black). Text is white / `text-gray-100`. Accent is `#01e46f` (bright green). Do not use light-mode styles — this site is permanently dark.

### No API calls

This site makes zero API calls to the backend. If you need to show demo data, hardcode it as constants in the file. If a feature requires a live backend call, it does not belong on this page — it belongs in `payminto/frontend/`.

### Planned future routes

- `/blog` — MDX-powered blog (not yet implemented)
- `/demo` — interactive live demo (not yet implemented)
- `/docs` — redirects to docs site or inline MDX docs (not yet implemented)

When implementing these, add them as additional `app/` route segments. Do not add them inline to `page.tsx`.

## Integration Points

This site is intentionally isolated:

| System | Direction | Details |
|--------|-----------|---------|
| None | — | No backend calls, no database, no auth |

The only external links are to GitHub, documentation, and the Discord community. The "Deploy Now" / "Get Started" CTA should link to the self-hosting docs or the GitHub release page.

## Key Files to Read First

1. `app/page.tsx` — the entire site (start here)
2. `app/layout.tsx` — dark mode setup, font loading, metadata (title, og:image)
3. `app/globals.css` — Tailwind v4 `@theme` tokens including brand colour
