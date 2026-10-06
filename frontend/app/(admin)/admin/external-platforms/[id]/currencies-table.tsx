"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import { Card } from "@/components/ui/card";
import { StatusBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState, ErrorState } from "@/components/ui/states";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  useExternalPlatformCurrencies,
  useUpsertEPBC,
} from "@/lib/query/hooks/use-admin";
import type {
  ExternalPlatformBlockchainCurrency,
  UpsertEPBCInput,
} from "@/lib/query/hooks/use-admin";

export function CurrenciesTable({ platformId }: { platformId: number }) {
  const { data, isLoading, error, refetch } =
    useExternalPlatformCurrencies(platformId);
  const upsert = useUpsertEPBC(platformId);
  const [editing, setEditing] =
    useState<ExternalPlatformBlockchainCurrency | null>(null);

  if (isLoading) {
    return (
      <Card className="p-0">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="flex items-center gap-4 border-b border-border px-4 py-3">
            <Skeleton className="h-4 w-16" />
            <Skeleton className="h-4 w-16" />
            <Skeleton className="ml-auto h-4 w-12" />
          </div>
        ))}
      </Card>
    );
  }
  if (error) {
    return (
      <ErrorState
        message={error instanceof Error ? error.message : "Failed to load"}
        retry={() => refetch()}
      />
    );
  }

  if ((data ?? []).length === 0) {
    return (
      <EmptyState
        title="No currencies configured"
        description="Currency settings will appear once configured."
      />
    );
  }

  return (
    <>
      <Card className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="pm-label">Chain</TableHead>
              <TableHead className="pm-label">Currency</TableHead>
              <TableHead className="pm-label">Status</TableHead>
              <TableHead className="pm-label tabular-nums">Auto-approve</TableHead>
              <TableHead className="pm-label tabular-nums">Hourly Cap</TableHead>
              <TableHead className="pm-label tabular-nums">Daily Cap</TableHead>
              <TableHead className="pm-label tabular-nums">Min</TableHead>
              <TableHead className="pm-label tabular-nums">Max</TableHead>
              <TableHead className="w-12" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(data ?? []).map((r) => (
              <TableRow key={r.id}>
                <TableCell className="text-[13px] font-medium">
                  {r.blockchainCode}
                </TableCell>
                <TableCell className="text-[13px] font-mono">
                  {r.currencyCode}
                </TableCell>
                <TableCell>
                  <StatusBadge status={r.active ? "active" : "inactive"} />
                </TableCell>
                <TableCell className="tabular-nums text-[13px]">
                  {r.autoApproveThreshold ?? "--"}
                </TableCell>
                <TableCell className="tabular-nums text-[13px]">
                  {r.hourlyCap ?? "--"}
                </TableCell>
                <TableCell className="tabular-nums text-[13px]">
                  {r.dailyCap ?? "--"}
                </TableCell>
                <TableCell className="tabular-nums text-[13px]">
                  {r.minAmount ?? "--"}
                </TableCell>
                <TableCell className="tabular-nums text-[13px]">
                  {r.maxAmount ?? "--"}
                </TableCell>
                <TableCell>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEditing(r)}
                  >
                    <Pencil className="size-3.5" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Card>

      {editing ? (
        <EditCurrencyDialog
          currency={editing}
          onClose={() => setEditing(null)}
          onSubmit={async (input) => {
            await upsert.mutateAsync(input);
            setEditing(null);
          }}
          isPending={upsert.isPending}
        />
      ) : null}
    </>
  );
}

function EditCurrencyDialog({
  currency,
  onClose,
  onSubmit,
  isPending,
}: {
  currency: ExternalPlatformBlockchainCurrency;
  onClose: () => void;
  onSubmit: (input: UpsertEPBCInput) => void;
  isPending: boolean;
}) {
  const [active, setActive] = useState(currency.active);
  const [threshold, setThreshold] = useState(currency.autoApproveThreshold ?? "");
  const [hourly, setHourly] = useState(currency.hourlyCap ?? "");
  const [daily, setDaily] = useState(currency.dailyCap ?? "");
  const [min, setMin] = useState(currency.minAmount ?? "");
  const [max, setMax] = useState(currency.maxAmount ?? "");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSubmit({
      blockchainCode: currency.blockchainCode,
      currencyCode: currency.currencyCode,
      active,
      autoApproveThreshold: threshold || undefined,
      hourlyCap: hourly || undefined,
      dailyCap: daily || undefined,
      minAmount: min || undefined,
      maxAmount: max || undefined,
    });
  }

  return (
    <Dialog open onOpenChange={() => onClose()}>
      <DialogContent>
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>
              Edit {currency.blockchainCode} / {currency.currencyCode}
            </DialogTitle>
          </DialogHeader>

          <div className="flex items-center gap-3">
            <Switch checked={active} onCheckedChange={setActive} />
            <Label className="text-sm">Active</Label>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="ebc-threshold" className="text-[12px]">Auto-approve Threshold</Label>
              <Input id="ebc-threshold" value={threshold} onChange={(e) => setThreshold(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ebc-hourly" className="text-[12px]">Hourly Cap</Label>
              <Input id="ebc-hourly" value={hourly} onChange={(e) => setHourly(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ebc-daily" className="text-[12px]">Daily Cap</Label>
              <Input id="ebc-daily" value={daily} onChange={(e) => setDaily(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ebc-min" className="text-[12px]">Min Amount</Label>
              <Input id="ebc-min" value={min} onChange={(e) => setMin(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ebc-max" className="text-[12px]">Max Amount</Label>
              <Input id="ebc-max" value={max} onChange={(e) => setMax(e.target.value)} />
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? "Saving..." : "Save"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
