# PayRam Reference

Canonical, browsable reverse-engineered specification of the **PayRam** merchant
dashboard, used as the source of truth when building the **Payminto** clone.
Nothing in this folder is shipped — it exists so engineers can answer "what does
PayRam actually do on screen X?" without re-running the live container.

## Folder layout

```
payram-reference/
├── README.md              this file
├── INVENTORY.md           every PayRam route + status + links
├── DESIGN_TOKENS.md       colors, fonts, radii, shadows, spacing (extracted CSS)
├── COMPONENT_LIBRARY.md   reusable UI primitives + DOM snippets + classes
├── API_CONTRACTS.md       all unique API endpoints observed during the crawl
├── FLOWS/                 multi-screen user journeys (auth, payments, sweep, …)
├── SCREENS/               one file per route — layout, buttons, network, states
├── _raw/                  Phase 1 dump from the running payram Docker container
│   ├── html/{app,pages}/  pre-rendered HTML for every route
│   ├── manifests/*.json   Next.js route → chunk maps
│   ├── chunks/            compiled JS + CSS bundles (gitignored, ~21MB)
│   ├── public/            static assets (logos, icons)
│   ├── package.json       confirms PayRam stack (Next 16, React 19, …)
│   └── next.config.mjs
├── _tools/
│   ├── crawl.mjs          Playwright crawler used in Phase 2
│   ├── route-index.json   per-route extract (titles, buttons, inputs, API calls)
│   └── storageState.json  Playwright auth state (gitignored)
└── captures/              Phase 2 outputs — gitignored, ~47MB
    ├── _summary.json
    └── <slug>/
        ├── screenshot.png        full-page 1440x900
        ├── dom.html              post-hydration DOM
        ├── network.json          all XHR/fetch with JSON bodies inlined
        └── interactions/         per-button click screenshots + DOM diffs
```

## How to regenerate

Prereqs: PayRam Docker container running (`docker ps | grep payram`),
Playwright Chromium installed locally, and PayRam's Postgres seeded with at
least one member whose password you know (see "Login bootstrap" below).

```bash
# Phase 1 — extract artifacts (one-shot, idempotent)
mkdir -p payminto/docs/payram-reference/_raw/{html,manifests,public,chunks}
docker cp payram:/web/.next/server/app   payminto/docs/payram-reference/_raw/html/
docker cp payram:/web/.next/server/pages payminto/docs/payram-reference/_raw/html/pages
docker cp payram:/web/.next/app-path-routes-manifest.json payminto/docs/payram-reference/_raw/manifests/
docker cp payram:/web/.next/prerender-manifest.json       payminto/docs/payram-reference/_raw/manifests/
docker cp payram:/web/.next/routes-manifest.json          payminto/docs/payram-reference/_raw/manifests/
docker cp payram:/web/.next/build-manifest.json           payminto/docs/payram-reference/_raw/manifests/
docker cp payram:/web/.next/react-loadable-manifest.json  payminto/docs/payram-reference/_raw/manifests/
docker cp payram:/web/package.json                        payminto/docs/payram-reference/_raw/
docker cp payram:/web/next.config.mjs                     payminto/docs/payram-reference/_raw/
docker cp payram:/web/public                              payminto/docs/payram-reference/_raw/
docker cp payram:/web/.next/static                        payminto/docs/payram-reference/_raw/chunks

# Phase 2 — crawl the live container
node payminto/docs/payram-reference/_tools/crawl.mjs
# env knobs:  PAYRAM_BASE, PAYRAM_EMAIL, PAYRAM_PASSWORD, LIMIT
```

## Login bootstrap

PayRam's signin endpoint is `POST /api/v1/signin` and returns a JWT in the
response body which the SPA persists into `localStorage` under
`payram_access_token` / `payram_refresh_token` / `payram_user`.

If you don't know any account's password, reset it directly in Postgres. The
backend uses bcrypt; **psql will silently strip `$2b$10$` if you pass it inline
as a string literal** (it parses `$...$` as dollar-quote markers), so build the
hash with `chr(36)`:

```bash
# Generate a bcrypt hash for "Demo1234!"
node -e "const b=require('bcryptjs');console.log(b.hashSync('Demo1234!',10));"
# -> $2b$10$1dEgJebI9ZtYKAyUDtQi3exdscYbKy3gA1euB5yUgON/OJ4iJoRiK

# PayRam's actual database lives on the address in the container's POSTGRES_HOST
# env var, NOT the localhost postgres bundled in the same container image.
docker exec payram sh -c 'env | grep POSTGRES'

docker exec payram sh -c "PGPASSWORD=payram123 psql -h \$POSTGRES_HOST -U payram -d payram \
  -c \"UPDATE members SET password = chr(36)||'2b'||chr(36)||'10'||chr(36)||'1dEgJebI9ZtYKAyUDtQi3exdscYbKy3gA1euB5yUgON/OJ4iJoRiK', reset_password_required=false WHERE id=1;\""
```

## Port-conflict gotcha (Colima / Docker Desktop)

PayRam exposes its Go API on host port **8080**. On Colima, ssh-mux forwarding
will fail silently if **anything else** is already bound to 8080 (a stray
`python3 -m http.server`, an ssh tunnel, etc.). Symptom: the browser loads the
SPA from `:8880` fine, but every API call returns either CORS errors or HTML
from the squatter. Fix: `lsof -iTCP:8080 -sTCP:LISTEN` and kill the squatter
**before** starting the crawl, then verify `curl http://localhost:8080/`
returns `Welcome to Payram Core`.

## Activity-log middleware noise

PayRam's `activity_log` middleware writes to a `SQL_ASCII` Postgres database
and crashes on any non-UTF-8 byte (`SQLSTATE 22021`). This produces a constant
trickle of `500`s in `/var/log/payram-core.err.log` that look scary but are
post-handler — your actual API calls succeed before the middleware fires. You
can ignore them when reading logs.

## Credits & relationship to other docs

- `research/PAYRAM_DEMO_RESEARCH.md` (1,545 lines) — manual page-by-page
  walkthrough; this folder supersedes its layout descriptions but its
  business-logic notes (sweep cycle, refund flow, role hierarchy) remain
  authoritative.
- `research/PAYRAM_TECHNICAL_DOCUMENTATION.md` (886 lines) — API + webhook +
  MCP reference. `API_CONTRACTS.md` here adds the endpoints observed at
  runtime that the public docs don't list.
- `screenshots/SCREENSHOT_INVENTORY.md` — covers landing/marketing site, not
  the dashboard. Complementary, not overlapping.
- `payminto/docs/PRODUCT_SPEC.md` and friends — describe what **Payminto**
  will be. This folder describes what **PayRam** **is**. Use them together.
