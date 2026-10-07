import { cn } from "@/lib/utils"
import type { BlockchainNetwork } from "@/lib/types"

interface BlockchainIconProps {
  blockchain: BlockchainNetwork
  size?: "sm" | "md" | "lg"
  showLabel?: boolean
  className?: string
}

/** Chain names and tickers. The mark is a neutral monogram; colour never identifies a chain. */
const CHAIN_CONFIG: Record<BlockchainNetwork, { label: string; abbr: string }> = {
  bitcoin: { label: "Bitcoin", abbr: "BTC" },
  ethereum: { label: "Ethereum", abbr: "ETH" },
  base: { label: "Base", abbr: "BASE" },
  polygon: { label: "Polygon", abbr: "POL" },
  tron: { label: "Tron", abbr: "TRX" },
}

const SIZE_MAP = {
  sm: "size-5 text-[11px]",
  md: "size-6 text-label",
  lg: "size-8 text-body-sm",
}

export function BlockchainIcon({
  blockchain,
  size = "md",
  showLabel = false,
  className,
}: BlockchainIconProps) {
  const config = CHAIN_CONFIG[blockchain]
  if (!config) return null

  return (
    <span className={cn("inline-flex items-center gap-1.5", className)}>
      <span
        aria-hidden={showLabel}
        title={config.label}
        className={cn(
          "inline-flex shrink-0 items-center justify-center rounded-full border border-line bg-surface-sunken font-mono font-medium text-ink-soft",
          SIZE_MAP[size]
        )}
      >
        {config.abbr.charAt(0)}
      </span>
      {showLabel && <span className="text-body text-ink">{config.label}</span>}
    </span>
  )
}

export function BlockchainLabel({
  blockchain,
  className,
}: {
  blockchain: BlockchainNetwork
  className?: string
}) {
  const config = CHAIN_CONFIG[blockchain]
  if (!config) return null

  return (
    <span className={cn("inline-flex items-center gap-1.5 text-body-sm text-ink", className)}>
      {config.label}
      <span className="font-mono text-label text-ink-soft">{config.abbr}</span>
    </span>
  )
}

export { CHAIN_CONFIG }
