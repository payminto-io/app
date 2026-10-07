import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * DataTable with the `Column` + `keyOf` API that existing pages use.
 * The `DataTableColumn` + `getRowId` variant lives at @/components/data-table.
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
    <div className="overflow-hidden rounded-md border border-line bg-surface">
      <div data-slot="table-container" className="w-full overflow-x-auto">
        <table className="w-full text-body-sm">
          <thead className="border-b border-line bg-surface-sunken/70 text-left">
            <tr>
              {columns.map((c) => (
                <th
                  key={c.key}
                  className={cn(
                    "h-9 px-3 text-label font-medium whitespace-nowrap text-ink-soft",
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
                  "border-b border-line last:border-0 transition-colors duration-120",
                  onRowClick && "cursor-pointer hover:bg-surface-sunken/60"
                )}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {columns.map((c) => (
                  <td key={c.key} className={cn("h-9 px-3 py-2 align-middle text-ink", c.className)}>
                    {c.cell(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="border-t border-line px-3 py-2 text-caption text-ink-soft num">
        {rows.length} {rows.length === 1 ? "result" : "results"}
      </div>
    </div>
  );
}
