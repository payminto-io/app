import * as React from "react"
import { cn } from "@/lib/utils"

interface MetricCardProps {
  label: string
  value: string | number
  sublabel?: string
  trend?: { value: string; direction: "up" | "down" | "neutral" }
  variant?: "primary" | "dark" | "lime"
  className?: string
}

/**
 * High-signal merchant metrics with one dark hero and quiet supporting cards.
 */
export function MetricCard({
  label,
  value,
  sublabel,
  trend,
  variant = "dark",
  className,
}: MetricCardProps) {
  if (variant === "lime") {
    return (
      <div
        className={cn(
          "relative overflow-hidden rounded-xl border border-emerald-200 bg-emerald-50 p-5 text-emerald-950",
          className
        )}
      >
        <div className="pm-label !text-emerald-700">{label}</div>
        <div className="pm-metric-value mt-3 !text-emerald-950">{value}</div>
        {sublabel && (
          <div className="mt-2 flex items-center gap-1.5 text-[12px] font-semibold text-emerald-700">
            <span className="size-1.5 rounded-full bg-emerald-500" />
            {sublabel}
          </div>
        )}
      </div>
    )
  }

  const isPrimary = variant === "primary"

  if (!isPrimary) {
    return (
      <div
        className={cn(
          "relative overflow-hidden rounded-xl border border-border bg-white p-5 shadow-[0_1px_2px_rgba(16,24,40,.03)]",
          className
        )}
      >
        <div className="pm-label">{label}</div>
        <div className="pm-metric-value mt-3">{value}</div>
        {(sublabel || trend) && (
          <div className="mt-2 flex items-center gap-2 text-[12px] text-muted-foreground">
            {trend ? (
              <span
                className={cn(
                  "font-semibold",
                  trend.direction === "up" && "text-emerald-700",
                  trend.direction === "down" && "text-red-700"
                )}
              >
                {trend.direction === "up" && "↑ "}
                {trend.direction === "down" && "↓ "}
                {trend.value}
              </span>
            ) : null}
            {sublabel ? <span>{sublabel}</span> : null}
          </div>
        )}
      </div>
    )
  }

  return (
    <div
      className={cn(
        "pm-metric-card p-5",
        isPrimary && "md:row-span-1",
        className
      )}
    >
      <div className="relative z-10">
        <div className="pm-label text-white/60">{label}</div>
        <div
          className={cn(
            "pm-metric-value mt-3 text-white",
            isPrimary && "text-[2.75rem]"
          )}
        >
          {value}
        </div>
        {(sublabel || trend) && (
          <div className="mt-2 flex items-center gap-2 text-[12px]">
            {trend && (
              <span
                className={cn(
                  "font-semibold",
                  trend.direction === "up" && "text-emerald-300",
                  trend.direction === "down" && "text-[oklch(0.68_0.22_22)]",
                  trend.direction === "neutral" && "text-white/60"
                )}
              >
                {trend.direction === "up" && "↑ "}
                {trend.direction === "down" && "↓ "}
                {trend.value}
              </span>
            )}
            {sublabel && <span className="text-white/50">{sublabel}</span>}
          </div>
        )}
      </div>
    </div>
  )
}
