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
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Permissions for {role.name}</DialogTitle>
          <DialogDescription>
            <span className="num">{selected.size}</span> selected. Changes apply on save.
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
                <h4 className="mb-2 text-label font-medium text-ink-soft">{category}</h4>
                <div className="divide-y divide-line rounded-sm border border-line">
                  {perms.map((p) => (
                    <label
                      key={p.name}
                      className="flex cursor-pointer items-start gap-3 px-3 py-2.5 transition-colors duration-120 hover:bg-surface-sunken/60"
                    >
                      <Checkbox
                        checked={selected.has(p.name)}
                        onCheckedChange={() => toggle(p.name)}
                      />
                      <div className="min-w-0">
                        <div className="font-mono text-label text-ink">{p.name}</div>
                        {p.description ? (
                          <div className="mt-0.5 text-caption text-ink-soft">{p.description}</div>
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
            {setPerms.isPending ? "Saving..." : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
