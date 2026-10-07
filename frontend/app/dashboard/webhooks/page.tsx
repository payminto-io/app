"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import { useWebhooksList, useCreateWebhook } from "@/lib/query/hooks/use-webhooks";
import { isApiError } from "@/lib/query/hooks/use-auth";
import type { CreatedWebhook, RedactedWebhook } from "@/lib/api/webhooks";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { EventList } from "@/components/event-list";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import { StatusBadge } from "@/components/ui/status-badge";

const AVAILABLE_EVENTS = [
  "payment.pending",
  "payment.filled",
  "payment.confirmed",
  "payment.expired",
  "payment.cancelled",
  "withdrawal.sent",
  "withdrawal.failed",
  "sweep.completed",
];

const COLUMNS: DataTableColumn<RedactedWebhook>[] = [
  {
    key: "url", stack: "lead",
    header: "Endpoint",
    cell: (wh) => (
      <Link
        href={`/dashboard/webhooks/${wh.id}`}
        onClick={(e) => e.stopPropagation()}
        className="tap block max-w-[320px] truncate rounded-xs font-mono text-label text-ink hover:text-tide"
        title={wh.url}
      >
        {wh.url}
      </Link>
    ),
  },
  { key: "events", stack: "meta", header: "Events", cell: (wh) => <EventList events={wh.events} max={2} /> },
  { key: "status", stack: "trail", header: "Status", cell: (wh) => <StatusBadge status={wh.active ? "active" : "inactive"} /> },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (wh) => <DateTime value={wh.createdAt} format="date" />,
  },
];

export default function WebhooksPage() {
  const router = useRouter();
  const { data, error, refetch } = useWebhooksList();
  const [addOpen, setAddOpen] = useState(false);

  return (
    <div className="space-y-5">
      <PageHeader title="Webhooks">
        <Button onClick={() => setAddOpen(true)}>
          <Plus />
          Add endpoint
        </Button>
      </PageHeader>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data ?? []}
          loading={!data}
          getRowId={(wh) => wh.id}
          onRowClick={(wh) => router.push(`/dashboard/webhooks/${wh.id}`)}
          emptyTitle="No endpoints yet."
          emptyDescription="Add one to receive payment events on your server."
          emptyAction={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              Add endpoint
            </Button>
          }
        />
      )}

      <AddWebhookDialog open={addOpen} onOpenChange={setAddOpen} />
    </div>
  );
}

function AddWebhookDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const create = useCreateWebhook();
  const [url, setUrl] = useState("");
  const [events, setEvents] = useState<string[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [created, setCreated] = useState<CreatedWebhook | null>(null);
  const [secretAcknowledged, setSecretAcknowledged] = useState(false);

  function reset() {
    setUrl("");
    setEvents([]);
    setErr(null);
    setCreated(null);
    setSecretAcknowledged(false);
  }

  function handleOpenChange(nextOpen: boolean) {
    if (!nextOpen && created && !secretAcknowledged) return;
    if (!nextOpen) reset();
    onOpenChange(nextOpen);
  }

  function toggleEvent(ev: string) {
    setEvents((prev) =>
      prev.includes(ev) ? prev.filter((e) => e !== ev) : [...prev, ev]
    );
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    if (events.length === 0) {
      setErr("Select at least one event.");
      return;
    }
    try {
      const result = await create.mutateAsync({ url, events });
      setCreated(result);
    } catch (error) {
      setErr(isApiError(error) ? error.message : "The endpoint could not be added.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{created ? "Save the signing secret" : "Add endpoint"}</DialogTitle>
          {created ? <DialogDescription>Shown once. Store it before closing.</DialogDescription> : null}
        </DialogHeader>
        {created ? (
          <div className="space-y-4">
            <CopyField value={created.secret} />
            <label className="flex cursor-pointer items-start gap-2 text-body-sm text-ink">
              <Checkbox
                checked={secretAcknowledged}
                onCheckedChange={(checked) =>
                  setSecretAcknowledged(checked === true)
                }
              />
              <span>I have stored the secret.</span>
            </label>
            <DialogFooter>
              <Button
                type="button"
                disabled={!secretAcknowledged}
                onClick={() => handleOpenChange(false)}
              >
                Done
              </Button>
            </DialogFooter>
          </div>
        ) : (
          <form onSubmit={onSubmit} className="space-y-4">
            <FormField label="Endpoint URL" htmlFor="wh-url">
              <TextInput
                id="wh-url"
                type="url"
                required
                placeholder="https://example.com/webhooks"
                className="font-mono"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </FormField>
            <fieldset className="space-y-2">
              <legend className="mb-1.5 text-label font-medium text-ink">Events</legend>
              <div className="grid grid-cols-1 gap-x-4 gap-y-2.5 sm:grid-cols-2">
                {AVAILABLE_EVENTS.map((ev) => (
                  <label key={ev} className="flex cursor-pointer items-center gap-2">
                    <Checkbox checked={events.includes(ev)} onCheckedChange={() => toggleEvent(ev)} />
                    <span className="font-mono text-label text-ink">{ev}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            {err ? <p role="alert" className="text-body-sm text-bad">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={create.isPending}>
                {create.isPending ? "Adding..." : "Add endpoint"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
