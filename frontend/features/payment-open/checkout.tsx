"use client";

import { useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";

import type { PaymentOpenClient } from "./client";
import {
  PaymentOpenError,
  chainPresentation,
  type OpenPayment,
  type OpenPaymentCommand,
} from "./model";

type CheckoutState =
  | { kind: "loading" }
  | { kind: "error" }
  | { kind: "access" }
  | { kind: "not-configured" }
  | { kind: "degraded" }
  | { kind: "data"; payment: OpenPayment };

export function PaymentOpenCheckout({
  command,
  client,
}: {
  command: OpenPaymentCommand;
  client: PaymentOpenClient;
}) {
  const [state, setState] = useState<CheckoutState>({ kind: "loading" });

  useEffect(() => {
    let active = true;
    const initial = window.setTimeout(() => {
      setState({ kind: "loading" });
      client.open(command).then(
        (payment) => active && setState({ kind: "data", payment }),
        (error) => active && setState(classifyError(error)),
      );
    }, 0);
    return () => {
      active = false;
      window.clearTimeout(initial);
    };
  }, [client, command]);

  if (state.kind === "loading") {
    return <StatePanel role="status" title="Preparing secure payment instructions" />;
  }
  if (state.kind === "access") {
    return <StatePanel title="Access required" detail="Sign in or request a new payment link." />;
  }
  if (state.kind === "not-configured") {
    return (
      <StatePanel
        title="Payment method not configured"
        detail="This merchant cannot accept the selected chain and asset."
      />
    );
  }
  if (state.kind === "degraded") {
    return (
      <StatePanel
        role="alert"
        title="Payment service degraded"
        detail="Payment instructions are temporarily unavailable. It is safe to retry."
      />
    );
  }
  if (state.kind === "error") {
    return (
      <StatePanel
        role="alert"
        title="We could not prepare this payment"
        detail="No payment instruction was issued. Contact the merchant before sending funds."
      />
    );
  }
  return <PaymentInstructions payment={state.payment} />;
}

function PaymentInstructions({ payment }: { payment: OpenPayment }) {
  const [copyState, setCopyState] = useState<"idle" | "copied" | "failed">("idle");
  const chain = chainPresentation[payment.paymentMethod.chainId];
  return (
    <main className="mx-auto flex min-h-screen w-full max-w-xl flex-col gap-5 bg-white px-4 py-6 text-slate-950 sm:px-8 sm:py-10">
      <header className="space-y-2">
        <p className="text-sm font-semibold text-indigo-700">Payminto secure checkout</p>
        <h1 className="text-2xl font-bold">Awaiting payment</h1>
        <p className="text-sm text-slate-600">
          Send the exact atomic-unit obligation to the assigned address. Detection is not confirmation.
        </p>
      </header>

      <section aria-labelledby="payment-obligation" className="rounded-2xl border border-slate-200 p-5 shadow-sm">
        <h2 id="payment-obligation" className="text-base font-semibold">Payment instruction</h2>
        <dl className="mt-4 grid gap-4">
          <Value label="Required atomic units" value={payment.quote.requiredAtomicUnits} mono />
          <Value label="Asset identity" value={payment.paymentMethod.assetId} mono />
          <Value label="Network" value={chain.name} />
          <Value label="Quote valid until" value={formatUTC(payment.quote.expiresAt)} />
          <Value label="Invoice expires" value={formatUTC(payment.expiresAt)} />
          <Value label="State" value="Open — awaiting payment" />
        </dl>
      </section>

      <section aria-labelledby="deposit-address" className="rounded-2xl border border-slate-200 p-5">
        <h2 id="deposit-address" className="text-base font-semibold">Deposit address</h2>
        <div className="mx-auto my-5 w-fit rounded-xl border border-slate-200 bg-white p-3">
          <QRCodeSVG
            value={payment.depositAddress.address}
            size={176}
            level="M"
            marginSize={1}
            title="Deposit address QR code"
          />
        </div>
        <p className="break-all rounded-lg bg-slate-100 p-3 font-mono text-sm" data-testid="deposit-address-value">
          {payment.depositAddress.address}
        </p>
        <button
          type="button"
          className="mt-3 min-h-11 w-full rounded-lg bg-indigo-700 px-4 py-2 font-semibold text-white outline-offset-4 hover:bg-indigo-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-indigo-700"
          onClick={() => copyAddress(payment.depositAddress.address, setCopyState)}
        >
          Copy deposit address
        </button>
        <p className="mt-2 min-h-6 text-center text-sm" aria-live="polite">
          {copyState === "copied" ? "Address copied" : copyState === "failed" ? "Copy failed; select the address above" : ""}
        </p>
      </section>

      <aside className="rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950">
        Verify the network and full asset identity in your wallet. Do not send on an unlisted chain.
      </aside>
    </main>
  );
}

function Value({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <dt className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</dt>
      <dd className={`mt-1 break-all text-sm ${mono ? "font-mono tabular-nums" : ""}`}>{value}</dd>
    </div>
  );
}

function StatePanel({ title, detail, role }: { title: string; detail?: string; role?: "status" | "alert" }) {
  return (
    <main className="mx-auto flex min-h-[50vh] max-w-xl items-center px-4 py-10">
      <section className="w-full rounded-2xl border border-slate-200 bg-white p-6 text-center" role={role}>
        <h1 className="text-xl font-bold text-slate-950">{title}</h1>
        {detail ? <p className="mt-2 text-sm text-slate-600">{detail}</p> : null}
      </section>
    </main>
  );
}

function classifyError(error: unknown): CheckoutState {
  if (!(error instanceof PaymentOpenError)) return { kind: "error" };
  if (error.code === "unauthenticated") return { kind: "access" };
  if (error.code === "unsupported_payment_method") return { kind: "not-configured" };
  if (
    error.code === "quote_unavailable" ||
    error.code === "quote_expired" ||
    error.code === "deposit_address_unavailable" ||
    error.code === "storage_unavailable" ||
    error.code === "transport_unavailable"
  ) {
    return { kind: "degraded" };
  }
  return { kind: "error" };
}

async function copyAddress(
  address: string,
  setState: (state: "copied" | "failed") => void,
): Promise<void> {
  try {
    await navigator.clipboard.writeText(address);
    setState("copied");
  } catch {
    setState("failed");
  }
}

function formatUTC(value: string): string {
  const date = new Date(value);
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  const hour = date.getUTCHours();
  const minutes = String(date.getUTCMinutes()).padStart(2, "0");
  const displayHour = hour % 12 || 12;
  const period = hour < 12 ? "am" : "pm";
  return `${date.getUTCDate()} ${months[date.getUTCMonth()]} ${date.getUTCFullYear()}, ${displayHour}:${minutes} ${period} UTC`;
}
