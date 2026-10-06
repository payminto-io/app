"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Building2, ChevronRight, Plus } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
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
  useExternalPlatforms,
  useCreateExternalPlatform,
} from "@/lib/query/hooks/use-admin";
import type { ExternalPlatform } from "@/lib/query/hooks/use-admin";
import { CreateProjectDialog } from "./create-project-dialog";
import { ApiKeyRevealDialog } from "./api-key-reveal-dialog";

function formatDate(d: string) {
  return new Date(d).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function ExternalPlatformsPage() {
  const router = useRouter();
  const { data, isLoading, error, refetch } = useExternalPlatforms();
  const createMutation = useCreateExternalPlatform();

  const [showCreate, setShowCreate] = useState(false);
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  async function handleCreate(input: { name: string; websiteURL?: string }) {
    const result = await createMutation.mutateAsync(input);
    setShowCreate(false);
    if (result.apiKey) {
      setRevealedKey(result.apiKey);
    }
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
        title="Projects"
        description="Manage external platform integrations"
        icon={<Building2 className="size-5" />}
        actions={
          <Button onClick={() => setShowCreate(true)}>
            <Plus className="size-4" />
            New Project
          </Button>
        }
      />

      {isLoading ? (
        <Card className="p-0">
          <div className="space-y-0">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
                <Skeleton className="size-8 rounded-md" />
                <Skeleton className="h-4 w-32" />
                <Skeleton className="ml-auto h-4 w-20" />
              </div>
            ))}
          </div>
        </Card>
      ) : (data ?? []).length === 0 ? (
        <EmptyState
          title="No projects yet"
          description="Create your first project to start accepting payments."
          icon={<Building2 className="size-5" />}
          action={
            <Button onClick={() => setShowCreate(true)}>
              <Plus className="size-4" />
              New Project
            </Button>
          }
        />
      ) : (
        <Card className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="pm-label">Project</TableHead>
                <TableHead className="pm-label">Website</TableHead>
                <TableHead className="pm-label">Status</TableHead>
                <TableHead className="pm-label">Created</TableHead>
                <TableHead className="w-12" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {(data ?? []).map((p: ExternalPlatform) => (
                <TableRow
                  key={p.id}
                  className="cursor-pointer"
                  onClick={() => router.push(`/admin/external-platforms/${p.id}`)}
                >
                  <TableCell>
                    <div className="flex items-center gap-2.5">
                      <div
                        className="size-8 rounded-md flex items-center justify-center text-[11px] font-bold text-white"
                        style={{ backgroundColor: "var(--pm-primary)" }}
                      >
                        {p.name.slice(0, 2).toUpperCase()}
                      </div>
                      <div className="text-[13px] font-medium text-foreground">
                        {p.name}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell className="text-[12px] text-muted-foreground font-mono">
                    {p.websiteURL ?? "--"}
                  </TableCell>
                  <TableCell>
                    <StatusBadge status={p.active ? "active" : "inactive"} />
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {formatDate(p.createdAt)}
                  </TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-1 text-[12px] font-medium text-primary hover:underline">
                      Manage
                      <ChevronRight className="size-3.5" />
                    </span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}

      <CreateProjectDialog
        open={showCreate}
        onOpenChange={setShowCreate}
        onSubmit={handleCreate}
        isPending={createMutation.isPending}
      />

      <ApiKeyRevealDialog
        apiKey={revealedKey}
        onClose={() => setRevealedKey(null)}
      />
    </div>
  );
}
