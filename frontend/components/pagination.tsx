"use client"

import { ChevronLeft, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

/** Offset pagination: "21-40 of 134" and two icon buttons. Renders nothing for one page. Fits `DataTable`'s footer. */
export function Pagination({
  offset,
  limit,
  total,
  onOffsetChange,
  className,
}: {
  offset: number
  limit: number
  total: number
  onOffsetChange: (offset: number) => void
  className?: string
}) {
  if (total <= limit) return null
  const from = offset + 1
  const to = Math.min(offset + limit, total)
  return (
    <div className={cn("flex items-center justify-between gap-3", className)}>
      <p className="num text-caption text-ink-soft">
        {from}-{to} of {total}
      </p>
      <div className="flex items-center gap-1">
        <Button
          variant="outline"
          size="icon-xs"
          aria-label="Previous page"
          disabled={offset === 0}
          onClick={() => onOffsetChange(Math.max(0, offset - limit))}
        >
          <ChevronLeft />
        </Button>
        <Button
          variant="outline"
          size="icon-xs"
          aria-label="Next page"
          disabled={offset + limit >= total}
          onClick={() => onOffsetChange(offset + limit)}
        >
          <ChevronRight />
        </Button>
      </div>
    </div>
  )
}
