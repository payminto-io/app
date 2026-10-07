"use client";

import { useEffect, useRef } from "react";
import { ArrowLeft } from "lucide-react";
import { railStep, viewState, type CheckoutPayment, type ViewState } from "@/lib/model";
import { CardSlot } from "./card-slot";
import { ChainPay } from "./chain-pay";
import { Methods, type MethodKind } from "./methods";
import { Cancelled, Confirming, Expired, Failed, Overpaid, Paid, Underpaid } from "./status";
import { Summary } from "./summary";
import { PoweredBy, Rail } from "./ui";

export interface CheckoutHandlers {
  onMethod: (m: MethodKind) => void;
  onNetwork: (code: string) => void;
  onContinue: () => void;
  onBack: () => void;
  onRetry?: () => void;
  onExpire?: () => void;
}

export interface CheckoutViewProps {
  payment: CheckoutPayment;
  method?: MethodKind;
  network?: string;
  busy?: boolean;
  error?: string;
  handlers: CheckoutHandlers;
  /** Set by the preview to force a state the data alone would not produce. */
  forceState?: ViewState;
}

function Awaiting({ payment, onBack, onExpire }: { payment: CheckoutPayment; onBack: () => void; onExpire?: () => void }) {
  const chain = payment.chain!;
  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-h2 font-semibold text-ink">
            Send {chain.asset} on {chain.networkName}
          </h1>
          <p className="text-body text-ink-soft" aria-live="polite">
            {chain.due ? "Send exactly this amount; the quote is held until the timer ends." : `The amount is counted in ${chain.asset} when it arrives.`}
          </p>
        </div>
        <button type="button" onClick={onBack} className="tap inline-flex shrink-0 items-center gap-1.5 pt-1 text-body-sm font-medium text-tide hover:text-tide-strong">
          <ArrowLeft size={14} strokeWidth={1.75} />
          Change
        </button>
      </div>
      <ChainPay chain={chain} onExpire={onExpire} />
      <Rail step={railStep(payment)} />
    </div>
  );
}

/** Pure render of one checkout state. The live container and the preview both use it. */
export function CheckoutView({ payment, method, network, busy, error, handlers, forceState }: CheckoutViewProps) {
  const state = forceState ?? viewState(payment, method);
  const panel = useRef<HTMLDivElement>(null);
  const previous = useRef(state);

  useEffect(() => {
    if (previous.current === state) return;
    previous.current = state;
    panel.current?.querySelector("h1")?.focus();
  }, [state]);

  let body: React.ReactNode;
  switch (state) {
    case "choose":
      body = <Methods payment={payment} method={method} network={network} onMethod={handlers.onMethod} onNetwork={handlers.onNetwork} onContinue={handlers.onContinue} busy={busy} error={error} />;
      break;
    case "card":
      body = <CardSlot payment={payment} onBack={handlers.onBack} />;
      break;
    case "awaiting":
      body = <Awaiting payment={payment} onBack={handlers.onBack} onExpire={handlers.onExpire} />;
      break;
    case "confirming":
      body = <Confirming payment={payment} />;
      break;
    case "paid":
      body = <Paid payment={payment} />;
      break;
    case "underpaid":
      body = <Underpaid payment={payment} onExpire={handlers.onExpire} />;
      break;
    case "overpaid":
      body = <Overpaid payment={payment} />;
      break;
    case "expired":
      body = <Expired payment={payment} onRetry={handlers.onRetry} busy={busy} />;
      break;
    case "cancelled":
      body = <Cancelled payment={payment} />;
      break;
    case "failed":
      body = <Failed payment={payment} onRetry={handlers.onRetry} busy={busy} />;
      break;
  }

  return (
    <main className="min-h-dvh">
      {payment.environment === "test" ? <div className="h-[3px] w-full bg-env-test" role="note" aria-label="Test payment" /> : null}
      <div className="mx-auto flex w-full max-w-[1040px] flex-col gap-4 px-4 py-4 wide:grid wide:grid-cols-[minmax(0,380px)_minmax(0,1fr)] wide:gap-16 wide:px-10 wide:py-16">
        <Summary payment={payment} />
        <section ref={panel} data-state={state} className="rounded-md border border-line bg-surface p-5 wide:p-8 [&_h1]:outline-none" aria-label="Payment">
          {body}
        </section>
      </div>
      <footer className="mx-auto flex w-full max-w-[1040px] items-center justify-between px-4 pb-8 wide:px-10">
        <PoweredBy />
        <span className="mono num text-caption text-ink-faint">{payment.referenceId}</span>
      </footer>
    </main>
  );
}
