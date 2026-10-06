import { cn } from "@/lib/utils";

/**
 * StatusBadge — merged version.
 *
 * Keeps all status keys from the current working frontend (payments,
 * withdrawals, generic) and adds the worktree design (oklch dot + pill).
 * Accepts optional className like the worktree version.
 */

type StatusTone = "success" | "warning" | "danger" | "info" | "neutral";

const STATUS_MAP: Record<string, { label: string; tone: StatusTone }> = {
  // payments
  created: { label: "Created", tone: "neutral" },
  confirming: { label: "Confirming", tone: "warning" },
  confirmed: { label: "Confirmed", tone: "success" },
  closed: { label: "Closed", tone: "success" },
  open: { label: "Open", tone: "warning" },
  cancelled: { label: "Cancelled", tone: "danger" },
  expired: { label: "Expired", tone: "neutral" },
  partially_filled: { label: "Partially Filled", tone: "info" },
  filled: { label: "Filled", tone: "success" },
  over_filled: { label: "Over Filled", tone: "info" },
  refunded: { label: "Refunded", tone: "warning" },
  // withdrawals
  pending_approval: { label: "Pending Approval", tone: "warning" },
  approved: { label: "Approved", tone: "info" },
  initiated: { label: "Initiated", tone: "info" },
  sent: { label: "Sent", tone: "success" },
  processed: { label: "Processed", tone: "success" },
  // generic
  active: { label: "Active", tone: "success" },
  inactive: { label: "Inactive", tone: "neutral" },
  pending: { label: "Pending", tone: "warning" },
  processing: { label: "Processing", tone: "info" },
  completed: { label: "Completed", tone: "success" },
  failed: { label: "Failed", tone: "danger" },
  delivered: { label: "Delivered", tone: "success" },
  success: { label: "Success", tone: "success" },
  failure: { label: "Failure", tone: "danger" },
  resolved: { label: "Resolved", tone: "success" },
  dismissed: { label: "Dismissed", tone: "neutral" },
};

const TONE_STYLES: Record<StatusTone, string> = {
  success:
    "bg-emerald-50 text-emerald-700 border-emerald-200",
  warning:
    "bg-amber-50 text-amber-700 border-amber-200",
  danger:
    "bg-red-50 text-red-700 border-red-200",
  info:
    "bg-blue-50 text-blue-700 border-blue-200",
  neutral:
    "bg-zinc-100 text-zinc-700 border-zinc-200",
};

const DOT_STYLES: Record<StatusTone, string> = {
  success: "bg-emerald-500",
  warning: "bg-amber-500",
  danger: "bg-red-500",
  info: "bg-blue-500",
  neutral: "bg-zinc-500",
};

export function StatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const key = status.toLowerCase();
  const config = STATUS_MAP[key] ?? {
    label: status.replace(/_/g, " "),
    tone: "neutral" as StatusTone,
  };

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-[11px] font-medium",
        TONE_STYLES[config.tone],
        className
      )}
    >
      <span className={cn("size-1.5 rounded-full", DOT_STYLES[config.tone])} />
      {config.label}
    </span>
  );
}
