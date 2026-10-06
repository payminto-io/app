"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ChevronRight, Globe, Trash2 } from "lucide-react";
import {
  useWebhook,
  useWebhookDeliveries,
  useUpdateWebhook,
  useDeleteWebhook,
} from "@/lib/query/hooks/use-webhooks";
import { ErrorState } from "@/components/ui/states";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { CopyButton } from "@/components/copy-button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
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
  DialogFooter,
} from "@/components/ui/dialog";
import type { WebhookDelivery } from "@/lib/query/hooks/use-webhooks";

export default function WebhookDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = Number(params.id);
  const { data, isLoading, error, refetch } = useWebhook(id || undefined);
  const deliveries = useWebhookDeliveries(id || undefined);
  const update = useUpdateWebhook();
  const remove = useDeleteWebhook();
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-6 w-64" />
        <Skeleton className="h-60 rounded-xl" />
      </div>
    );
  }
  if (error) return <ErrorState message={error.message} retry={refetch} />;
  if (!data) return <ErrorState message="Webhook not found" />;

  async function toggleActive() {
    if (!data) return;
    await update.mutateAsync({ id: data.id, input: { active: !data.active } });
    refetch();
  }

  async function onDelete() {
    if (!data) return;
    await remove.mutateAsync(data.id);
    router.push("/dashboard/webhooks");
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1.5 text-[13px] text-muted-foreground">
        <Link href="/dashboard" className="hover:text-foreground transition-colors">Dashboard</Link>
        <ChevronRight className="size-3.5" />
        <Link href="/dashboard/webhooks" className="hover:text-foreground transition-colors">Webhooks</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground font-medium">#{data.id}</span>
      </nav>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <Globe className="size-5 text-muted-foreground" />
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight">Webhook #{data.id}</h1>
            <p className="text-[12px] font-mono text-muted-foreground truncate max-w-xs">
              {data.url}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={toggleActive}
            disabled={update.isPending}
          >
            {data.active ? "Deactivate" : "Activate"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="size-3.5" />
            Delete
          </Button>
        </div>
      </div>

      {/* Tabs */}
      <Tabs defaultValue="overview">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="deliveries">Deliveries</TabsTrigger>
          <TabsTrigger value="events">Events</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-4">
          <Card className="border-border shadow-sm">
            <CardContent className="p-6">
              <div className="grid gap-5 sm:grid-cols-2">
                <DetailRow label="URL">
                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[12px] break-all">{data.url}</span>
                    <CopyButton value={data.url} size="sm" variant="ghost" label="" />
                  </div>
                </DetailRow>
                <DetailRow label="Status">
                  <StatusBadge status={data.active ? "active" : "inactive"} />
                </DetailRow>
                <DetailRow
                  label="Created"
                  value={new Date(data.createdAt).toLocaleString()}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="deliveries" className="mt-4">
          <DeliveriesTab deliveries={deliveries} />
        </TabsContent>

        <TabsContent value="events" className="mt-4">
          <Card className="border-border shadow-sm">
            <CardHeader>
              <CardTitle className="text-[15px] font-semibold">Subscribed Events</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap gap-2">
                {data.events.map((ev) => (
                  <span
                    key={ev}
                    className="inline-flex rounded-md border border-border bg-muted/50 px-3 py-1.5 text-[12px] font-medium text-muted-foreground"
                  >
                    {ev}
                  </span>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Delete dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Webhook</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            This will permanently delete the webhook and all delivery history.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>Cancel</Button>
            <Button variant="destructive" onClick={onDelete} disabled={remove.isPending}>
              {remove.isPending ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function DetailRow({
  label,
  value,
  children,
}: {
  label: string;
  value?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <span className="pm-label">{label}</span>
      {value ? <p className="text-[13px]">{value}</p> : children}
    </div>
  );
}

function DeliveriesTab({
  deliveries,
}: {
  deliveries: {
    data: WebhookDelivery[] | undefined;
    isLoading: boolean;
    error: Error | null;
    refetch: () => void;
  };
}) {
  if (deliveries.isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-12 w-full rounded-lg" />
        ))}
      </div>
    );
  }

  if (deliveries.error) {
    return <ErrorState message={deliveries.error.message} retry={deliveries.refetch} />;
  }

  if (!deliveries.data || deliveries.data.length === 0) {
    return (
      <Card className="border-border shadow-sm">
        <CardContent className="flex h-32 items-center justify-center">
          <p className="text-sm text-muted-foreground">No deliveries yet</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border-border shadow-sm overflow-hidden">
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow className="border-border hover:bg-transparent">
              <TableHead className="pm-label pl-6">Event</TableHead>
              <TableHead className="pm-label">Status</TableHead>
              <TableHead className="pm-label">Attempts</TableHead>
              <TableHead className="pm-label">HTTP Code</TableHead>
              <TableHead className="pm-label text-right pr-6">Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {deliveries.data.map((d) => (
              <TableRow key={d.id} className="border-border/60 hover:bg-muted/30">
                <TableCell className="pl-6 text-[12px] font-mono">{d.event}</TableCell>
                <TableCell><StatusBadge status={d.status} /></TableCell>
                <TableCell className="text-[13px] tabular-nums">{d.attempts}</TableCell>
                <TableCell className="text-[13px] tabular-nums text-muted-foreground">
                  {d.statusCode ?? "\u2014"}
                </TableCell>
                <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                  {new Date(d.createdAt).toLocaleString()}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
