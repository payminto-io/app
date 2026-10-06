#!/usr/bin/env node
/**
 * PayRam live-container crawler.
 *
 * Drives the running `payram` Docker container (http://localhost:8880) with
 * Playwright, capturing for every route:
 *   - captures/<slug>/screenshot.png   full-page screenshot (1440x900 viewport)
 *   - captures/<slug>/dom.html         post-hydration DOM
 *   - captures/<slug>/network.json     request/response log (JSON bodies inlined)
 *   - captures/<slug>/interactions/*   per-button click screenshots + DOM diffs
 *
 * Route list is read from
 *   payminto/docs/payram-reference/_raw/manifests/app-path-routes-manifest.json
 * Route groups like (auth)/(dashboard)/(public)/(referral) are stripped. Routes
 * with [param] placeholders are filled with sensible defaults (id=1, projectId=1,
 * roleId=1, promoterId=1, token=demo) so they're reachable without real data.
 *
 * Auth: guest login is open. We log in once at /login with
 * email=demo@payminto.dev, persist storageState to storageState.json in this
 * tools dir, and reuse it for every subsequent page navigation.
 *
 * Usage:
 *   cd payramclone
 *   node payminto/docs/payram-reference/_tools/crawl.mjs
 *   # optional: LIMIT=5 to crawl only the first 5 routes while iterating
 */

import { chromium } from 'playwright';
import { readFile, writeFile, mkdir } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const REF_DIR = path.resolve(__dirname, '..');
const MANIFEST = path.join(REF_DIR, '_raw/manifests/app-path-routes-manifest.json');
const CAPTURES = path.join(REF_DIR, 'captures');
const STORAGE = path.join(__dirname, 'storageState.json');

const BASE = process.env.PAYRAM_BASE || 'http://localhost:8880';
const LOGIN_EMAIL = process.env.PAYRAM_EMAIL || 'recsagarjethi@gmail.com';
const LOGIN_PASSWORD = process.env.PAYRAM_PASSWORD || 'Demo1234!';
const LIMIT = process.env.LIMIT ? parseInt(process.env.LIMIT, 10) : null;
const NAV_TIMEOUT = 15000;
const NETIDLE_TIMEOUT = 5000;
const INTERACTIONS_PER_PAGE = 10;

// Param substitutions for dynamic routes.
// projectId defaults to 'all' — PayRam's pseudo-project that aggregates every
// real project; every project-scoped route is mounted there by default on load.
const PARAM_DEFAULTS = {
  id: '1',
  projectId: 'all',
  roleId: '1',
  promoterId: '1',
  token: 'demo',
};

// Destructive / dangerous labels to skip when clicking through buttons.
const UNSAFE_LABEL_RE =
  /\b(logout|log\s*out|sign\s*out|delete|remove|destroy|revoke|ban|disable|terminate|submit|save|create|confirm|approve|withdraw|send|pay|deploy|upload)\b/i;

function routeSlug(urlPath) {
  if (urlPath === '/') return 'root';
  return urlPath.replace(/^\//, '').replace(/\[|\]/g, '').replace(/\//g, '-') || 'root';
}

function resolveParams(urlPath) {
  return urlPath.replace(/\[([^\]]+)\]/g, (_, name) => PARAM_DEFAULTS[name] || '1');
}

/**
 * Turn the app-path-routes-manifest map into a deduped ordered list of real
 * URL paths, stripping route-group parents and resolving [param] placeholders.
 */
async function loadRoutes() {
  const raw = JSON.parse(await readFile(MANIFEST, 'utf8'));
  const seen = new Set();
  const routes = [];
  for (const [, urlPath] of Object.entries(raw)) {
    if (!urlPath.startsWith('/')) continue;
    // Skip internal next routes & assets.
    if (urlPath.startsWith('/_') || urlPath === '/favicon.ico') continue;
    // Skip /logout — it destroys our auth session for every subsequent route.
    if (urlPath === '/logout' || urlPath.endsWith('/logout')) continue;
    const resolved = resolveParams(urlPath);
    if (seen.has(resolved)) continue;
    seen.add(resolved);
    routes.push({ original: urlPath, path: resolved, slug: routeSlug(urlPath) });
  }
  return routes;
}

