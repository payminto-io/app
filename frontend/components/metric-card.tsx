import * as React from "react"
import { cn } from "@/lib/utils"

interface MetricCardProps {
  label: string
  value: string | number
  sublabel?: string
  trend?: { value: string; direction: "up" | "down" | "neutral" }
  /** Kept for API stability; every variant renders the same quiet card. */
  variant?: "primary" | "dark" | "lime"
  className?: string
}

/**
 * A metric is a label, a tabular value and a caption. Never a hero panel.
 * A trend is rendered only when the caller computed it from two real periods.
 */
export function MetricCard({
  label,
  value,
  sublabel,
  trend,
  className,
}: MetricCardProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-2 rounded-md border border-line bg-surface p-5 max-sm:p-4",
        className
      )}
    >
      <div className="text-label font-medium text-ink-soft">{label}</div>
      <div className="num text-h1 font-semibold text-ink">{value}</div>
      {(sublabel || trend) && (
        <div className="flex items-center gap-2 text-caption text-ink-soft">
          {trend ? (
            <span
              className={cn(
                "num font-medium",
                trend.direction === "up" && "text-ok",
                trend.direction === "down" && "text-bad",
                trend.direction === "neutral" && "text-ink-soft"
              )}
            >
              {trend.direction === "up" && "+"}
              {trend.direction === "down" && "-"}
              {trend.value}
            </span>
          ) : null}
          {sublabel ? <span>{sublabel}</span> : null}
        </div>
      )}
    </div>
  )
}
