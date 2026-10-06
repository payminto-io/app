"use client";

import { useState } from "react";
import { Search } from "lucide-react";
import { useCustomersList } from "@/lib/query/hooks/use-customers";
import { ErrorState } from "@/components/ui/states";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Customer } from "@/lib/query/hooks/use-customers";

const PAGE_SIZE = 25;

export default function CustomersPage() {
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");

  const filters = { limit: PAGE_SIZE, offset };
  const { data, isLoading, error, refetch } = useCustomersList(filters);

  const filtered = data?.customers.filter((c: Customer) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      c.name.toLowerCase().includes(q) ||
      c.email?.toLowerCase().includes(q) ||
      c.customerID?.toLowerCase().includes(q)
    );
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Customers</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Customers auto-created from payment requests.
          </p>
        </div>
      </div>

      {/* Search */}
      <div className="relative max-w-sm">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <input
          type="text"
          placeholder="Search by name, email, or customer ID..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full rounded-lg border border-border bg-background pl-10 pr-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>

      {/* Error state */}
      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {/* Table */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      ) : filtered && filtered.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">Name</TableHead>
                  <TableHead className="pm-label">Email</TableHead>
                  <TableHead className="pm-label">Customer ID</TableHead>
                  <TableHead className="pm-label">Status</TableHead>
                  <TableHead className="pm-label text-right pr-6">
                    Created
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((c: Customer) => (
                  <TableRow
                    key={c.id}
                    className="border-border/60 hover:bg-muted/30"
                  >
                    <TableCell className="pl-6 font-medium text-[13px]">
                      {c.name}
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">
                      {c.email ?? "\u2014"}
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">
                      {c.customerID || "\u2014"}
                    </TableCell>
                    <TableCell>
                      <span
                        className={
                          c.state === "active"
                            ? "inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-400"
                            : "inline-flex items-center rounded-full bg-zinc-500/10 px-2 py-0.5 text-[11px] font-medium text-zinc-400"
                        }
                      >
                        {c.state}
                      </span>
                    </TableCell>
                    <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                      {new Date(c.createdAt).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-border shadow-sm">
          <CardContent className="flex h-40 items-center justify-center">
            <div className="text-center">
              <p className="text-sm text-muted-foreground">
                No customers found
              </p>
              <p className="text-xs text-muted-foreground/60 mt-1">
                Customers appear here when payments are created with a customer
                email.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Pagination */}
      {data && data.total > PAGE_SIZE ? (
        <div className="flex items-center justify-between">
          <p className="text-[13px] text-muted-foreground tabular-nums">
            Showing {offset + 1}&ndash;
            {Math.min(offset + PAGE_SIZE, data.total)} of {data.total}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={offset + PAGE_SIZE >= data.total}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              Next
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
