"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Plus } from "lucide-react";
import type { LinkStatus, PaymentLink } from "@/lib/api/links";
import { useLinksList } from "@/lib/query/hooks/use-links";
import { PageHeader } from "@/components/page-header";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { Pagination } from "@/components/pagination";
import { CurrencyDisplay } from "@/components/currency-display";
import { DateTime } from "@/components/date-time";
import { StatusBadge } from "@/components/status-badge";
import { buttonVariants } from "@/components/ui/button";
import { ErrorState } from "@/components/ui/states";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { LINKS_COPY } from "../copy";
import { effectiveUseLimit } from "../model";
import { ShortLink } from "./link-share";

const PAGE_SIZE = 25;
const TABS = ["all", "active", "draft", "paused", "archived"] as const;
const C = LINKS_COPY.columns;

function columns(basePath: string): DataTableColumn<PaymentLink>[] {
  return [
    {
      key: "title",
      stack: "lead",
      header: C.title,
      cell: (l) => (
        <Link
          href={`${basePath}/${l.id}`}
          onClick={(e) => e.stopPropagation()}
          className="tap block max-w-[28ch] truncate rounded-xs font-medium text-ink hover:text-tide"
        >
          {l.title || LINKS_COPY.untitled}
        </Link>
      ),
    },
    { key: "status", stack: "trail", header: C.status, cell: (l) => <StatusBadge status={l.status} /> },
    {
      key: "amount",
      stack: "meta",
      header: C.amount,
      align: "right",
      cell: (l) =>
        l.total && l.currency ? (
          <CurrencyDisplay amount={l.total} currency={l.currency} size="sm" />
        ) : l.amount_mode === "customer" ? (
          <span className="text-body-sm text-ink-soft">{LINKS_COPY.customerSets}</span>
        ) : null,
    },
    {
      key: "uses",
      stack: "detail",
      header: C.uses,
      align: "right",
      cell: (l) => {
        const cap = effectiveUseLimit(l);
        return (
          <span className="num text-body-sm text-ink">
            {l.uses_count}
            {cap !== null ? <span className="text-ink-soft"> / {cap}</span> : null}
          </span>
        );
      },
    },
    {
      key: "url",
      stack: "detail",
      header: C.url,
      cell: (l) => (l.url ? <ShortLink url={l.url} title={l.title} className="max-w-[30ch]" /> : null),
    },
    {
      key: "created",
      stack: "meta",
      header: C.created,
      align: "right",
      className: "text-ink-soft",
      cell: (l) => <DateTime value={l.created_at} format="date" />,
    },
  ];
}

export function LinksListPage({ basePath = "/dashboard/links" }: { basePath?: string }) {
  const router = useRouter();
  const [tab, setTab] = useState<(typeof TABS)[number]>("all");
  const [offset, setOffset] = useState(0);
  const status: LinkStatus | undefined = tab === "all" ? undefined : tab;
  const { data, error, refetch } = useLinksList({ status, limit: PAGE_SIZE, offset });

  return (
    <div className="space-y-5">
      <PageHeader title={LINKS_COPY.listTitle}>
        <Link href={`${basePath}/new`} className={buttonVariants()}>
          <Plus />
          {LINKS_COPY.newLink}
        </Link>
      </PageHeader>

      <div className="-mx-4 min-w-0 overflow-x-auto px-4 sm:mx-0 sm:px-0">
        <Tabs
          value={tab}
          onValueChange={(v) => {
            setTab(v as (typeof TABS)[number]);
            setOffset(0);
          }}
        >
          <TabsList variant="line">
            {TABS.map((t) => (
              <TabsTrigger key={t} value={t}>
                {LINKS_COPY.tabs[t]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={columns(basePath)}
          rows={data?.links ?? []}
          loading={!data}
          getRowId={(l) => l.id}
          onRowClick={(l) => router.push(`${basePath}/${l.id}`)}
          total={data?.total}
          footer={
            data && data.total > PAGE_SIZE ? (
              <Pagination offset={offset} limit={PAGE_SIZE} total={data.total} onOffsetChange={setOffset} />
            ) : undefined
          }
          emptyTitle={tab === "all" ? LINKS_COPY.emptyTitle : LINKS_COPY.emptyFiltered}
          emptyDescription={tab === "all" ? LINKS_COPY.emptyBody : undefined}
          emptyAction={
            tab === "all" ? (
              <Link href={`${basePath}/new`} className={buttonVariants({ size: "sm" })}>
                {LINKS_COPY.newLink}
              </Link>
            ) : undefined
          }
        />
      )}
    </div>
  );
}
