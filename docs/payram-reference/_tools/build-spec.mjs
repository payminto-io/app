#!/usr/bin/env node
/**
 * Build the SCREENS/, INVENTORY.md, and API_CONTRACTS.md spec files from
 * captures/ + route-index.json. Run after crawl.mjs.
 *
 *   node payminto/docs/payram-reference/_tools/build-spec.mjs
 */
import { readFile, readdir, writeFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REF = path.resolve(__dirname, '..');
const CAP = path.join(REF, 'captures');
const SCREENS = path.join(REF, 'SCREENS');
const FLOWS = path.join(REF, 'FLOWS');

const index = JSON.parse(await readFile(path.join(__dirname, 'route-index.json'), 'utf8'));
const summary = JSON.parse(await readFile(path.join(CAP, '_summary.json'), 'utf8'));

await mkdir(SCREENS, { recursive: true });
await mkdir(FLOWS, { recursive: true });

// ── classify routes by group ─────────────────────────────────────────────────
const groupOf = (route) => {
  if (/^\/(login|signup|forgotPassword|setPassword|createPassword|logout)/.test(route)) return 'auth';
  if (/^\/(pay|addresses|success|payments|referral)/.test(route)) return 'public';
  if (/^\/dashboard|^\/project\/[^/]+\/dashboard/.test(route)) return 'dashboard';
  if (/^\/project\/[^/]+\/payments/.test(route)) return 'payments';
  if (/^\/project\/[^/]+\/customers/.test(route)) return 'customers';
  if (/^\/project\/[^/]+\/growth/.test(route)) return 'growth';
  if (/^\/project\/[^/]+\/accounts/.test(route)) return 'accounts';
  if (/^\/onramp/.test(route)) return 'onramp';
  if (/^\/sweepIn/.test(route)) return 'sweep';
  if (/^\/manageWallet/.test(route)) return 'wallets';
  if (/^\/withdraw/.test(route)) return 'withdraw';
  if (/^\/developers/.test(route)) return 'developers';
  if (/^\/settings/.test(route)) return 'settings';
  return 'misc';
};

// ── INVENTORY.md ─────────────────────────────────────────────────────────────
{
  const rows = index
    .map((r) => {
      const group = groupOf(r.route || '');
      const auth = group === 'auth' || group === 'public' ? 'no' : 'yes';
      const params = /\d|\/all\b/.test(r.route || '') && /\/(\d+|all)/.test(r.route) ? 'yes' : 'no';
      const screen = `SCREENS/${r.slug}.md`;
      const shot = `captures/${r.slug}/screenshot.png`;
      const dom = `captures/${r.slug}/dom.html`;
      const apis = (r.apiCalls || []).length;
      const interactions = r.interactions || 0;
      return `| \`${r.route}\` | ${group} | ${auth} | ${params} | ${apis} | ${interactions} | [shot](${shot}) · [dom](${dom}) · [spec](${screen}) |`;
    })
    .join('\n');

  const md = `# PayRam Route Inventory

Canonical list of every route exposed by PayRam's Next.js dashboard build,
extracted from \`_raw/manifests/app-path-routes-manifest.json\` and verified by
crawling each one against the live container at \`http://localhost:8880\`.

- **Total routes:** ${index.length}
- **Crawl summary:** ${summary.succeeded}/${summary.total} succeeded, ${summary.totalInteractions} interactions captured
- **Crawled at:** ${summary.ranAt}

\`auth\` = whether a logged-in JWT is required to render the page meaningfully.
Auth/public routes redirect logged-in users elsewhere; dashboard routes redirect
unauthenticated users to \`/login\`.

\`api\` = number of distinct \`/api/v1/*\` endpoints called on first paint
(useful gap-analysis signal — high-API routes are data-heavy).

\`int\` = number of safe interactions Playwright successfully clicked through
during the crawl (modals/tabs/dropdowns; capped at 10 per route).

| Route | Group | Auth | Params | API | Int | Artifacts |
|---|---|---|---|---|---|---|
${rows}

## Routes intentionally skipped

- \`/_not-found\`, \`/_global-error\` — Next.js internal error pages.
- \`/favicon.ico\` — asset, not a page.
- \`/logout\` — would destroy our crawler's auth session for every subsequent
  route. Captured by hand instead (it just clears localStorage and redirects
  to \`/login\`).
- \`/(auth)/page\` (\`/\`) — captured as the \`root\` slug; in practice it
  immediately redirects to \`/login\` or \`/project/all/dashboard\` depending
  on auth state.

## Routes that don't exist in the Payminto clone yet

Compared to \`payminto/frontend/src/app/\`:

- \`/manageWallet/gasFeeWallet\` (+ \`/add\`, \`/edit\`)
- \`/manageWallet/sweepContract\` (+ \`/deploy\`)
- \`/manageWallet/wallets/cold\`, \`/manageWallet/wallets/details/[id]\`
- \`/developers/documentation\`
- \`/onramp-payments\` and \`/project/all/accounts/revenue\` (Card Onramp)
- \`/sweepIn/{btc,eth,usdc,usdt}\` (per-asset sweep pages)
- \`/settings/{adminControl,api,checkoutAppearance,mobileApp,paymentsApp,paymentMethods,policyManagement,testTokens,updater,walletManagement,webhook}\`
- \`/project/[id]/payments/{invoice,missed-payments}\`

These are tracked in \`GAP_ANALYSIS.md\` (Phase 4).
`;
  await writeFile(path.join(REF, 'INVENTORY.md'), md);
  console.log('wrote INVENTORY.md');
}

// ── API_CONTRACTS.md ─────────────────────────────────────────────────────────
{
  const all = new Map();
  for (const slug of (await readdir(CAP, { withFileTypes: true })).filter((d) => d.isDirectory()).map((d) => d.name)) {
    try {
      const j = JSON.parse(await readFile(path.join(CAP, slug, 'network.json'), 'utf8'));
      const reqByUrl = new Map();
      for (const e of j) {
        if (!e.url.includes('/api/v1/')) continue;
        if (e.kind === 'request') reqByUrl.set(e.method + '|' + e.url, e);
        if (e.kind === 'response') {
          let p;
          try {
            p = new URL(e.url).pathname;
          } catch {
            continue;
          }
          const norm = p.replace(/\/(\d+|all)(?=\/|$)/g, '/{id}');
          const key = e.method + ' ' + norm;
          if (!all.has(key)) {
            all.set(key, {
              method: e.method,
              path: norm,
              statuses: new Set(),
              count: 0,
              respShape: null,
              reqShape: null,
              callers: new Set(),
              query: new Set(),
            });
          }
          const r = all.get(key);
          r.statuses.add(e.status);
          r.count++;
          r.callers.add(slug);
          try {
            const u = new URL(e.url);
            for (const k of u.searchParams.keys()) r.query.add(k);
          } catch {}
          if (!r.respShape && e.body && typeof e.body === 'object') {
            r.respShape = Array.isArray(e.body)
              ? `array<${typeof e.body[0]}>`
              : '{ ' + Object.keys(e.body).slice(0, 12).join(', ') + ' }';
          }
          const req = reqByUrl.get(e.method + '|' + e.url);
          if (req && req.postData && !r.reqShape) {
            try {
              const b = JSON.parse(req.postData);
              r.reqShape = Array.isArray(b)
                ? `array<${typeof b[0]}>`
                : '{ ' + Object.keys(b).slice(0, 12).join(', ') + ' }';
            } catch {
              r.reqShape = '<form/binary>';
            }
          }
        }
      }
    } catch {}
  }

  const grouped = {};
  for (const [k, v] of all) {
    const seg = v.path.split('/').slice(0, 4).join('/') || v.path;
    (grouped[seg] ||= []).push(v);
  }
  let body = '';
  for (const seg of Object.keys(grouped).sort()) {
    body += `\n### \`${seg}\`\n\n`;
    for (const r of grouped[seg].sort((a, b) => a.method.localeCompare(b.method) || a.path.localeCompare(b.path))) {
      body += `- **${r.method} ${r.path}** — status \`${[...r.statuses].join(',')}\` · seen ${r.count}× across ${r.callers.size} route(s)\n`;
      if (r.query.size) body += `  - query: ${[...r.query].map((q) => '`' + q + '`').join(', ')}\n`;
      if (r.reqShape) body += `  - request: \`${r.reqShape}\`\n`;
      if (r.respShape) body += `  - response: \`${r.respShape}\`\n`;
      if (r.callers.size <= 4) body += `  - callers: ${[...r.callers].map((c) => `[${c}](SCREENS/${c}.md)`).join(', ')}\n`;
    }
  }

  const md = `# PayRam API Contracts (observed)

Aggregated from every \`network.json\` produced by the Phase 2 crawl. This
captures only endpoints actually invoked by the dashboard during initial page
loads + 158 safe button interactions — write-mutating endpoints that require
form submissions (create payment link, deploy contract, invite user, etc.) are
documented in \`FLOWS/*.md\` instead.

- **Unique endpoints:** ${all.size}
- **Base URL:** \`http://localhost:8080/api/v1/\` (or whatever
  \`NEXT_PUBLIC_BACKEND_URL\` points at)
- **Auth:** \`Authorization: Bearer <jwt>\` from \`localStorage.payram_access_token\`.
- **Refresh:** \`localStorage.payram_refresh_token\` is exchanged via the same
  \`/signin\`-style flow when a 401 is received (not directly observed).

## Conventions

- Numeric IDs and the literal string \`all\` (the meta-project) are normalized
  to \`{id}\` in paths below.
- The pseudo-project \`/project/all/...\` aggregates every real project; PayRam
  ships with this default and most dashboard pages call the \`all\` variant.
- Many endpoints respond with an array directly at the root (no envelope) —
  these show as \`{ 0, 1, 2, ... }\` in the response shape column.
- A \`POST /api/v1/websocket-token/create\` is fired on every page load —
  PayRam opens a websocket per browser tab for live deposit/sweep updates.

## Endpoints by namespace
${body}
## Known 404 / wishful endpoints

The frontend code calls these but they 404 against the current build —
either dead code, feature-flagged, or stripped from the OSS build:

- \`GET /api/v1/health\`, \`/api/v1/version\`
- \`GET /api/v1/system/updater/{status,inspect,history}\` — settings/updater UI
  is wired up but the backend route is gone
- \`GET /api/v1/secrets-vaults\`, \`/api/v1/contract-address/blockchain/*/contract/sweep_approval\`
- \`GET /api/v1/config/smtp/\` — \`settings/emailConfig\` calls this
- \`GET /null/api/v1/...\` (literal \`null\` in the URL) — bug in the SPA when
  a project context is missing; harmless but worth fixing in our clone
`;
  await writeFile(path.join(REF, 'API_CONTRACTS.md'), md);
  console.log('wrote API_CONTRACTS.md', all.size, 'endpoints');
}

// ── SCREENS/<slug>.md (one per route) ────────────────────────────────────────
{
  for (const r of index) {
    const group = groupOf(r.route || '');
    const auth = group === 'auth' || group === 'public' ? '**no** (public)' : '**yes** (JWT in `localStorage.payram_access_token`)';
    const buttons = (r.buttons || []).filter(Boolean);
    const inputs = r.inputs || [];
    const apis = r.apiCalls || [];
    const interactionsDir = `../captures/${r.slug}/interactions/`;
    let body = `# ${r.title || r.slug}

- **Route:** \`${r.route}\`
- **Slug:** \`${r.slug}\`
- **Group:** ${group}
- **Auth required:** ${auth}
- **Screenshot:** [\`captures/${r.slug}/screenshot.png\`](../captures/${r.slug}/screenshot.png)
- **Raw DOM:** [\`captures/${r.slug}/dom.html\`](../captures/${r.slug}/dom.html)
- **Network log:** [\`captures/${r.slug}/network.json\`](../captures/${r.slug}/network.json)
- **Interactions captured:** ${r.interactions || 0} ([browse](${interactionsDir}))

## Snapshot

> ${(r.snippet || '').slice(0, 300).replace(/\n/g, ' ')}

## Network calls observed on first paint

${apis.length ? apis.map((a) => `- \`${a}\``).join('\n') : '_(none — static page)_'}

See [\`API_CONTRACTS.md\`](../API_CONTRACTS.md) for request/response shapes.

## Form inputs

${inputs.length ? inputs.map((i) => `- \`${i}\``).join('\n') : '_(none detected in initial DOM — may be lazy-rendered behind a button or modal; check interactions/)_'}

