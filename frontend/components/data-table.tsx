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

/**
 * Where a column goes in the stacked row shown below the tablet breakpoint
 * (1024px, DESIGN.md section 11):
 * - `lead`: line one, left (the amount, or the row's name)
 * - `trail`: line one, right (the status)
 * - `meta`: line two, soft, no label (reference, customer, date)
 * - `detail`: a labelled row under line two (the default)
 * - `action`: the row kebab, top right
 * - `hidden`: not shown when stacked
 */
export type DataTableStack = "lead" | "trail" | "meta" | "detail" | "action" | "hidden"

export type DataTableColumn<T> = {
  key: string
  header: React.ReactNode
  cell: (row: T) => React.ReactNode
  className?: string
  align?: "left" | "right" | "center"
  stack?: DataTableStack
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
  /** Replaces the result count under the table, e.g. with `Pagination`; `null` hides it. */
  footer?: React.ReactNode
}

/** An unannotated table still stacks: first column leads, `status` trails, `actions` is the kebab. */
export function stackRole<T>(col: DataTableColumn<T>, index: number): DataTableStack {
  if (col.stack) return col.stack
  if (col.key === "actions") return "action"
  if (col.key === "status") return "trail"
  return index === 0 ? "lead" : "detail"
}

function isBlank(node: React.ReactNode) {
  return node === null || node === undefined || node === false || node === ""
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
  const [scrolled, setScrolled] = React.useState(false)

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

  const roles = columns.map((c, i) => stackRole(c, i))
  const pick = (role: DataTableStack) => columns.filter((_, i) => roles[i] === role)
  const lead = pick("lead")
  const trail = pick("trail")
  const meta = pick("meta")
  const detail = pick("detail")
  const action = pick("action")

  return (
    <div className={cn("overflow-hidden rounded-md border border-line bg-surface", className)}>
      <ul data-slot="data-table-stack" className="divide-y divide-line lg:hidden">
        {rows.map((row, i) => {
          const id = getRowId ? getRowId(row, i) : i
          const metaCells = meta.map((c) => [c, c.cell(row)] as const).filter(([, v]) => !isBlank(v))
          const detailCells = detail.map((c) => [c, c.cell(row)] as const).filter(([, v]) => !isBlank(v))
          return (
            <li
              key={id}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
              className={cn(
                "px-3 py-3 transition-colors duration-120",
                onRowClick && "cursor-pointer hover:bg-surface-sunken/60"
              )}
            >
              <div className="flex min-h-6 items-center justify-between gap-3">
                <div className="flex min-w-0 items-center gap-2 text-body font-medium text-ink">
                  {lead.map((c) => (
                    <div key={c.key} className="min-w-0 truncate">
                      {c.cell(row)}
                    </div>
                  ))}
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  {trail.map((c) => (
                    <div key={c.key}>{c.cell(row)}</div>
                  ))}
                  {action.map((c) => (
                    <div key={c.key}>{c.cell(row)}</div>
                  ))}
                </div>
              </div>
              {metaCells.length ? (
                <div className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-0.5 text-body-sm text-ink-soft">
                  {metaCells.map(([c, v]) => (
                    <div key={c.key} className="min-w-0 max-w-full truncate">
                      {v}
                    </div>
                  ))}
                </div>
              ) : null}
              {detailCells.length ? (
                <dl className="mt-2 grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] gap-x-4 gap-y-1 text-body-sm">
                  {detailCells.map(([c, v]) => (
                    <React.Fragment key={c.key}>
                      <dt className="text-ink-soft">{c.header}</dt>
                      <dd className="min-w-0 truncate text-right text-ink">{v}</dd>
                    </React.Fragment>
                  ))}
                </dl>
              ) : null}
            </li>
          )
        })}
      </ul>

      <div className="hidden lg:block">
        <Table
          containerProps={{
            className: "group/scroll",
            onScroll: (e) => setScrolled(e.currentTarget.scrollLeft > 0),
            ...{ "data-scrolled": scrolled ? "true" : undefined },
          }}
        >
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              {columns.map((col, ci) => (
                <TableHead
                  key={col.key}
                  className={cn(
                    col.align === "right" && "text-right",
                    col.align === "center" && "text-center",
                    ci === 0 && STICKY_HEAD,
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
                  className={cn("group/row", onRowClick && "cursor-pointer")}
                >
                  {columns.map((col, ci) => (
                    <TableCell
                      key={col.key}
                      className={cn(
                        col.align === "right" && "num text-right",
                        col.align === "center" && "text-center",
                        ci === 0 && STICKY_CELL,
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
      </div>
      {footer !== null ? (
        <div className="num border-t border-line px-3 py-2 text-caption text-ink-soft">
          {footer ?? `${total ?? rows.length} ${(total ?? rows.length) === 1 ? "result" : "results"}`}
        </div>
      ) : null}
    </div>
  )
}

/* Sticky first column for wide tables (DESIGN.md section 11); fills are opaque mixes of the translucent row and header tints. */
const STICKY_BASE =
  "sticky left-0 z-[1] group-data-[scrolled=true]/scroll:shadow-[inset_-1px_0_0_var(--line)]"
const STICKY_HEAD = cn(STICKY_BASE, "bg-[color-mix(in_srgb,var(--surface-sunken)_70%,var(--surface))]")
const STICKY_CELL = cn(
  STICKY_BASE,
  "bg-surface group-hover/row:bg-[color-mix(in_srgb,var(--surface-sunken)_60%,var(--surface))]"
)
