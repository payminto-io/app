"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import {
  usePermissions,
  useSetRolePermissions,
} from "@/lib/query/hooks/use-admin";
import type { Role } from "@/lib/query/hooks/use-admin";

export function RolePermissionsEditor({
  role,
  onClose,
}: {
  role: Role;
  onClose: () => void;
}) {
  const { data: allPermissions, isLoading, error } = usePermissions();
  const setPerms = useSetRolePermissions();
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(role.permissions)
  );

  function toggle(name: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  }

  async function handleSave() {
    await setPerms.mutateAsync({
      id: role.id,
      permissions: Array.from(selected),
    });
    onClose();
  }

  const grouped = (allPermissions ?? []).reduce<
    Record<string, { name: string; description?: string }[]>
  >((acc, p) => {
    const cat = p.category ?? "General";
    if (!acc[cat]) acc[cat] = [];
    acc[cat].push({ name: p.name, description: p.description });
    return acc;
  }, {});

  return (
    <Dialog open onOpenChange={() => onClose()}>
      <DialogContent className="max-h-[80vh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Permissions for {role.name}</DialogTitle>
          <DialogDescription>
            Toggle permissions for this role. Changes take effect immediately
            on save.
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-3 w-20" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
              </div>
            ))}
          </div>
        ) : error ? (
          <ErrorState
            message={error instanceof Error ? error.message : "Failed to load"}
          />
        ) : (
          <div className="space-y-5">
            {Object.entries(grouped).map(([category, perms]) => (
              <div key={category}>
                <h4 className="pm-label mb-2.5 text-muted-foreground uppercase tracking-wider">
                  {category}
                </h4>
                <div className="rounded-lg border border-border divide-y divide-border">
                  {perms.map((p) => (
                    <label
                      key={p.name}
                      className="flex items-center gap-3 px-3 py-2.5 text-sm hover:bg-muted/50 cursor-pointer transition-colors"
                    >
                      <Checkbox
                        checked={selected.has(p.name)}
                        onCheckedChange={() => toggle(p.name)}
                      />
                      <div className="min-w-0">
                        <span className="text-[13px] font-medium">{p.name}</span>
                        {p.description ? (
                          <span className="ml-2 text-[11px] text-muted-foreground">
                            {p.description}
                          </span>
                        ) : null}
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            ))}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={setPerms.isPending}>
            {setPerms.isPending ? "Saving..." : "Save Permissions"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
