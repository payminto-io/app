import { cn } from "@/lib/utils"

interface StatusBadgeProps {
  status: string
  className?: string
}

type StatusTone = "success" | "warning" | "danger" | "info" | "neutral"

const STATUS_MAP: Record<string, { label: string; tone: StatusTone }> = {
  created: { label: "Created", tone: "neutral" },
  confirming: { label: "Confirming", tone: "warning" },
  confirmed: { label: "Confirmed", tone: "success" },
  closed: { label: "Closed", tone: "success" },
  open: { label: "Open", tone: "warning" },
  cancelled: { label: "Cancelled", tone: "danger" },
  expired: { label: "Expired", tone: "neutral" },
  active: { label: "Active", tone: "success" },
  inactive: { label: "Inactive", tone: "neutral" },
  pending: { label: "Pending", tone: "warning" },
  processing: { label: "Processing", tone: "info" },
  completed: { label: "Completed", tone: "success" },
  failed: { label: "Failed", tone: "danger" },
  delivered: { label: "Delivered", tone: "success" },
  success: { label: "Success", tone: "success" },
  failure: { label: "Failure", tone: "danger" },
}

const TONE_STYLES: Record<StatusTone, string> = {
  success:
    "bg-[oklch(0.93_0.22_128_/_0.15)] text-[oklch(0.35_0.15_140)] dark:text-[oklch(0.85_0.22_140)]",
  warning:
    "bg-[oklch(0.82_0.17_80_/_0.18)] text-[oklch(0.42_0.15_75)] dark:text-[oklch(0.85_0.17_80)]",
  danger:
    "bg-[oklch(0.68_0.22_22_/_0.12)] text-[oklch(0.5_0.22_22)] dark:text-[oklch(0.8_0.22_22)]",
  info: "bg-[oklch(0.7_0.18_250_/_0.12)] text-[oklch(0.45_0.2_250)] dark:text-[oklch(0.85_0.18_250)]",
  neutral: "bg-muted text-muted-foreground",
}

const DOT_STYLES: Record<StatusTone, string> = {
  success: "bg-[oklch(0.72_0.22_140)]",
  warning: "bg-[oklch(0.78_0.17_80)]",
  danger: "bg-[oklch(0.68_0.22_22)]",
  info: "bg-[oklch(0.65_0.18_250)]",
  neutral: "bg-muted-foreground/50",
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const key = status.toLowerCase()
  const config = STATUS_MAP[key] ?? { label: status, tone: "neutral" as StatusTone }

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-medium",
        TONE_STYLES[config.tone],
        className
      )}
    >
      <span className={cn("size-1.5 rounded-full", DOT_STYLES[config.tone])} />
      {config.label}
    </span>
  )
}
