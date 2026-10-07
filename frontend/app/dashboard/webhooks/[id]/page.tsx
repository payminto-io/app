"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  useWebhook,
  useWebhookDeliveries,
  useUpdateWebhook,
  useDeleteWebhook,
  type WebhookDelivery,
} from "@/lib/query/hooks/use-webhooks";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { DetailItem, DetailList } from "@/components/detail-list";
import { EventList } from "@/components/event-list";
import { RowActions } from "@/components/row-actions";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

const DELIVERY_COLUMNS: DataTableColumn<WebhookDelivery>[] = [
  { key: "event", stack: "lead", header: "Event", cell: (d) => <span className="font-mono text-label">{d.event}</span> },
  { key: "status", stack: "trail", header: "Status", cell: (d) => <StatusBadge status={d.status} /> },
  {
    key: "code", stack: "meta",
    header: "HTTP",
    align: "right",
    className: "text-ink-soft",
    cell: (d) => (d.statusCode !== undefined ? d.statusCode : null),
  },
  { key: "attempts", stack: "meta", header: "Attempts", align: "right", cell: (d) => d.attempts },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (d) => <DateTime value={d.createdAt} />,
  },
];

function hostOf(url: string): string {
  try {
    return new URL(url).host;
  } catch {
    return url;
  }
}

export default function WebhookDetailPage() {
  const params = useParams<{ id: string }>();
  const router = useRouter();
  const id = Number(params.id);
  const { data, error, refetch } = useWebhook(id || undefined);
  const deliveries = useWebhookDeliveries(id || undefined);
  const update = useUpdateWebhook();
  const remove = useDeleteWebhook();
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (error) return <ErrorState message={error.message} retry={refetch} />;
  if (!data) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-4 w-40" />
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-40 w-full rounded-md" />
      </div>
    );
  }

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
      <PageHeader
        breadcrumbs={[{ label: "Webhooks", href: "/dashboard/webhooks" }, { label: `Endpoint ${data.id}` }]}
        title={
          <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="min-w-0 break-all">{hostOf(data.url)}</span>
            <StatusBadge status={data.active ? "active" : "inactive"} />
          </span>
        }
      >
        <Button variant="outline" onClick={toggleActive} disabled={update.isPending}>
          {data.active ? "Disable" : "Enable"}
        </Button>
        <RowActions label="More actions">
          <DropdownMenuItem variant="destructive" onClick={() => setDeleteOpen(true)}>
            Delete endpoint
          </DropdownMenuItem>
        </RowActions>
      </PageHeader>

      <Card>
        <CardContent>
          <DetailList columns={2}>
            <DetailItem label="Endpoint URL" className="sm:col-span-2">
              <CopyField value={data.url} />
            </DetailItem>
            <DetailItem label="Events">
              <EventList events={data.events} />
            </DetailItem>
            <DetailItem label="Created">
              <DateTime value={data.createdAt} />
            </DetailItem>
          </DetailList>
        </CardContent>
      </Card>

      <section className="space-y-3">
        <h2 className="text-h3 font-semibold text-ink">Deliveries</h2>
        {deliveries.error ? (
          <ErrorState message={deliveries.error.message} retry={deliveries.refetch} />
        ) : (
          <DataTable
            columns={DELIVERY_COLUMNS}
            rows={deliveries.data ?? []}
            loading={!deliveries.data}
            getRowId={(d) => d.id}
            emptyTitle="No deliveries yet."
            emptyDescription="Each event sent to this endpoint is listed here."
          />
        )}
      </section>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete this endpoint?</DialogTitle>
            <DialogDescription>Its delivery history is deleted with it. This cannot be undone.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={onDelete} disabled={remove.isPending}>
              {remove.isPending ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
