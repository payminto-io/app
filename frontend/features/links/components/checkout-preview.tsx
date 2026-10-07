"use client";

import { useState } from "react";
import { Check, Monitor, Smartphone } from "lucide-react";
import type { RenderMethod, RenderModel } from "@/lib/api/links";
import { CurrencyDisplay } from "@/components/currency-display";
import { cn } from "@/lib/utils";
import { formatDecimal } from "@/lib/money";
import { LINKS_COPY, methodLabel } from "../copy";

const C = LINKS_COPY.checkout;

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

function Money({ amount, currency, size = "md" }: { amount: string; currency: string; size?: "sm" | "md" | "lg" | "display" }) {
  return <CurrencyDisplay amount={amount} currency={currency || " "} size={size} />;
}

function MockInput({ label, optional, wide = true }: { label: string; optional?: boolean; wide?: boolean }) {
  return (
    <div className={cn("space-y-1", wide && "col-span-2")}>
      <p className="text-label text-ink-soft">
        {label}
        {optional ? <span className="text-ink-faint"> ({C.optional.toLowerCase()})</span> : null}
      </p>
      <div className="h-9 rounded-sm border border-line-strong bg-surface" />
    </div>
  );
}

/**
 * The hosted checkout, drawn from a render model (docs/API_SPECIFICATION.md 4.44).
 * It shows only what the model carries, so a preview cannot promise more than checkout does.
 */
