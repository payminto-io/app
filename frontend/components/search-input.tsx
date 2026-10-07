"use client"

import * as React from "react"
import { Search } from "lucide-react"

import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

/** Input with a leading search glyph. The placeholder is an example, so `aria-label` names the field. */
export function SearchInput({
  className,
  ...props
}: Omit<React.ComponentProps<typeof Input>, "type"> & { "aria-label": string }) {
  return (
    <div className={cn("relative w-full sm:max-w-xs", className)}>
      <Search
        aria-hidden
        className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-ink-faint"
      />
      <Input type="search" className="pl-9" {...props} />
    </div>
  )
}
