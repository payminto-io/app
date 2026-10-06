# Payminto Landing — Design System

> **Source of truth.** This document is the contract every landing-page change must respect. Inspired by the Wise design language ([reference](./WISE_DESIGN_REFERENCE.md)) but adapted to Payminto's purple brand identity. If a design choice in the code conflicts with this doc, the doc wins — update the code, not the doc.

---

## 1. Atmosphere

Payminto is a **warm, light, confident** fintech landing in the Wise tradition: massive type, billboard-bold display weight, generous whitespace, ring-shadow cards, pill buttons that physically grow on hover. The single accent is **light purple `#a78bfa`** (not Wise lime green) — fresh, modern, distinctive in a fintech category dominated by corporate blues and crypto neon-greens.

**Light, never dark.** No section uses a black background except the very last footer (inversion accent).

---

## 2. Color palette

### Brand
| Token | Value | Use |
|---|---|---|
| `--brand` | `#a78bfa` (light purple) | Primary CTA background, accents, brand chips |
| `--brand-ink` | `#3b1d8a` (deep purple) | Text on `--brand` background, ring borders |
| `--brand-hover` | `#c4b5fd` (pastel purple) | Hover state on primary buttons |
| `--brand-soft` | `#ede9fe` (lavender mist) | Soft surface, badge background, mint-equivalent |

### Canvas
| Token | Value | Use |
|---|---|---|
| `--background` | `#fafaf7` (warm off-white) | Page canvas |
| `--surface` | `#ffffff` (pure white) | Cards |
| `--surface-soft` | `#f4f5f1` (warm gray) | Alternate section bg |
| `--surface-mint` | `#ede9fe` (lavender mist) | Tinted accent sections |

### Type
| Token | Value | Use |
|---|---|---|
| `--foreground` | `#0e0f0c` (near-black) | Primary text |
| `--foreground-soft` | `#454745` (warm dark) | Secondary text |
| `--foreground-muted` | `#868685` (gray) | Captions, eyebrows |

### Borders & semantic
| Token | Value | Use |
|---|---|---|
| `--border` | `rgba(14,15,12,0.12)` | All ring shadows / hairlines |
| `--positive` | `#054d28` | Success state |
| `--danger` | `#d03238` | Destructive / "old way" comparison |
| `--warning` | `#ffd11a` | Warning |

### Rules
- **One accent only.** Light purple. No competing cyan / amber / gradient mash. Amber/red appear only as semantic states inside cards.
- Light purple is for **buttons, brand chips, underlines, and small accents only** — never as a large-area background. (Wise rule: "Don't use the brand color as a large surface.")
- Text on `--brand` is always `--brand-ink` (deep purple), never white.

---

## 3. Typography

### Fonts
| Role | Font | Notes |
|---|---|---|
| Display | **Inter weight 900** | Wise Sans is proprietary; Inter Black is the closest free fallback. Used at 56–126px. |
| Body | **Inter weight 600** as the *default* | Confident, never light. Drop to 400 only for very long-form paragraphs. |
| Mono | **Geist Mono** | Code blocks, terminal, step numbers |
| Feature | `font-feature-settings: "calt"` applied **globally** | Mandatory — see Wise principle. |

### Scale (must match)
| Role | Size | Weight | Line-height | Letter-spacing |
|---|---|---|---|---|
| Display Mega | 126 px | 900 | **0.85** | -0.02em |
| Display Hero | 96 px | 900 | 0.85 | -0.02em |
| Section H2 | 72–84 px | 900 | 0.85 | -0.02em |
| Sub-heading | 40–48 px | 900 | 0.85 | -0.02em |
| Card title | 22–26 px | 700 | 1.23 | -0.01em |
| Body | 18 px | 600 | 1.44 | -0.108px |
| Caption | 14 px | 600 | 1.5 | -0.084px |
| Small | 12 px | 700 (uppercase) | 1.0 | 0.18em |

**The 0.85 line-height on display is non-negotiable — it is the visual signature.**

### Style helpers
- **`.brand-underline`** — a thick light-purple bar sitting *behind* the last phrase of every H2 (e.g., `you actually own.` / `move money.`). One signature device used consistently across all section headings.
- All-caps eyebrow labels at 11–12 px / weight 700 / `tracking-[0.18em]` above each H2.

---

## 4. Components

### Buttons — pill + scale
| Variant | Background | Text | Padding | Radius |
|---|---|---|---|---|
| `.btn-primary` | `#a78bfa` | `#3b1d8a` | 13 / 22 px | 9999 px |
| `.btn-secondary` | `rgba(59,29,138,0.08)` | `#0e0f0c` | 13 / 22 px | 9999 px |

**Mandatory motion:**
- Hover → `transform: scale(1.05)` (the button physically grows)
- Active → `transform: scale(0.96)`
- Focus → 2px outline `#3b1d8a` offset 3 px
- Transition: `180ms cubic-bezier(0.2, 0.8, 0.2, 1)`

Never use color-only hover. Always the scale.

### Cards — ring shadow only
| Class | Radius | Use |
|---|---|---|
| `.card-ring` | 30 px | Standard feature card |
| `.card-ring-lg` | 40 px | Hero image, dashboard, testimonial |

```css
box-shadow: 0 0 0 1px rgba(14,15,12,0.12);
background: #ffffff;
```

**No drop shadows. Ever.** Depth comes from the brand accent against the warm canvas, not blur.

