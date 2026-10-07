"use client";

/**
 * Address pool of one HD wallet, filterable by status. The wallet summary
 * comes from the cached wallets list. Backed by GET /wallets and
 * GET /wallets/:id/addresses.
 */
import { use, useMemo, useState } from "react";
import { useWalletAddresses, useWalletsList, type AddressPoolItem } from "@/lib/query/hooks/use-wallets";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Pagination } from "@/components/pagination";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

const PAGE_SIZE = 20;

const STATUS_TABS = [
  { value: "all", label: "All" },
  { value: "available", label: "Available" },
  { value: "used", label: "Used" },
  { value: "locked", label: "Locked" },
] as const;

type StatusTabValue = (typeof STATUS_TABS)[number]["value"];

function truncateMiddle(value: string, head: number, tail: number): string {
  if (value.length <= head + tail + 3) return value;
  return `${value.slice(0, head)}...${value.slice(-tail)}`;
}

const COLUMNS: DataTableColumn<AddressPoolItem>[] = [
  {
    key: "address", stack: "lead",
    header: "Address",
    cell: (a) => (
      <CopyField value={a.address} display={truncateMiddle(a.address, 10, 8)} boxed={false} className="max-w-[280px]" />
    ),
  },
  { key: "index", stack: "meta", header: "Index", align: "right", className: "text-ink-soft", cell: (a) => a.pathIndex },
  { key: "status", stack: "trail", header: "Status", cell: (a) => <StatusBadge status={a.status} /> },
  {
    key: "created", stack: "meta",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (a) => <DateTime value={a.createdAt} />,
  },
];

export default function WalletAddressesPage({ params }: { params: Promise<{ id: string }> }) {
  const { id: rawId } = use(params);
  const walletID = Number(rawId);
  const valid = Number.isFinite(walletID) && walletID > 0;

  const [tab, setTab] = useState<StatusTabValue>("all");
  const [offset, setOffset] = useState(0);

  const { data: walletsData } = useWalletsList();
  const wallet = useMemo(() => walletsData?.wallets.find((w) => w.id === walletID), [walletsData, walletID]);

  const { data, error, refetch } = useWalletAddresses(valid ? walletID : undefined, {
    status: tab === "all" ? undefined : tab,
    limit: PAGE_SIZE,
    offset,
  });

  const crumbs = [{ label: "Wallets", href: "/dashboard/wallets" }, { label: wallet?.name ?? `Wallet ${rawId}` }];

  if (!valid) {
    return (
      <div className="space-y-5">
        <PageHeader breadcrumbs={crumbs} title="Wallet not found" />
        <ErrorState message={`There is no wallet with id ${rawId}.`} />
      </div>
    );
  }

  const meta = [
    wallet?.blockchainFamily?.name,
    wallet ? `${wallet.addressCount.toLocaleString("en-US")} addresses` : undefined,
  ].filter(Boolean);

  return (
    <div className="space-y-5">
      <PageHeader
        breadcrumbs={crumbs}
        title={wallet?.name ?? `Wallet ${walletID}`}
        description={meta.length ? meta.join(", ") : undefined}
      >
        {wallet ? <StatusBadge status={wallet.status} /> : null}
      </PageHeader>

      <div className="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
        <Tabs
          value={tab}
          onValueChange={(v) => {
            setTab(v as StatusTabValue);
            setOffset(0);
          }}
        >
          <TabsList variant="line">
            {STATUS_TABS.map((t) => (
              <TabsTrigger key={t.value} value={t.value}>
                {t.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {error ? (
        <ErrorState message={error.message} retry={() => refetch()} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data?.addresses ?? []}
          loading={!data}
          getRowId={(a) => a.id}
          total={data?.total}
          footer={
            data && data.total > PAGE_SIZE ? (
              <Pagination offset={offset} limit={PAGE_SIZE} total={data.total} onOffsetChange={setOffset} />
            ) : undefined
          }
          emptyTitle={tab === "all" ? "This pool has no addresses yet." : `No ${tab} addresses.`}
        />
      )}
    </div>
  );
}
