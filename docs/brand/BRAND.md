# Payminto brand identity

Status: locked 2026-10-07 on branch `brand`.
This is the single source for every brand asset: the mark, the wordmark, icons, illustrations, 3D renders, animation and marketing images.
Nothing is generated outside the prompt kit in section 10, and `brand.yaml` next to this file is the machine-readable copy of that kit.
Product tokens, type scale and component rules live in `../design/DESIGN.md` and are not repeated here; this document only says which of them the brand uses and how.

## 1. Positioning

The gateway where a card and a stablecoin are the same money: one ledger, one rail, settled where you choose, on servers you own.

Three words carry it: Received, Final, Settled.
Every brand asset is a physical rendering of that rail.

## 2. The mark

### Concept

A rounded square holding three bars.
Two full bars are the ledger: two sides of an entry.
The third bar is short and teal: the segment of the rail that is still filling.
The shape is the product thesis drawn once: money moves along the same track whatever rail it arrived on.

It is not a hamburger menu and not a text icon; the third bar's length and colour are what separate it from both, so they are never changed.

### Construction grid

Drawn on a 28-unit square, 7 modules of 4.

| Element | x | y | w | h | Radius | Fill |
| --- | --- | --- | --- | --- | --- | --- |
| Square | 0 | 0 | 28 | 28 | 7 | square |
| Bar 1 | 7 | 8.5 | 14 | 2.5 | 1.25 | bar |
| Bar 2 | 7 | 12.75 | 14 | 2.5 | 1.25 | bar |
| Bar 3 | 7 | 17 | 8 | 2.5 | 1.25 | accent |

Inner padding is 7 on every side (one quarter of the square).
Bars are 2.5 tall with a 1.75 gap; the bar group is 11 tall and sits 0.25 above true centre, which reads as optically centred.
Bar 3 is 8 wide: four sevenths of a full bar, never half.
The square's radius is one quarter of its side, so the ratio holds at every size.

### Colour

| Scheme | Square | Bars | Accent |
| --- | --- | --- | --- |
| On light surfaces | `ink` light `#15181D` | `ink-inverse` light `#FFFFFF` | `#3FB8CC` |
| On dark surfaces | `ink` dark `#EDEFF2` | `ink-inverse` dark `#0F1115` | `#0B7285` |

The accent is the opposite scheme's `tide`.
Tide light on the ink square is 1.9:1 and vanishes; tide dark on it is 7.2:1.
So the rule is "tide on ink", not "tide of the current scheme", and `components/logo.tsx` encodes it as `fill-[#3FB8CC] dark:fill-[#0B7285]`.
Both values are existing tokens; no new colour was introduced.

The mark never appears in tide, never outlined, never on a photograph, never with a shadow or gradient.
A single-colour version (square in `ink`, bars knocked out, accent bar also knocked out) is allowed only where one ink is physically required, such as a laser etch or a stamp.

### Clear space and minimum size

Clear space on every side is 7 units at 28 (one quarter of the side, the same as the inner padding).
Nothing sits inside that space, including the wordmark, which is why the lockup gap is 10 and not 7: the gap is the clear space plus the optical weight of the "P".

| Use | Minimum |
| --- | --- |
| Mark on screen | 16 px, and at 16 to 24 use the favicon construction below |
| Mark in print | 6 mm |
| Lockup on screen | 20 px tall |
| Lockup in print | 8 mm tall |

### Favicon

`app/icon.svg` is not the mark scaled; at 16 px the 2.5-unit bars become 1.4 px and smear.
It is redrawn on a 16-unit grid with every edge on a whole pixel: square radius 4, bars 8 wide and 2 tall at y = 4, 7, 10 with 1-unit gaps, bar 3 is 5 wide.
At 32 px it is exactly 2x and stays crisp.
Above 32 px use the mark.

### Do and don't

Do: place the mark on `canvas`, `surface` or their dark equivalents; keep the 28-grid ratios; use the light-surface build on light and the dark-surface build on dark.

Don't: rotate, skew, add bars, lengthen bar 3, change which bar is teal, put the mark in a circle, add a stroke, put it on tide or amber, animate the square, or scale it below 16 px.

## 3. Wordmark

