"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import { useExternalPlatforms, useCreateExternalPlatform, type ExternalPlatform } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { CreateProjectDialog } from "./create-project-dialog";
import { ApiKeyRevealDialog } from "./api-key-reveal-dialog";

const COLUMNS: DataTableColumn<ExternalPlatform>[] = [
  {
    key: "name", stack: "lead",
    header: "Project",
    cell: (p) => (
      <Link
        href={`/admin/external-platforms/${p.id}`}
        onClick={(e) => e.stopPropagation()}
        className="tap rounded-xs font-medium text-ink hover:text-tide"
      >
        {p.name}
      </Link>
    ),
  },
  {
    key: "website", stack: "meta",
    header: "Website",
    cell: (p) => (p.websiteURL ? <span className="font-mono text-label text-ink-soft">{p.websiteURL}</span> : null),
  },
  { key: "status", stack: "trail", header: "Status", cell: (p) => <StatusBadge status={p.active ? "active" : "inactive"} /> },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (p) => <DateTime value={p.createdAt} format="date" />,
  },
];

export default function ExternalPlatformsPage() {
  const router = useRouter();
  const { data, error, refetch } = useExternalPlatforms();
  const createMutation = useCreateExternalPlatform();
  const [showCreate, setShowCreate] = useState(false);
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  async function handleCreate(input: { name: string; websiteURL?: string }) {
    const result = await createMutation.mutateAsync(input);
    setShowCreate(false);
    if (result.apiKey) setRevealedKey(result.apiKey);
  }

  return (
    <div className="space-y-5">
      <PageHeader title="Projects">
        <Button onClick={() => setShowCreate(true)}>
          <Plus />
          New project
        </Button>
      </PageHeader>

      {error ? (
        <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data ?? []}
          loading={!data}
          getRowId={(p) => p.id}
          onRowClick={(p) => router.push(`/admin/external-platforms/${p.id}`)}
          emptyTitle="No projects yet."
          emptyDescription="Each project gets its own API key and currency limits."
          emptyAction={
            <Button size="sm" onClick={() => setShowCreate(true)}>
              New project
            </Button>
          }
        />
      )}

      <CreateProjectDialog
        open={showCreate}
        onOpenChange={setShowCreate}
        onSubmit={handleCreate}
        isPending={createMutation.isPending}
      />
      <ApiKeyRevealDialog apiKey={revealedKey} onClose={() => setRevealedKey(null)} />
    </div>
  );
}
