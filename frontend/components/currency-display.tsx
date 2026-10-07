import { cn } from "@/lib/utils"
import { formatDecimal, formatMinor } from "@/lib/money"

interface CurrencyDisplayProps {
  /** Decimal string as the API returns it, e.g. "1250.5". Ignored when `minor` is given. */
  amount: string
  /** Minor units; when present it is the source of truth. */
  minor?: bigint | string
  currency: string
  size?: "sm" | "md" | "lg" | "display"
  /** Signed ledger view: "+" colours ok, "-" stays ink. */
  signed?: boolean
  className?: string
}

const SIZE_MAP = {
  sm: "text-body-sm",
  md: "text-body",
  lg: "text-h2",
  display: "text-display",
}

/**
 * Amount in ink, code after it in ink-soft at 0.75em. Never a symbol.
 * Rules: docs/design/DESIGN.md section 8.
 */
export function CurrencyDisplay({
  amount,
  minor,
  currency,
  size = "md",
  signed = false,
  className,
}: CurrencyDisplayProps) {
  const text = minor !== undefined ? formatMinor(minor, currency) : formatDecimal(amount, currency)
  const positive = signed && !text.startsWith("-")
  return (
    <span
      className={cn(
        "num inline-flex items-baseline gap-[0.35em] whitespace-nowrap",
        SIZE_MAP[size],
        positive ? "text-ok" : "text-ink",
        className
      )}
    >
      <span className={cn(size === "display" || size === "lg" ? "font-semibold" : "font-medium")}>
        {positive ? "+" : ""}
        {text}
      </span>
      <span className="text-[max(0.75em,11px)] font-medium text-ink-soft">{currency.toUpperCase()}</span>
    </span>
  )
}
