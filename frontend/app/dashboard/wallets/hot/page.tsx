"use client";

/**
 * Hot Wallet setup page.
 *
 * Hot wallets hold funds online to pay blockchain transaction fees when
 * SmartSweep moves deposits from customer addresses to the merchant's cold
 * wallet. Only EVM (ETH/Base/Polygon) and Tron chains use hot wallets —
 * Bitcoin uses a different sweep mechanism that does not require a
 * separately-funded gas wallet.
 *
 * Backed by POST/GET /wallets/hot on the Payminto backend. Private keys
 * are stored in the SecretsVault on the backend and never returned from
 * any GET endpoint.
 */

import { useState, type FormEvent } from "react";
import Link from "next/link";
import {
  ChevronRight,
  Flame,
  Plus,
  AlertTriangle,
  Info,
  Eye,
  EyeOff,
} from "lucide-react";
import {
  useHotWallets,
  useHotWalletBalance,
  useRegisterHotWallet,
  type HotWallet,
  type RegisterHotWalletInput,
} from "@/lib/query/hooks/use-wallets";
import { ErrorState, EmptyState } from "@/components/ui/states";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { BlockchainIcon } from "@/components/blockchain-icon";
import { CopyButton } from "@/components/copy-button";
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
import type { BlockchainNetwork } from "@/lib/types";
import { cn } from "@/lib/utils";

/* ── Family metadata ───────────────────────────────────── */

interface FamilyMeta {
  label: string;
  network: BlockchainNetwork;
  description: string;
}

/**
 * Maps the backend `blockchainFamilyCode` (as stored in blockchain_families)
 * to display metadata. Only EVM and Tron support hot wallets.
 */
const FAMILY_META: Record<string, FamilyMeta> = {
  evm: {
    label: "EVM",
    network: "ethereum",
    description: "Ethereum, Base, Polygon",
  },
  EVM: {
    label: "EVM",
    network: "ethereum",
    description: "Ethereum, Base, Polygon",
  },
  trx: { label: "Tron", network: "tron", description: "TRX / TRC-20" },
  TRX: { label: "Tron", network: "tron", description: "TRX / TRC-20" },
};

const FAMILY_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "evm", label: "EVM (ETH / Base / Polygon)" },
  { value: "trx", label: "Tron (TRX / TRC-20)" },
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

/** Format a backend timestamp as a compact human string. */
function formatCreatedAt(raw: string): string {
  if (!raw) return "\u2014";
  try {
    const d = new Date(raw);
    if (Number.isNaN(d.getTime())) return raw;
    return d.toLocaleString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return raw;
  }
}

/* ── Page ──────────────────────────────────────────────── */

