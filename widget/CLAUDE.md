# Payminto Widget — CLAUDE.md

## What This Is

The Payminto widget is a lightweight, embeddable JavaScript snippet that merchants drop into any website to accept cryptocurrency payments with a single `<script>` tag. It has zero dependencies — no React, no framework, no external CSS. It is written in plain TypeScript, bundled via esbuild into a single minified IIFE (~2.4kb), and loaded from `https://payminto.com/widget/payminto-widget.js`. When loaded, it reads configuration from `data-*` attributes on its own `<script>` tag, injects a "Pay with Payminto" button into the host page, and opens an overlay modal when clicked.

## Tech Stack

| Component | Version / Package |
|-----------|------------------|
| Language | TypeScript 5.7 |
| Bundler | esbuild ^0.24.0 |
| Output format | IIFE (Immediately Invoked Function Expression) |
| Output file | `dist/payminto-widget.js` (~2.4kb minified) |
| Module type | ESM source → IIFE output |
| Runtime | Any modern browser (no Node.js) |

No runtime dependencies. No npm packages installed at runtime. `esbuild` is the only devDependency besides TypeScript.

## Directory Layout

```
widget/
├── src/
│   └── index.ts          # Entire widget implementation — single file
├── dist/
│   └── payminto-widget.js  # Compiled + minified output (commit this for CDN deploy)
├── tsconfig.json
└── package.json
```

## Common Commands

```bash
# Build minified production bundle
npm run build
# equivalent: esbuild src/index.ts --bundle --minify --outfile=dist/payminto-widget.js --format=iife

# Development: rebuild on every file change (no minify, faster)
npm run dev
# equivalent: esbuild src/index.ts --bundle --outfile=dist/payminto-widget.js --format=iife --watch

# Check bundle size
wc -c dist/payminto-widget.js

# Type-check without bundling
npx tsc --noEmit
```

## How It Works (Runtime Flow)

1. Merchant adds the `<script>` tag to their page:
   ```html
   <script
     src="https://payminto.com/widget/payminto-widget.js"
     data-payminto-url="https://my-payminto.example.com"
     data-api-key="pk_live_..."
     data-amounts="10,25,50,100"
     data-theme="dark"
     data-customer-email="optional@email.com"
   ></script>
   ```

2. When the script loads, the IIFE executes immediately.

3. It locates its own `<script>` tag by searching `document.currentScript` (during load) or by matching `src` attribute.

4. It reads `data-*` attributes from that tag:
   - `data-payminto-url` — base URL of the merchant's self-hosted Payminto instance
   - `data-api-key` — public API key for creating payments
   - `data-amounts` — comma-separated list of preset amounts (USD)
   - `data-theme` — `"dark"` (default) or `"light"`
   - `data-customer-email` — optional pre-fill for customer email

5. It injects a `<button>Pay with Payminto</button>` into `document.body` (or adjacent to the `<script>` tag).

6. On click, it renders an overlay modal with amount selector buttons.

7. When an amount is selected, it opens a new tab to the payment URL at `data-payminto-url`.

> **Note:** The current implementation opens an external URL in a new tab. Direct API calls (creating invoices via `POST /api/v1/payments`) are planned but not yet implemented. The widget today is a lightweight redirect launcher.

## Code Conventions

### One file only

All widget logic lives in `src/index.ts`. Do not split into multiple files — the output must be a single bundled file, and esbuild handles the bundling, but keeping the source in one file avoids accidental complexity.

### DOM API only — no abstractions

Use raw DOM APIs: `document.createElement`, `element.appendChild`, `element.addEventListener`. No virtual DOM, no event delegation libraries.

```ts
// CORRECT
const button = document.createElement("button")
button.textContent = "Pay with Payminto"
button.style.background = "#01e46f"
button.style.color = "#000"
button.addEventListener("click", openModal)
document.body.appendChild(button)

// WRONG — no framework allowed
render(<Button onClick={openModal}>Pay with Payminto</Button>, document.body)
```

### Inline styles only — no external CSS

The widget must not load any external CSS files (no `<link>` tags, no `import './style.css'`). All styles are applied via `element.style.*` or injected as a `<style>` tag with a unique prefix to avoid conflicts with the host page's CSS.

Use a unique prefix for any injected CSS class names (e.g., `pmw-` prefix) to avoid collisions:
```ts
const style = document.createElement("style")
style.textContent = `.pmw-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.8); }`
document.head.appendChild(style)
```

### Keep bundle small

The target is under 5kb minified. Avoid:
- Importing any npm packages (even tiny utilities)
- Using `async/await` unnecessarily (adds regenerator runtime)
- Unnecessary abstractions

### TypeScript

Strict mode is on. The IIFE format means there is no `exports` — declare all code as module-internal. Use `window` and `document` directly.

## Embed Usage for Merchants

```html
<!-- Minimal embed -->
<script
  src="https://payminto.com/widget/payminto-widget.js"
  data-payminto-url="https://pay.yourstore.com"
  data-api-key="pk_live_abc123"
></script>

<!-- Full configuration -->
<script
  src="https://payminto.com/widget/payminto-widget.js"
  data-payminto-url="https://pay.yourstore.com"
  data-api-key="pk_live_abc123"
  data-amounts="5,10,25,50,100"
  data-theme="dark"
  data-customer-email="customer@example.com"
></script>
```

For self-hosted merchants, replace `https://payminto.com/widget/payminto-widget.js` with their own CDN or Nginx-served path (Nginx config in `payminto/docker/nginx/nginx.conf` serves `/widget/` from the built file).

## Integration Points

| System | Direction | Details |
|--------|-----------|---------|
| Merchant's Payminto instance | outbound | Opens payment URL in new tab (future: POST /api/v1/payments) |
| Host website | n/a | Injected into any webpage via `<script>` tag |
| Nginx (docker) | serves widget file | `/widget/payminto-widget.js` served as static asset |

## Key Files to Read First

1. `src/index.ts` — the entire widget implementation
2. `package.json` — esbuild commands and bundle output configuration
3. `tsconfig.json` — TypeScript config (lib includes DOM, strict mode)
