/**
 * Money formatting. Rules: docs/design/DESIGN.md section 8.
 * Amounts arrive as minor units (bigint or string of digits) with a currency
 * or asset code; this is the only place that divides.
 */

const FIAT_DECIMALS: Record<string, number> = { USD: 2, EUR: 2, GBP: 2, INR: 2, SGD: 2, MYR: 2, JPY: 0 };
const ASSET_DECIMALS: Record<string, number> = {
  USDC: 6, USDT: 6, SOL: 9, BTC: 8, ETH: 18, TRX: 6, MATIC: 18, BNB: 18,
};

/** Display precision: fiat fixed, stablecoins up to 6, native assets up to 8 (or 9 for SOL). */
function displayPrecision(code: string): { min: number; max: number } {
  const c = code.toUpperCase();
  if (c in FIAT_DECIMALS) return { min: FIAT_DECIMALS[c], max: FIAT_DECIMALS[c] };
  if (c === "USDC" || c === "USDT") return { min: 2, max: 6 };
  if (c === "SOL") return { min: 2, max: 9 };
  if (c === "BTC") return { min: 2, max: 8 };
  if (c in ASSET_DECIMALS) return { min: 2, max: 8 };
  return { min: 2, max: 8 };
}

export function minorUnitDecimals(code: string): number {
  const c = code.toUpperCase();
  return FIAT_DECIMALS[c] ?? ASSET_DECIMALS[c] ?? 2;
}

function groupThousands(intPart: string): string {
  return intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/**
 * Format a decimal string ("1250.5") for display: grouped, with the
 * precision rules for the code. Never rounds beyond the display max; the
 * caller keeps the full value for tooltips.
 */
export function formatDecimal(amount: string, code: string): string {
  const { min, max } = displayPrecision(code);
  const neg = amount.trim().startsWith("-");
  const [rawInt = "0", rawFrac = ""] = amount.trim().replace(/^[-+]/, "").split(".");
  let frac = rawFrac.slice(0, max);
  frac = frac.replace(/0+$/, "");
  if (frac.length < min) frac = frac.padEnd(min, "0");
  const intPart = groupThousands(rawInt.replace(/^0+(?=\d)/, "") || "0");
  return `${neg ? "-" : ""}${intPart}${frac ? "." + frac : ""}`;
}

/** Minor units (bigint or digit string) to a decimal string using the code's own decimals. */
export function minorToDecimal(minor: bigint | string, code: string): string {
  const decimals = minorUnitDecimals(code);
  const value = typeof minor === "bigint" ? minor : BigInt(minor);
  const neg = value < BigInt(0);
  const abs = neg ? -value : value;
  const s = abs.toString().padStart(decimals + 1, "0");
  const intPart = s.slice(0, s.length - decimals);
  const frac = decimals ? s.slice(s.length - decimals) : "";
  return `${neg ? "-" : ""}${intPart}${frac ? "." + frac : ""}`;
}

export function formatMinor(minor: bigint | string, code: string): string {
  return formatDecimal(minorToDecimal(minor, code), code);
}