async function ensureDir(p) {
  await mkdir(p, { recursive: true });
}

/**
 * Attempt login once and persist storage state. Idempotent across runs.
 * Guest login is open; any email works.
 */
async function login(browser) {
  if (existsSync(STORAGE)) {
    try {
      const s = JSON.parse(await readFile(STORAGE, 'utf8'));
      if (s && (s.cookies?.length || s.origins?.length)) return;
    } catch {}
  }
  const ctx = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    ignoreHTTPSErrors: true,
  });
  const page = await ctx.newPage();
  console.log('[login] navigating to /login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: NAV_TIMEOUT });
  // Best-effort: fill first email-like input and submit.
  try {
    const emailInput = page.locator('input[type="email"], input[name*="email" i], input[placeholder*="email" i]').first();
    await emailInput.waitFor({ timeout: 5000 });
    await emailInput.fill(LOGIN_EMAIL);
    const pwd = page.locator('input[type="password"]').first();
    if (await pwd.count()) {
      await pwd.fill(LOGIN_PASSWORD);
    }
    const submit = page
      .locator('button[type="submit"], button:has-text("Sign In"), button:has-text("Login"), button:has-text("Continue")')
      .first();
    await submit.click({ timeout: 3000 }).catch(() => {});
    // Wait long enough for the signin POST + redirect to /project/all/dashboard
    // and for the client to persist JWT into localStorage.
    await page.waitForURL(/\/(dashboard|project\/.+\/dashboard)/, { timeout: 8000 }).catch(() => {});
    await page.waitForLoadState('networkidle', { timeout: NETIDLE_TIMEOUT }).catch(() => {});
    await page.waitForTimeout(1000);
  } catch (e) {
    console.warn('[login] interactive login failed, continuing with whatever state exists:', e.message);
  }
  await ctx.storageState({ path: STORAGE });
  await ctx.close();
  console.log('[login] storage state saved');
}

/**
 * Attach request/response listeners that accumulate into `entries`.
 * We try to inline JSON response bodies (<256KB) for later analysis.
 */
function attachNetworkRecorder(page, entries) {
  page.on('request', (req) => {
    try {
      entries.push({
        kind: 'request',
        t: Date.now(),
        method: req.method(),
        url: req.url(),
        resourceType: req.resourceType(),
        postData: req.postData() || null,
        headers: req.headers(),
      });
    } catch {}
  });
  page.on('response', async (res) => {
    try {
      const req = res.request();
      const ct = (res.headers()['content-type'] || '').toLowerCase();
      const entry = {
        kind: 'response',
        t: Date.now(),
        method: req.method(),
        url: res.url(),
        status: res.status(),
        contentType: ct,
        body: null,
      };
      if (ct.includes('application/json')) {
        try {
          const buf = await res.body();
          if (buf.length < 256 * 1024) entry.body = JSON.parse(buf.toString('utf8'));
          else entry.body = `<${buf.length} bytes omitted>`;
        } catch {}
      }
      entries.push(entry);
    } catch {}
  });
}

/**
 * Safely click up to N buttons on the page and capture the resulting state.
 * Skips destructive labels. Presses Escape between clicks to close modals.
 */
