#!/usr/bin/env node
/**
 * Payminto landing page asset generator.
 *
 * Generates all hero images, mockups, and feature icons via Replicate
 * and saves them to public/generated/.
 *
 * Usage:
 *   node scripts/generate-assets.mjs            # only generates missing files
 *   node scripts/generate-assets.mjs --force    # regenerates everything
 *   node scripts/generate-assets.mjs hero-orb   # generates a single asset by name
 *
 * Env:
 *   REPLICATE_API_TOKEN - falls back to the user-provided token below.
 */

import { writeFile, mkdir, access } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const OUT_DIR = join(__dirname, "..", "public", "generated");

const TOKEN = process.env.REPLICATE_API_TOKEN;
if (!TOKEN) {
  console.error("Set REPLICATE_API_TOKEN to run this script.");
  process.exit(1);
}

const SEEDREAM_MODEL = "bytedance/seedream-4";
const NANO_MODEL = "google/nano-banana-pro";
const SDXL_ICONS_VERSION =
  "5839ce85291601c6af252443a642a1cbd12eea8c83e41f27946b9212ff845dbf";

// ─── Asset specs ──────────────────────────────────────────────────────────────

/** @type {Array<{name: string, file: string, kind: 'seedream'|'sdxl-icon', input: object}>} */
const ASSETS = [
  // Hero / mockups / illustrations (Seedream-4)
  {
    name: "hero-orb",
    file: "hero-orb.png",
    kind: "seedream",
    input: {
      prompt:
        "Abstract 3D rendered glowing violet and cyan orb with concentric light rings, payment network visualization, particles flowing inward, dark near-black background, soft volumetric light, cinematic, no text, premium fintech aesthetic",
      size: "2K",
      aspect_ratio: "16:9",
    },
  },
  {
    name: "card-to-crypto",
    file: "card-to-crypto.png",
    kind: "seedream",
    input: {
      prompt:
        "Isometric illustration of a credit card morphing into a glowing cryptocurrency token, violet and cyan gradient, abstract geometric flow lines, dark background, premium 3D render, no text, no logos",
      size: "2K",
      aspect_ratio: "3:2",
    },
  },
  {
    name: "flow-diagram",
    file: "flow-diagram.png",
    kind: "seedream",
    input: {
      prompt:
        "Isometric diagram of a payment orchestration system: customer to checkout to processor to blockchain to cold storage wallet, violet and cyan accent nodes connected by glowing lines, dark background, technical illustration, no text labels",
      size: "2K",
      aspect_ratio: "16:9",
    },
  },
  {
    name: "dashboard-mockup",
    file: "dashboard-mockup.png",
    kind: "seedream",
    input: {
      prompt:
        "Modern fintech analytics dashboard UI, dark theme with violet accent, charts showing payment volume, recent transactions list, KPI cards, glassmorphism panels, photorealistic 4K render, no real text just placeholder bars and shapes",
      size: "4K",
      aspect_ratio: "3:2",
    },
  },
  {
    name: "mobile-app",
    file: "mobile-app.png",
    kind: "seedream",
    input: {
      prompt:
        "Smartphone mockup floating on dark background, dark fintech app UI on screen with violet accents, transaction list and balance card visible, soft cyan glow behind phone, photorealistic 3D render, no text",
      size: "2K",
      aspect_ratio: "3:4",
    },
  },
  {
    name: "og-image",
    file: "og-image.png",
    kind: "seedream",
    input: {
      prompt:
        "Wide social card: dark gradient background, abstract violet and cyan light rays converging at center, premium fintech aesthetic, cinematic, no text",
      size: "2K",
      aspect_ratio: "16:9",
    },
  },
  {
    name: "pattern-grid",
    file: "pattern-grid.png",
    kind: "seedream",
    input: {
      prompt:
        "Subtle geometric grid pattern on near-black background, faint violet glow at intersections, seamless tileable, minimal, designer texture",
      size: "2K",
      aspect_ratio: "1:1",
    },
  },
  {
    name: "logo-mark",
    file: "logo-mark.png",
    kind: "seedream",
    input: {
      prompt:
        "Minimal abstract geometric letter P logo mark, violet to cyan gradient, flat vector style, on dark background, premium fintech brand, single color shape, designer logo, no text",
      size: "1K",
      aspect_ratio: "1:1",
    },
  },

  // High-fidelity upgrades (nano-banana-pro)
  {
    name: "dashboard-mockup",
    file: "dashboard-mockup.png",
    kind: "nano",
    input: {
      prompt:
        "Ultra realistic screenshot of a modern fintech analytics dashboard, dark mode (#1a1a1f background), violet (#7c5cff) accent, sidebar navigation on left labeled Dashboard / Payments / Wallets / Sweeps / Settings, main area shows: large area chart of payment volume over 30 days with a violet gradient fill, KPI cards showing Total Volume $1.2M / Active Wallets 142 / Success Rate 99.8%, a transactions table below with rows showing crypto symbols BTC ETH USDC and amounts, glassmorphism panels, sharp typography (Inter font), 4K screenshot quality, no logos, no watermarks, centered composition",
      aspect_ratio: "3:2",
      output_format: "png",
    },
  },
  {
    name: "mobile-app",
    file: "mobile-app.png",
    kind: "nano",
    input: {
      prompt:
        "Realistic iPhone 15 Pro mockup floating on transparent dark background, screen shows a modern fintech mobile app in dark mode with violet (#7c5cff) accents, displaying a balance card at top showing $48,392.10 in big white text, below it a list of recent transactions (USDC, BTC, ETH) with green/red amount changes, a violet primary action button at bottom labeled Send, soft cyan glow under the phone, photorealistic 3D render, no real text other than placeholder, premium fintech aesthetic, centered",
      aspect_ratio: "3:4",
      output_format: "png",
    },
  },
  {
    name: "checkout-screen",
    file: "checkout-screen.png",
    kind: "nano",
    input: {
      prompt:
        "Photorealistic screenshot of a modern crypto payment checkout page, dark mode UI with violet (#7c5cff) accent, centered card showing: merchant logo placeholder at top, headline 'Pay $49.99', three payment method tabs (Card / Crypto / Apple Pay) with Crypto selected, a QR code in the middle, wallet address below the QR in monospace, an amber 'Awaiting payment' status pill, countdown timer 14:32, dark glassmorphism card on a soft violet gradient background, premium clean design, no real text outside the elements described, 4K quality",
      aspect_ratio: "3:2",
      output_format: "png",
    },
  },
  {
    name: "testimonial-avatar",
    file: "testimonial-avatar.png",
    kind: "nano",
    input: {
      prompt:
        "Professional studio portrait of a 34 year old asian woman with shoulder length black hair, confident warm smile, wearing a charcoal blazer over a black tee, soft rim lighting on a deep dark gray background, head and shoulders crop, photorealistic, shallow depth of field, premium magazine quality headshot, no text",
      aspect_ratio: "1:1",
      output_format: "png",
    },
  },
  {
    name: "infographic-custody",
    file: "infographic-custody.png",
    kind: "nano",
    input: {
      prompt:
        "Editorial infographic illustration in a clean modern flat-vector style on a warm off-white background (#fafaf7). Side-by-side comparison with a thin vertical divider. Left side labeled 'TRADITIONAL PROCESSOR': a customer figure on the left with a credit card icon, an arrow flowing into a large middleman building/vault icon in the center labeled 'Custodian', then a smaller arrow flowing out to a merchant figure on the right - emphasize the funds get TRAPPED at the custodian with a soft red highlight. Right side labeled 'PAYMINTO': customer figure with credit card flowing through a small clean arrow DIRECTLY to a merchant wallet icon, no middleman, with a soft light purple #a78bfa highlight on the direct path. Use Inter typography, deep purple #3b1d8a labels, light purple #a78bfa accents, no random extra elements, premium fintech editorial illustration like a Wise or Stripe explainer, no logos, no real text other than the two section labels, ultra clean composition",
      aspect_ratio: "21:9",
      output_format: "png",
    },
  },
  {
    name: "infographic-architecture",
    file: "infographic-architecture.png",
    kind: "nano",
    input: {
      prompt:
        "Editorial infographic of a self-hosted crypto payment gateway architecture, clean modern flat-vector illustration on warm off-white background (#fafaf7). Center: a single server rack icon labeled 'Your VPS' in a soft purple rounded card. Around it, four smaller connected component cards labeled 'API', 'Dashboard', 'MCP Server', 'Block Monitor', each connected to the central VPS with thin dashed lines. Below the VPS, an arrow flows down to a wallet icon labeled 'Your Cold Wallet'. Above the VPS, three small chain icons (Bitcoin, Ethereum, Base) flowing down into the VPS. Use deep purple #3b1d8a for labels, light purple #a78bfa for accents and connector lines, lavender mist #ede9fe for card backgrounds with subtle 1px ring borders. Inter typography, premium fintech editorial style like Stripe documentation. No real text other than the component labels, 16:9 aspect ratio, designer composition",
      aspect_ratio: "16:9",
      output_format: "png",
    },
  },

  // Feature icons (SDXL app icons)
  {
    name: "icon-custody",
    file: "icon-custody.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, vault with violet glow, dark background, 3D render, premium",
    },
  },
  {
    name: "icon-payments",
    file: "icon-payments.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, stacked coins with violet glow, dark background, 3D render",
    },
  },
  {
    name: "icon-auth",
    file: "icon-auth.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, glowing key on a shield, violet and cyan, dark background",
    },
  },
  {
    name: "icon-funds",
    file: "icon-funds.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, abstract flowing river of light into a vault, violet, dark background",
    },
  },
  {
    name: "icon-compliance",
    file: "icon-compliance.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, document with checkmark, violet glow, dark background",
    },
  },
  {
    name: "icon-automation",
    file: "icon-automation.png",
    kind: "sdxl-icon",
    input: {
      prompt:
        "minimalist app icon, stylized robot head with circuit lines, violet and cyan, dark background",
    },
  },
];