### Borders
- All hairlines use `border-color: rgba(14,15,12,0.12)`
- A brand-tinted border (`1px solid #a78bfa`) is allowed only for the *single* card you want the eye to land on per section.

---

## 5. Layout & spacing

- **Base unit:** 8 px
- **Section padding:** `py-28` (112 px) on every major section
- **Container:** `max-w-6xl mx-auto px-6`
- **Asymmetry rule:** at least 2 sections per page must break the centered grid (e.g., headline left, image right at unequal widths). The current page does this via Card-to-Crypto and Mobile-App.
- **Bento break:** the first card in any 3-column grid spans 2 columns when it can carry the meaning (FeaturesGrid does this with the lavender card-1).

---

## 6. Motion

### Page-level
- **Lenis** for smooth scrolling (already wired in `app/smooth-scroll.tsx`).
- **GSAP + ScrollTrigger** for all reveal animations.
- `[data-reveal]` → fade + 60 px slide up, 1 s `power3.out`.
- `[data-reveal-stagger]` → children stagger 0.08 s.
- Every effect respects `prefers-reduced-motion`.

### Memorable moments
The page is allowed **two** screenshot-worthy moments:

1. **Hero — Draggable purple `$` orb (the "Tug me" moment).**
   A glowing light-purple coin sits in the hero. The user can grab it with `gsap.Draggable`, drag it anywhere on the viewport, release it, and watch it elastically snap back to its anchor with `ease: "elastic.out(1, 0.4)"`. A faint dashed line connects the orb to its anchor and updates as it's dragged. This is the *first* tactile moment the visitor encounters and immediately signals "this site is alive."

2. **Flow section — Coin-on-SVG-path scroll-scrub.**
   A pinned section with a hand-authored SVG curve from "Capture" → "Verify" → "Sweep". A second `$` coin travels along the path via `MotionPathPlugin`, scrubbed by scroll. The path draws via `strokeDashoffset`. Step cards fade in as the coin reaches each node.

**Both moments must:**
- Be gated behind `prefers-reduced-motion`
- Use only GSAP plugins shipped in core 3.14 (no Club / no new deps)
- Use the *same* `$` coin styling so the two moments feel like one product

### Forbidden clichés
- No `scale-1.02` button hovers (use 1.05)
- No emoji icons
- No gradient mash with 5 colors competing for attention
- No fake "stat ring" cards with 1 px dividers
- No drop shadows under cards
- No light-mode `text-gray-400` body text — bump to weight 600 muted instead

---

## 7. Imagery

### What we generate
- **Photoreal product mockups** via `google/nano-banana-pro` (dashboard, mobile, hosted checkout)
- **Photoreal portrait** for testimonial via nano-banana-pro
- **Abstract illustration** (hero orb, flow diagram bg) via `bytedance/seedream-4`

### What we never generate
- AI-generated UI dashboards from Seedream (too uncanny — use nano-banana-pro)
- AI-generated faces from Seedream (use nano-banana-pro)
- "AI app icon pack" SDXL feature icons — use `lucide-react` or hand-authored SVG instead

### Pipeline
`payminto/landing/scripts/generate-assets.mjs` — `npm run generate`
Each spec lists: name, file, kind (`seedream` | `nano` | `sdxl-icon`), input prompt.
Add new assets to the array, run with the asset name as a CLI arg, or `--force` to regenerate everything.

---

## 8. Page section order

| # | Section | Notes |
|---|---|---|
| 1 | Navbar | Pill CTAs right-aligned, hover bg `rgba(167,139,250,0.18)` |
| 2 | Hero | Display Mega headline + draggable `$` orb (memorable moment 1) |
| 3 | Trust strip | Press marks in muted small caps |
| 4 | Card-to-Crypto | Live badge + checkout-screen mockup |
| 5 | Setup | Fake terminal with type-line animation |
| 6 | Features grid | First card spans 2 cols (bento break) |
| 7 | Flow diagram | Pinned scroll-scrub coin (memorable moment 2) |
| 8 | AI Agents | Old-way ↔ Payminto comparison + MCP code snippet |
| 9 | Dashboard showcase | Real product screenshot via nano-banana-pro |
| 10 | Mobile app | Phone mockup + 3 features |
| 11 | Supported chains | 6 chain chips |
| 12 | Testimonial | Pull quote in lavender card with real avatar |
| 13 | FAQ | Accordion with rotating `+` button |
| 14 | CTA banner | Final billboard headline + dual pill CTAs |
| 15 | Footer | **Only** dark surface, 4-column links |

---

## 9. Don'ts (the AI-slop checklist)

If any of these creep back in, the page has slipped:

- ❌ Pure `#000000` background
- ❌ Hex colors hard-coded in components instead of going through tokens
- ❌ Display headings without `line-height: 0.85`
- ❌ Body text under weight 600
- ❌ Drop-shadowed cards
- ❌ Color-only button hovers (must include the scale)
- ❌ Five accent colors competing in a single section
- ❌ Emoji feature icons
- ❌ A flow diagram that is a generated PNG
- ❌ Text overlays on AI dashboard mockups that try to be "the product"
- ❌ Phantom footer link grids with 16 dead `href="#"`s
- ❌ **"Pulse dot + status pill"** combos anywhere on the page — the `● v0.1 — Now in private beta` / `● Live` pattern is the single most overused Vercel-template tell. Never ship a pulsing colored dot next to a status label. If a section genuinely needs a live state, use the word alone (e.g., "Live onramp") without any dot, or nothing at all.
