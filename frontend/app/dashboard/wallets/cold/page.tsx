"use client";

/** Cold wallets receive swept deposits; only the public address is stored. Backed by GET/POST /wallets/cold. */

import { useState, type FormEvent } from "react";
import { Plus } from "lucide-react";
import {
  useColdWallets,
  useConfigureColdWallet,
  type ColdWallet,
  type ConfigureColdWalletInput,
} from "@/lib/query/hooks/use-wallets";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormField, TextInput } from "@/components/ui/form-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ErrorState } from "@/components/ui/states";
import { chainName } from "@/lib/chains";
import { WalletTabs } from "../_components/wallet-tabs";

/* ── Chain metadata ────────────────────────────────────── */


const CHAIN_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "ETH", label: "Ethereum" },
  { value: "BASE", label: "Base" },
  { value: "POLYGON", label: "Polygon" },
  { value: "BTC", label: "Bitcoin" },
  { value: "TRX", label: "Tron" },
];

/** Truncate long addresses for display: 0x1234…abcd */
function truncateAddress(address: string): string {
  if (address.length <= 16) return address;
  return `${address.slice(0, 8)}\u2026${address.slice(-6)}`;
}

/**
 * Validate an address for a given blockchain code.
 * Returns an error string or null when valid.
 */
function validateAddress(
  blockchainCode: string,
  address: string
): string | null {
  const trimmed = address.trim();
  if (!trimmed) return "Address is required";

  switch (blockchainCode) {
    case "ETH":
    case "BASE":
    case "POLYGON":
      if (!/^0x[a-fA-F0-9]{40}$/.test(trimmed)) {
        return "Must be a valid EVM address (0x followed by 40 hex characters)";
      }
      return null;
    case "BTC":
      if (!/^(bc1|tb1|1|3)/.test(trimmed)) {
        return "Must start with bc1, tb1, 1, or 3";
      }
      return null;
    case "TRX":
      if (!/^T/.test(trimmed)) {
        return "Tron addresses must start with T";
      }
      return null;
    default:
      return null;
  }
}

/* ── Page ──────────────────────────────────────────────── */

const COLUMNS: DataTableColumn<ColdWallet>[] = [
  { key: "name", header: "Name", className: "font-medium", cell: (w) => w.name || null },
  {
    key: "chain",
    header: "Chain",
    cell: (w) => (
      <span className="inline-flex items-baseline gap-1.5">
        <span className="text-ink">{chainName(w.blockchainCode)}</span>
        <span className="font-mono text-label text-ink-soft">{w.blockchainCode}</span>
      </span>
    ),
  },
  {
    key: "address",
    header: "Address",
    cell: (w) => (
      <CopyField value={w.address} display={truncateAddress(w.address)} boxed={false} className="max-w-[240px]" />
    ),
  },
];

export default function ColdWalletsPage() {
  const { data, error, refetch } = useColdWallets();
  const [addOpen, setAddOpen] = useState(false);

  return (
    <div className="space-y-5">
      <PageHeader title="Wallets">
        <Button onClick={() => setAddOpen(true)}>
          <Plus />
          Add cold wallet
        </Button>
      </PageHeader>
      <WalletTabs />

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data?.coldWallets ?? []}
          loading={!data}
          getRowId={(w) => `${w.blockchainCode}-${w.address}`}
          emptyTitle="No cold wallets yet."
          emptyDescription="Sweeps send deposits to the cold wallet of each chain."
          emptyAction={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              Add cold wallet
            </Button>
          }
        />
      )}

      <AddColdWalletDialog open={addOpen} onOpenChange={setAddOpen} />
    </div>
  );
}

/* ── Add Cold Wallet dialog ────────────────────────────── */

function AddColdWalletDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const configure = useConfigureColdWallet();
  const [blockchainCode, setBlockchainCode] = useState<string>("");
  const [address, setAddress] = useState("");
  const [name, setName] = useState("");
  const [addressError, setAddressError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function reset() {
    setBlockchainCode("");
    setAddress("");
    setName("");
    setAddressError(null);
    setSubmitError(null);
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset();
    onOpenChange(next);
  }

  function handleAddressChange(value: string) {
    setAddress(value);
    if (addressError && blockchainCode) {
      // Live-clear the error once the user starts editing again.
      const err = validateAddress(blockchainCode, value);
      setAddressError(err);
    }
  }

  function handleBlockchainChange(value: string) {
    setBlockchainCode(value);
    if (address) {
      setAddressError(validateAddress(value, address));
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitError(null);

    if (!blockchainCode) {
      setSubmitError("Choose a chain.");
      return;
    }
    if (!name.trim()) {
      setSubmitError("Enter a name.");
      return;
    }
    const addrErr = validateAddress(blockchainCode, address);
    if (addrErr) {
      setAddressError(addrErr);
      return;
    }

    const input: ConfigureColdWalletInput = {
      blockchainCode,
      address: address.trim(),
      name: name.trim(),
    };

    try {
      await configure.mutateAsync(input);
      reset();
      onOpenChange(false);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "The cold wallet could not be added."
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Add cold wallet</DialogTitle>
            <DialogDescription>Swept deposits on this chain go to this address.</DialogDescription>
          </DialogHeader>

          <FormField label="Chain" htmlFor="cw-blockchain">
            <Select
              value={blockchainCode}
              onValueChange={(v) => handleBlockchainChange(v ?? "")}
            >
              <SelectTrigger
                id="cw-blockchain"
                className="w-full justify-between"
              >
                <SelectValue placeholder="Choose a chain" />
              </SelectTrigger>
              <SelectContent>
                {CHAIN_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          <FormField
            label="Address"
            htmlFor="cw-address"
            error={addressError ?? undefined}
            hint={
              blockchainCode
                ? undefined
                : "Choose a chain first"
            }
          >
            <TextInput
              id="cw-address"
              required
              placeholder={
                blockchainCode === "BTC"
                  ? "bc1..."
                  : blockchainCode === "TRX"
                  ? "T..."
                  : blockchainCode
                  ? "0x..."
                  : ""
              }
              value={address}
              onChange={(e) => handleAddressChange(e.target.value)}
              className="font-mono"
            />
          </FormField>

          <FormField label="Name" htmlFor="cw-name">
            <TextInput
              id="cw-name"
              required
              placeholder="Main cold storage"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>

          {submitError ? (
            <p role="alert" className="text-body-sm text-bad">
              {submitError}
            </p>
          ) : null}

          <DialogFooter>
            <Button
              variant="outline"
              type="button"
              onClick={() => handleOpenChange(false)}
              disabled={configure.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={configure.isPending}>
              {configure.isPending ? "Adding..." : "Add cold wallet"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