// ─── Replicate helpers ────────────────────────────────────────────────────────

async function createPrediction(spec) {
  const body =
    spec.kind === "sdxl-icon"
      ? { version: SDXL_ICONS_VERSION, input: spec.input }
      : { input: spec.input };

  const modelPath =
    spec.kind === "nano" ? NANO_MODEL : SEEDREAM_MODEL;
  const url =
    spec.kind === "sdxl-icon"
      ? "https://api.replicate.com/v1/predictions"
      : `https://api.replicate.com/v1/models/${modelPath}/predictions`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      "Content-Type": "application/json",
      Prefer: "wait=60",
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Replicate POST failed (${res.status}): ${text}`);
  }
  return res.json();
}

async function pollPrediction(prediction) {
  let current = prediction;
  while (
    current.status !== "succeeded" &&
    current.status !== "failed" &&
    current.status !== "canceled"
  ) {
    await new Promise((r) => setTimeout(r, 2000));
    const res = await fetch(current.urls.get, {
      headers: { Authorization: `Bearer ${TOKEN}` },
    });
    if (!res.ok) {
      throw new Error(`Replicate poll failed: ${res.status}`);
    }
    current = await res.json();
  }
  return current;
}

function extractOutputUrl(prediction) {
  const output = prediction.output;
  if (!output) return null;
  if (Array.isArray(output)) return output[0];
  if (typeof output === "string") return output;
  return null;
}

async function downloadTo(url, dest) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Download failed ${res.status}: ${url}`);
  const buf = Buffer.from(await res.arrayBuffer());
  await writeFile(dest, buf);
  return buf.length;
}

