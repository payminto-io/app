import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

/** Label/value rows. `columns={2}` splits into a two-column grid from `sm` up. */
export function DetailList({
  children,
  columns = 1,
  className,
}: {
  children: ReactNode
  columns?: 1 | 2
  className?: string
}) {
  return (
    <dl
      className={cn(
        "grid gap-x-8 gap-y-4",
        columns === 2 && "sm:grid-cols-2",
        className
      )}
    >
      {children}
    </dl>
  )
}

/** One row. Renders nothing when `children` is empty, so a missing field is omitted rather than dashed. */
export function DetailItem({
  label,
  children,
  className,
}: {
  label: string
  children: ReactNode
  className?: string
}) {
  if (children === null || children === undefined || children === "" || children === false) return null
  return (
    <div className={cn("flex min-w-0 flex-col gap-1", className)}>
      <dt className="text-label font-medium text-ink-soft">{label}</dt>
      <dd className="min-w-0 text-body text-ink">{children}</dd>
    </div>
  )
}
