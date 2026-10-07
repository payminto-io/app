"use client";

import { ArrowLeft } from "lucide-react";
import type { CheckoutPayment } from "@/lib/model";
import { Amount } from "./ui";

/**
 * Card step. The provider's hosted fields mount into the labelled container;
 * nothing here ever sees a PAN. Ticket 07 owns the mount; the mount id is the
 * contract. CLAUDE.md: no card data on these servers.
 */
export const CARD_MOUNT_ID = "card-hosted-fields";

export function CardSlot({ payment, onBack, mounted = false }: { payment: CheckoutPayment; onBack: () => void; mounted?: boolean }) {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-h2 font-semibold text-ink">Card</h1>
        <button type="button" onClick={onBack} className="tap inline-flex items-center gap-1.5 text-body-sm font-medium text-tide hover:text-tide-strong">
          <ArrowLeft size={14} strokeWidth={1.75} />
          Change method
        </button>
      </div>

      <div className="flex flex-col gap-1.5">
        <span className="text-label font-medium text-ink-soft">Card details</span>
        <div
          id={CARD_MOUNT_ID}
          data-hosted-fields-mount
          className="min-h-[148px] rounded-md border border-dashed border-line-strong bg-surface-sunken"
          aria-live="polite"
        >
          {!mounted ? (
            <div className="flex h-full min-h-[148px] flex-col items-center justify-center gap-1 p-4 text-center">
              <span className="text-body-sm font-medium text-ink">Card fields load here</span>
              <span className="text-caption text-ink-soft">Provided by the card processor. Card numbers never pass through this page.</span>
            </div>
          ) : null}
        </div>
      </div>

      <button type="button" className="btn sticky bottom-4 w-full wide:static" disabled={!mounted}>
        Pay <Amount money={payment.total} className="text-[15px] [&>span:last-child]:text-ink-inverse/70" />
      </button>
    </div>
  );
}
