"use client";

import { useState } from "react";
import { useCustomersList, type Customer } from "@/lib/query/hooks/use-customers";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Pagination } from "@/components/pagination";
import { SearchInput } from "@/components/search-input";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

const PAGE_SIZE = 25;

const COLUMNS: DataTableColumn<Customer>[] = [
  { key: "name", stack: "lead", header: "Name", className: "font-medium", cell: (c) => c.name },
  { key: "email", stack: "meta", header: "Email", className: "text-ink-soft", cell: (c) => c.email ?? null },
  {
    key: "id", stack: "meta",
    header: "Customer ID",
    cell: (c) => (c.customerID ? <span className="font-mono text-label text-ink-soft">{c.customerID}</span> : null),
  },
  { key: "state", stack: "trail", header: "Status", cell: (c) => <StatusBadge status={c.state} /> },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (c) => <DateTime value={c.createdAt} format="date" />,
  },
];

export default function CustomersPage() {
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const { data, error, refetch } = useCustomersList({ limit: PAGE_SIZE, offset });

  const q = search.trim().toLowerCase();
  const searching = q.length > 0;
  const rows =
    data?.customers.filter(
      (c) =>
        !q ||
        c.name.toLowerCase().includes(q) ||
        c.email?.toLowerCase().includes(q) ||
        c.customerID?.toLowerCase().includes(q)
    ) ?? [];

  return (
    <div className="space-y-5">
      <PageHeader title="Customers" />
      <SearchInput
        aria-label="Search this page by name, email or customer ID"
        placeholder="Name, email or ID"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />
      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={rows}
          loading={!data}
          getRowId={(c) => c.id}
          total={searching ? undefined : data?.total}
          footer={
            data && !searching && data.total > PAGE_SIZE ? (
              <Pagination offset={offset} limit={PAGE_SIZE} total={data.total} onOffsetChange={setOffset} />
            ) : undefined
          }
          emptyTitle={searching ? "No customers on this page match." : "No customers yet."}
          emptyDescription={searching ? undefined : "A customer is added the first time a payment names their email."}
        />
      )}
    </div>
  );
}
