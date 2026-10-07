"use client";

import { CreditCard, TriangleAlert } from "lucide-react";
import type { CheckoutPayment, NetworkOption } from "@/lib/model";
import { Amount, cx } from "./ui";

export type MethodKind = "card" | "stablecoin";

function UsdcMark() {
  return (
    <span aria-hidden className="inline-grid h-5 w-5 place-items-center rounded-full text-[11px] font-semibold text-white" style={{ background: "#2775CA" }}>
      $
    </span>
  );
}

function MethodRow({ selected, onSelect, mark, title, detail, children }: {
  selected: boolean;
  onSelect: () => void;
  mark: React.ReactNode;
  title: string;
  detail?: string;
  children?: React.ReactNode;
}) {
  return (
    <div className={cx("rounded-md border transition-colors duration-[120ms]", selected ? "border-ink" : "border-line hover:border-line-strong")}>
      <label className="flex min-h-14 cursor-pointer items-center gap-3 px-4 py-3">
        <input type="radio" name="method" className="peer sr-only" checked={selected} onChange={onSelect} />
        <span aria-hidden className={cx("grid h-4 w-4 shrink-0 place-items-center rounded-full border", selected ? "border-ink" : "border-line-strong")}>
          {selected ? <span className="h-2 w-2 rounded-full bg-ink" /> : null}
        </span>
        {mark}
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="text-body font-medium text-ink">{title}</span>
          {detail ? <span className="text-caption text-ink-soft">{detail}</span> : null}
        </span>
      </label>
      {selected && children ? <div className="border-t border-line px-4 py-3">{children}</div> : null}
    </div>
  );
}

export function Methods({ payment, method, network, onMethod, onNetwork, onContinue, busy, error }: {
  payment: CheckoutPayment;
  method?: MethodKind;
  network?: string;
  onMethod: (m: MethodKind) => void;
  onNetwork: (code: string) => void;
  onContinue: () => void;
  busy?: boolean;
  error?: string;
}) {
  const stable = payment.methods.stablecoin;
  const networks: NetworkOption[] = stable?.networks ?? [];
  const chosenNetwork = networks.find((n) => n.code === network);
  const nothing = !payment.methods.card && !stable;
  const ready = method === "card" || (method === "stablecoin" && Boolean(chosenNetwork));

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-h2 font-semibold text-ink">Pay with</h1>
      {nothing ? (
        <p className="rounded-md border border-dashed border-line-strong p-4 text-body-sm text-ink-soft">No payment method is available for this link yet.</p>
      ) : (
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="sr-only">Payment method</legend>
          {payment.methods.card ? (
            <MethodRow
              selected={method === "card"}
              onSelect={() => onMethod("card")}
              mark={<CreditCard size={20} strokeWidth={1.75} className="text-ink-soft" />}
              title="Card"
              detail="Debit or credit"
            />
          ) : null}
          {stable ? (
            <MethodRow
              selected={method === "stablecoin"}
              onSelect={() => onMethod("stablecoin")}
              mark={<UsdcMark />}
              title={stable.asset}
              detail={networks.length === 1 ? `On ${networks[0].name}` : "Choose a network"}
            >
              {networks.length > 1 ? (
                <div role="radiogroup" aria-label="Network" className="flex flex-wrap gap-2">
                  {networks.map((n) => {
                    const on = n.code === network;
                    return (
                      <button
                        key={n.code}
                        type="button"
                        role="radio"
                        aria-checked={on}
                        onClick={() => onNetwork(n.code)}
                        className={cx(
                          "tap min-h-9 rounded-sm border px-3 text-body-sm font-medium transition-colors duration-[120ms]",
                          on ? "border-ink bg-ink text-ink-inverse" : "border-line-strong text-ink hover:bg-surface-sunken",
                        )}
                      >
                        {n.name}
                      </button>
                    );
                  })}
                </div>
              ) : null}
            </MethodRow>
          ) : null}
        </fieldset>
      )}

      {error ? (
        <p role="alert" className="flex items-start gap-2 rounded-sm bg-bad-tint p-3 text-body-sm text-bad">
          <TriangleAlert size={16} strokeWidth={1.75} className="mt-0.5" />
          {error}
        </p>
      ) : null}

      {!nothing ? (
        <button type="button" className="btn sticky bottom-4 w-full wide:static" disabled={!ready || busy} onClick={onContinue}>
          {busy ? "Preparing" : method === "stablecoin" && chosenNetwork ? `Pay with ${stable?.asset} on ${chosenNetwork.name}` : <>Pay <Amount money={payment.total} className="text-[15px] [&>span:last-child]:text-ink-inverse/70" /></>}
        </button>
      ) : null}
    </div>
  );
}
