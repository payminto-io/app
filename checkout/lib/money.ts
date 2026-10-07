/** Money formatting. Rules: docs/design/DESIGN.md section 8. Decimal strings in, display strings out. */

const FIAT: Record<string, number> = { USD: 2, EUR: 2, GBP: 2, INR: 2, SGD: 2, MYR: 2, JPY: 0 };
const STABLE = new Set(["USDC", "USDT", "DAI", "PYUSD", "USDP", "TUSD"]);

export function isFiat(code: string): boolean {
  return code.toUpperCase() in FIAT;
}

export function isStablecoin(code: string): boolean {
  return STABLE.has(code.toUpperCase());
}

function precision(code: string): { min: number; max: number } {
  const c = code.toUpperCase();
  if (c in FIAT) return { min: FIAT[c], max: FIAT[c] };
  if (STABLE.has(c)) return { min: 2, max: 6 };
  if (c === "SOL") return { min: 2, max: 9 };
  return { min: 2, max: 8 };
}

function group(intPart: string): string {
  return intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/** "1250.5" + "USD" -> "1,250.50". Truncates past the display max; never rounds. */
export function formatAmount(amount: string, code: string): string {
  const { min, max } = precision(code);
  const trimmed = amount.trim();
  if (!/^[-+]?\d*(\.\d+)?$/.test(trimmed) || trimmed === "" || trimmed === "-" || trimmed === "+") return amount;
  const neg = trimmed.startsWith("-");
  const [rawInt = "0", rawFrac = ""] = trimmed.replace(/^[-+]/, "").split(".");
  let frac = rawFrac.slice(0, max).replace(/0+$/, "");
  if (frac.length < min) frac = frac.padEnd(min, "0");
  const intPart = group(rawInt.replace(/^0+(?=\d)/, "") || "0");
  return `${neg ? "-" : ""}${intPart}${frac ? "." + frac : ""}`;
}

/** Exact decimal arithmetic on strings, enough for received/remaining sentences. */
export function subtractDecimal(a: string, b: string): string {
  const scale = Math.max(fracLength(a), fracLength(b));
  const diff = toScaled(a, scale) - toScaled(b, scale);
  const neg = diff < BigInt(0);
  const abs = (neg ? -diff : diff).toString().padStart(scale + 1, "0");
  const intPart = abs.slice(0, abs.length - scale);
  const frac = scale ? abs.slice(abs.length - scale) : "";
  return `${neg ? "-" : ""}${intPart}${frac ? "." + frac : ""}`;
}

export function compareDecimal(a: string, b: string): -1 | 0 | 1 {
  const scale = Math.max(fracLength(a), fracLength(b));
  const x = toScaled(a, scale);
  const y = toScaled(b, scale);
  return x < y ? -1 : x > y ? 1 : 0;
}

function fracLength(value: string): number {
  const i = value.indexOf(".");
  return i === -1 ? 0 : value.length - i - 1;
}

function toScaled(value: string, scale: number): bigint {
  const neg = value.trim().startsWith("-");
  const [intPart = "0", frac = ""] = value.trim().replace(/^[-+]/, "").split(".");
  const digits = `${intPart}${frac.padEnd(scale, "0")}`.replace(/^0+(?=\d)/, "") || "0";
  const n = BigInt(digits);
  return neg ? -n : n;
}
