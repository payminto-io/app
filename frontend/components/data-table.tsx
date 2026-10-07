"use client"

import * as React from "react"

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { LoadingRows } from "@/components/ui/states"
import { TableEmpty } from "@/components/table-empty"
import { cn } from "@/lib/utils"

export type DataTableColumn<T> = {
  key: string
  header: React.ReactNode
  cell: (row: T) => React.ReactNode
  className?: string
  align?: "left" | "right" | "center"
}

export type DataTableProps<T> = {
  columns: DataTableColumn<T>[]
  rows: T[]
  loading?: boolean
  onRowClick?: (row: T) => void
  emptyTitle?: string
  emptyDescription?: string
  emptyAction?: React.ReactNode
  className?: string
  getRowId?: (row: T, index: number) => string | number
  /** Server total when the rows are one page of a longer list. */
  total?: number
  /** Replaces the result count under the table, e.g. with `Pagination`. */
  footer?: React.ReactNode
}

export function DataTable<T>({
  columns,
  rows,
  loading,
  onRowClick,
  emptyTitle = "Nothing here yet",
  emptyDescription,
  emptyAction,
  className,
  getRowId,
  total,
  footer,
}: DataTableProps<T>) {
  if (loading) {
    return <LoadingRows rows={5} />
  }

  if (!rows.length) {
    return (
      <TableEmpty
        headers={columns.map((c) => c.header)}
        title={emptyTitle}
        description={emptyDescription}
        action={emptyAction}
        className={className}
      />
    )
  }

  return (
    <div className={cn("overflow-hidden rounded-md border border-line bg-surface", className)}>
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((col) => (
              <TableHead
                key={col.key}
                className={cn(
                  col.align === "right" && "text-right",
                  col.align === "center" && "text-center",
                  col.className
                )}
              >
                {col.header}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, i) => {
            const id = getRowId ? getRowId(row, i) : i
            return (
              <TableRow
                key={id}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                className={cn(onRowClick && "cursor-pointer")}
              >
                {columns.map((col) => (
                  <TableCell
                    key={col.key}
                    className={cn(
                      col.align === "right" && "num text-right",
                      col.align === "center" && "text-center",
                      col.className
                    )}
                  >
                    {col.cell(row)}
                  </TableCell>
                ))}
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
      <div className="num border-t border-line px-3 py-2 text-caption text-ink-soft">
        {footer ?? `${total ?? rows.length} ${(total ?? rows.length) === 1 ? "result" : "results"}`}
      </div>
    </div>
  )
}
