"use client";

/**
 * Deposit (HD) wallets for the current platform, one per blockchain family.
 * Each row opens its pre-generated address pool. Backed by GET /wallets.
 */
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useWalletsList, type Wallet } from "@/lib/query/hooks/use-wallets";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { WalletTabs } from "./_components/wallet-tabs";

const FAMILY_LABEL: Record<string, string> = {
  evm: "EVM",
  eth_family: "EVM",
  btc: "Bitcoin",
  btc_family: "Bitcoin",
  trx: "Tron",
  trx_family: "Tron",
};

function familyLabel(wallet: Wallet): string | undefined {
  for (const c of [wallet.blockchainFamily?.code, wallet.blockchainFamily?.family]) {
    if (c && FAMILY_LABEL[c.toLowerCase()]) return FAMILY_LABEL[c.toLowerCase()];
  }
  return wallet.blockchainFamily?.name;
}

function kindLabel(kind: string): string {
  const key = kind.toLowerCase();
  if (key === "hd") return "HD";
  if (key === "hot") return "Hot";
  if (key === "cold") return "Cold";
  if (key === "sc" || key === "scw") return "Smart contract";
  return kind;
}

const COLUMNS: DataTableColumn<Wallet>[] = [
  {
    key: "name", stack: "lead",
    header: "Wallet",
    cell: (w) => (
      <Link
        href={`/dashboard/wallets/${w.id}/addresses`}
        onClick={(e) => e.stopPropagation()}
        className="tap rounded-xs font-medium text-ink hover:text-tide"
      >
        {w.name}
      </Link>
    ),
  },
  { key: "family", stack: "meta", header: "Chain", className: "text-ink-soft", cell: (w) => familyLabel(w) ?? null },
  { key: "kind", stack: "meta", header: "Type", className: "text-ink-soft", cell: (w) => kindLabel(w.kind) },
  {
    key: "path", stack: "detail",
    header: "Derivation path",
    cell: (w) =>
      w.blockchainFamily?.path ? (
        <span className="font-mono text-label text-ink-soft">{w.blockchainFamily.path}</span>
      ) : null,
  },
  { key: "status", stack: "trail", header: "Status", cell: (w) => <StatusBadge status={w.status} /> },
  {
    key: "addresses", stack: "detail",
    header: "Addresses",
    align: "right",
    cell: (w) => w.addressCount.toLocaleString("en-US"),
  },
];

export default function WalletsPage() {
  const router = useRouter();
  const { data, error, refetch } = useWalletsList();

  return (
    <div className="space-y-5">
      <PageHeader title="Wallets" />
      <WalletTabs />
      {error ? (
        <ErrorState message={error.message} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data?.wallets ?? []}
          loading={!data}
          getRowId={(w) => w.id}
          onRowClick={(w) => router.push(`/dashboard/wallets/${w.id}/addresses`)}
          emptyTitle="No deposit wallets yet."
          emptyDescription="A wallet appears here once a chain is enabled for this account."
        />
      )}
    </div>
  );
}