"Payminto" in IBM Plex Sans SemiBold (600), sentence case, letter spacing -0.01em, no ligatures.
The exported SVG is the glyph outlines of IBM Plex Sans SemiBold 3.005, so it renders identically without the font installed.
Set at 15 px beside the 28 px mark: cap height 10.47 px, baseline at y = 19.24 so the caps are centred on the mark.
The wordmark is only ever `ink` (light `#15181D`, dark `#EDEFF2`); never tide, never on a tint.
It never takes a tagline, a trademark glyph or a different weight.

## 4. Exported files

| File | Content |
| --- | --- |
| `brand/mark.svg`, `brand/mark-dark.svg` | 28 x 28 mark, light-surface and dark-surface builds |
| `brand/wordmark.svg`, `brand/wordmark-dark.svg` | 64.61 x 15.5 outlines |
| `brand/lockup.svg`, `brand/lockup-dark.svg` | 102.61 x 28, mark + 10 gap + wordmark |
| `app/icon.svg` | 16 x 16 favicon construction |

They live under `frontend/public/`, `checkout/public/` and `landing/public/` (and `app/icon.svg` in each of the three apps).
The three copies are byte-identical and were written by one script in one pass; when one changes, all three change, and `components/logo.tsx` changes with them.
Check with `shasum */public/brand/*.svg | awk '{print $1}' | sort | uniq -c`: six distinct hashes, each three times.

## 5. Colour on brand surfaces

Brand surfaces (landing, OG image, social, docs covers, decks) use a subset of the DESIGN.md tokens and nothing else:

| Role | Token | Light | Dark |
| --- | --- | --- | --- |
| Page | `canvas` | `#F4F5F7` | `#0F1115` |
| Panel | `surface` | `#FFFFFF` | `#171A20` |
| Rule | `line` | `#E1E4E9` | `#262B33` |
| Text | `ink` | `#15181D` | `#EDEFF2` |
| Secondary text | `ink-soft` | `#5B6472` | `#A7AFBA` |
| The one accent | `tide` | `#0B7285` | `#3FB8CC` |
| Settled, final | `ok` | `#1A7F4B` | `#3DBB78` |

Rules: tide is a line, a bar or a word, never an area larger than a button.
`ok` appears only on a filled rail segment.
Amber belongs to the test environment and never appears on a brand surface.
No colour is ever a background wash; the page is `canvas`, panels are `surface`, and the rest is ink.
Status colours other than `ok` do not appear in marketing.

## 6. Typography for brand

IBM Plex Sans for everything readable, IBM Plex Mono for anything a machine would read: an amount, an address, an API key, a command.
Sentence case everywhere, no uppercase tracking, no italic.

| Role | Size | Line | Weight | Tracking |
| --- | --- | --- | --- | --- |
| Display | 56 to 72 | 1.05 | 600 | -0.025em |
| Headline | 28 to 40 | 1.15 | 600 | -0.02em |
| Lead | 18 | 1.5 | 400 | -0.01em |
| Body | 16 | 1.5 | 400 | -0.01em |
| Mono | 14 to 16 | 1.5 | 400 | 0 |

The audience does not read much: a headline is one sentence, a block is at most two, and a number is shown with its unit in mono.
Amounts follow DESIGN.md section 8 even in a deck: `1,250.00 USDC`, never `$`.

## 7. Imagery direction

Every image is a product render of one small physical object, as if a machinist had cut the idea out of metal and photographed it on a bench.
This is the only imagery family; there are no photographs, no scenes and no characters.

**Material.** Matte anodised graphite aluminium in `ink` (near-black), lightly bead-blasted, with one element in matte `tide` (`#3FB8CC`).
No chrome, no glass, no plastic gloss, no wood, no fabric.

**Lighting.** One large softbox upper-left, a weak fill from the right, a soft contact shadow under the object and nothing else.
No rim light, no glow, no coloured light, no caustics.

**Camera.** Three-quarter view from slightly above, around 30 degrees, long lens (50 to 85 mm equivalent) so verticals stay nearly parallel.
Object centred with generous margin: it fills about 55 percent of a square frame and about 40 percent of a wide frame.

**Composition.** One object.
If the idea needs a second part (a slot, a track, a base plate) it is part of the same assembly, touching.
Nothing floats, nothing explodes, nothing is in a grid.

**Background.** Transparent for anything placed in the product; `canvas` for anything standalone.
Never a gradient, never a vignette, never an environment.

**Palette constraints.** Ink, the one teal, and the grey of the shadow.
No second hue.
The teal element is small: a bar, a rim, a tip, a tile, never a whole object.

