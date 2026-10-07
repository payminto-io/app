"use client";

import { useEffect } from "react";
import { ArrowUp, Check, Clock3, Minus, X } from "lucide-react";
import { overage, railStep, remainingDue, type CheckoutPayment } from "@/lib/model";
import { formatAmount } from "@/lib/money";
import { ChainPay } from "./chain-pay";
import { Amount, cx, ExplorerLink, formatWhen, Pulse, Rail, Row, truncateMiddle } from "./ui";

type Tone = "ok" | "wait" | "bad" | "note" | "mute";

const TONE: Record<Tone, string> = {
  ok: "bg-ok-tint text-ok",
  wait: "bg-wait-tint text-wait",
  bad: "bg-bad-tint text-bad",
  note: "bg-note-tint text-note",
  mute: "bg-mute-tint text-mute",
};

function Glyph({ tone, icon }: { tone: Tone; icon: React.ReactNode }) {
  return <span className={cx("grid h-10 w-10 shrink-0 place-items-center rounded-full", TONE[tone])}>{icon}</span>;
}

/** Heading plus the state sentence; the sentence is the live region. DESIGN.md section 12. */
function Headline({ tone, icon, title, sentence }: { tone: Tone; icon: React.ReactNode; title: string; sentence: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3">
      <Glyph tone={tone} icon={icon} />
      <div className="flex min-w-0 flex-col gap-1 pt-1">
        <h1 className="text-h2 font-semibold text-ink">{title}</h1>
        <p className="text-body text-ink-soft" aria-live="polite">{sentence}</p>
      </div>
    </div>
  );
}

function Receipt({ payment }: { payment: CheckoutPayment }) {
  const chain = payment.chain;
  const paid = chain?.received ?? chain?.due;
  return (
    <div className="flex flex-col divide-y divide-line rounded-md border border-line px-4">
      {payment.merchant.name ? <Row label="To">{payment.merchant.name}</Row> : null}
      {paid ? <Row label="Paid"><Amount money={paid} size="body-sm" /></Row> : <Row label="Amount due"><Amount money={payment.total} size="body-sm" /></Row>}
      {payment.card?.brand || payment.card?.last4 ? (
        <Row label="Card">{[payment.card.brand, payment.card.last4 ? `•••• ${payment.card.last4}` : undefined].filter(Boolean).join(" ")}</Row>
      ) : null}
      {chain ? <Row label="Network">{chain.networkName}</Row> : null}
      {chain?.txHash ? (
        <Row label="Transaction" mono>
          {chain.explorerUrl ? <ExplorerLink href={chain.explorerUrl}>{truncateMiddle(chain.txHash)}</ExplorerLink> : truncateMiddle(chain.txHash)}
        </Row>
      ) : null}
      {payment.paidAt ? <Row label="Date">{formatWhen(payment.paidAt)}</Row> : null}
      <Row label="Reference" mono>{payment.referenceId}</Row>
    </div>
  );
}

export function Confirming({ payment }: { payment: CheckoutPayment }) {
  const chain = payment.chain;
  const count = chain?.confirmations !== undefined && chain.requiredConfirmations !== undefined ? `${chain.confirmations} of ${chain.requiredConfirmations}` : undefined;
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="wait"
        icon={<Pulse />}
        title="Confirming"
        sentence={
          chain ? (
            <>
              Seen on {chain.networkName}.{count ? <> Confirmations <span className="num">{count}</span>.</> : null} Keep this page open, or close it; the merchant is told either way.
            </>
          ) : (
            "Seen. Keep this page open, or close it; the merchant is told either way."
          )
        }
      />
      <Rail step={railStep(payment)} />
      {chain?.received ? <Row label="Amount"><Amount money={chain.received} size="body-sm" /></Row> : null}
      {chain?.explorerUrl ? <ExplorerLink href={chain.explorerUrl} /> : null}
    </div>
  );
}

export function Paid({ payment }: { payment: CheckoutPayment }) {
  const redirect = payment.success?.redirectUrl;
  useEffect(() => {
    if (!redirect) return;
    const t = window.setTimeout(() => window.location.assign(redirect), 1500);
    return () => window.clearTimeout(t);
  }, [redirect]);
  const paid = payment.chain?.received ?? payment.chain?.due ?? payment.total;
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="ok"
        icon={<Check size={20} strokeWidth={2} />}
        title="Paid"
        sentence={
          payment.success?.message ?? (
            <>
              <Amount money={paid} size="body-sm" className="text-ink" /> {payment.merchant.name ? `to ${payment.merchant.name} ` : ""}is final.
              {redirect ? " Taking you back." : ""}
            </>
          )
        }
      />
      <Rail step={railStep(payment)} timestamps={payment.paidAt ? [undefined, formatWhen(payment.paidAt), undefined] : undefined} />
      <Receipt payment={payment} />
      {redirect ? (
        <a href={redirect} className="btn btn-outline w-full">Continue</a>
      ) : null}
    </div>
  );
}

