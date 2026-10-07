"use client";

/**
 * Hot wallets pay gas when SmartSweep moves deposits to the cold wallet; only
 * EVM and Tron use them. Backed by POST/GET /wallets/hot. Private keys go to
 * the backend SecretsVault (encrypted) and are never returned.
 */

import { useState, type FormEvent } from "react";
import { Eye, EyeOff, Plus } from "lucide-react";
import {
  useHotWallets,
  useHotWalletBalance,
  useRegisterHotWallet,
  type HotWallet,
  type RegisterHotWalletInput,
} from "@/lib/query/hooks/use-wallets";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { Notice } from "@/components/notice";
import { Badge } from "@/components/ui/badge";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import type { BlockchainNetwork } from "@/lib/types";
import { WalletTabs } from "../_components/wallet-tabs";

/* ── Family metadata ───────────────────────────────────── */

interface FamilyMeta {
  label: string;
  network: BlockchainNetwork;
  description: string;
}

/** Backend `blockchainFamilyCode` to display metadata. Only EVM and Tron support hot wallets. */
const FAMILY_META: Record<string, FamilyMeta> = {
  evm: { label: "EVM", network: "ethereum", description: "Ethereum, Base, Polygon" },
  EVM: { label: "EVM", network: "ethereum", description: "Ethereum, Base, Polygon" },
  trx: { label: "Tron", network: "tron", description: "TRX, TRC-20" },
  TRX: { label: "Tron", network: "tron", description: "TRX, TRC-20" },
};

const FAMILY_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "evm", label: "EVM (Ethereum, Base, Polygon)" },
  { value: "trx", label: "Tron (TRX, TRC-20)" },
];

/** Truncate long addresses for display. */
function truncateAddress(address: string): string {
  if (address.length <= 16) return address;
  return `${address.slice(0, 8)}\u2026${address.slice(-6)}`;
}

/**
 * Validate a private key for a given family. Returns an error string or
 * null when valid. Deliberately lenient — the backend enforces the real
 * rules when storing in the vault.
 */
function validatePrivateKey(
  familyCode: string,
  privateKey: string
): string | null {
  const trimmed = privateKey.trim();
  if (!trimmed) return "Private key is required";

  const lower = familyCode.toLowerCase();
  if (lower === "evm") {
    // 0x + 64 hex chars, or bare 64 hex chars.
    if (!/^(0x)?[a-fA-F0-9]{64}$/.test(trimmed)) {
      return "Must be 64 hex chars, optionally prefixed with 0x";
    }
    return null;
  }
  if (lower === "trx") {
    // Tron private keys are 64 hex chars (no 0x prefix expected but tolerated).
    if (!/^(0x)?[a-fA-F0-9]{64}$/.test(trimmed)) {
      return "Must be 64 hex characters";
    }
    return null;
  }
  return null;
}

/** Validate the public address for a given family. */
function validateAddress(
  familyCode: string,
  address: string
): string | null {
  const trimmed = address.trim();
  if (!trimmed) return "Public address is required";

  const lower = familyCode.toLowerCase();
  if (lower === "evm") {
    if (!/^0x[a-fA-F0-9]{40}$/.test(trimmed)) {
      return "Must be a valid EVM address (0x followed by 40 hex characters)";
    }
    return null;
  }
  if (lower === "trx") {
    if (!/^T[1-9A-HJ-NP-Za-km-z]{33}$/.test(trimmed)) {
      return "Must be a valid Tron address (starts with T, base58)";
    }
    return null;
  }
  return null;
}

/** Resolve metadata for a family code returned by the backend. */
function resolveFamily(code: string | undefined): FamilyMeta {
  if (!code) {
    return {
      label: "Unknown",
      network: "ethereum",
      description: "Unknown family",
    };
  }
  return (
    FAMILY_META[code] ??
    FAMILY_META[code.toLowerCase()] ?? {
      label: code.toUpperCase(),
      network: "ethereum",
      description: code,
    }
  );
}

/* ── Page ──────────────────────────────────────────────── */

