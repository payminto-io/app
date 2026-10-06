"use client";

import { useState } from "react";
import { use } from "react";
import { KeyRound, RotateCcw } from "lucide-react";
import { PageHeader } from "@/components/ui/page-header";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  useExternalPlatform,
  useRegenerateEPAPIKey,
} from "@/lib/query/hooks/use-admin";
import { ApiKeyRevealDialog } from "../api-key-reveal-dialog";
import { CurrenciesTable } from "./currencies-table";

export const dynamic = "force-dynamic";

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Skeleton className="size-10 rounded-lg" />
        <div className="space-y-2">
          <Skeleton className="h-6 w-48" />
          <Skeleton className="h-4 w-32" />
        </div>
      </div>
      <Card>
        <CardContent className="pt-6">
          <div className="grid grid-cols-4 gap-6">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="space-y-2">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-5 w-24" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

export default function ExternalPlatformDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id: rawId } = use(params);
  const id = Number(rawId);
  const { data: platform, isLoading, error, refetch } = useExternalPlatform(id);
  const regenerate = useRegenerateEPAPIKey();
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  async function handleRegenerate() {
    if (!confirm("Regenerate API key? The old key will stop working immediately.")) {
      return;
    }
    const result = await regenerate.mutateAsync(id);
    if (result.apiKey) {
      setRevealedKey(result.apiKey);
    }
  }

  if (isLoading) return <DetailSkeleton />;
  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }
  if (!platform) return null;

  return (
    <div className="space-y-6">
      <PageHeader
        title={platform.name}
        description={platform.websiteURL ?? undefined}
        breadcrumbs={[
          { label: "Projects", href: "/admin/external-platforms" },
          { label: platform.name },
        ]}
        actions={
          <Button
            variant="outline"
            onClick={handleRegenerate}
            disabled={regenerate.isPending}
          >
            <RotateCcw className="size-4" />
            {regenerate.isPending ? "Regenerating..." : "Regenerate API Key"}
          </Button>
        }
      />

      <Tabs defaultValue="info">
        <TabsList variant="line">
          <TabsTrigger value="info">Info</TabsTrigger>
          <TabsTrigger value="currencies">Currencies</TabsTrigger>
          <TabsTrigger value="api-keys">
            <KeyRound className="size-3.5" />
            API Keys
          </TabsTrigger>
        </TabsList>

        <TabsContent value="info">
          <Card className="mt-4">
            <CardContent className="pt-6">
              <dl className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm sm:grid-cols-4">
                <div>
                  <dt className="pm-label text-muted-foreground">Status</dt>
                  <dd className="mt-1.5">
                    <StatusBadge status={platform.active ? "active" : "inactive"} />
                  </dd>
                </div>
                <div>
                  <dt className="pm-label text-muted-foreground">Website</dt>
                  <dd className="mt-1.5 text-[13px]">
                    {platform.websiteURL ?? "--"}
                  </dd>
                </div>
                <div>
                  <dt className="pm-label text-muted-foreground">Created</dt>
                  <dd className="mt-1.5 text-[13px] tabular-nums">
                    {new Date(platform.createdAt).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </dd>
                </div>
                <div>
                  <dt className="pm-label text-muted-foreground">Updated</dt>
                  <dd className="mt-1.5 text-[13px] tabular-nums">
                    {new Date(platform.updatedAt).toLocaleDateString("en-US", {
                      month: "short",
                      day: "numeric",
                      year: "numeric",
                    })}
                  </dd>
                </div>
              </dl>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="currencies">
          <div className="mt-4">
            <CurrenciesTable platformId={id} />
          </div>
        </TabsContent>

        <TabsContent value="api-keys">
          <Card className="mt-4">
            <CardContent className="pt-6 space-y-4">
              <p className="text-[13px] text-muted-foreground">
                API keys are shown only once when created or regenerated. If you
                lose your key, regenerate a new one.
              </p>
              <Button
                variant="outline"
                onClick={handleRegenerate}
                disabled={regenerate.isPending}
              >
                <KeyRound className="size-4" />
                {regenerate.isPending ? "Regenerating..." : "Regenerate API Key"}
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <ApiKeyRevealDialog
        apiKey={revealedKey}
        onClose={() => setRevealedKey(null)}
      />
    </div>
  );
}
