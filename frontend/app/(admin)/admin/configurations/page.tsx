"use client";

import { useState, useMemo } from "react";
import { Settings2, Pencil } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  useConfigurations,
  useSetConfiguration,
} from "@/lib/query/hooks/use-admin";
import type { Configuration } from "@/lib/query/hooks/use-admin";

export default function ConfigurationsPage() {
  const { data, isLoading, error, refetch } = useConfigurations();
  const setConfig = useSetConfiguration();
  const [editing, setEditing] = useState<Configuration | null>(null);
  const [editValue, setEditValue] = useState("");

  function openEdit(config: Configuration) {
    setEditing(config);
    setEditValue(config.value);
  }

  async function handleSave() {
    if (!editing) return;
    await setConfig.mutateAsync({ key: editing.key, value: editValue });
    setEditing(null);
  }

  const categories = useMemo(() => {
    if (!data) return [];
    const grouped = new Map<string, Configuration[]>();
    for (const c of data) {
      const cat = c.category ?? "General";
      if (!grouped.has(cat)) grouped.set(cat, []);
      grouped.get(cat)!.push(c);
    }
    return Array.from(grouped.entries()).sort(([a], [b]) => a.localeCompare(b));
  }, [data]);

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
        title="Runtime Configuration"
        description="View and edit system configuration values"
        icon={<Settings2 className="size-5" />}
      />

      {isLoading ? (
        <Card className="p-0">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
              <Skeleton className="h-4 w-40" />
              <Skeleton className="h-4 w-48" />
              <Skeleton className="ml-auto h-4 w-16" />
            </div>
          ))}
        </Card>
      ) : (data ?? []).length === 0 ? (
        <EmptyState
          title="No configurations"
          description="No runtime configuration values found."
          icon={<Settings2 className="size-5" />}
        />
      ) : categories.length <= 1 ? (
        <ConfigTable configs={data ?? []} onEdit={openEdit} />
      ) : (
        <Tabs defaultValue={categories[0]?.[0] ?? "General"}>
          <TabsList variant="line">
            {categories.map(([cat]) => (
              <TabsTrigger key={cat} value={cat}>
                {cat}
              </TabsTrigger>
            ))}
          </TabsList>
          {categories.map(([cat, configs]) => (
            <TabsContent key={cat} value={cat}>
              <div className="mt-4">
                <ConfigTable configs={configs} onEdit={openEdit} />
              </div>
            </TabsContent>
          ))}
        </Tabs>
      )}

      <Dialog
        open={editing !== null}
        onOpenChange={(open) => {
          if (!open) setEditing(null);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Configuration</DialogTitle>
            {editing?.description ? (
              <DialogDescription>{editing.description}</DialogDescription>
            ) : null}
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="cfg-key" className="text-[12px]">Key</Label>
            <Input
              id="cfg-key"
              value={editing?.key ?? ""}
              readOnly
              className="font-mono text-[12px] bg-muted/50"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="cfg-value" className="text-[12px]">Value</Label>
            <Input
              id="cfg-value"
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
            />
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setEditing(null)}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={setConfig.isPending}>
              {setConfig.isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ConfigTable({
  configs,
  onEdit,
}: {
  configs: Configuration[];
  onEdit: (c: Configuration) => void;
}) {
  return (
    <Card className="p-0">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="pm-label">Key</TableHead>
            <TableHead className="pm-label">Value</TableHead>
            <TableHead className="pm-label">Updated</TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {configs.map((r) => (
            <TableRow key={r.key}>
              <TableCell className="font-mono text-[12px] font-medium">
                {r.key}
              </TableCell>
              <TableCell>
                <span className="max-w-xs truncate block text-[13px] text-muted-foreground">
                  {r.value}
                </span>
              </TableCell>
              <TableCell className="text-muted-foreground text-sm tabular-nums">
                {new Date(r.updatedAt).toLocaleDateString("en-US", {
                  month: "short",
                  day: "numeric",
                  year: "numeric",
                })}
              </TableCell>
              <TableCell>
                <Button variant="ghost" size="sm" onClick={() => onEdit(r)}>
                  <Pencil className="size-3.5" />
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  );
}
