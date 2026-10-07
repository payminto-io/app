import type { ReactNode } from "react"
import { AlertTriangle, Info, XCircle } from "lucide-react"

import { cn } from "@/lib/utils"

const TONE = {
  note: { box: "border-note/25 bg-note-tint", icon: Info, glyph: "text-note" },
  wait: { box: "border-wait/30 bg-wait-tint", icon: AlertTriangle, glyph: "text-wait" },
  bad: { box: "border-bad/30 bg-bad-tint", icon: XCircle, glyph: "text-bad" },
} as const

/** One-line strip for a condition the page cannot hide. Text stays in ink; tone is the border, tint and glyph. */
export function Notice({
  tone = "note",
  children,
  action,
  className,
}: {
  tone?: keyof typeof TONE
  children: ReactNode
  action?: ReactNode
  className?: string
}) {
  const t = TONE[tone]
  const Icon = t.icon
  return (
    <div
      role={tone === "bad" ? "alert" : "status"}
      className={cn("flex items-start gap-2.5 rounded-sm border px-3 py-2.5", t.box, className)}
    >
      <Icon aria-hidden className={cn("mt-0.5 size-4 shrink-0", t.glyph)} />
      <div className="min-w-0 flex-1 text-body-sm text-ink">{children}</div>
      {action ? <div className="shrink-0">{action}</div> : null}
    </div>
  )
}