export default function HotWalletsPage() {
  const { data, isLoading, error, refetch } = useHotWallets();
  const [addOpen, setAddOpen] = useState(false);

  const hotWallets: HotWallet[] = data?.hotWallets ?? [];

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
        <span className="text-foreground font-medium">Hot Wallets</span>
      </nav>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Hot Wallet</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Configure gas-fee wallets for sweeping deposits to cold storage
            (EVM + Tron only)
          </p>
        </div>
        <Button
          size="sm"
          className="h-9 rounded-lg"
          onClick={() => setAddOpen(true)}
        >
          <Plus className="size-4" />
          Add Hot Wallet
        </Button>
      </div>

      {/* Security warning banner */}
      <div className="flex items-start gap-3 rounded-lg border border-amber-500/30 bg-amber-500/5 p-4">
        <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-amber-500/10 text-amber-500">
          <AlertTriangle className="size-4" />
        </div>
        <div className="space-y-1">
          <h3 className="text-[13px] font-semibold text-foreground">
            Security notice
          </h3>
          <p className="text-[12px] text-muted-foreground leading-relaxed">
            Hot wallets hold funds online to pay transaction fees. Keep only a
            minimum balance (e.g., $50&ndash;100 worth). Never store significant
            funds in a hot wallet. Private keys are encrypted at rest in
            Payminto&apos;s secrets vault.
          </p>
        </div>
      </div>

      {/* Info banner: what is a hot wallet? */}
      <div className="flex items-start gap-3 rounded-lg border border-[var(--pm-primary)]/20 bg-[var(--pm-primary)]/5 p-4">
        <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md bg-[var(--pm-primary)]/10 text-[var(--pm-primary)]">
          <Info className="size-4" />
        </div>
        <div className="space-y-1">
          <h3 className="text-[13px] font-semibold text-foreground">
            What is a hot wallet?
          </h3>
          <p className="text-[12px] text-muted-foreground leading-relaxed">
            Hot wallets pay gas fees when SmartSweep moves funds from customer
            deposit addresses to your cold wallet. Only EVM (ETH, Base,
            Polygon) and Tron chains need hot wallets &mdash; Bitcoin uses a
            different sweep mechanism that does not require a separately-funded
            gas wallet.
          </p>
        </div>
      </div>

      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
      ) : hotWallets.length === 0 ? (
        <EmptyState
          title="No hot wallets configured"
          description="Add a hot wallet for EVM or Tron to enable automatic sweeping of deposits to your cold storage."
          icon={<Flame className="size-5" />}
          action={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              <Plus className="size-4" />
              Add Hot Wallet
            </Button>
          }
        />
      ) : (
        <div className="space-y-3">
          {hotWallets.map((wallet) => (
            <HotWalletCard key={wallet.id} wallet={wallet} />
          ))}
        </div>
      )}

      <AddHotWalletDialog open={addOpen} onOpenChange={setAddOpen} />
    </div>
  );
}

/* ── Hot wallet card (live balance) ────────────────────── */

// lowBalanceThreshold returns the native-coin floor below which a hot wallet is
// flagged "low" (it can no longer reliably fund gas / sweeps).
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

