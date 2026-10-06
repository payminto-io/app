"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { ChevronRight, Shield, Snowflake, Plus, Trash2 } from "lucide-react";
import {
  useColdWallets,
  useConfigureColdWallet,
  type ColdWallet,
  type ConfigureColdWalletInput,
} from "@/lib/query/hooks/use-wallets";
import { ErrorState, EmptyState } from "@/components/ui/states";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { BlockchainIcon } from "@/components/blockchain-icon";
import { CopyButton } from "@/components/copy-button";
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
import type { BlockchainNetwork } from "@/lib/types";

/* ── Chain metadata ────────────────────────────────────── */

interface ChainMeta {
  name: string;
  network: BlockchainNetwork;
}

const CHAIN_META: Record<string, ChainMeta> = {
  ETH: { name: "Ethereum", network: "ethereum" },
  BASE: { name: "Base", network: "base" },
  POLYGON: { name: "Polygon", network: "polygon" },
  BTC: { name: "Bitcoin", network: "bitcoin" },
  TRX: { name: "Tron", network: "tron" },
};

const CHAIN_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "ETH", label: "Ethereum (ETH)" },
  { value: "BASE", label: "Base (BASE)" },
  { value: "POLYGON", label: "Polygon (POLYGON)" },
  { value: "BTC", label: "Bitcoin (BTC)" },
  { value: "TRX", label: "Tron (TRX)" },
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

export default function ColdWalletsPage() {
  const { data, isLoading, error, refetch } = useColdWallets();
  const [addOpen, setAddOpen] = useState(false);

  const coldWallets: ColdWallet[] = data?.coldWallets ?? [];

  return (
    <div className="space-y-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1.5 text-[13px] text-muted-foreground">
        <Link
          href="/dashboard"
          className="hover:text-foreground transition-colors"
        >
          Dashboard
        </Link>
        <ChevronRight className="size-3.5" />
        <Link
          href="/dashboard/wallets"
          className="hover:text-foreground transition-colors"
        >
          Wallets
        </Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground font-medium">Cold Wallets</span>
      </nav>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Cold Wallet</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Configure secure offline wallets that receive swept funds from
            customer deposits
          </p>
        </div>
        <Button
          size="sm"
          className="h-9 rounded-lg"
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-4" />
          Add Cold Wallet
        </Button>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-3 rounded-lg border border-[var(--pm-primary)]/20 bg-[var(--pm-primary)]/5 p-4">
        <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-[var(--pm-primary)]/10 text-[var(--pm-primary)]">
          <Shield className="size-4" />
        </div>
        <div className="space-y-1">
          <h3 className="text-[13px] font-semibold text-foreground">
            What is a cold wallet?
          </h3>
          <p className="text-[12px] text-muted-foreground leading-relaxed">
            A cold wallet is an offline wallet where swept funds are stored for
            security. SmartSweep automatically transfers deposits from customer
            addresses to your cold wallet. Use a hardware wallet (Ledger,
            Trezor) or an air-gapped device &mdash; never a hot wallet.
          </p>
        </div>
      </div>

      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
      ) : coldWallets.length === 0 ? (
        <EmptyState
          title="No cold wallet configured"
          description="Add your first cold wallet to enable automatic fund consolidation from customer deposit addresses."
          icon={<Snowflake className="size-5" />}
          action={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              <Plus className="size-4" />
              Add Cold Wallet
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {coldWallets.map((wallet) => {
            const meta = CHAIN_META[wallet.blockchainCode];
            return (
              <Card
                key={`${wallet.blockchainCode}-${wallet.address}`}
                className="border-border shadow-sm"
              >
                <CardContent className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
                  <div className="flex items-center gap-4 min-w-0">
                    <BlockchainIcon
                      blockchain={meta?.network ?? "ethereum"}
                      size="lg"
                    />
                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <h3 className="text-[14px] font-semibold">
                          {meta?.name ?? wallet.blockchainCode}
                        </h3>
                        <span className="pm-label">
                          {wallet.blockchainCode}
                        </span>
                      </div>
                      <p className="mt-0.5 text-[12px] text-muted-foreground truncate">
                        {wallet.name}
                      </p>
                      <div className="mt-1.5 flex items-center gap-2">
                        <code
                          className="font-mono text-[12px] text-foreground/90"
                          title={wallet.address}
                        >
                          {truncateAddress(wallet.address)}
                        </code>
                        <CopyButton
                          value={wallet.address}
                          label=""
                          size="icon"
                          variant="ghost"
                          className="size-7"
                        />
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled
                      title="Coming soon"
                      className="h-8 gap-1.5 text-[12px]"
                    >
                      <Trash2 className="size-3.5" />
                      Remove
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
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
      setSubmitError("Please select a blockchain");
      return;
    }
    if (!name.trim()) {
      setSubmitError("Wallet name is required");
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
        err instanceof Error ? err.message : "Failed to configure cold wallet"
      );
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Add Cold Wallet</DialogTitle>
            <DialogDescription>
              Configure a cold wallet for a blockchain. Swept funds will be
              sent here automatically.
            </DialogDescription>
          </DialogHeader>

          <FormField label="Blockchain" htmlFor="cw-blockchain">
            <Select
              value={blockchainCode}
              onValueChange={(v) => handleBlockchainChange(v ?? "")}
            >
              <SelectTrigger
                id="cw-blockchain"
                className="w-full justify-between"
              >
                <SelectValue placeholder="Select a blockchain" />
              </SelectTrigger>
              <SelectContent>
                {CHAIN_OPTIONS.map((opt) => {
                  const meta = CHAIN_META[opt.value];
                  return (
                    <SelectItem key={opt.value} value={opt.value}>
                      <span className="flex items-center gap-2">
                        <BlockchainIcon
                          blockchain={meta?.network ?? "ethereum"}
                          size="sm"
                        />
                        {opt.label}
                      </span>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </FormField>

          <FormField
            label="Wallet Address"
            htmlFor="cw-address"
            error={addressError ?? undefined}
            hint={
              blockchainCode
                ? undefined
                : "Select a blockchain first to validate the address format"
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
                  : "Paste your cold wallet address"
              }
              value={address}
              onChange={(e) => handleAddressChange(e.target.value)}
              className="font-mono"
            />
          </FormField>

          <FormField label="Wallet Name" htmlFor="cw-name">
            <TextInput
              id="cw-name"
              required
              placeholder="e.g. Main Cold Storage"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>

          {submitError ? (
            <p role="alert" className="text-xs text-destructive">
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
              {configure.isPending ? "Saving..." : "Add Cold Wallet"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