**Never appears.** Faces, hands, hands holding phones, people of any kind, coins with currency symbols, currency symbols at all, glowing blockchain cubes, chains of cubes, globes, rockets, padlocks, shields, gradient washes, purple-blue neon, glassmorphism, particles, bokeh, lens flare, stock-photo anything, text, letters, numbers, logos (including ours; the mark is vector and is placed afterwards).

## 8. Icon style

Interface icons are Lucide, as DESIGN.md section 9 says, and the brand does not draw its own line icons.
Where a marketing surface needs a line icon it uses Lucide at 24 px, stroke 1.75, round caps and joins, on a 24-unit grid with 2-unit padding, corner radius 2, in `ink` or `ink-soft`, never tide.
The 3D icon set in section 9 is for landing and docs covers only and is never mixed with line icons in the same row.

## 9. 3D style

One material family and one light setup, exactly as section 7, with the camera pulled in so the object fills about 55 percent of a square frame.
Edges carry a small, even radius like a machined part; nothing is sharp and nothing is bulbous.
Six rail icons exist: card, bank, stablecoin, chain finality, settlement, custody.
Each is one assembly with one teal element, rendered on transparent, delivered at 256 px for display at 64 to 128 px.
New 3D icons are added by adding an asset to `brand.yaml`, not by prompting freehand.

## 10. Motion principles for brand animation

1. Motion is the rail filling: segments fill left to right over 400 ms `ease-in-out`, one after another, never together.
2. Nothing is ambient: no floating, breathing, orbiting or looping, except the product's three-dot confirming pulse.
3. An object may arrive by a 4 px rise and a fade over 160 ms; it never bounces, spins or scales from zero.
4. Camera does not move; the object does, and only along one axis.
5. One animation per frame; two things never move at once.
6. The mark is static; only its third bar may fill, once, as a loading indicator.
7. `prefers-reduced-motion` makes every brand animation a cut.

## 11. Prompt kit

Every image prompt is the fixed preamble, then one asset template with its subject filled in, then the fixed negative block.
Nothing is prompted without all three.
The machine copy is `brand.yaml`; the text below is the human copy and they are kept identical.

### Style preamble (fixed)

