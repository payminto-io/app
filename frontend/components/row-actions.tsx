"use client"

import type { ReactNode } from "react"
import { MoreHorizontal } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

/** The kebab at the end of a table row. Children are `DropdownMenuItem`s. Never a hover reveal. */
export function RowActions({ children, label = "Row actions" }: { children: ReactNode; label?: string }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="ghost" size="icon-sm" className="-my-1.5" aria-label={label} onClick={(e) => e.stopPropagation()} />}
      >
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-auto min-w-40">
        {children}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
