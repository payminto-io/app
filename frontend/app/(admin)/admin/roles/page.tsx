"use client";

import { useState } from "react";
import { useRoles, type Role } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/states";
import { RolePermissionsEditor } from "./role-permissions-editor";

export default function RolesPage() {
  const { data: roles, error, refetch } = useRoles();
  const [editingRole, setEditingRole] = useState<Role | null>(null);

  const columns: DataTableColumn<Role>[] = [
    {
      key: "name", stack: "lead",
      header: "Role",
      cell: (r) => (
        <span className="inline-flex items-center gap-2">
          <span className="font-medium">{r.name}</span>
          {r.builtIn ? <Badge variant="secondary">Built in</Badge> : null}
        </span>
      ),
    },
    {
      key: "description", stack: "meta",
      header: "Description",
      className: "text-ink-soft max-w-[420px] truncate",
      cell: (r) => r.description ?? null,
    },
    { key: "permissions", stack: "detail", header: "Permissions", align: "right", cell: (r) => r.permissions.length },
    {
      key: "edit", stack: "action",
      header: <span className="sr-only">Edit</span>,
      align: "right",
      className: "w-0",
      cell: (r) => (
        <Button variant="outline" size="xs" onClick={() => setEditingRole(r)}>
          Edit
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="Roles" />

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={columns}
          rows={roles ?? []}
          loading={!roles}
          getRowId={(r) => r.id}
          emptyTitle="No roles defined."
        />
      )}

      {editingRole ? <RolePermissionsEditor role={editingRole} onClose={() => setEditingRole(null)} /> : null}
    </div>
  );
}