function HotWalletCard({ wallet }: { wallet: HotWallet }) {
  const meta = resolveFamily(
    wallet.blockchainFamilyCode ?? wallet.blockchainFamily?.code
  );
  const address = wallet.address;
  const { data: bal, isLoading: balLoading } = useHotWalletBalance(
    wallet.id,
    Boolean(address)
  );

  const balanceNum = bal ? parseFloat(bal.balance) : null;
  const lowBalance =
    balanceNum !== null &&
    bal !== undefined &&
    balanceNum < lowBalanceThreshold(bal.symbol);

  return (
    <Card className="border-border shadow-sm">
      <CardContent className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4 min-w-0">
          <BlockchainIcon blockchain={meta.network} size="lg" />
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3 className="text-[14px] font-semibold">{wallet.name}</h3>
              <span className="pm-label">{meta.label}</span>
              {lowBalance ? (
                <span className="inline-flex items-center gap-1 rounded-md border border-amber-500/40 bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-400">
                  <AlertTriangle className="size-3" />
                  Low balance
                </span>
              ) : null}
            </div>
            <p className="mt-0.5 text-[12px] text-muted-foreground">
              {meta.description} &middot; Created{" "}
              {formatCreatedAt(wallet.createdAt)}
            </p>
            {address ? (
              <div className="mt-1.5 flex items-center gap-2">
                <code
                  className="font-mono text-[12px] text-foreground/90"
                  title={address}
                >
                  {truncateAddress(address)}
                </code>
                <CopyButton
                  value={address}
                  label=""
                  size="icon"
                  variant="ghost"
                  className="size-7"
                />
              </div>
            ) : (
              <p className="mt-1.5 text-[12px] text-muted-foreground italic">
                Address unavailable
              </p>
            )}
          </div>
        </div>
        <div className="flex flex-col items-end gap-1.5">
          <span
            className={cn(
              "inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[11px] font-medium",
              wallet.status === "active"
                ? "bg-emerald-500/10 text-emerald-400"
                : "bg-muted text-muted-foreground"
            )}
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                wallet.status === "active"
                  ? "bg-emerald-400"
                  : "bg-muted-foreground"
              )}
            />
            {wallet.status || "unknown"}
          </span>
          {address ? (
            <span className="font-mono text-[13px] tabular-nums text-foreground/90">
              {balLoading ? (
                <span className="text-muted-foreground">…</span>
              ) : bal ? (
                `${bal.balance} ${bal.symbol}`
              ) : (
                <span className="text-muted-foreground" title="Balance unavailable (no active RPC)">
                  —
                </span>
              )}
            </span>
          ) : null}
        </div>
      </CardContent>
    </Card>
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
      setSubmitError("Please confirm the security notice before submitting");
      return;
    }
    if (!familyCode) {
      setSubmitError("Please select a blockchain family");
      return;
    }
    if (!name.trim()) {
      setSubmitError("Wallet name is required");
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
        err instanceof Error ? err.message : "Failed to register hot wallet"
      );
    }
  }

  const selectedMeta = familyCode ? resolveFamily(familyCode) : null;
  const canSubmit = confirmed && !register.isPending;

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Add Hot Wallet</DialogTitle>
            <DialogDescription>
              Register a gas-fee wallet for SmartSweep. The private key is
              encrypted at rest in Payminto&apos;s secrets vault and is never
              returned by any API.
            </DialogDescription>
          </DialogHeader>

          {/* Step 1: Security confirmation ─────────────────── */}
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 p-3">
            <label className="flex items-start gap-2.5 cursor-pointer">
              <span className="mt-0.5">
                <Checkbox
                  checked={confirmed}
                  onCheckedChange={(next) => setConfirmed(next === true)}
                />
              </span>
              <span className="text-[12px] leading-relaxed text-foreground">
                I understand I&apos;m entering a private key and it will be
                encrypted at rest. I confirm I&apos;m on a trusted device.
              </span>
            </label>
          </div>

          {/* Blockchain family ─────────────────────────────── */}
          <FormField label="Blockchain Family" htmlFor="hw-family">
            <Select
              value={familyCode}
              onValueChange={(v) => handleFamilyChange(v ?? "")}
              disabled={!confirmed}
            >
              <SelectTrigger
                id="hw-family"
                className="w-full justify-between"
              >
                <SelectValue placeholder="Select a family" />
              </SelectTrigger>
              <SelectContent>
                {FAMILY_OPTIONS.map((opt) => {
                  const meta = resolveFamily(opt.value);
                  return (
                    <SelectItem key={opt.value} value={opt.value}>
                      <span className="flex items-center gap-2">
                        <BlockchainIcon blockchain={meta.network} size="sm" />
                        {opt.label}
                      </span>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </FormField>

          {/* Wallet name ───────────────────────────────────── */}
          <FormField label="Wallet Name" htmlFor="hw-name">
            <TextInput
              id="hw-name"
              required
              disabled={!confirmed}
              placeholder="e.g. EVM Gas Wallet"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>

          {/* Private key ───────────────────────────────────── */}
          <FormField
            label="Private Key"
            htmlFor="hw-private-key"
            error={pkError ?? undefined}
            hint={
              pkError
                ? undefined
                : selectedMeta
                ? "64 hex chars starting with 0x (EVM) or 64 hex chars (Tron)"
                : "64 hex chars starting with 0x (EVM) or Base58 (Tron)"
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
                className="absolute inset-y-0 right-0 flex items-center pr-3 text-muted-foreground hover:text-foreground disabled:opacity-50"
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
            label="Public Address"
            htmlFor="hw-address"
            error={addrError ?? undefined}
            hint={
              addrError
                ? undefined
                : "The public address derived from the private key above"
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

          {/* Final warning ─────────────────────────────────── */}
          <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3">
            <p className="text-[11px] leading-relaxed text-destructive/90">
              <span className="font-semibold">NEVER</span> paste a private key
              that holds significant funds. Use a dedicated hot wallet with
              only enough balance to cover sweep gas fees.
            </p>
          </div>

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
              disabled={register.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={!canSubmit}>
              {register.isPending ? "Saving..." : "Add Hot Wallet"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
