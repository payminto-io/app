"use client";

/**
 * Wallet management — main overview.
 *
 * Lists the HD wallets provisioned for the current platform (one per
 * blockchain family). Each card links to its pre-generated address pool
 * at `/dashboard/wallets/[id]/addresses`.
 *
 * Backed by GET /wallets.
 */
import Link from "next/link";
import {
  ChevronRight,
  Flame,
  KeyRound,
  MoreHorizontal,
  Snowflake,
  Wallet as WalletIcon,
} from "lucide-react";
import { useWalletsList } from "@/lib/query/hooks/use-wallets";
import type { Wallet } from "@/lib/query/hooks/use-wallets";
import { ErrorState, EmptyState } from "@/components/ui/states";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { BlockchainIcon } from "@/components/blockchain-icon";
import type { BlockchainNetwork } from "@/lib/types";

/* ── Blockchain family → display metadata ────────────── */

type FamilyMeta = {
  displayName: string;
  network: BlockchainNetwork;
  description: string;
};

const FAMILY_META: Record<string, FamilyMeta> = {
  evm: {
    displayName: "EVM Wallet",
    network: "ethereum",
    description: "Ethereum, Base, Polygon and other EVM-compatible chains.",
  },
  eth_family: {
    displayName: "EVM Wallet",
    network: "ethereum",
    description: "Ethereum, Base, Polygon and other EVM-compatible chains.",
  },
  btc: {
    displayName: "Bitcoin Wallet",
    network: "bitcoin",
    description: "Native SegWit deposit addresses for BTC.",
  },
  btc_family: {
    displayName: "Bitcoin Wallet",
    network: "bitcoin",
    description: "Native SegWit deposit addresses for BTC.",
  },
  trx: {
    displayName: "Tron Wallet",
    network: "tron",
    description: "TRX and TRC-20 token deposit addresses.",
  },
  trx_family: {
    displayName: "Tron Wallet",
    network: "tron",
    description: "TRX and TRC-20 token deposit addresses.",
  },
};

function familyMetaFor(wallet: Wallet): FamilyMeta {
  const candidates = [
    wallet.blockchainFamily?.code,
    wallet.blockchainFamily?.family,
    wallet.blockchainFamily?.name,
  ];
  for (const candidate of candidates) {
    if (!candidate) continue;
    const meta = FAMILY_META[candidate.toLowerCase()];
    if (meta) return meta;
  }
  return {
    displayName: wallet.name || "Wallet",
    network: "ethereum",
    description: "Merchant deposit wallet.",
  };
}

function kindLabel(kind: string): string {
  const key = kind.toLowerCase();
  if (key === "hd") return "HD Wallet";
  if (key === "hot") return "Hot Wallet";
  if (key === "cold") return "Cold Wallet";
  if (key === "sc" || key === "scw") return "Smart Contract";
  return kind;
}

/* ── Page component ───────────────────────────────────── */

export default function WalletsPage() {
  const { data, isLoading, error, refetch } = useWalletsList();
  const wallets = data?.wallets ?? [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Wallet Management</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            HD wallets and their pre-generated deposit address pools
          </p>
        </div>
      </div>

      {/* Setup guide — 3 wallet types */}
      <SetupGuide />

      {error ? (
        <ErrorState message={error.message} retry={() => refetch()} />
      ) : null}

      {/* Wallet grid */}
      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-56 rounded-xl" />
          ))}
        </div>
      ) : wallets.length === 0 && !error ? (
        <EmptyState
          title="No wallets provisioned"
          description="HD wallets will appear here once your backend has provisioned blockchain families. Contact your administrator to enable chains."
          icon={<WalletIcon className="size-5" />}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {wallets.map((wallet) => (
            <WalletCard key={wallet.id} wallet={wallet} />
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Wallet card ──────────────────────────────────────── */

function WalletCard({ wallet }: { wallet: Wallet }) {
  const meta = familyMetaFor(wallet);
  const href = `/dashboard/wallets/${wallet.id}/addresses`;

  return (
    <Card className="border-border shadow-sm hover:border-border/80 transition-colors">
      <CardContent className="p-5 space-y-4">
        {/* Top row: icon + name + kind badge */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3 min-w-0">
            <BlockchainIcon blockchain={meta.network} size="lg" />
            <div className="min-w-0">
              <h3 className="text-[14px] font-semibold truncate">
                {meta.displayName}
              </h3>
              <p className="pm-label truncate">
                {wallet.blockchainFamily?.name ?? wallet.name}
              </p>
            </div>
          </div>
          <Badge variant="outline" className="shrink-0">
            {kindLabel(wallet.kind)}
          </Badge>
        </div>

        {/* Meta: address count + status */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-[12px] text-muted-foreground">
              Address pool
            </span>
            <span className="text-[13px] font-semibold tabular-nums">
              {wallet.addressCount.toLocaleString()} addresses
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-[12px] text-muted-foreground">Status</span>
            <StatusBadge status={wallet.status} />
          </div>
          {wallet.blockchainFamily?.path ? (
            <div className="flex items-center justify-between">
              <span className="text-[12px] text-muted-foreground">
                Derivation
              </span>
              <span className="font-mono text-[11px] text-muted-foreground truncate max-w-[180px]">
                {wallet.blockchainFamily.path}
              </span>
            </div>
          ) : null}
        </div>

        {/* Actions */}
        <div className="flex gap-2 pt-1">
          <Link href={href} className="flex-1">
            <Button
              variant="outline"
              size="sm"
              className="w-full h-9 text-[12px]"
            >
              View Addresses
              <ChevronRight className="size-3.5" />
            </Button>
          </Link>
          <Button
            variant="outline"
            size="sm"
            className="h-9 w-9 p-0"
            aria-label="Manage wallet"
            disabled
            title="Wallet management actions coming soon"
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

/* ── Setup guide ──────────────────────────────────────── */

function SetupGuide() {
  const items = [
    {
      icon: <KeyRound className="size-4" />,
      title: "Deposit Wallet (HD)",
      description:
        "Hierarchical-deterministic wallet that generates unique deposit addresses for every payment, derived from a single seed.",
    },
    {
      icon: <Flame className="size-4" />,
      title: "Hot Wallet",
      description:
        "Keeps a small operating balance online to pay gas for sweeps, ERC-20 approvals and merchant payouts.",
    },
    {
      icon: <Snowflake className="size-4" />,
      title: "Cold Wallet",
      description:
        "Offline destination for swept funds. Payminto only stores the public address — your keys stay in self-custody.",
    },
  ];

  return (
    <Card className="border-border shadow-sm">
      <CardHeader className="pb-2">
        <CardTitle className="text-[14px] font-semibold">
          Wallet architecture
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="grid gap-4 md:grid-cols-3">
          {items.map((item) => (
            <div key={item.title} className="flex gap-3">
              <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                {item.icon}
              </div>
              <div className="min-w-0">
                <div className="text-[13px] font-semibold">{item.title}</div>
                <p className="text-[12px] text-muted-foreground leading-relaxed mt-0.5">
                  {item.description}
                </p>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
