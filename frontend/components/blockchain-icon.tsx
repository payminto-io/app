import { cn } from "@/lib/utils"
import type { BlockchainNetwork } from "@/lib/types"

interface BlockchainIconProps {
  blockchain: BlockchainNetwork
  size?: "sm" | "md" | "lg"
  showLabel?: boolean
  className?: string
}

const CHAIN_CONFIG: Record<
  BlockchainNetwork,
  { label: string; abbr: string; color: string; bgColor: string }
> = {
  bitcoin: {
    label: "Bitcoin",
    abbr: "BTC",
    color: "text-orange-400",
    bgColor: "bg-orange-500/10",
  },
  ethereum: {
    label: "Ethereum",
    abbr: "ETH",
    color: "text-blue-400",
    bgColor: "bg-blue-500/10",
  },
  base: {
    label: "Base",
    abbr: "BASE",
    color: "text-blue-300",
    bgColor: "bg-blue-400/10",
  },
  polygon: {
    label: "Polygon",
    abbr: "POL",
    color: "text-purple-400",
    bgColor: "bg-purple-500/10",
  },
  tron: {
    label: "Tron",
    abbr: "TRX",
    color: "text-red-400",
    bgColor: "bg-red-500/10",
  },
}

const SIZE_MAP = {
  sm: "size-5 text-[10px]",
  md: "size-6 text-xs",
  lg: "size-8 text-sm",
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
        className={cn(
          "inline-flex items-center justify-center rounded-full font-mono font-semibold",
          config.bgColor,
          config.color,
          SIZE_MAP[size]
        )}
      >
        {config.abbr.charAt(0)}
      </span>
      {showLabel && (
        <span className={cn("text-sm font-medium", config.color)}>
          {config.label}
        </span>
      )}
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
    <span className={cn("inline-flex items-center gap-1.5", className)}>
      <span className={cn("size-2 rounded-full", config.bgColor, config.color)} style={{ backgroundColor: "currentColor" }} />
      <span className="text-sm">{config.label}</span>
    </span>
  )
}

export { CHAIN_CONFIG }
