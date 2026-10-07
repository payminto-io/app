"use client";

import { useState, use } from "react";
import { RotateCcw } from "lucide-react";
import { useExternalPlatform, useRegenerateEPAPIKey } from "@/lib/query/hooks/use-admin";
import { PageHeader } from "@/components/page-header";
import { DateTime } from "@/components/date-time";
import { DetailItem, DetailList } from "@/components/detail-list";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { ApiKeyRevealDialog } from "../api-key-reveal-dialog";
import { CurrenciesTable } from "./currencies-table";

export const dynamic = "force-dynamic";

export default function ExternalPlatformDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: rawId } = use(params);
  const id = Number(rawId);
  const { data: platform, error, refetch } = useExternalPlatform(id);
  const regenerate = useRegenerateEPAPIKey();
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  async function handleRegenerate() {
    if (!confirm("Regenerate the API key? The current key stops working at once.")) return;
    const result = await regenerate.mutateAsync(id);
    if (result.apiKey) setRevealedKey(result.apiKey);
  }

  if (error) {
    return <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />;
  }
  if (!platform) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-8 w-56" />
        <Skeleton className="h-24 w-full rounded-md" />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <PageHeader
        breadcrumbs={[{ label: "Projects", href: "/admin/external-platforms" }, { label: platform.name }]}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            {platform.name}
            <StatusBadge status={platform.active ? "active" : "inactive"} />
          </span>
        }
      >
        <Button variant="outline" onClick={handleRegenerate} disabled={regenerate.isPending}>
          <RotateCcw />
          {regenerate.isPending ? "Regenerating..." : "Regenerate API key"}
        </Button>
      </PageHeader>

      <Card>
        <CardContent>
          <DetailList className="grid-cols-1 sm:grid-cols-3">
            <DetailItem label="Website">
              {platform.websiteURL ? <span className="font-mono text-body-sm break-all">{platform.websiteURL}</span> : null}
            </DetailItem>
            <DetailItem label="Created">
              <DateTime value={platform.createdAt} format="date" />
            </DetailItem>
            <DetailItem label="Updated">
              <DateTime value={platform.updatedAt} format="date" />
            </DetailItem>
          </DetailList>
        </CardContent>
      </Card>

      <section className="space-y-3">
        <h2 className="text-h3 font-semibold text-ink">Currencies</h2>
        <CurrenciesTable platformId={id} />
      </section>

      <ApiKeyRevealDialog apiKey={revealedKey} onClose={() => setRevealedKey(null)} />
    </div>
  );
}
