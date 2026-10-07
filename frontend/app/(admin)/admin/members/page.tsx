"use client";

import { useState } from "react";
import { UserPlus } from "lucide-react";
import { useAdminMembers, useDeactivateMember, type Member } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { RowActions } from "@/components/row-actions";
import { Button } from "@/components/ui/button";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { InviteMemberDialog } from "./invite-member-dialog";

export default function MembersPage() {
  const { data, error, refetch } = useAdminMembers();
  const deactivate = useDeactivateMember();
  const [showInvite, setShowInvite] = useState(false);

  async function handleDeactivate(member: Member) {
    if (!confirm(`Deactivate ${member.name} (${member.email})? They lose access at once.`)) {
      return;
    }
    await deactivate.mutateAsync(member.id);
  }

  const columns: DataTableColumn<Member>[] = [
    { key: "name", header: "Name", className: "font-medium", cell: (m) => m.name },
    { key: "email", header: "Email", className: "text-ink-soft", cell: (m) => m.email },
    {
      key: "role",
      header: "Role",
      cell: (m) => (m.memberType ? <span className="capitalize">{m.memberType}</span> : null),
    },
    { key: "status", header: "Status", cell: (m) => <StatusBadge status={m.active ? "active" : "inactive"} /> },
    {
      key: "joined",
      header: "Joined",
      align: "right",
      className: "text-ink-soft",
      cell: (m) => <DateTime value={m.createdAt} format="date" />,
    },
    {
      key: "actions",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      className: "w-0",
      cell: (m) =>
        m.active ? (
          <RowActions label={`Actions for ${m.name}`}>
            <DropdownMenuItem variant="destructive" onClick={() => handleDeactivate(m)}>
              Deactivate
            </DropdownMenuItem>
          </RowActions>
        ) : null,
    },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="Members">
        <Button onClick={() => setShowInvite(true)}>
          <UserPlus />
          Invite member
        </Button>
      </PageHeader>

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(m) => m.id}
          emptyTitle="No members yet."
          emptyAction={
            <Button size="sm" onClick={() => setShowInvite(true)}>
              Invite member
            </Button>
          }
        />
      )}

      <InviteMemberDialog open={showInvite} onOpenChange={setShowInvite} />
    </div>
  );
}
