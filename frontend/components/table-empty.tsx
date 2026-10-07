import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/**
 * Empty list that keeps the table's shape: the real header, two ghost rows,
 * one sentence and one action. references.md, "Empty and error states".
 */
export function TableEmpty({
  headers,
  title,
  description,
  action,
  className,
}: {
  headers: ReactNode[]
  title: string
  description?: string
  action?: ReactNode
  className?: string
}) {
  return (
    <div className={cn("overflow-hidden rounded-md border border-line bg-surface", className)}>
      <div aria-hidden className="overflow-hidden">
        <div className="flex h-9 items-center gap-6 border-b border-line bg-surface-sunken/70 px-3">
          {headers.map((h, i) => (
            <span key={i} className="min-w-0 flex-1 truncate text-label font-medium text-ink-soft last:text-right">
              {h}
            </span>
          ))}
        </div>
        {[0, 1].map((r) => (
          <div key={r} className="flex h-9 items-center gap-6 border-b border-line px-3">
            {headers.map((_, i) => (
              <span key={i} className="flex min-w-0 flex-1 last:justify-end">
                <span className={cn("h-2 rounded-full bg-surface-sunken", r === 0 ? "w-3/5" : "w-2/5")} />
              </span>
            ))}
          </div>
        ))}
      </div>
      <div className="flex flex-col items-center px-6 py-8 text-center">
        <p className="text-body font-medium text-ink">{title}</p>
        {description ? <p className="mt-1 max-w-[44ch] text-body-sm text-ink-soft">{description}</p> : null}
        {action ? <div className="mt-4">{action}</div> : null}
      </div>
    </div>
  )
}
