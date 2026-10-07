"use client"

import { CopyButton } from "@/components/copy-button"
import { cn } from "@/lib/utils"

/**
 * A long identifier (address, link, key, hash) in mono, truncated to one line,
 * with a copy button. The full value is in the title and on the clipboard.
 */
export function CopyField({
  value,
  display,
  className,
  boxed = true,
}: {
  value: string
  /** Shown instead of `value`, e.g. a truncated hash. */
  display?: string
  className?: string
  /** Sunken box for a hero value; false for an inline value in a detail list. */
  boxed?: boolean
}) {
  return (
    <div
      className={cn(
        "flex min-w-0 items-center gap-1.5",
        boxed && "rounded-sm border border-line bg-surface-sunken py-1 pr-1 pl-3",
        className
      )}
    >
      <code title={value} className={cn("min-w-0 truncate font-mono text-body-sm text-ink", boxed && "flex-1")}>
        {display ?? value}
      </code>
      <CopyButton value={value} variant="ghost" size="icon" label="" className={cn("shrink-0", boxed ? "size-7" : "size-6")} />
    </div>
  )
}