export function CheckoutPreview({ model, className }: { model: RenderModel; className?: string }) {
  const [device, setDevice] = useState<"desktop" | "phone">("desktop");
  const [tab, setTab] = useState<"pay" | "done">("pay");
  const [picked, setPicked] = useState(0);
  const method: RenderMethod | undefined = model.methods[Math.min(picked, model.methods.length - 1)];
  const accent = model.branding.accent_color;
  const phone = device === "phone";

  return (
    <div className={cn("overflow-hidden rounded-lg border border-line bg-surface-sunken", className)}>
      <div className="flex items-center justify-between gap-2 border-b border-line bg-surface px-3 py-2">
        <div role="tablist" aria-label={LINKS_COPY.builder.preview} className="flex items-center gap-1">
          {(["pay", "done"] as const).map((t) => (
            <button
              key={t}
              role="tab"
              type="button"
              aria-selected={tab === t}
              onClick={() => setTab(t)}
              className={cn(
                "tap h-7 rounded-xs px-2.5 text-label font-medium transition-colors duration-120 outline-none focus-visible:outline-2 focus-visible:outline-tide",
                tab === t ? "bg-tide-tint text-ink" : "text-ink-soft hover:text-ink"
              )}
            >
              {C.tabs[t]}
            </button>
          ))}
        </div>
        <div role="radiogroup" aria-label="Device" className="hidden items-center gap-0.5 sm:flex">
          {(["desktop", "phone"] as const).map((d) => {
            const Icon = d === "desktop" ? Monitor : Smartphone;
            return (
              <button
                key={d}
                type="button"
                role="radio"
                aria-checked={device === d}
                aria-label={C.device[d]}
                onClick={() => setDevice(d)}
                className={cn(
                  "tap flex size-7 items-center justify-center rounded-xs transition-colors duration-120 outline-none focus-visible:outline-2 focus-visible:outline-tide",
                  device === d ? "bg-tide-tint text-ink" : "text-ink-faint hover:text-ink"
                )}
              >
                <Icon className="size-4" aria-hidden />
              </button>
            );
          })}
        </div>
      </div>

      <div className={cn("p-3 sm:p-5", phone && "flex justify-center")}>
        <div
          aria-label="Checkout preview"
          className={cn(
            "overflow-hidden rounded-md border border-line bg-surface shadow-1",
            phone ? "w-[340px] max-w-full" : "w-full"
          )}
        >
          {tab === "done" ? (
            <DonePane model={model} />
          ) : (
            <div className={cn("grid", !phone && "sm:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]")}>
              <SummaryPane model={model} method={method} phone={phone} />
              <PayPane model={model} picked={picked} onPick={setPicked} method={method} accent={accent} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
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

function SummaryPane({ model, method, phone }: { model: RenderModel; method?: RenderMethod; phone: boolean }) {
  const cur = model.currency;
  const surcharge = method?.customer_total ? method : null;
  const due = surcharge?.customer_total ?? model.amount;
  return (
    <div className={cn("space-y-4 p-5", !phone && "sm:border-r sm:border-line", phone && "border-b border-line")}>
      <Merchant model={model} />
      <div className="space-y-1">
        <p className={cn("text-body-sm text-ink-soft", !model.title && "text-ink-faint")}>{model.title || C.untitled}</p>
        {model.amount_mode === "customer" ? (
          <p className="text-caption text-ink-soft">{C.range(model.amount_min, model.amount_max, cur)}</p>
        ) : model.amount ? (
          <Money amount={model.amount} currency={cur} size="display" />
        ) : model.amount_mode === "line_items" ? (
          <p className="text-body-sm text-ink-faint">{C.serverTotalPending}</p>
        ) : null}
      </div>
      {model.description ? <p className="text-body-sm whitespace-pre-line text-ink-soft">{model.description}</p> : null}

      {model.line_items.length > 0 ? (
        <ul className="divide-y divide-line border-y border-line">
          {model.line_items.map((li, i) => (
            <li key={i} className="flex items-baseline justify-between gap-3 py-2">
              <span className="min-w-0">
                <span className="block truncate text-body-sm text-ink">{li.name || C.untitled}</span>
                <span className="num text-caption text-ink-soft">
                  {li.quantity} x {li.unit_price} {cur}
                  {Number(li.tax_rate) > 0 ? `, tax ${li.tax_rate}%` : ""}
                </span>
              </span>
              {li.total ? <Money amount={li.total} currency={cur} size="sm" /> : null}
            </li>
          ))}
        </ul>
      ) : null}

      {surcharge && model.amount ? (
        <dl className="space-y-1.5 text-body-sm">
          <Line label={model.amount_mode === "line_items" ? "Subtotal" : "Amount"} amount={model.amount} currency={cur} />
          <Line label={C.fee} amount={surcharge.fee} currency={cur} />
          {surcharge.tax && Number(surcharge.tax) > 0 ? <Line label={C.feeTax} amount={surcharge.tax} currency={cur} /> : null}
        </dl>
      ) : null}
      {due && (surcharge || model.line_items.length > 0) ? (
        <div className="flex items-baseline justify-between border-t border-line pt-3">
          <span className="text-body-sm font-semibold text-ink">{C.totalDue}</span>
          <Money amount={due} currency={cur} />
        </div>
      ) : null}
      {model.expires_at ? (
        <p className="num text-caption text-ink-soft">
          {C.ends} {new Date(model.expires_at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" })}
        </p>
      ) : null}
    </div>
  );
}

function Line({ label, amount, currency }: { label: string; amount: string | null; currency: string }) {
  if (!amount) return null;
  return (
    <div className="flex items-baseline justify-between gap-3">
      <dt className="text-ink-soft">{label}</dt>
      <dd>
        <Money amount={amount} currency={currency} size="sm" />
      </dd>
    </div>
  );
}

function PayPane({
  model,
  picked,
  onPick,
  method,
  accent,
}: {
  model: RenderModel;
  picked: number;
  onPick: (i: number) => void;
  method?: RenderMethod;
  accent: string | null;
}) {
  const f = model.customer_fields;
  const contact = (["email", "name", "phone"] as const).filter((k) => f[k].mode !== "hidden");
  const label = { name: "Name", email: "Email", phone: "Phone" } as const;
  const due = method?.customer_total ?? model.amount;
  return (
    <div className="space-y-4 p-5">
      {model.amount_mode === "customer" ? <MockInput label={C.customerAmount} /> : null}

      {contact.length > 0 ? (
        <div className="space-y-2">
          <p className="text-label font-medium text-ink">{C.contact}</p>
          <div className="grid grid-cols-2 gap-2">
            {contact.map((k) => (
              <div key={k} className="col-span-2 space-y-1">
                <p className="text-label text-ink-soft">
                  {label[k]}
                  {f[k].mode === "optional" ? <span className="text-ink-faint"> ({C.optional.toLowerCase()})</span> : null}
                </p>
                <div className="flex h-9 items-center rounded-sm border border-line-strong bg-surface px-3 text-body-sm text-ink">
                  {f[k].prefill ?? ""}
                </div>
              </div>
            ))}
          </div>
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
              <p className="text-label text-ink-soft">
                {q.label}
                {!q.required ? <span className="text-ink-faint"> ({C.optional.toLowerCase()})</span> : null}
              </p>
              <div className="flex h-9 items-center justify-between rounded-sm border border-line-strong bg-surface px-3 text-body-sm text-ink-faint">
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
                <span
                  aria-hidden
                  className={cn("size-3.5 shrink-0 rounded-full border", i === picked ? "border-[5px] border-ink" : "border-line-strong")}
                />
                <span className="min-w-0 truncate">{methodLabel(m)}</span>
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
          {due ? ` ${formatDecimal(due, model.currency)} ${model.currency}` : ""}
        </div>
      ) : (
        <p role="status" className="rounded-sm border border-wait/30 bg-wait-tint px-3 py-2.5 text-body-sm text-ink">
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