## Buttons (initial DOM)

${buttons.length ? buttons.map((b) => `- "${b}"`).join('\n') : '_(none in initial DOM — likely icon buttons without text content; check interactions/ for the screenshots)_'}

## Layout & components

The page extends the dashboard shell defined in [\`COMPONENT_LIBRARY.md\`](../COMPONENT_LIBRARY.md):
- **Top bar** — Testnet badge, project switcher, profile menu
- **Left sidebar** — collapsible nav with the groups Dashboard / Payments /
  Onramp / Growth / Funds Consolidation / Wallet management / Withdraw /
  Developers / Settings / Profile
- **Main pane** — page-specific content (see screenshot)

## Notes

${
  group === 'sweep'
    ? '- Part of the **Funds Consolidation (sweepIn)** flow. See [`FLOWS/sweep-cycle.md`](../FLOWS/sweep-cycle.md).'
    : group === 'wallets'
      ? '- Part of the **Wallet management** flow. See [`FLOWS/deposit-wallet-setup.md`](../FLOWS/deposit-wallet-setup.md).'
      : group === 'payments'
        ? '- Part of the **Payments** flow. See [`FLOWS/create-payment-link.md`](../FLOWS/create-payment-link.md).'
        : group === 'developers'
          ? '- Part of the **Developers** area. See [`FLOWS/webhook-config.md`](../FLOWS/webhook-config.md).'
          : group === 'auth'
            ? '- Part of the **Auth** flow. See [`FLOWS/auth.md`](../FLOWS/auth.md).'
            : '- General dashboard route.'
}
- This file was generated from the Phase 2 crawl. For complex routes
  (\`dashboard\`, \`payments-allPayments\`, \`manageWallet/*\`, \`settings/*\`)
  the captured screenshot is the source of truth — open it side-by-side with
  the Payminto implementation when closing gaps.
`;
    await writeFile(path.join(SCREENS, `${r.slug}.md`), body);
  }
  console.log('wrote', index.length, 'SCREENS/*.md');
}
