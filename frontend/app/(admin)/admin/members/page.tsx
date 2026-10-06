"use client";

import { useState } from "react";
import { Users, UserPlus, MoreHorizontal, Ban } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  useAdminMembers,
  useDeactivateMember,
} from "@/lib/query/hooks/use-admin";
import type { Member } from "@/lib/query/hooks/use-admin";
import { InviteMemberDialog } from "./invite-member-dialog";

const TYPE_COLORS: Record<string, string> = {
  admin: "bg-purple-500/10 text-purple-400 border-purple-500/20",
  root: "bg-blue-500/10 text-blue-400 border-blue-500/20",
  operator: "bg-emerald-500/10 text-emerald-400 border-emerald-500/20",
};

function formatDate(d: string) {
  return new Date(d).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function getInitials(name: string) {
  return name
    .split(" ")
    .map((w) => w[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

export default function MembersPage() {
  const { data, isLoading, error, refetch } = useAdminMembers();
  const deactivate = useDeactivateMember();
  const [showInvite, setShowInvite] = useState(false);

  async function handleDeactivate(member: Member) {
    if (
      !confirm(
        `Deactivate ${member.name} (${member.email})? They will lose access.`
      )
    ) {
      return;
    }
    await deactivate.mutateAsync(member.id);
  }

  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Members"
        description="Invite and manage team members"
        icon={<Users className="size-5" />}
        actions={
          <Button onClick={() => setShowInvite(true)}>
            <UserPlus className="size-4" />
            Invite Member
          </Button>
        }
      />

      {isLoading ? (
        <Card className="p-0">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
              <Skeleton className="size-8 rounded-full" />
              <div className="space-y-1.5">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-40" />
              </div>
              <Skeleton className="ml-auto h-5 w-16 rounded-full" />
            </div>
          ))}
        </Card>
      ) : (data ?? []).length === 0 ? (
        <EmptyState
          title="No members"
          description="Invite your first team member."
          icon={<Users className="size-5" />}
          action={
            <Button onClick={() => setShowInvite(true)}>
              <UserPlus className="size-4" />
              Invite
            </Button>
          }
        />
      ) : (
        <Card className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pm-label">User</TableHead>
                <TableHead className="pm-label">Role</TableHead>
                <TableHead className="pm-label">Status</TableHead>
                <TableHead className="pm-label">Joined</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data ?? []).map((member: Member) => (
                <TableRow
                  key={member.id}
                  className={!member.active ? "opacity-50" : ""}
                >
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <Avatar className="size-8">
                        <AvatarFallback className="text-xs">
                          {getInitials(member.name)}
                        </AvatarFallback>
                      </Avatar>
                      <div>
                        <div className="text-[13px] font-medium text-foreground">
                          {member.name}
                        </div>
                        <div className="text-[11px] text-muted-foreground">
                          {member.email}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    {member.memberType ? (
                      <span
                        className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${
                          TYPE_COLORS[member.memberType] ??
                          "bg-muted text-muted-foreground border-border"
                        }`}
                      >
                        {member.memberType}
                      </span>
                    ) : (
                      <span className="text-[12px] text-muted-foreground">--</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={member.active ? "active" : "inactive"} />
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm tabular-nums">
                    {formatDate(member.createdAt)}
                  </TableCell>
                  <TableCell>
                    {member.active ? (
                      <DropdownMenu>
                        <DropdownMenuTrigger
                          render={
                            <Button variant="ghost" size="sm">
                              <MoreHorizontal className="size-4" />
                            </Button>
                          }
                        />
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => handleDeactivate(member)}
                          >
                            <Ban className="size-3.5" />
                            Deactivate
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    ) : null}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <InviteMemberDialog
        open={showInvite}
        onOpenChange={setShowInvite}
      />
    </div>
  );
}
