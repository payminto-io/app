/**
 * Tempo rail. Tempo pays a merchant's fixed receiving address and tells
 * payments apart by a 32-byte memo on the TIP-20 `transferWithMemo` call, so
 * the checkout shows the memo next to the address instead of a per-payment
 * deposit address. The watcher matches the memo against the payment
 * reference, so the encoding here must stay byte-identical to the backend's.
 */

/** Blockchain code the API uses for the Tempo rail. */
export const TEMPO_CODE = "TEMPO";

/** Currency code the API uses for pathUSD. Display uses `TEMPO_TESTNET.token.symbol`. */
export const PATHUSD_CODE = "PATHUSD";

export const TEMPO_TESTNET = {
  chainId: 42431,
  name: "Tempo testnet",
  rpcUrl: "https://rpc.moderato.tempo.xyz",
  explorerUrl: "https://explore.testnet.tempo.xyz",
  token: {
    symbol: "pathUSD",
    standard: "TIP-20",
    address: "0x20c0000000000000000000000000000000000000",
    decimals: 6,
  },
} as const;

export const MEMO_BYTES = 32;

export class MemoError extends Error {}

/**
 * Payment reference -> bytes32 memo: UTF-8 bytes, left aligned, zero padded,
 * 0x-prefixed lowercase hex. Throws when the reference is empty or longer
 * than 32 bytes, because a truncated memo would never match.
 */
export function encodeMemo(reference: string): `0x${string}` {
  if (!reference) throw new MemoError("A memo needs a payment reference");
  const bytes = new TextEncoder().encode(reference);
  if (bytes.length > MEMO_BYTES) throw new MemoError(`Payment reference is ${bytes.length} bytes; a Tempo memo holds ${MEMO_BYTES}`);
  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
  return `0x${hex.padEnd(MEMO_BYTES * 2, "0")}`;
}

/** The memo for a reference, or undefined when it cannot be one. */
export function tryEncodeMemo(reference: string | undefined): `0x${string}` | undefined {
  if (!reference) return undefined;
  try {
    return encodeMemo(reference);
  } catch {
    return undefined;
  }
}

/** Decimal string -> integer base units, exact. "128.5" at 6 decimals -> 128500000n. */
export function toBaseUnits(amount: string, decimals: number): bigint {
  const trimmed = amount.trim();
  if (!/^\d+(\.\d+)?$/.test(trimmed)) throw new Error(`Invalid amount ${amount}`);
  const [whole, frac = ""] = trimmed.split(".");
  if (frac.replace(/0+$/, "").length > decimals) throw new Error(`Amount ${amount} has more than ${decimals} decimals`);
  return BigInt(whole + frac.padEnd(decimals, "0").slice(0, decimals));
}
