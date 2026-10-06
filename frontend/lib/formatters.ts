/**
 * Shared formatting utilities used across all pages.
 */

/** Truncate a hex hash or address for display. */
export function truncateHash(hash: string, chars = 6): string {
  if (!hash) return ""
  if (hash.length <= chars * 2 + 2) return hash
  return `${hash.slice(0, chars + 2)}...${hash.slice(-chars)}`
}

/** Alias for truncateHash — semantically clearer for addresses. */
export const truncateAddress = truncateHash

/** Format a date string to a human-readable format. */
export function formatDate(
  dateStr: string,
  options: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }
): string {
  return new Date(dateStr).toLocaleDateString("en-US", options)
}

/** Format a date string to a short date (e.g., "Apr 7, 2026"). */
export function formatShortDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

/** Format a date string to relative time (e.g., "2 hours ago"). */
export function formatRelativeTime(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const diffMin = Math.floor(diffMs / 60_000)
  const diffHour = Math.floor(diffMs / 3_600_000)
  const diffDay = Math.floor(diffMs / 86_400_000)

  if (diffMin < 1) return "just now"
  if (diffMin < 60) return `${diffMin}m ago`
  if (diffHour < 24) return `${diffHour}h ago`
  if (diffDay < 30) return `${diffDay}d ago`
  return formatShortDate(dateStr)
}

/** Get a block explorer URL for a transaction hash. */
export function getExplorerUrl(
  blockchain: string,
  txHash: string,
  type: "tx" | "address" = "tx"
): string {
  const explorers: Record<string, string> = {
    ethereum: "https://etherscan.io",
    bitcoin: "https://blockchair.com/bitcoin",
    base: "https://basescan.org",
    polygon: "https://polygonscan.com",
    tron: "https://tronscan.org",
  }
  const base = explorers[blockchain.toLowerCase()] ?? "https://etherscan.io"

  if (blockchain.toLowerCase() === "bitcoin") {
    return `${base}/${type === "tx" ? "transaction" : "address"}/${txHash}`
  }
  if (blockchain.toLowerCase() === "tron") {
    return `${base}/#/${type === "tx" ? "transaction" : "address"}/${txHash}`
  }
  return `${base}/${type}/${txHash}`
}

/** Format a crypto amount for display (trim trailing zeros). */
export function formatCryptoAmount(amount: string | number): string {
  const num = typeof amount === "string" ? parseFloat(amount) : amount
  if (isNaN(num)) return "0"
  if (num === 0) return "0"
  if (num >= 1) return num.toLocaleString("en-US", { maximumFractionDigits: 6 })
  return num.toFixed(8).replace(/0+$/, "").replace(/\.$/, "")
}

/** Copy text to clipboard and return a promise. */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