async function captureInteractions(page, routeDir, routePath) {
  const interactionsDir = path.join(routeDir, 'interactions');
  await ensureDir(interactionsDir);

  const handles = await page
    .locator('button, [role="button"], [role="tab"], summary')
    .elementHandles();

  let captured = 0;
  for (let i = 0; i < handles.length && captured < INTERACTIONS_PER_PAGE; i++) {
    const h = handles[i];
    try {
      const visible = await h.isVisible().catch(() => false);
      const disabled = await h.isDisabled().catch(() => false);
      if (!visible || disabled) continue;
      const label = (
        (await h.getAttribute('aria-label').catch(() => null)) ||
        (await h.textContent().catch(() => null)) ||
        ''
      )
        .trim()
        .replace(/\s+/g, ' ')
        .slice(0, 60);
      if (!label) continue;
      if (UNSAFE_LABEL_RE.test(label)) continue;
      // Avoid type=submit (would mutate).
      const type = await h.getAttribute('type').catch(() => null);
      if (type === 'submit') continue;

      await h.click({ timeout: 1500, trial: false }).catch(() => {});
      await page.waitForTimeout(500);

      const safeLabel = label.replace(/[^a-z0-9]+/gi, '_').toLowerCase().slice(0, 40);
      const idx = String(captured).padStart(2, '0');
      await page
        .screenshot({ path: path.join(interactionsDir, `${idx}-${safeLabel}.png`), fullPage: true })
        .catch(() => {});
      const html = await page.content().catch(() => '');
      await writeFile(path.join(interactionsDir, `${idx}-${safeLabel}.html`), html).catch(() => {});
      captured++;

      // Close any modal/dropdown that may have opened.
      await page.keyboard.press('Escape').catch(() => {});
      await page.waitForTimeout(200);
      // If we somehow navigated away, bail out — caller handles recovery.
      if (!page.url().includes(routePath)) break;
    } catch (e) {
      // ignore single-button failures
    }
  }
  return captured;
}

async function crawlRoute(browser, route) {
  const routeDir = path.join(CAPTURES, route.slug);
  await ensureDir(routeDir);

  const ctx = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    ignoreHTTPSErrors: true,
    storageState: STORAGE,
  });
  const page = await ctx.newPage();
  const network = [];
  attachNetworkRecorder(page, network);

  const result = { route: route.path, slug: route.slug, ok: false, interactions: 0, error: null };
  try {
    await page.goto(`${BASE}${route.path}`, { waitUntil: 'domcontentloaded', timeout: NAV_TIMEOUT });
    await page.waitForLoadState('networkidle', { timeout: NETIDLE_TIMEOUT }).catch(() => {});
    await page.waitForTimeout(500);

    await page
      .screenshot({ path: path.join(routeDir, 'screenshot.png'), fullPage: true })
      .catch((e) => console.warn(`  screenshot failed: ${e.message}`));
    const html = await page.content();
    await writeFile(path.join(routeDir, 'dom.html'), html);

    result.interactions = await captureInteractions(page, routeDir, route.path);
    result.ok = true;
  } catch (e) {
    result.error = e.message;
    console.warn(`  ! ${route.path} failed: ${e.message}`);
  } finally {
    await writeFile(path.join(routeDir, 'network.json'), JSON.stringify(network, null, 2)).catch(
      () => {}
    );
    await ctx.close().catch(() => {});
  }
  return result;
}

async function main() {
  await ensureDir(CAPTURES);
  const routes = await loadRoutes();
  const toCrawl = LIMIT ? routes.slice(0, LIMIT) : routes;
  console.log(`Loaded ${routes.length} routes; crawling ${toCrawl.length}`);

  const browser = await chromium.launch({ headless: true });
  try {
    await login(browser);

    const results = [];
    let idx = 0;
    for (const route of toCrawl) {
      idx++;
      console.log(`[${idx}/${toCrawl.length}] ${route.path}`);
      const r = await crawlRoute(browser, route);
      results.push(r);
    }

    const succeeded = results.filter((r) => r.ok).length;
    const failed = results.length - succeeded;
    const totalInteractions = results.reduce((a, r) => a + r.interactions, 0);
    const summary = {
      base: BASE,
      total: results.length,
      succeeded,
      failed,
      totalInteractions,
      results,
      ranAt: new Date().toISOString(),
    };
    await writeFile(path.join(CAPTURES, '_summary.json'), JSON.stringify(summary, null, 2));
    console.log('');
    console.log(`=== Crawl summary ===`);
    console.log(`Routes:       ${results.length}`);
    console.log(`Succeeded:    ${succeeded}`);
    console.log(`Failed:       ${failed}`);
    console.log(`Interactions: ${totalInteractions}`);
    console.log(`Summary:      ${path.join(CAPTURES, '_summary.json')}`);
  } finally {
    await browser.close();
  }
}

main().catch((e) => {
  console.error('fatal:', e);
  process.exit(1);
});