> Product render for a payments infrastructure brand. One small physical object, machined from matte anodised graphite aluminium in near-black (#15181D), lightly bead-blasted, with exactly one element in matte teal (#3FB8CC). Studio lighting: one large softbox upper-left, weak fill from the right, a soft contact shadow directly beneath, nothing else. Camera: three-quarter view from about 30 degrees above, long lens, verticals nearly parallel, object centred with generous empty margin. Surfaces plain and precise, edges with a small even radius like a machined part. Quiet, premium, restrained.

### Negative block (fixed)

> No text, no letters, no numbers, no logos, no currency symbols, no people, no faces, no hands, no phones, no coins with symbols, no cubes, no chains of cubes, no globes, no rockets, no padlocks, no shields, no glow, no neon, no gradients, no glass, no transparency effects, no particles, no bokeh, no lens flare, no environment, no second colour.

### Templates

| Template id | Use | Size | Background | Body |
| --- | --- | --- | --- | --- |
| `empty_state` | Dashboard empty states | 1536 x 1024, delivered 720 x 480 | transparent | "The object: {subject}. It is small in a wide frame, filling about 40 percent of the width, resting on nothing. The scene says '{meaning}' without words." |
| `rail_icon` | 3D icons for the rails | 1024 x 1024, delivered 256 x 256 | transparent | "The object: {subject}. It fills about 55 percent of the square frame, seen from the front three-quarter. It is a single assembly." |
| `og_background` | Open Graph and social backgrounds | 1536 x 1024, delivered 1200 x 630 | `#F4F5F7` | "The object: {subject}. It sits in the right third of a wide frame on a plain light grey (#F4F5F7) bench that fades to nothing; the left two thirds are empty light grey for text placed later." |

### Subjects

| Asset id | Template | Subject | Meaning |
| --- | --- | --- | --- |
| `empty-payments` | `empty_state` | a flat rounded rectangular tile lying on a short straight track made of three rounded segments, the first segment teal | nothing has arrived yet |
| `empty-payouts` | `empty_state` | a shallow rounded slot in a flat plate with a thin rounded bar emerging halfway out of it, the tip of the bar teal | nothing has gone out yet |
| `empty-wallets` | `empty_state` | an open shallow rounded tray seen from above at an angle, empty, with a small teal tab on its front edge | nothing is held yet |
| `empty-webhooks` | `empty_state` | a thin rounded rod bent into a smooth hook, its end finished with a small teal disc | nothing has been sent yet |
| `empty-api-keys` | `empty_state` | a flat key: a rounded bar with two square notches on one edge and a small ring at the other end, one notch teal | no key exists yet |
| `empty-customers` | `empty_state` | three short rounded pegs standing in a row on a flat base, the middle peg teal | no one has paid yet |
| `icon-card` | `rail_icon` | a flat rounded rectangular slab slightly thicker than a card, with one short raised teal bar near its lower edge | - |
| `icon-bank` | `rail_icon` | a flat rectangular base carrying three short upright rounded pillars and a flat lid, the middle pillar teal | - |
| `icon-stablecoin` | `rail_icon` | a thick flat disc with a completely plain face and a thin teal inset ring near its edge | - |
| `icon-chain-finality` | `rail_icon` | three rounded bars in a row, joined end to end on a flat base, the last bar teal | - |
| `icon-settlement` | `rail_icon` | a flat rounded bar sliding into a matching slot in a flat base plate, the rim of the slot teal | - |
| `icon-custody` | `rail_icon` | a rounded square housing with a smaller rounded square tile recessed inside it, the tile teal | - |
| `og-background` | `og_background` | a straight track of three rounded segments on a thin base, the first two segments teal and the third graphite | - |

The OG template is `docs/brand/og-template.svg`: the background render under the dark lockup at the left, with one headline slot in Plex Sans SemiBold 56 px and one line of mono.
Fill the slot, render at 1200 x 630, nothing else moves.

## 12. Generating assets

```bash
node scripts/brand/generate.mjs --dry-run               # print every prompt
node scripts/brand/generate.mjs --only icon-card         # 1 candidate
node scripts/brand/generate.mjs --only icon-card -n 3    # up to 3 candidates
node scripts/brand/generate.mjs --promote icon-card=2    # pick candidate 2, resize, write png + webp
```

The script reads the key from `PAYMENTS_SECRETS_FILE` (default `../.secrets/providers.yaml`, outside the repository), never prints it and never writes it.
It refuses to generate more than 3 candidates per asset and more than 30 images per budget (`--budget`), counting what `manifest.json` already records.
Candidates land in `frontend/public/brand/generated/candidates/`, which is not committed; promoted files land in `frontend/public/brand/generated/` with `manifest.json` carrying id, model, prompt, size and sha256.

Curation is by eye and the bar is section 7: one object, one material, one teal element, no text, no second hue, no scene.
Anything that looks like a stock 3D icon pack is rejected even if it is on palette.

## 13. Asset log

Pass 1, 2026-10-07: 0 images generated, 0 kept, 0 rejected.
Every call to the Images API returned `429 insufficient_quota` (`credit_balance_exhausted`) for the organisation behind the key in `.secrets/providers.yaml`, on `gpt-image-2` and on `gpt-image-1-mini` alike.
The pipeline is complete and dry-runs cleanly; once credits exist, the first pass is one candidate per asset (13 images), then at most two more per rejected asset within the budget of 30:

```bash
node scripts/brand/generate.mjs
node scripts/brand/generate.mjs --only <id> -n 2      # only for rejects
node scripts/brand/generate.mjs --promote <id>=<k>    # per kept candidate
```

Until then `frontend/public/brand/generated/` does not exist, `EmptyState`'s `illustration` prop is wired but unused by any page, and the OG template renders its vector rail fallback.
Record each pass here as: asset id, candidates seen, kept candidate, reason for each rejection in the words of section 7.

## 14. Proposed token changes

None of the DESIGN.md tokens needed to change.
Two notes for the next DESIGN.md pass, written here rather than edited in:

1. Section 1 could record the "tide on ink" rule for the mark's accent (the opposite scheme's tide), since the component now encodes it.
2. Section 13 lists "decorative illustrations in empty states".
   The renders in this pass are not decorative scenes; they are the brand's one object family at 120 px, and `EmptyState` takes them only through an optional `illustration` prop that no page sets yet.
   If the owner wants that line to stay absolute, the prop can be removed without touching a page.
