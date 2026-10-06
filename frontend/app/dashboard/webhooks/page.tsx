"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Globe, Plus } from "lucide-react";
import {
  useWebhooksList,
  useCreateWebhook,
} from "@/lib/query/hooks/use-webhooks";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { Checkbox } from "@/components/ui/checkbox";
import { FormField, TextInput } from "@/components/ui/form-field";
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
import { isApiError } from "@/lib/query/hooks/use-auth";
import { CopyButton } from "@/components/copy-button";
import type { CreatedWebhook } from "@/lib/api/webhooks";

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

export default function WebhooksPage() {
  const router = useRouter();
  const { data, isLoading, error, refetch } = useWebhooksList();
  const [addOpen, setAddOpen] = useState(false);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Webhooks</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Receive real-time event notifications via HTTP callbacks
          </p>
        </div>
        <Button
          onClick={() => setAddOpen(true)}
          className="h-9 rounded-lg bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
        >
          <Plus className="size-4" />
          Add Webhook
        </Button>
      </div>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full rounded-lg" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">URL</TableHead>
                  <TableHead className="pm-label">Events</TableHead>
                  <TableHead className="pm-label">Status</TableHead>
                  <TableHead className="pm-label text-right pr-6">Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((wh) => (
                  <TableRow
                    key={wh.id}
                    className="border-border/60 hover:bg-muted/30 cursor-pointer"
                    onClick={() => router.push(`/dashboard/webhooks/${wh.id}`)}
                  >
                    <TableCell className="pl-6">
                      <div className="flex items-center gap-2">
                        <Globe className="size-4 text-muted-foreground shrink-0" />
                        <span className="font-mono text-[12px] truncate max-w-[300px]">
                          {wh.url}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1">
                        {wh.events.slice(0, 3).map((ev) => (
                          <span
                            key={ev}
                            className="inline-flex rounded-md border border-border bg-muted/50 px-2 py-0.5 text-[10px] text-muted-foreground"
                          >
                            {ev}
                          </span>
                        ))}
                        {wh.events.length > 3 && (
                          <span className="text-[10px] text-muted-foreground">
                            +{wh.events.length - 3} more
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={wh.active ? "active" : "inactive"} />
                    </TableCell>
                    <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                      {new Date(wh.createdAt).toLocaleDateString()}
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
              <p className="text-sm text-muted-foreground">No webhooks yet</p>
              <p className="text-xs text-muted-foreground/60 mt-1">
                Add a webhook to receive event notifications.
              </p>
            </div>
          </CardContent>
        </Card>
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
      setErr(isApiError(error) ? error.message : "Failed to create webhook.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{created ? "Save Signing Secret" : "Add Webhook"}</DialogTitle>
        </DialogHeader>
        {created ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground" role="status">
              This signing secret is shown once. Copy it now and store it securely.
            </p>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-lg border border-border bg-muted/50 px-3 py-2 font-mono text-xs">
                {created.secret}
              </code>
              <CopyButton
                value={created.secret}
                size="sm"
                variant="outline"
                label="Copy secret"
              />
            </div>
            <label className="flex items-start gap-2 text-sm">
              <Checkbox
                checked={secretAcknowledged}
                onCheckedChange={(checked) =>
                  setSecretAcknowledged(checked === true)
                }
              />
              <span>I have stored this signing secret securely.</span>
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
            <FormField label="URL" htmlFor="wh-url">
              <TextInput
                id="wh-url"
                type="url"
                required
                placeholder="https://example.com/webhook"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
              />
            </FormField>
            <fieldset className="space-y-2">
              <legend className="pm-label">Events</legend>
              <div className="grid grid-cols-2 gap-2">
                {AVAILABLE_EVENTS.map((ev) => (
                  <label key={ev} className="flex items-center gap-2 text-[13px] cursor-pointer">
                    <Checkbox
                      checked={events.includes(ev)}
                      onCheckedChange={() => toggleEvent(ev)}
                    />
                    <span className="text-muted-foreground">{ev}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            {err ? <p role="alert" className="text-sm text-destructive">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={create.isPending} className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
                {create.isPending ? "Creating..." : "Create Webhook"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
