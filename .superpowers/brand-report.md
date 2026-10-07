# Brand report

Branch: `brand`, worktree `gateway-wt-brand`. Date: 2026-10-07.
Commits: `b769fbb` (mark, wordmark, lockup, favicon, logo.tsx), `ad30953` (BRAND.md, brand.yaml, generate.mjs), `2255b32` (EmptyState wiring, OG template, asset log), plus this report.

## Checks

- `npx tsc --noEmit`: pass
- `npm run lint`: pass (no output)
- `npm test`: 2 files, 19 tests, pass

## Images generated

0.
Every Images API call returned `429 insufficient_quota` / `credit_balance_exhausted` for the organisation behind `openai.api_key` in `.secrets/providers.yaml`.
Tried `gpt-image-2` (the best usable model by the preference list) and `gpt-image-1-mini` (the cheapest) to rule out a per-model limit.
The models endpoint lists `gpt-image-1`, `gpt-image-1-mini`, `gpt-image-1.5`, `gpt-image-2`, `gpt-image-2-2026-04-21`, and the `gpt-image-2.5-flare` / `-sunburst` variants; the script prefers `gpt-image-2` and `--model` can override.

## What shipped

### The mark (`b769fbb`)

- Geometry kept from `logo.tsx` (28-grid, 7 padding, bars 14 x 2.5 with 1.75 gaps, third bar 8 wide) and now documented as a construction grid in BRAND.md.
- One real fix: in light mode the accent bar was `tide` light (`#0B7285`) on the ink square, 1.9:1 and nearly invisible. The rule is now "tide on ink": the opposite scheme's tide. `logo.tsx` uses `fill-[#3FB8CC] dark:fill-[#0B7285]`; both values are existing tokens.
- Wordmark: IBM Plex Sans SemiBold 3.005 outlines (from IBM's repository, OFL) converted to path data with opentype.js in a scratch folder, 15 px, -0.01em, caps centred on the mark, 10 px gap.
- Favicon redrawn on a 16-unit grid with 2 px bars and 1 px gaps (every edge on a whole pixel); the previous file was the mark scaled.
- Exports: `mark`, `wordmark`, `lockup` in light and dark under `frontend/public/brand/`, `checkout/public/brand/` (folder created; checkout had no `public/`), `landing/public/brand/`, and `app/icon.svg` in all three apps. `landing/app/favicon.ico` (the old Payminto favicon) removed. Six distinct hashes each present three times, verified with `shasum`.

### Identity and kit (`ad30953`)

- `docs/brand/BRAND.md`: positioning, mark, wordmark, exports, colour subset, type, imagery direction (one object, matte graphite aluminium, one teal element, one softbox, three-quarter camera, the never-appears list), icon style (Lucide, no custom line icons), 3D style, motion principles, the prompt kit (fixed preamble, fixed negative block, three templates, 13 subjects), generation instructions, asset log, and "Proposed token changes" (none needed; two notes for DESIGN.md).
- `docs/brand/brand.yaml`: machine copy of the kit, 13 assets with id, type, template, subject, output path and copies.
- `scripts/brand/generate.mjs`: Node 20, no dependencies, fetch only. YAML-subset parser (nested maps, lists of scalars or maps, `|` blocks, flow lists, quoted strings). Reads the key from `PAYMENTS_SECRETS_FILE` (default `../.secrets/providers.yaml`), never prints or writes it. `--models`, `--dry-run`, `--only`, `-n` (1 to 3), `--model`, `--budget` (default 30, counted against `manifest.json` across runs), `--promote <id>=<k>` (centre-crop to the delivered aspect, `sips` resize, `cwebp` at q88 with lossless alpha, copies to other apps, manifest entry with id, model, prompt, size, sha256 per file). Transparent background requested for empty states and icons. `candidates/` is git-ignored.

### Wiring (`2255b32`)

- `frontend/lib/brand/illustrations.ts`: `EMPTY_ILLUSTRATIONS` typed map (payments, payouts, wallets, webhooks, api-keys, customers) to the webp output paths from brand.yaml.
- `EmptyState` (`ui/states.tsx`, re-exported by `components/empty-state.tsx` with the map and type) takes an optional `illustration` id, rendered with `next/image` at 240 x 160 (180 below `sm`) in place of the icon box. No page was changed.
- `docs/brand/og-template.svg`: 1200 x 630, canvas, dark lockup top-left, headline and mono slots, vector rail at the right as the fallback layer, commented `<image>` layer for the render.

## Concerns

1. Deliverable 5 is unmet: no credits on the OpenAI organisation. Once topped up, the whole first pass is one command and roughly 13 images at medium quality (OG at high); curation is then by eye against BRAND.md section 7.
2. DESIGN.md section 13 lists "decorative illustrations in empty states" as a slop tell. The owner asked for empty-state illustrations; I reconciled it by keeping the prop optional, unused, and sized at 120 to 160 px, and recorded the tension under "Proposed token changes". The owner may want to delete the prop instead.
3. The DESIGN.md statement "the brand mark is the rail: a rounded square containing three short bars" is kept literally. I considered a horizontal three-segment mark closer to the product rail and rejected it: at 16 px three segments in a row are 3 px each and unreadable.
4. The `gpt-image-2.5-*` variants are newer than `gpt-image-2` but unknown to me in behaviour and price, so they are not in the preference list; `--model` can select one deliberately.
5. `checkout/app/page.tsx` still renders a violet `ShieldCheck` as its "brand mark" (pre-redesign code); out of scope here but it should use `checkout/public/brand/mark.svg`.