async function fileExists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

// ─── Main ─────────────────────────────────────────────────────────────────────

async function generate(spec) {
  const dest = join(OUT_DIR, spec.file);
  const label = `[${spec.name.padEnd(18)}]`;

  process.stdout.write(`${label} creating prediction...\n`);
  const prediction = await createPrediction(spec);

  let final = prediction;
  if (
    prediction.status !== "succeeded" &&
    prediction.status !== "failed" &&
    prediction.status !== "canceled"
  ) {
    process.stdout.write(`${label} polling...\n`);
    final = await pollPrediction(prediction);
  }

  if (final.status !== "succeeded") {
    throw new Error(
      `${spec.name} ended with status=${final.status} error=${
        final.error || "unknown"
      }`
    );
  }

  const outUrl = extractOutputUrl(final);
  if (!outUrl) throw new Error(`${spec.name} no output url in prediction`);

  const bytes = await downloadTo(outUrl, dest);
  process.stdout.write(`${label} ✓ ${(bytes / 1024).toFixed(0)} KB → ${spec.file}\n`);
}

async function main() {
  await mkdir(OUT_DIR, { recursive: true });

  const args = process.argv.slice(2);
  const force = args.includes("--force");
  const filterNames = args.filter((a) => !a.startsWith("--"));

  const targets = filterNames.length
    ? ASSETS.filter((a) => filterNames.includes(a.name))
    : ASSETS;

  if (filterNames.length && targets.length === 0) {
    console.error(`No assets matched: ${filterNames.join(", ")}`);
    console.error(`Available: ${ASSETS.map((a) => a.name).join(", ")}`);
    process.exit(1);
  }

  console.log(
    `Generating ${targets.length} assets to ${OUT_DIR}${force ? " (force)" : ""}`
  );

  let done = 0;
  let skipped = 0;
  let failed = 0;

  for (const spec of targets) {
    const dest = join(OUT_DIR, spec.file);
    if (!force && (await fileExists(dest))) {
      console.log(`[${spec.name.padEnd(18)}] ↷ exists, skipping`);
      skipped++;
      continue;
    }
    try {
      await generate(spec);
      done++;
    } catch (err) {
      console.error(`[${spec.name.padEnd(18)}] ✗ ${err.message}`);
      failed++;
    }
  }

  console.log(`\nDone. generated=${done} skipped=${skipped} failed=${failed}`);
  if (failed > 0) process.exit(1);
}

main().catch((err) => {
  console.error("Fatal:", err);
  process.exit(1);
});