export function Underpaid({ payment, onExpire }: { payment: CheckoutPayment; onExpire?: () => void }) {
  const chain = payment.chain;
  const remaining = remainingDue(chain);
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="wait"
        icon={<Clock3 size={20} strokeWidth={1.75} />}
        title="Under paid"
        sentence={
          chain?.received && chain.due && remaining ? (
            <>
              Received <Amount money={chain.received} size="body-sm" className="text-ink" /> of <Amount money={chain.due} size="body-sm" className="text-ink" />. Send the remaining{" "}
              <Amount money={remaining} size="body-sm" className="text-ink" /> to the same address.
            </>
          ) : (
            "Part of the amount has arrived. Send the rest to the same address."
          )
        }
      />
      <Rail step={railStep(payment)} />
      {chain ? <ChainPay chain={chain} due={remaining} onExpire={onExpire} note="Waiting for the rest" /> : null}
      {chain?.explorerUrl ? <ExplorerLink href={chain.explorerUrl} /> : null}
    </div>
  );
}

export function Overpaid({ payment }: { payment: CheckoutPayment }) {
  const chain = payment.chain;
  const extra = overage(chain);
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="note"
        icon={<ArrowUp size={20} strokeWidth={1.75} />}
        title="Paid, with extra"
        sentence={
          chain?.received && chain.due && extra ? (
            <>
              Received <Amount money={chain.received} size="body-sm" className="text-ink" /> against <Amount money={chain.due} size="body-sm" className="text-ink" />. The payment is complete;{" "}
              {payment.merchant.name ?? "the merchant"} holds the extra <Amount money={extra} size="body-sm" className="text-ink" />.
            </>
          ) : (
            <>More than the amount due arrived. The payment is complete; {payment.merchant.name ?? "the merchant"} holds the difference.</>
          )
        }
      />
      <Rail step={railStep(payment)} />
      <Receipt payment={payment} />
    </div>
  );
}

export function Expired({ payment, onRetry, busy }: { payment: CheckoutPayment; onRetry?: () => void; busy?: boolean }) {
  const chain = payment.chain;
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="mute"
        icon={<Minus size={20} strokeWidth={1.75} />}
        title="Expired"
        sentence={
          chain
            ? `This ${chain.asset} quote ran out before a transfer arrived. Do not send to the old address.`
            : "This payment link ran out of time."
        }
      />
      {payment.retryAllowed && onRetry ? (
        <button type="button" className="btn w-full" onClick={onRetry} disabled={busy}>{busy ? "Preparing" : "Get a new quote"}</button>
      ) : (
        <p className="text-body-sm text-ink-soft">Ask {payment.merchant.name ?? "the merchant"} for a new link.</p>
      )}
    </div>
  );
}

export function Cancelled({ payment }: { payment: CheckoutPayment }) {
  return (
    <div className="flex flex-col gap-6">
      <Headline tone="bad" icon={<X size={20} strokeWidth={1.75} />} title="Cancelled" sentence={`${payment.merchant.name ?? "The merchant"} cancelled this payment. Nothing is due.`} />
      <Row label="Reference" mono>{payment.referenceId}</Row>
    </div>
  );
}

export function Failed({ payment, onRetry, busy }: { payment: CheckoutPayment; onRetry?: () => void; busy?: boolean }) {
  return (
    <div className="flex flex-col gap-6">
      <Headline
        tone="bad"
        icon={<X size={20} strokeWidth={1.75} />}
        title="Payment failed"
        sentence={payment.failureReason ?? "The payment did not go through. Nothing was charged."}
      />
      <Rail step={0} failed />
      {payment.retryAllowed && onRetry ? (
        <button type="button" className="btn w-full" onClick={onRetry} disabled={busy}>Try another method</button>
      ) : null}
      <Row label="Reference" mono>{payment.referenceId}</Row>
    </div>
  );
}

export function dueLabel(payment: CheckoutPayment): string {
  return `${formatAmount(payment.total.amount, payment.total.code)} ${payment.total.code}`;
}
