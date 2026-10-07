/** Display names for the backend's chain codes. Unknown codes render as the code itself. */
const CHAIN_NAME: Record<string, string> = {
  ETH: "Ethereum",
  BASE: "Base",
  POLYGON: "Polygon",
  BTC: "Bitcoin",
  TRX: "Tron",
  TRON: "Tron",
};

export function chainName(code: string | undefined): string {
  if (!code) return "";
  return CHAIN_NAME[code.toUpperCase()] ?? code;
}

/** Chain codes accepted by the merchant payout and recipient forms. */
export const PAYOUT_CHAINS = ["ETH", "BTC", "BASE", "POLYGON", "TRON"] as const;
