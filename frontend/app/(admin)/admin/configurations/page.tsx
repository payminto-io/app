"use client";

import { useMemo, useState } from "react";
import { Pencil } from "lucide-react";
import { useConfigurations, useSetConfiguration, type Configuration } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormField, TextInput } from "@/components/ui/form-field";
import { ErrorState } from "@/components/ui/states";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function ConfigurationsPage() {
  const { data, error, refetch } = useConfigurations();
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

  const columns: DataTableColumn<Configuration>[] = [
    { key: "key", header: "Key", cell: (r) => <span className="font-mono text-label font-medium">{r.key}</span> },
    {
      key: "value",
      header: "Value",
      cell: (r) => (
        <span className="block max-w-xs truncate font-mono text-label text-ink-soft" title={r.value}>
          {r.value}
        </span>
      ),
    },
    {
      key: "updated",
      header: "Updated",
      className: "text-ink-soft",
      cell: (r) => <DateTime value={r.updatedAt} format="date" />,
    },
    {
      key: "edit",
      header: <span className="sr-only">Edit</span>,
      align: "right",
      className: "w-0",
      cell: (r) => (
        <Button variant="ghost" size="icon-sm" className="-my-1.5" aria-label={`Edit ${r.key}`} onClick={() => openEdit(r)}>
          <Pencil />
        </Button>
      ),
    },
  ];

  const table = (rows: Configuration[]) => (
    <DataTable
      columns={columns}
      rows={rows}
      loading={!data}
      getRowId={(r) => r.key}
      emptyTitle="No configuration values."
    />
  );

  return (
    <div className="space-y-5">
      <PageHeader title="Configuration" />

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : categories.length <= 1 ? (
        table(data ?? [])
      ) : (
        <Tabs defaultValue={categories[0]?.[0] ?? "General"} className="gap-5">
          <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
            <TabsList variant="line">
              {categories.map(([cat]) => (
                <TabsTrigger key={cat} value={cat}>
                  {cat}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
          {categories.map(([cat, configs]) => (
            <TabsContent key={cat} value={cat}>
              {table(configs)}
            </TabsContent>
          ))}
        </Tabs>
      )}

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit value</DialogTitle>
            {editing?.description ? <DialogDescription>{editing.description}</DialogDescription> : null}
          </DialogHeader>
          <FormField label="Key" htmlFor="cfg-key">
            <TextInput id="cfg-key" value={editing?.key ?? ""} readOnly disabled className="font-mono" />
          </FormField>
          <FormField label="Value" htmlFor="cfg-value">
            <TextInput id="cfg-value" value={editValue} onChange={(e) => setEditValue(e.target.value)} className="font-mono" />
          </FormField>
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
