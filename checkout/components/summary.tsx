"use client";

import { useId, useState } from "react";
import { ChevronDown } from "lucide-react";
import type { CheckoutPayment } from "@/lib/model";
import { Amount, cx } from "./ui";

/* eslint-disable @next/next/no-img-element -- merchant logos are arbitrary remote URLs */
export function MerchantMark({ merchant, size = 32 }: { merchant: CheckoutPayment["merchant"]; size?: number }) {
  const initial = (merchant.name ?? "").trim().slice(0, 1).toUpperCase();
  if (merchant.logoUrl) {
    return <img src={merchant.logoUrl} alt="" width={size} height={size} className="shrink-0 rounded-sm object-cover" style={{ width: size, height: size }} />;
  }
  return (
    <span
      aria-hidden
      className="inline-grid shrink-0 place-items-center rounded-sm bg-ink text-ink-inverse font-semibold"
      style={{ width: size, height: size, fontSize: Math.round(size * 0.44) }}
    >
      {initial || "·"}
    </span>
  );
}

function Lines({ payment, label }: { payment: CheckoutPayment; label: string }) {
  const items = payment.lineItems ?? [];
  return (
    <dl className="flex flex-col">
      {items.map((item, i) => (
        <div key={i} className="flex items-baseline justify-between gap-4 py-2 text-body-sm">
          <dt className="min-w-0 text-ink">
            {item.name}
            {item.quantity && item.quantity > 1 ? <span className="num text-ink-soft"> × {item.quantity}</span> : null}
          </dt>
          <dd className="m-0 shrink-0"><Amount money={item.amount} size="body-sm" /></dd>
        </div>
      ))}
      {payment.description && items.length === 0 ? <p className="py-2 text-body-sm text-ink-soft">{payment.description}</p> : null}
      <div className={cx("flex items-baseline justify-between gap-4 py-2 text-body", (items.length > 0 || payment.description) && "mt-1 border-t border-line pt-3")}>
        <dt className="font-medium text-ink">{label}</dt>
        <dd className="m-0"><Amount money={payment.total} /></dd>
      </div>
    </dl>
  );
}

/** Wide: the left column. Narrow: a header row that discloses the lines. DESIGN.md section 11. */
export function Summary({ payment }: { payment: CheckoutPayment }) {
  const [open, setOpen] = useState(false);
  const id = useId();
  const hasDetail = (payment.lineItems?.length ?? 0) > 0 || Boolean(payment.description);
  const name = payment.merchant.name;
  const totalLabel = payment.state === "open" || payment.state === "confirming" || payment.state === "underpaid" ? "Total due" : "Total";

  return (
    <>
      <aside className="hidden wide:flex wide:flex-col wide:gap-8">
        <div className="flex items-center gap-3">
          <MerchantMark merchant={payment.merchant} size={36} />
          <div className="flex min-w-0 flex-col">
            {name ? <span className="truncate text-body font-medium text-ink">{name}</span> : null}
            <span className="text-label text-ink-soft">Payment</span>
          </div>
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-label text-ink-soft">{totalLabel}</span>
          <Amount money={payment.total} size="display" />
        </div>
        {hasDetail ? <Lines payment={payment} label={totalLabel} /> : null}
      </aside>

      <div className="wide:hidden rounded-md border border-line bg-surface">
        <button
          type="button"
          className="flex w-full items-center gap-3 p-4 text-left"
          aria-expanded={hasDetail ? open : undefined}
          aria-controls={hasDetail ? id : undefined}
          onClick={() => hasDetail && setOpen((v) => !v)}
          disabled={!hasDetail}
          style={{ cursor: hasDetail ? "pointer" : "default" }}
        >
          <MerchantMark merchant={payment.merchant} size={32} />
          <span className="flex min-w-0 flex-1 flex-col">
            {name ? <span className="truncate text-body-sm font-medium text-ink">{name}</span> : null}
            <span className="text-caption text-ink-soft">{totalLabel}</span>
          </span>
          <Amount money={payment.total} size="h2" />
          {hasDetail ? (
            <ChevronDown size={16} strokeWidth={1.75} className={cx("text-ink-soft transition-transform duration-[140ms]", open && "rotate-180")} aria-hidden />
          ) : null}
        </button>
        {hasDetail && open ? (
          <div id={id} className="disclose border-t border-line px-4 pb-2 pt-1">
            <Lines payment={payment} label={totalLabel} />
          </div>
        ) : null}
      </div>
    </>
  );
}
