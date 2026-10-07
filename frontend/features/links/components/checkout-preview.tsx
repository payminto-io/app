"use client";

import { useId, useRef, useState, type KeyboardEvent } from "react";
import { Check } from "lucide-react";
import type { RenderMethod, RenderModel } from "@/lib/api/links";
import { CurrencyDisplay } from "@/components/currency-display";
import { formatDecimal } from "@/lib/money";
import { cn } from "@/lib/utils";
import { LINKS_COPY, methodLabel } from "../copy";
import { Segmented } from "./controls";

const C = LINKS_COPY.checkout;

export type PreviewState = "loading" | "ready" | "updating" | "paused" | "failed";

/** Readable text on a merchant's accent: ink or white, whichever contrasts more. */
function onAccent(hex: string): string {
  const n = Number.parseInt(hex.slice(1), 16);
  const lin = (c: number) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  const l = 0.2126 * lin((n >> 16) & 255) + 0.7152 * lin((n >> 8) & 255) + 0.0722 * lin(n & 255);
  return l > 0.4 ? "#15181D" : "#FFFFFF";
}

/**
 * The hosted checkout drawn from the server's render model (`POST /links/preview`), and nothing else.
 * While a new model is on its way the last complete one stays, dimmed, so numbers never mix.
 */
export function CheckoutPreview({
  model,
  state,
  className,
}: {
  model: RenderModel | null;
  state: PreviewState;
  className?: string;
}) {
  const [device, setDevice] = useState<"desktop" | "phone">("desktop");
  const [tab, setTab] = useState<"pay" | "done">("pay");
  const [picked, setPicked] = useState(0);
  const base = useId();
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const tabs = ["pay", "done"] as const;
  const phone = device === "phone";

  const onTabKey = (e: KeyboardEvent, i: number) => {
    if (e.key !== "ArrowRight" && e.key !== "ArrowLeft") return;
    e.preventDefault();
    const next = (i + (e.key === "ArrowRight" ? 1 : tabs.length - 1)) % tabs.length;
    setTab(tabs[next]);
    tabRefs.current[next]?.focus();
  };

  return (
    <section aria-label={C.region} className={cn("overflow-hidden rounded-lg border border-line bg-surface-sunken", className)}>
      <div className="flex items-center justify-between gap-2 border-b border-line bg-surface px-3 py-2">
        <div role="tablist" aria-label={C.region} className="flex items-center gap-1">
          {tabs.map((t, i) => (
            <button
              key={t}
              ref={(el) => {
                tabRefs.current[i] = el;
              }}
              id={`${base}-tab-${t}`}
              role="tab"
              type="button"
              aria-selected={tab === t}
              aria-controls={`${base}-panel`}
              tabIndex={tab === t ? 0 : -1}
              onClick={() => setTab(t)}
              onKeyDown={(e) => onTabKey(e, i)}
              className={cn(
                "tap h-8 rounded-xs px-2.5 text-label font-medium transition-colors duration-120 outline-none focus-visible:outline-2 focus-visible:outline-tide",
                tab === t ? "bg-tide-tint text-ink" : "text-ink-soft hover:text-ink"
              )}
            >
              {C.tabs[t]}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-3">
          {state === "updating" || state === "loading" ? (
            <span className="text-caption text-ink-soft" aria-hidden>
              {C.updating}
            </span>
          ) : null}
          <Segmented
            label={C.device.label}
            value={device}
            className="hidden sm:inline-flex"
            options={[
              { value: "desktop", label: C.device.desktop },
              { value: "phone", label: C.device.phone },
            ]}
            onChange={setDevice}
          />
        </div>
      </div>

      <div
        id={`${base}-panel`}
        role="tabpanel"
        aria-labelledby={`${base}-tab-${tab}`}
        aria-busy={state === "updating" || state === "loading"}
        className={cn("p-3 sm:p-5", phone && "flex justify-center")}
      >
        <div className={cn("overflow-hidden rounded-md border border-line bg-surface shadow-1", phone ? "w-[340px] max-w-full" : "w-full")}>
          {state === "paused" || state === "failed" || !model ? (
            <p className="px-6 py-12 text-center text-body-sm text-ink-soft">
              {state === "failed" ? C.failed : state === "paused" ? C.paused : C.updating}
            </p>
          ) : (
            <div className={cn("transition-opacity duration-120", state === "updating" && "opacity-50")}>
              {tab === "done" ? (
                <DonePane model={model} />
              ) : (
                <div className={cn("grid", !phone && "sm:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]")}>
                  <SummaryPane model={model} method={model.methods[Math.min(picked, model.methods.length - 1)]} phone={phone} />
                  <PayPane model={model} picked={Math.min(picked, model.methods.length - 1)} onPick={setPicked} />
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function Money({ amount, currency, size = "md" }: { amount: string; currency: string; size?: "sm" | "md" | "lg" | "display" }) {
  return <CurrencyDisplay amount={amount} currency={currency} size={size} />;
}

function Merchant({ model }: { model: RenderModel }) {
  const { logo_url } = model.branding;
  if (!logo_url && !model.merchant_name) return null;
  return (
    <div className="flex items-center gap-2">
      {logo_url ? (
        // Merchant-supplied URL; the hosted checkout renders it the same way.
        // eslint-disable-next-line @next/next/no-img-element
        <img src={logo_url} alt="" className="size-6 rounded-xs border border-line object-cover" />
      ) : null}
      {model.merchant_name ? <span className="text-body-sm font-medium text-ink">{model.merchant_name}</span> : null}
    </div>
  );
}

function Line({ label, amount, currency, strong }: { label: string; amount: string | null; currency: string; strong?: boolean }) {
  if (!amount) return null;
  return (
    <div className="flex items-baseline justify-between gap-3">
      <dt className={strong ? "font-semibold text-ink" : "text-ink-soft"}>{label}</dt>
      <dd>
        <Money amount={amount} currency={currency} size={strong ? "md" : "sm"} />
      </dd>
    </div>
  );
}

function rangeText(model: RenderModel): string {
  const f = (v: string | null) => (v ? `${formatDecimal(v, model.currency)} ${model.currency}` : null);
  const min = f(model.amount_min);
  const max = f(model.amount_max);
  if (min && max) return `${min} ${C.to} ${max}`;
  if (min) return `${C.from} ${min}`;
  if (max) return `${C.upTo} ${max}`;
  return "";
}

function SummaryPane({ model, method, phone }: { model: RenderModel; method?: RenderMethod; phone: boolean }) {
  const cur = model.currency;
  const surcharge = method?.customer_total ? method : null;
  const items = model.amount_mode === "line_items";
  const due = surcharge?.customer_total ?? model.amount;
  const hero = model.amount_mode === "fixed" && model.amount && !surcharge;
  return (
    <div className={cn("space-y-4 border-b border-line p-5", !phone && "sm:border-r sm:border-b-0")}>
      <Merchant model={model} />
      <div className="space-y-1">
        <p className={cn("text-body-sm", model.title ? "text-ink-soft" : "text-ink-faint")}>{model.title || C.untitled}</p>
        {!cur ? (
          <p className="text-body-sm text-ink-faint">{C.addCurrency}</p>
        ) : hero && model.amount ? (
          <Money amount={model.amount} currency={cur} size={formatDecimal(model.amount, cur).length > 6 ? "lg" : "display"} />
        ) : model.amount_mode === "customer" ? (
          <p className="num text-body-sm text-ink-soft">{rangeText(model)}</p>
        ) : null}
      </div>
      {model.description ? <p className="text-body-sm whitespace-pre-line text-ink-soft">{model.description}</p> : null}

      {items && model.line_items.length > 0 ? (
        <ul className="divide-y divide-line border-t border-line">
          {model.line_items.map((li, i) => (
            <li key={i} className="flex items-baseline justify-between gap-3 py-2">
              <span className="min-w-0">
                <span className="block truncate text-body-sm text-ink">{li.name || C.untitled}</span>
                {cur ? (
                  <span className="num text-caption text-ink-soft">
                    {li.quantity} x {formatDecimal(li.unit_price, cur)} {cur}
                  </span>
                ) : null}
              </span>
              {cur ? <Money amount={li.subtotal} currency={cur} size="sm" /> : null}
            </li>
          ))}
        </ul>
      ) : null}

      {cur && (items || surcharge) ? (
        <dl className="space-y-1.5 border-t border-line pt-3 text-body-sm">
          {items ? (
            <>
              <Line label={C.subtotal} amount={model.subtotal} currency={cur} />
              {model.tax_total && Number(model.tax_total) > 0 ? <Line label={C.tax} amount={model.tax_total} currency={cur} /> : null}
            </>
          ) : (
            <Line label={C.amount} amount={model.amount} currency={cur} />
          )}
          {surcharge ? (
            <>
              <Line label={C.fee} amount={surcharge.fee} currency={cur} />
              {surcharge.tax && Number(surcharge.tax) > 0 ? <Line label={C.feeTax} amount={surcharge.tax} currency={cur} /> : null}
            </>
          ) : null}
          <div className="border-t border-line pt-2">
            <Line label={C.totalDue} amount={due} currency={cur} strong />
          </div>
        </dl>
      ) : null}

      {model.expires_at ? (
        <p className="num text-caption text-ink-soft">
          {C.ends} {new Date(model.expires_at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })}
        </p>
      ) : null}
    </div>
  );
}

function MockLabel({ text, optional }: { text: string; optional?: boolean }) {
  return (
    <p className="text-label text-ink-soft">
      {text}
      {optional ? <span className="text-ink-faint"> ({C.optional})</span> : null}
    </p>
  );
}

function PayPane({ model, picked, onPick }: { model: RenderModel; picked: number; onPick: (i: number) => void }) {
  const f = model.customer_fields;
  const contact = (["email", "name", "phone"] as const).filter((k) => f[k].mode !== "hidden");
  const method = model.methods[picked];
  const due = method?.customer_total ?? model.amount;
  const accent = model.branding.accent_color;
  return (
    <div className="space-y-4 p-5">
      {model.amount_mode === "customer" ? (
        <div className="space-y-1">
          <MockLabel text={C.customerAmount} />
          <div className="h-9 rounded-sm border border-line-strong bg-surface" />
        </div>
      ) : null}

      {contact.length > 0 ? (
        <div className="space-y-2">
          <p className="text-label font-medium text-ink">{C.contact}</p>
          {contact.map((k) => (
            <div key={k} className="space-y-1">
              <MockLabel text={C.contactLabels[k]} optional={f[k].mode === "optional"} />
              <div className="flex h-9 items-center rounded-sm border border-line-strong bg-surface px-3 text-body-sm text-ink">{f[k].prefill ?? ""}</div>
            </div>
          ))}
        </div>
      ) : null}

      {model.billing_required ? <AddressBlock title={C.billing} /> : null}
      {model.shipping_required ? <AddressBlock title={C.shipping} /> : null}

      {model.questions.map((q) => (
        <div key={q.key} className="space-y-1">
          {q.type === "checkbox" ? (
            <div className="flex items-center gap-2">
              <span className="size-4 rounded-xs border border-line-strong" aria-hidden />
              <span className="text-body-sm text-ink">{q.label}</span>
            </div>
          ) : (
            <>
              <MockLabel text={q.label} optional={!q.required} />
              <div className="flex h-9 items-center rounded-sm border border-line-strong bg-surface px-3 text-body-sm text-ink-faint">
                {q.type === "select" ? (q.options[0] ?? "") : ""}
              </div>
            </>
          )}
        </div>
      ))}

      <div className="space-y-2">
        <p className="text-label font-medium text-ink">{C.payWith}</p>
        {model.methods.length === 0 ? (
          <p className="rounded-sm border border-dashed border-line-strong px-3 py-2.5 text-body-sm text-ink-faint">{C.noMethods}</p>
        ) : (
          <div role="radiogroup" aria-label={C.payWith} className="grid grid-cols-2 gap-2">
            {model.methods.map((m, i) => (
              <button
                key={`${m.method}-${m.chain}-${m.asset}`}
                type="button"
                role="radio"
                aria-checked={i === picked}
                onClick={() => onPick(i)}
                className={cn(
                  "tap flex min-h-11 items-center gap-2 rounded-sm border px-3 text-left text-body-sm transition-colors duration-120 outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide",
                  i === picked ? "border-ink text-ink" : "border-line-strong text-ink-soft hover:text-ink"
                )}
              >
                <span aria-hidden className={cn("size-3.5 shrink-0 rounded-full border", i === picked ? "border-[5px] border-ink" : "border-line-strong")} />
                <span className="min-w-0">
                  <span className="block truncate">{m.method === "crypto" ? m.asset : methodLabel(m)}</span>
                  {m.method === "crypto" && m.chain ? <span className="block truncate text-caption text-ink-soft">{m.chain}</span> : null}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      {model.available ? (
        <div
          className={cn("num flex h-11 items-center justify-center rounded-sm px-3 text-body font-medium", !accent && "bg-ink text-ink-inverse")}
          style={accent ? { background: accent, color: onAccent(accent) } : undefined}
        >
          {C.pay}
          {due && model.currency ? ` ${formatDecimal(due, model.currency)} ${model.currency}` : ""}
        </div>
      ) : (
        <p className="rounded-sm border border-wait/30 bg-wait-tint px-3 py-2.5 text-body-sm text-ink">
          {model.unavailable_reason ? C.unavailable[model.unavailable_reason] : null}
        </p>
      )}
    </div>
  );
}

function AddressBlock({ title }: { title: string }) {
  return (
    <div className="space-y-2">
      <p className="text-label font-medium text-ink">{title}</p>
      <div className="grid grid-cols-2 gap-2">
        <div className="col-span-2 h-9 rounded-sm border border-line-strong" />
        <div className="h-9 rounded-sm border border-line-strong" />
        <div className="h-9 rounded-sm border border-line-strong" />
      </div>
    </div>
  );
}

function DonePane({ model }: { model: RenderModel }) {
  return (
    <div className="flex flex-col items-center gap-3 px-6 py-12 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-ok-tint text-ok">
        <Check className="size-5" aria-hidden />
      </span>
      {model.success_mode === "redirect" ? (
        <p className="text-body text-ink">{C.redirect}</p>
      ) : (
        <p className="max-w-[40ch] text-body whitespace-pre-line text-ink">{model.success_message || C.paid}</p>
      )}
      {model.receipt_email ? <p className="text-caption text-ink-soft">{C.receipt}</p> : null}
    </div>
  );
}
