import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * DataTable — backward-compatible version.
 *
 * Keeps the existing `Column` type and `keyOf` prop that all current
 * pages use. The worktree `DataTableColumn` + `getRowId` variant lives
 * at @/components/data-table for new pages.
 */
export interface Column<T> {
  key: string;
  header: ReactNode;
  cell: (row: T) => ReactNode;
  className?: string;
}

export function DataTable<T>({
  columns,
  rows,
  keyOf,
  onRowClick,
  empty,
}: {
  columns: Column<T>[];
  rows: T[];
  keyOf: (row: T) => string | number;
  onRowClick?: (row: T) => void;
  empty?: ReactNode;
}) {
  if (rows.length === 0) {
    return empty ?? null;
  }
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <table className="w-full text-sm">
        <thead className="border-b border-border bg-muted/30 text-left">
          <tr>
            {columns.map((c) => (
              <th
                key={c.key}
                className={cn(
                  "px-4 py-3 pm-label text-muted-foreground",
                  c.className
                )}
              >
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={keyOf(row)}
              className={cn(
                "border-b border-border last:border-0",
                onRowClick &&
                  "cursor-pointer transition-colors hover:bg-muted/30"
              )}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
            >
              {columns.map((c) => (
                <td key={c.key} className={cn("px-4 py-3", c.className)}>
                  {c.cell(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
