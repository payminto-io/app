import { cn } from "@/lib/utils"

interface CurrencyDisplayProps {
  amount: string
  currency: string
  size?: "sm" | "md" | "lg"
  className?: string
}

const SIZE_MAP = {
  sm: "text-sm",
  md: "text-base",
  lg: "text-2xl",
}

export function CurrencyDisplay({
  amount,
  currency,
  size = "md",
  className,
}: CurrencyDisplayProps) {
  return (
    <span className={cn("inline-flex items-baseline gap-1.5 font-mono", SIZE_MAP[size], className)}>
      <span className="tabular-nums">{amount}</span>
      <span className="text-muted-foreground text-[0.75em] font-sans font-medium uppercase">
        {currency}
      </span>
    </span>
  )
}