export default function HotWalletsPage() {
  const { data, error, refetch } = useHotWallets();
  const [addOpen, setAddOpen] = useState(false);

  const addButton = (
    <Button onClick={() => setAddOpen(true)}>
      <Plus />
      Add hot wallet
    </Button>
  );

  return (
    <div className="space-y-5">
      <PageHeader title="Wallets">{addButton}</PageHeader>
      <WalletTabs />

      <Notice tone="wait">Keep only enough here to pay sweep gas. Hot wallets are online.</Notice>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data?.hotWallets ?? []}
          loading={!data}
          getRowId={(w) => w.id}
          emptyTitle="No hot wallets yet."
          emptyDescription="Add one for EVM or Tron so deposits can be swept to cold storage."
          emptyAction={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              Add hot wallet
            </Button>
          }
        />
      )}

      <AddHotWalletDialog open={addOpen} onOpenChange={setAddOpen} />
    </div>
  );
}

/* ── Table ─────────────────────────────────────────────── */

/** Native-coin floor below which a hot wallet can no longer reliably fund gas. */
function lowBalanceThreshold(symbol: string): number {
  switch (symbol) {
    case "ETH":
      return 0.01;
    case "BTC":
      return 0.0005;
    case "TRX":
      return 50;
    default:
      return 0;
  }
}

const COLUMNS: DataTableColumn<HotWallet>[] = [
  { key: "name", header: "Name", className: "font-medium", cell: (w) => w.name },
  {
    key: "chain",
    header: "Chain",
    className: "text-ink-soft",
    cell: (w) => resolveFamily(w.blockchainFamilyCode ?? w.blockchainFamily?.code).label,
  },
  {
    key: "address",
    header: "Address",
    cell: (w) =>
      w.address ? (
        <CopyField value={w.address} display={truncateAddress(w.address)} boxed={false} className="max-w-[220px]" />
      ) : null,
  },
  { key: "status", header: "Status", cell: (w) => <StatusBadge status={w.status || "inactive"} /> },
  { key: "balance", header: "Balance", align: "right", cell: (w) => <HotBalance wallet={w} /> },
];

/** Live native balance, fetched per row so the list never waits on RPC. */
function HotBalance({ wallet }: { wallet: HotWallet }) {
  const { data: bal, isLoading } = useHotWalletBalance(wallet.id, Boolean(wallet.address));
  if (!wallet.address) return null;
  if (isLoading) return <Skeleton className="ml-auto h-3 w-20" />;
  if (!bal) return <span className="text-caption text-ink-faint">Unavailable</span>;
  const low = parseFloat(bal.balance) < lowBalanceThreshold(bal.symbol);
  return (
    <span className="inline-flex items-center gap-2">
      {low ? <Badge variant="wait">Low</Badge> : null}
      <CurrencyDisplay amount={bal.balance} currency={bal.symbol} size="sm" />
    </span>
  );
}

/* ── Add Hot Wallet dialog ─────────────────────────────── */

function AddHotWalletDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const register = useRegisterHotWallet();

  const [confirmed, setConfirmed] = useState(false);
  const [familyCode, setFamilyCode] = useState<string>("");
  const [name, setName] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [address, setAddress] = useState("");
  const [showKey, setShowKey] = useState(false);

  const [pkError, setPkError] = useState<string | null>(null);
  const [addrError, setAddrError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);

  function reset() {
    setConfirmed(false);
    setFamilyCode("");
    setName("");
    setPrivateKey("");
    setAddress("");
    setShowKey(false);
    setPkError(null);
    setAddrError(null);
    setSubmitError(null);
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset();
    onOpenChange(next);
  }

  function handleFamilyChange(value: string) {
    setFamilyCode(value);
    if (privateKey) setPkError(validatePrivateKey(value, privateKey));
    if (address) setAddrError(validateAddress(value, address));
  }

  function handlePrivateKeyChange(value: string) {
    setPrivateKey(value);
    if (pkError && familyCode) {
      setPkError(validatePrivateKey(familyCode, value));
    }
  }

  function handleAddressChange(value: string) {
    setAddress(value);
    if (addrError && familyCode) {
      setAddrError(validateAddress(familyCode, value));
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitError(null);

    if (!confirmed) {
      setSubmitError("Confirm the device and balance check first.");
      return;
    }
    if (!familyCode) {
      setSubmitError("Choose a chain.");
      return;
    }
    if (!name.trim()) {
      setSubmitError("Enter a name.");
      return;
    }

    const pkErr = validatePrivateKey(familyCode, privateKey);
    if (pkErr) {
      setPkError(pkErr);
      return;
    }
    const addrErr = validateAddress(familyCode, address);
    if (addrErr) {
      setAddrError(addrErr);
      return;
    }

    const input: RegisterHotWalletInput = {
      blockchainFamilyCode: familyCode,
      privateKey: privateKey.trim(),
      name: name.trim(),
      address: address.trim(),
    };

    try {
      await register.mutateAsync(input);
      reset();
      onOpenChange(false);
    } catch (err) {
      setSubmitError(
        err instanceof Error ? err.message : "The hot wallet could not be added."
      );
    }
  }

  const canSubmit = confirmed && !register.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Add hot wallet</DialogTitle>
            <DialogDescription>
              The private key is encrypted at rest and never returned by the API.
            </DialogDescription>
          </DialogHeader>

          {/* Step 1: Security confirmation ─────────────────── */}
          <label className="flex cursor-pointer items-start gap-2.5 rounded-sm border border-wait/30 bg-wait-tint p-3">
            <span className="mt-0.5">
              <Checkbox
                checked={confirmed}
                onCheckedChange={(next) => setConfirmed(next === true)}
              />
            </span>
            <span className="text-body-sm text-ink">
              I am on a trusted device, and this key holds only gas money.
            </span>
          </label>

          {/* Blockchain family ─────────────────────────────── */}
          <FormField label="Chain" htmlFor="hw-family">
            <Select
              value={familyCode}
              onValueChange={(v) => handleFamilyChange(v ?? "")}
              disabled={!confirmed}
            >
              <SelectTrigger
                id="hw-family"
                className="w-full justify-between"
              >
                <SelectValue placeholder="Choose a chain" />
              </SelectTrigger>
              <SelectContent>
                {FAMILY_OPTIONS.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          {/* Wallet name ───────────────────────────────────── */}
          <FormField label="Name" htmlFor="hw-name">
            <TextInput
              id="hw-name"
              required
              disabled={!confirmed}
              placeholder="EVM gas wallet"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>

          {/* Private key ───────────────────────────────────── */}
          <FormField
            label="Private key"
            htmlFor="hw-private-key"
            error={pkError ?? undefined}
            hint={
              pkError
                ? undefined
                : "64 hex characters, 0x prefix optional"
            }
          >
            <div className="relative">
              <TextInput
                id="hw-private-key"
                required
                disabled={!confirmed}
                type={showKey ? "text" : "password"}
                placeholder="0x..."
                autoComplete="off"
                spellCheck={false}
                value={privateKey}
                onChange={(e) => handlePrivateKeyChange(e.target.value)}
                className="font-mono pr-10"
              />
              <button
                type="button"
                aria-label={showKey ? "Hide private key" : "Show private key"}
                onClick={() => setShowKey((s) => !s)}
                disabled={!confirmed}
                className="tap absolute inset-y-0 right-0 flex items-center pr-3 text-ink-faint transition-colors duration-120 hover:text-ink disabled:opacity-40"
              >
                {showKey ? (
                  <EyeOff className="size-4" />
                ) : (
                  <Eye className="size-4" />
                )}
              </button>
            </div>
          </FormField>

          {/* Public address ────────────────────────────────── */}
          <FormField
            label="Public address"
            htmlFor="hw-address"
            error={addrError ?? undefined}
            hint={
              addrError
                ? undefined
                : "The address of the key above"
            }
          >
            <TextInput
              id="hw-address"
              required
              disabled={!confirmed}
              placeholder={
                familyCode.toLowerCase() === "trx" ? "T..." : "0x..."
              }
              value={address}
              onChange={(e) => handleAddressChange(e.target.value)}
              className="font-mono"
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
              disabled={register.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!canSubmit}>
              {register.isPending ? "Adding..." : "Add hot wallet"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
