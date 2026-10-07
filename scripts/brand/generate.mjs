#!/usr/bin/env node
// Generates brand assets from docs/brand/brand.yaml through the OpenAI Images API.
// Rules and curation bar: docs/brand/BRAND.md sections 11 and 12. Node 20, no dependencies.
//
//   node scripts/brand/generate.mjs --dry-run              print every prompt, call nothing
//   node scripts/brand/generate.mjs [--only <id>] [-n 1..3] generate candidates
//   node scripts/brand/generate.mjs --promote <id>=<k>      resize candidate k into the delivered files
//   options: --model <id>  --budget <n>  --models (list usable image models)
//
// The API key comes from PAYMENTS_SECRETS_FILE (openai.api_key). It is never printed or written.

import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, writeFileSync, copyFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const YAML_PATH = resolve(ROOT, "docs/brand/brand.yaml");
const SECRETS_PATH = process.env.PAYMENTS_SECRETS_FILE || resolve(ROOT, "../.secrets/providers.yaml");
const OUT_DIR = resolve(ROOT, "frontend/public/brand/generated");
const CANDIDATES_DIR = resolve(OUT_DIR, "candidates");
const MANIFEST_PATH = resolve(OUT_DIR, "manifest.json");

// ---------------------------------------------------------------------------
// Minimal YAML subset parser (maps, lists of scalars or maps, "|" blocks, flow lists).
// ---------------------------------------------------------------------------

