"use client";

import { useState } from "react";
import { Key, Plus } from "lucide-react";
import { useApiKeys } from "@/lib/query/hooks/use-merchant-misc";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  CreateKeyDialog,
  RevealKeyDialog,
  RevokeConfirmDialog,
} from "./api-key-dialogs";

export default function ApiKeysPage() {
  const { data, isLoading, error, refetch } = useApiKeys();
  const [createOpen, setCreateOpen] = useState(false);
  const [revokeId, setRevokeId] = useState<number | null>(null);
  const [revealedKey, setRevealedKey] = useState<string | null>(null);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">API Keys</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Manage authentication keys for your integrations
          </p>
        </div>
        <Button
          onClick={() => setCreateOpen(true)}
          className="h-9 rounded-lg bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
        >
          <Plus className="size-4" />
          Generate Key
        </Button>
      </div>

      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-lg" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">Name</TableHead>
                  <TableHead className="pm-label">Prefix</TableHead>
                  <TableHead className="pm-label">Status</TableHead>
                  <TableHead className="pm-label">Last Used</TableHead>
                  <TableHead className="pm-label">Created</TableHead>
                  <TableHead className="pm-label text-right pr-6">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((k) => (
                  <TableRow key={k.id} className="border-border/60 hover:bg-muted/30">
                    <TableCell className="pl-6">
                      <div className="flex items-center gap-2">
                        <Key className="size-3.5 text-muted-foreground" />
                        <span className="font-medium text-[13px]">{k.name}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <code className="rounded-md border border-border bg-muted/50 px-2 py-0.5 font-mono text-[11px]">
                        {k.prefix}...
                      </code>
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={k.active ? "active" : "inactive"} />
                    </TableCell>
                    <TableCell className="text-[12px] text-muted-foreground">
                      {k.lastUsedAt ? new Date(k.lastUsedAt).toLocaleDateString() : "Never"}
                    </TableCell>
                    <TableCell className="text-[12px] text-muted-foreground">
                      {new Date(k.createdAt).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right pr-6">
                      {k.active ? (
                        <Button
                          variant="destructive"
                          size="sm"
                          className="h-7 text-[11px]"
                          onClick={(e) => {
                            e.stopPropagation();
                            setRevokeId(k.id);
                          }}
                        >
                          Revoke
                        </Button>
                      ) : (
                        <span className="text-[11px] text-muted-foreground">Revoked</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-border shadow-sm">
          <CardContent className="flex h-40 items-center justify-center">
            <div className="text-center">
              <Key className="mx-auto size-8 text-muted-foreground/40 mb-2" />
              <p className="text-sm text-muted-foreground">No API keys yet</p>
              <p className="text-xs text-muted-foreground/60 mt-1">
                Generate your first key to start integrating.
              </p>
            </div>
          </CardContent>
        </Card>
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
