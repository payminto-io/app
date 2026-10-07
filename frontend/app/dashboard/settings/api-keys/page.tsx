"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { useApiKeys } from "@/lib/query/hooks/use-merchant-misc";
import type { APIKey } from "@/lib/api/api-keys";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { RowActions } from "@/components/row-actions";
import { Button } from "@/components/ui/button";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { SettingsTabs } from "../_components/settings-tabs";
import { CreateKeyDialog, RevealKeyDialog, RevokeConfirmDialog } from "./api-key-dialogs";

export default function ApiKeysPage() {
  const { data, error, refetch } = useApiKeys();
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeId, setRevokeId] = useState<number | null>(null);
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  const columns: DataTableColumn<APIKey>[] = [
    { key: "name", stack: "lead", header: "Name", className: "font-medium", cell: (k) => k.name },
    {
      key: "prefix", stack: "meta",
      header: "Key",
      cell: (k) => <code className="font-mono text-label text-ink-soft">{k.prefix}...</code>,
    },
    { key: "status", stack: "trail", header: "Status", cell: (k) => <StatusBadge status={k.active ? "active" : "inactive"} /> },
    {
      key: "used", stack: "detail",
      header: "Last used",
      className: "text-ink-soft",
      cell: (k) => (k.lastUsedAt ? <DateTime value={k.lastUsedAt} /> : "Never"),
    },
    { key: "created", stack: "detail", header: "Created", className: "text-ink-soft", cell: (k) => <DateTime value={k.createdAt} format="date" /> },
    {
      key: "actions", stack: "action",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      className: "w-0",
      cell: (k) =>
        k.active ? (
          <RowActions label={`Actions for ${k.name}`}>
            <DropdownMenuItem variant="destructive" onClick={() => setRevokeId(k.id)}>
              Revoke
            </DropdownMenuItem>
          </RowActions>
        ) : null,
    },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="Settings">
        <Button onClick={() => setCreateOpen(true)}>
          <Plus />
          Generate key
        </Button>
      </PageHeader>
      <SettingsTabs />

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(k) => k.id}
          emptyTitle="No API keys yet."
          emptyDescription="A key lets your server create payments."
          emptyAction={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              Generate key
            </Button>
          }
        />
      )}

      <CreateKeyDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(rawKey) => {
          setCreateOpen(false);
          setRevealedKey(rawKey);
        }}
      />
      <RevealKeyDialog rawKey={revealedKey} onClose={() => setRevealedKey(null)} />
      <RevokeConfirmDialog id={revokeId} onClose={() => setRevokeId(null)} />
    </div>
  );
}