function parseYaml(text) {
  const lines = text.split(/\r?\n/);
  const rows = [];
  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i];
    if (!raw.trim() || /^\s*#/.test(raw)) {
      rows.push({ indent: -1, text: "", blank: true, line: i });
      continue;
    }
    rows.push({ indent: raw.search(/\S/), text: raw.trim(), blank: false, line: i, raw });
  }

  function scalar(s) {
    s = s.trim();
    if (s === "" || s === "~" || s === "null") return null;
    if (/^".*"$/.test(s)) return JSON.parse(s);
    if (/^'.*'$/.test(s)) return s.slice(1, -1).replace(/''/g, "'");
    if (/^\[.*\]$/.test(s)) {
      const inner = s.slice(1, -1).trim();
      return inner ? inner.split(",").map((x) => scalar(x)) : [];
    }
    if (s === "true") return true;
    if (s === "false") return false;
    if (/^-?\d+(\.\d+)?$/.test(s)) return Number(s);
    return s.replace(/\s+#.*$/, "");
  }

  function block(start, indent) {
    const out = [];
    let i = start;
    while (i < rows.length && (rows[i].blank || rows[i].indent >= indent)) {
      out.push(rows[i].blank ? "" : rows[i].raw.slice(indent));
      i++;
    }
    while (out.length && out[out.length - 1] === "") out.pop();
    return { value: out.join("\n") + "\n", next: i };
  }

  function nextNonBlank(i) {
    while (i < rows.length && rows[i].blank) i++;
    return i;
  }

  function parseValueAfterKey(rest, i, keyIndent) {
    if (rest === "|" || rest === "|-") {
      const j = nextNonBlank(i + 1);
      const { value, next } = block(j, rows[j].indent);
      return { value: rest === "|-" ? value.replace(/\n$/, "") : value, next };
    }
    if (rest !== "") return { value: scalar(rest), next: i + 1 };
    const j = nextNonBlank(i + 1);
    if (j >= rows.length || rows[j].indent <= keyIndent) return { value: null, next: j };
    return rows[j].text.startsWith("- ") || rows[j].text === "-" ? parseList(j, rows[j].indent) : parseMap(j, rows[j].indent);
  }

  function parseMap(i, indent) {
    const obj = {};
    while (i < rows.length) {
      i = nextNonBlank(i);
      if (i >= rows.length || rows[i].indent < indent) break;
      if (rows[i].indent > indent) throw new Error(`brand.yaml:${rows[i].line + 1}: unexpected indent`);
      const m = rows[i].text.match(/^([^:#]+):(?:\s+(.*))?$/);
      if (!m) throw new Error(`brand.yaml:${rows[i].line + 1}: expected "key: value"`);
      const r = parseValueAfterKey((m[2] ?? "").trim(), i, indent);
      obj[m[1].trim()] = r.value;
      i = r.next;
    }
    return { value: obj, next: i };
  }

  function parseList(i, indent) {
    const arr = [];
    while (i < rows.length) {
      i = nextNonBlank(i);
      if (i >= rows.length || rows[i].indent !== indent || !(rows[i].text.startsWith("- ") || rows[i].text === "-")) break;
      const body = rows[i].text.slice(1).trim();
      const km = body.match(/^([^:#]+):(?:\s+(.*))?$/);
      if (km && !/^["'\[]/.test(body)) {
        // list item that is a map: rewrite the first key as a row at indent+2 and parse a map
        const itemIndent = indent + 2;
        rows[i] = { ...rows[i], indent: itemIndent, text: body, raw: " ".repeat(itemIndent) + body };
        const r = parseMap(i, itemIndent);
        arr.push(r.value);
        i = r.next;
      } else {
        arr.push(scalar(body));
        i++;
      }
    }
    return { value: arr, next: i };
  }

  return parseMap(nextNonBlank(0), rows[nextNonBlank(0)].indent).value;
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

function args() {
  const a = process.argv.slice(2);
  const opt = { only: [], n: 1, dryRun: false, promote: [], model: null, budget: null, listModels: false };
  for (let i = 0; i < a.length; i++) {
    const k = a[i];
    if (k === "--dry-run") opt.dryRun = true;
    else if (k === "--models") opt.listModels = true;
    else if (k === "--only") opt.only.push(a[++i]);
    else if (k === "-n" || k === "--candidates") opt.n = Number(a[++i]);
    else if (k === "--promote") opt.promote.push(a[++i]);
    else if (k === "--model") opt.model = a[++i];
    else if (k === "--budget") opt.budget = Number(a[++i]);
    else throw new Error(`unknown argument ${k}`);
  }
  return opt;
}

function readKey() {
  if (!existsSync(SECRETS_PATH)) throw new Error(`secrets file not found: ${SECRETS_PATH}`);
  const doc = parseYaml(readFileSync(SECRETS_PATH, "utf8"));
  const key = doc?.openai?.api_key;
  if (!key || typeof key !== "string") throw new Error("openai.api_key missing in secrets file");
  return key;
}

function readManifest() {
  if (!existsSync(MANIFEST_PATH)) return { generated: [], assets: {} };
  return JSON.parse(readFileSync(MANIFEST_PATH, "utf8"));
}

function writeManifest(m) {
  mkdirSync(OUT_DIR, { recursive: true });
  writeFileSync(MANIFEST_PATH, JSON.stringify(m, null, 2) + "\n");
}

const sha256 = (buf) => createHash("sha256").update(buf).digest("hex");

function buildPrompt(brand, asset) {
  const t = brand.templates[asset.template];
  if (!t) throw new Error(`asset ${asset.id}: unknown template ${asset.template}`);
  const body = t.body
    .replace("{subject}", asset.subject)
    .replace("{meaning}", asset.meaning ?? "")
    .trim();
  return [brand.style_preamble.trim(), body, brand.negative_block.trim()].join("\n\n");
}

async function usableImageModels(key) {
  const res = await fetch("https://api.openai.com/v1/models", { headers: { Authorization: `Bearer ${key}` } });
  if (!res.ok) throw new Error(`GET /v1/models failed: ${res.status}`);
  const { data } = await res.json();
  return data.map((m) => m.id).filter((id) => /image|dall-e/i.test(id)).sort();
}

async function pickModel(key, brand, override) {
  const available = await usableImageModels(key);
  if (override) {
    if (!available.includes(override)) throw new Error(`model ${override} not usable; usable: ${available.join(", ")}`);
    return override;
  }
  const chosen = brand.model.prefer.find((m) => available.includes(m));
  if (!chosen) throw new Error(`none of ${brand.model.prefer.join(", ")} usable; usable: ${available.join(", ")}`);
  return chosen;
}

async function generateOne(key, brand, model, asset, prompt, n) {
  const t = brand.templates[asset.template];
  const body = {
    model,
    prompt,
    n,
    size: t.size,
    quality: t.quality,
    output_format: "png",
    background: t.background === "transparent" ? "transparent" : "opaque",
  };
  const res = await fetch(brand.model.endpoint, {
    method: "POST",
    headers: { Authorization: `Bearer ${key}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`images API ${res.status} for ${asset.id}: ${text.slice(0, 400)}`);
  }
  const json = await res.json();
  return json.data.map((d) => Buffer.from(d.b64_json, "base64"));
}

function nextCandidateIndex(manifest, id) {
  const used = manifest.generated.filter((g) => g.id === id).map((g) => g.candidate);
  return used.length ? Math.max(...used) + 1 : 1;
}

// ---------------------------------------------------------------------------
// Promote: resize a chosen candidate into the delivered png + webp (+ copies).
// ---------------------------------------------------------------------------

function hasBin(name) {
  try {
    execFileSync("which", [name], { stdio: "ignore" });
    return true;
  } catch {
    return false;
  }
}

function promote(brand, manifest, spec) {
  const [id, kStr] = spec.split("=");
  const k = Number(kStr);
  const asset = brand.assets.find((a) => a.id === id);
  if (!asset || !k) throw new Error(`--promote expects <id>=<k>, got ${spec}`);
  const gen = manifest.generated.find((g) => g.id === id && g.candidate === k);
  if (!gen) throw new Error(`no candidate ${k} for ${id} in manifest`);
  const src = resolve(ROOT, gen.file);
  const t = brand.templates[asset.template];
  const [dw, dh] = t.deliver.split("x").map(Number);
  const [sw, sh] = t.size.split("x").map(Number);
  const outPng = resolve(ROOT, asset.output + ".png");
  mkdirSync(dirname(outPng), { recursive: true });
  copyFileSync(src, outPng);
  // crop to the delivered aspect first, centred, then resample (sips -z would distort otherwise).
  const targetRatio = dw / dh;
  if (Math.abs(sw / sh - targetRatio) > 0.01) {
    const cw = sw / sh > targetRatio ? Math.round(sh * targetRatio) : sw;
    const ch = sw / sh > targetRatio ? sh : Math.round(sw / targetRatio);
    execFileSync("sips", ["-c", String(ch), String(cw), outPng], { stdio: "ignore" });
  }
  execFileSync("sips", ["-z", String(dh), String(dw), outPng], { stdio: "ignore" });
  const outputs = [{ file: asset.output + ".png", sha256: sha256(readFileSync(outPng)) }];
  if (hasBin("cwebp")) {
    const outWebp = resolve(ROOT, asset.output + ".webp");
    execFileSync("cwebp", ["-quiet", "-q", "88", "-alpha_q", "100", outPng, "-o", outWebp], { stdio: "ignore" });
    outputs.push({ file: asset.output + ".webp", sha256: sha256(readFileSync(outWebp)) });
  }
  for (const copy of asset.copies ?? []) {
    for (const o of [...outputs]) {
      const ext = o.file.slice(o.file.lastIndexOf("."));
      const dest = resolve(ROOT, copy + ext);
      mkdirSync(dirname(dest), { recursive: true });
      copyFileSync(resolve(ROOT, o.file), dest);
      outputs.push({ file: copy + ext, sha256: o.sha256 });
    }
  }
  manifest.assets[id] = {
    id,
    type: asset.type,
    model: gen.model,
    prompt: gen.prompt,
    size: t.size,
    delivered: t.deliver,
    candidate: k,
    source_sha256: gen.sha256,
    outputs,
  };
  console.log(`promoted ${id} candidate ${k} -> ${outputs.map((o) => o.file).join(", ")}`);
}

// ---------------------------------------------------------------------------

async function main() {
  const opt = args();
  const brand = parseYaml(readFileSync(YAML_PATH, "utf8"));
  const manifest = readManifest();
  const perAsset = brand.limits.candidates_per_asset;
  const budget = opt.budget ?? brand.limits.images_per_budget;

  if (opt.promote.length) {
    for (const p of opt.promote) promote(brand, manifest, p);
    writeManifest(manifest);
    return;
  }

  const assets = brand.assets.filter((a) => !opt.only.length || opt.only.includes(a.id));
  if (!assets.length) throw new Error(`no asset matches --only ${opt.only.join(", ")}`);
  if (opt.n < 1 || opt.n > perAsset) throw new Error(`-n must be 1..${perAsset}`);

  if (opt.dryRun) {
    for (const a of assets) {
      const t = brand.templates[a.template];
      console.log(`\n== ${a.id} (${t.size}, ${t.quality}, ${t.background}) -> ${a.output}\n`);
      console.log(buildPrompt(brand, a));
    }
    return;
  }

  const key = readKey();
  if (opt.listModels) {
    console.log((await usableImageModels(key)).join("\n"));
    return;
  }
  const model = await pickModel(key, brand, opt.model);
  console.log(`model: ${model}; generated so far: ${manifest.generated.length}/${budget}`);

  for (const a of assets) {
    const have = nextCandidateIndex(manifest, a.id) - 1;
    const room = Math.min(opt.n, perAsset - have, budget - manifest.generated.length);
    if (room <= 0) {
      console.log(`skip ${a.id}: ${have}/${perAsset} candidates exist or budget ${budget} reached`);
      continue;
    }
    const prompt = buildPrompt(brand, a);
    process.stdout.write(`generating ${a.id} x${room} ... `);
    const images = await generateOne(key, brand, model, a, prompt, room);
    mkdirSync(CANDIDATES_DIR, { recursive: true });
    for (const buf of images) {
      const k = nextCandidateIndex(manifest, a.id);
      const file = `frontend/public/brand/generated/candidates/${a.id}-${k}.png`;
      writeFileSync(resolve(ROOT, file), buf);
      manifest.generated.push({
        id: a.id,
        candidate: k,
        model,
        prompt,
        size: brand.templates[a.template].size,
        file,
        sha256: sha256(buf),
        at: new Date().toISOString(),
      });
      writeManifest(manifest);
    }
    console.log(`ok (${images.length})`);
  }
  console.log(`total generated: ${manifest.generated.length}/${budget}`);
}

main().catch((e) => {
  console.error(e.message);
  process.exit(1);
});
