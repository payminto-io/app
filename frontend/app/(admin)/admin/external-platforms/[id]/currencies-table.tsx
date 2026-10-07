"use client";

import { useState } from "react";
import { Pencil } from "lucide-react";
import {
  useExternalPlatformCurrencies,
  useUpsertEPBC,
  type ExternalPlatformBlockchainCurrency,
  type UpsertEPBCInput,
} from "@/lib/query/hooks/use-admin";
import { chainName } from "@/lib/chains";
import { CurrencyDisplay } from "@/components/currency-display";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { FormField, TextInput } from "@/components/ui/form-field";
import { Label } from "@/components/ui/label";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { Switch } from "@/components/ui/switch";

type Row = ExternalPlatformBlockchainCurrency;

function amountCell(r: Row, key: "autoApproveThreshold" | "hourlyCap" | "dailyCap" | "minAmount" | "maxAmount") {
  const value = r[key];
  return value ? <CurrencyDisplay amount={value} currency={r.currencyCode} size="sm" /> : null;
}

export function CurrenciesTable({ platformId }: { platformId: number }) {
  const { data, error, refetch } = useExternalPlatformCurrencies(platformId);
  const upsert = useUpsertEPBC(platformId);
  const [editing, setEditing] = useState<Row | null>(null);

  const columns: DataTableColumn<Row>[] = [
    {
      key: "asset",
      header: "Asset",
      cell: (r) => (
        <span className="inline-flex items-baseline gap-1.5">
          <span className="font-medium text-ink">{r.currencyCode}</span>
          <span className="text-label text-ink-soft">on {chainName(r.blockchainCode)}</span>
        </span>
      ),
    },
    { key: "status", header: "Status", cell: (r) => <StatusBadge status={r.active ? "active" : "inactive"} /> },
    { key: "auto", header: "Auto-approve up to", align: "right", cell: (r) => amountCell(r, "autoApproveThreshold") },
    { key: "hourly", header: "Hourly cap", align: "right", cell: (r) => amountCell(r, "hourlyCap") },
    { key: "daily", header: "Daily cap", align: "right", cell: (r) => amountCell(r, "dailyCap") },
    { key: "min", header: "Min", align: "right", cell: (r) => amountCell(r, "minAmount") },
    { key: "max", header: "Max", align: "right", cell: (r) => amountCell(r, "maxAmount") },
    {
      key: "edit",
      header: <span className="sr-only">Edit</span>,
      align: "right",
      className: "w-0",
      cell: (r) => (
        <Button
          variant="ghost"
          size="icon-sm"
          className="-my-1.5"
          aria-label={`Edit ${r.currencyCode} on ${chainName(r.blockchainCode)}`}
          onClick={() => setEditing(r)}
        >
          <Pencil />
        </Button>
      ),
    },
  ];

  if (error) {
    return <ErrorState message={error instanceof Error ? error.message : "Failed to load"} retry={() => refetch()} />;
  }

  return (
    <>
      <DataTable
        columns={columns}
        rows={data ?? []}
        loading={!data}
        getRowId={(r) => r.id}
        emptyTitle="No currencies configured."
      />
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
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {currency.currencyCode} on {chainName(currency.blockchainCode)}
            </DialogTitle>
          </DialogHeader>

          <div className="flex items-center gap-3">
            <Switch id="ebc-active" checked={active} onCheckedChange={setActive} />
            <Label htmlFor="ebc-active">Active</Label>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Auto-approve up to" htmlFor="ebc-threshold">
              <TextInput id="ebc-threshold" inputMode="decimal" className="num" value={threshold} onChange={(e) => setThreshold(e.target.value)} />
            </FormField>
            <FormField label="Hourly cap" htmlFor="ebc-hourly">
              <TextInput id="ebc-hourly" inputMode="decimal" className="num" value={hourly} onChange={(e) => setHourly(e.target.value)} />
            </FormField>
            <FormField label="Daily cap" htmlFor="ebc-daily">
              <TextInput id="ebc-daily" inputMode="decimal" className="num" value={daily} onChange={(e) => setDaily(e.target.value)} />
            </FormField>
            <FormField label="Min amount" htmlFor="ebc-min">
              <TextInput id="ebc-min" inputMode="decimal" className="num" value={min} onChange={(e) => setMin(e.target.value)} />
            </FormField>
            <FormField label="Max amount" htmlFor="ebc-max">
              <TextInput id="ebc-max" inputMode="decimal" className="num" value={max} onChange={(e) => setMax(e.target.value)} />
            </FormField>
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
