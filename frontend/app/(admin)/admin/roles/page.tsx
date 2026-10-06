"use client";

import { useState } from "react";
import { Shield, Pencil } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { useRoles } from "@/lib/query/hooks/use-admin";
import type { Role } from "@/lib/query/hooks/use-admin";
import { RolePermissionsEditor } from "./role-permissions-editor";

export default function RolesPage() {
  const { data: roles, isLoading, error, refetch } = useRoles();
  const [editingRole, setEditingRole] = useState<Role | null>(null);

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
        title="Roles & Permissions"
        description="Define granular access controls for your team."
        icon={<Shield className="size-5" />}
      />

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-5 w-24" />
                <Skeleton className="mt-2 h-3 w-40" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-4 w-32" />
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {(roles ?? []).map((role) => (
            <Card
              key={role.id}
              className="group relative transition-colors hover:border-primary/30"
            >
              <CardHeader className="pb-3">
                <div className="flex items-center gap-2.5">
                  <div className="flex size-8 items-center justify-center rounded-md bg-primary/10 text-primary">
                    <Shield className="size-4" />
                  </div>
                  <div>
                    <CardTitle className="text-[14px]">{role.name}</CardTitle>
                  </div>
                </div>
                {role.description ? (
                  <CardDescription className="mt-2 text-[12px]">
                    {role.description}
                  </CardDescription>
                ) : null}
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <Badge variant="secondary" className="text-[11px]">
                    {role.permissions.length} permission
                    {role.permissions.length !== 1 ? "s" : ""}
                  </Badge>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setEditingRole(role)}
                  >
                    <Pencil className="size-3" />
                    Edit
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {editingRole ? (
        <RolePermissionsEditor
          role={editingRole}
          onClose={() => setEditingRole(null)}
        />
      ) : null}
    </div>
  );
}
