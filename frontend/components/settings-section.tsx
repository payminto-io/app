import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/**
 * A settings group: title and one line on the left, the controls in a card
 * on the right from `lg` up; stacked below it.
 */
export function SettingsSection({
  title,
  description,
  children,
  className,
}: {
  title: string
  description?: string
  children: ReactNode
  className?: string
}) {
  return (
    <section className={cn("grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,2fr)] lg:gap-8", className)}>
      <div className="min-w-0">
        <h2 className="text-h3 font-semibold text-ink">{title}</h2>
        {description ? <p className="mt-1 text-body-sm text-ink-soft">{description}</p> : null}
      </div>
      <div className="min-w-0 rounded-md border border-line bg-surface p-5 max-sm:p-4">{children}</div>
    </section>
  )
}
