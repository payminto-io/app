"use client";

import { Lock, Plus, Trash2 } from "lucide-react";
import type { MethodSpec, PayMethod } from "@/lib/api/links";
import { CurrencyDisplay } from "@/components/currency-display";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { LINKS_COPY, methodLabel } from "../copy";
import { emptyLineItem, emptyQuestion, type FieldForm, type LinkForm } from "../model";
import type { MethodFee } from "../preview";
import { Field, GroupLabel, NativeSelect, Segmented, ToggleRow } from "./controls";

const F = LINKS_COPY.fields;

export interface StepProps {
  form: LinkForm;
  /** Apply a change and clear the API's message for `path` (and anything under it). */
  edit: (path: string, fn: (f: LinkForm) => LinkForm) => void;
  err: (path: string, nested?: boolean) => string | undefined;
  /** Amount, currency, methods and line items are fixed on a published link. */
  locked: boolean;
}

const set =
  <K extends keyof LinkForm>(k: K, v: LinkForm[K]) =>
  (f: LinkForm): LinkForm => ({ ...f, [k]: v });

const COMMON_CURRENCIES = ["USD", "EUR", "GBP", "INR", "SGD", "AED", "USDC", "USDT"];

export function LockedNote({ onDuplicate }: { onDuplicate?: () => void }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-sm border border-line bg-surface-sunken px-3 py-2">
      <span className="flex items-center gap-2 text-body-sm text-ink-soft">
        <Lock className="size-3.5 shrink-0" aria-hidden />
        {LINKS_COPY.builder.locked}
      </span>
      {onDuplicate ? (
        <Button type="button" size="xs" variant="outline" onClick={onDuplicate}>
          {LINKS_COPY.builder.duplicateToChange}
        </Button>
      ) : null}
    </div>
  );
}

export function ItemStep({ form, edit, err, locked, onDuplicate }: StepProps & { onDuplicate?: () => void }) {
  const mode = form.amount_mode;
  return (
    <>
      <Field label={F.title} htmlFor="lf-title" error={err("title")}>
        <Input id="lf-title" value={form.title} maxLength={200} aria-invalid={Boolean(err("title")) || undefined} onChange={(e) => edit("title", set("title", e.target.value))} />
      </Field>
      <Field label={F.description} htmlFor="lf-desc" error={err("description")}>
        <Textarea id="lf-desc" rows={2} value={form.description} maxLength={5000} onChange={(e) => edit("description", set("description", e.target.value))} />
      </Field>

      {locked ? <LockedNote onDuplicate={onDuplicate} /> : null}

      <Field label={F.amountMode} error={err("amount_mode")}>
        <Segmented
          label={F.amountMode}
          value={mode}
          disabled={locked}
          className="w-full sm:w-auto"
          options={(["fixed", "customer", "line_items"] as const).map((v) => ({ value: v, label: F.modes[v] }))}
          onChange={(v) => edit("amount_mode", set("amount_mode", v))}
        />
      </Field>

      <div className="grid grid-cols-2 gap-3">
        {mode === "fixed" ? (
          <Field label={F.amount} htmlFor="lf-amount" error={err("amount")}>
            <Input id="lf-amount" inputMode="decimal" className="num" value={form.amount} disabled={locked} aria-invalid={Boolean(err("amount")) || undefined} onChange={(e) => edit("amount", set("amount", e.target.value))} />
          </Field>
        ) : null}
        <Field label={F.currency} htmlFor="lf-currency" error={err("currency")} className={mode === "fixed" ? "" : "col-span-2 sm:col-span-1"}>
          <Input
            id="lf-currency"
            list="lf-currencies"
            autoCapitalize="characters"
            className="uppercase"
            maxLength={12}
            value={form.currency}
            disabled={locked}
            aria-invalid={Boolean(err("currency")) || undefined}
            onChange={(e) => edit("currency", set("currency", e.target.value.toUpperCase()))}
          />
          <datalist id="lf-currencies">
            {COMMON_CURRENCIES.map((c) => (
              <option key={c} value={c} />
            ))}
          </datalist>
        </Field>
      </div>

      {mode === "customer" ? (
        <div className="grid grid-cols-2 gap-3">
          <Field label={F.min} htmlFor="lf-min" error={err("amount_min")}>
            <Input id="lf-min" inputMode="decimal" className="num" value={form.amount_min} disabled={locked} aria-invalid={Boolean(err("amount_min")) || undefined} onChange={(e) => edit("amount_min", set("amount_min", e.target.value))} />
          </Field>
          <Field label={F.max} htmlFor="lf-max" error={err("amount_max")}>
            <Input id="lf-max" inputMode="decimal" className="num" value={form.amount_max} disabled={locked} aria-invalid={Boolean(err("amount_max")) || undefined} onChange={(e) => edit("amount_max", set("amount_max", e.target.value))} />
          </Field>
        </div>
      ) : null}

      {mode === "line_items" ? <LineItems form={form} edit={edit} err={err} locked={locked} /> : null}

      <div className="grid grid-cols-2 gap-3">
        <Field label={F.reference} htmlFor="lf-ref" error={err("reference_id")}>
          <Input id="lf-ref" className="font-mono" maxLength={100} value={form.reference_id} onChange={(e) => edit("reference_id", set("reference_id", e.target.value))} />
        </Field>
        <Field label={F.category} htmlFor="lf-cat" error={err("category")}>
          <Input id="lf-cat" maxLength={64} value={form.category} onChange={(e) => edit("category", set("category", e.target.value))} />
        </Field>
      </div>

      <Metadata form={form} edit={edit} err={err} locked={locked} />
    </>
  );
}

function LineItems({ form, edit, err, locked }: StepProps) {
  const items = form.line_items;
  const change = (i: number, k: keyof LinkForm["line_items"][number], v: string) =>
    edit(`line_items[${i}].${k}`, (f) => ({ ...f, line_items: f.line_items.map((li, j) => (j === i ? { ...li, [k]: v } : li)) }));
  return (
    <div className="space-y-2">
      <GroupLabel>{F.lineItems}</GroupLabel>
      {err("line_items") ? (
        <p role="alert" className="text-label text-bad">
          {err("line_items")}
        </p>
      ) : null}
      <ul className="space-y-2">
        {items.map((li, i) => {
          const e = (k: string) => err(`line_items[${i}].${k}`);
          const rowErr = e("name") ?? e("quantity") ?? e("unit_price") ?? e("tax_rate");
          return (
            <li key={i} className="rounded-sm border border-line p-2.5">
              <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <Input aria-label={`${F.itemName} ${i + 1}`} placeholder={F.itemName} value={li.name} disabled={locked} aria-invalid={Boolean(e("name")) || undefined} onChange={(ev) => change(i, "name", ev.target.value)} />
                <Button type="button" variant="ghost" size="icon" aria-label={`Remove item ${i + 1}`} disabled={locked} onClick={() => edit("line_items", (f) => ({ ...f, line_items: f.line_items.filter((_, j) => j !== i) }))}>
                  <Trash2 />
                </Button>
              </div>
              <div className="mt-2 grid grid-cols-[4.5rem_minmax(0,1fr)_5rem] gap-2">
                <Input aria-label={`${F.qty} ${i + 1}`} inputMode="numeric" className="num" value={li.quantity} disabled={locked} aria-invalid={Boolean(e("quantity")) || undefined} onChange={(ev) => change(i, "quantity", ev.target.value)} />
                <Input aria-label={`${F.unitPrice} ${i + 1}`} placeholder={F.unitPrice} inputMode="decimal" className="num" value={li.unit_price} disabled={locked} aria-invalid={Boolean(e("unit_price")) || undefined} onChange={(ev) => change(i, "unit_price", ev.target.value)} />
                <Input aria-label={`${F.taxRate} ${i + 1}`} placeholder={F.taxRate} inputMode="decimal" className="num" value={li.tax_rate} disabled={locked} aria-invalid={Boolean(e("tax_rate")) || undefined} onChange={(ev) => change(i, "tax_rate", ev.target.value)} />
              </div>
              <div className="mt-1 grid grid-cols-[4.5rem_minmax(0,1fr)_5rem] gap-2 text-caption text-ink-faint" aria-hidden>
                <span>{F.qty}</span>
                <span>{F.unitPrice}</span>
                <span>{F.taxRate}</span>
              </div>
              {rowErr ? (
                <p role="alert" className="mt-1 text-label text-bad">
                  {rowErr}
                </p>
              ) : null}
            </li>
          );
        })}
      </ul>
      <Button type="button" size="sm" variant="outline" disabled={locked} onClick={() => edit("line_items", (f) => ({ ...f, line_items: [...f.line_items, emptyLineItem()] }))}>
        <Plus />
        {F.addItem}
      </Button>
    </div>
  );
}

function Metadata({ form, edit, err }: StepProps) {
  const rows = form.metadata;
  const change = (i: number, k: "key" | "value", v: string) =>
    edit("metadata", (f) => ({ ...f, metadata: f.metadata.map((r, j) => (j === i ? { ...r, [k]: v } : r)) }));
  return (
    <div className="space-y-2">
      <GroupLabel>{F.metadata}</GroupLabel>
      {rows.map((r, i) => (
        <div key={i} className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto] gap-2">
          <Input aria-label={`${F.key} ${i + 1}`} placeholder={F.key} className="font-mono" value={r.key} onChange={(e) => change(i, "key", e.target.value)} />
          <Input aria-label={`${F.value} ${i + 1}`} placeholder={F.value} value={r.value} onChange={(e) => change(i, "value", e.target.value)} />
          <Button type="button" variant="ghost" size="icon" aria-label={`Remove field ${i + 1}`} onClick={() => edit("metadata", (f) => ({ ...f, metadata: f.metadata.filter((_, j) => j !== i) }))}>
            <Trash2 />
          </Button>
        </div>
      ))}
      {err("metadata") ? (
        <p role="alert" className="text-label text-bad">
          {err("metadata")}
        </p>
      ) : null}
      <Button type="button" size="sm" variant="ghost" className="-ml-2" onClick={() => edit("metadata", (f) => ({ ...f, metadata: [...f.metadata, { key: "", value: "" }] }))}>
        <Plus />
        {F.addMetadata}
      </Button>
    </div>
  );
}

export function CustomerStep({ form, edit, err }: StepProps) {
  const fields = ["name", "email", "phone"] as const;
  const setField = (k: (typeof fields)[number], v: Partial<FieldForm>) => (f: LinkForm): LinkForm => ({
    ...f,
    customer: { ...f.customer, [k]: { ...f.customer[k], ...v } },
  });
  return (
    <>
      <div className="space-y-3">
        {fields.map((k) => {
          const base = `customer_field_policy.${k}`;
          const modeErr = err(`${base}.mode`);
          const prefillErr = err(`${base}.prefill`);
          return (
            <div key={k} className="space-y-1.5">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-body-sm font-medium text-ink">{F.customerFields[k]}</span>
                <Segmented
                  size="sm"
                  label={`${F.customerFields[k]} on checkout`}
                  value={form.customer[k].mode}
                  invalid={Boolean(modeErr)}
                  options={(["required", "optional", "hidden"] as const).map((v) => ({ value: v, label: F.fieldModes[v] }))}
                  onChange={(v) => edit(base, setField(k, { mode: v }))}
                />
              </div>
              <Input
                aria-label={`${F.customerFields[k]} ${F.prefill.toLowerCase()}`}
                placeholder={`${F.prefill} (${k === "phone" ? "+14155550123" : k === "email" ? "name@example.com" : F.customerFields.name.toLowerCase()})`}
                type={k === "email" ? "email" : k === "phone" ? "tel" : "text"}
                value={form.customer[k].prefill}
                aria-invalid={Boolean(prefillErr) || undefined}
                onChange={(e) => edit(`${base}.prefill`, setField(k, { prefill: e.target.value }))}
              />
              {modeErr ?? prefillErr ? (
                <p role="alert" className="text-label text-bad">
                  {modeErr ?? prefillErr}
                </p>
              ) : null}
            </div>
          );
        })}
      </div>

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.billing} checked={form.billing_required} onChange={(v) => edit("billing_required", set("billing_required", v))} error={err("billing_required")} />
        <ToggleRow label={F.shipping} checked={form.shipping_required} onChange={(v) => edit("shipping_required", set("shipping_required", v))} error={err("shipping_required")} />
      </div>

      <Questions form={form} edit={edit} err={err} locked={false} />

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.multiUse} checked={form.multi_use} onChange={(v) => edit("multi_use", set("multi_use", v))} error={err("multi_use")} />
        {form.multi_use ? (
          <Field label={F.useLimit} htmlFor="lf-uselimit" error={err("use_limit")} className="max-w-48">
            <Input id="lf-uselimit" inputMode="numeric" className="num" value={form.use_limit} aria-invalid={Boolean(err("use_limit")) || undefined} onChange={(e) => edit("use_limit", set("use_limit", e.target.value))} />
          </Field>
        ) : null}
      </div>
    </>
  );
}

function Questions({ form, edit, err }: StepProps) {
  const change = (i: number, patch: Partial<LinkForm["questions"][number]>, key: string) =>
    edit(`questions[${i}].${key}`, (f) => ({ ...f, questions: f.questions.map((q, j) => (j === i ? { ...q, ...patch } : q)) }));
  return (
    <div className="space-y-2 border-t border-line pt-4">
      <GroupLabel>{F.questions}</GroupLabel>
      {err("questions") ? (
        <p role="alert" className="text-label text-bad">
          {err("questions")}
        </p>
      ) : null}
      {form.questions.map((q, i) => {
        const e = (k: string) => err(`questions[${i}].${k}`);
        return (
          <div key={i} className="space-y-2 rounded-sm border border-line p-2.5">
            <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
              <Input aria-label={`${F.questionLabel} ${i + 1}`} placeholder={F.questionLabel} value={q.label} aria-invalid={Boolean(e("label")) || undefined} onChange={(ev) => change(i, { label: ev.target.value }, "label")} />
              <Button type="button" variant="ghost" size="icon" aria-label={`Remove question ${i + 1}`} onClick={() => edit("questions", (f) => ({ ...f, questions: f.questions.filter((_, j) => j !== i) }))}>
                <Trash2 />
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input aria-label={`${F.questionKey} ${i + 1}`} placeholder={`${F.questionKey} (size)`} className="font-mono" value={q.key} aria-invalid={Boolean(e("key")) || undefined} onChange={(ev) => change(i, { key: ev.target.value }, "key")} />
              <NativeSelect aria-label={`${F.questionType} ${i + 1}`} value={q.type} onChange={(ev) => change(i, { type: ev.target.value as typeof q.type }, "type")}>
                {(["text", "select", "checkbox"] as const).map((t) => (
                  <option key={t} value={t}>
                    {F.questionTypes[t]}
                  </option>
                ))}
              </NativeSelect>
            </div>
            {q.type === "select" ? (
              <Textarea aria-label={`${F.options} ${i + 1}`} placeholder={F.options} rows={3} value={q.options} aria-invalid={Boolean(e("options")) || undefined} onChange={(ev) => change(i, { options: ev.target.value }, "options")} />
            ) : null}
            <div className="flex flex-wrap gap-x-5 gap-y-2">
              <CheckLabel label={F.required} checked={q.required} onChange={(v) => change(i, { required: v }, "required")} />
              <CheckLabel label={F.perOrder} checked={q.per_order} onChange={(v) => change(i, { per_order: v }, "per_order")} />
            </div>
            {e("key") ?? e("label") ?? e("type") ?? e("options") ? (
              <p role="alert" className="text-label text-bad">
                {e("key") ?? e("label") ?? e("type") ?? e("options")}
              </p>
            ) : null}
          </div>
        );
      })}
      <Button type="button" size="sm" variant="ghost" className="-ml-2" onClick={() => edit("questions", (f) => ({ ...f, questions: [...f.questions, emptyQuestion()] }))}>
        <Plus />
        {F.addQuestion}
      </Button>
    </div>
  );
}

function CheckLabel({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="tap inline-flex cursor-pointer items-center gap-2 text-body-sm text-ink">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="size-4 accent-[var(--ink)]" />
      {label}
    </label>
  );
}

const FIAT_METHODS: PayMethod[] = ["card", "upi", "bank"];

export function PaymentStep({
  form,
  edit,
  err,
  locked,
  fees,
  cryptoOptions,
  onDuplicate,
}: StepProps & { fees: MethodFee[]; cryptoOptions: { chain: string; asset: string }[]; onDuplicate?: () => void }) {
  const has = (m: PayMethod) => form.methods.some((x) => x.method === m);
  const indexOf = (spec: MethodSpec) =>
    form.methods.findIndex((x) => x.method === spec.method && (x.chain ?? "") === (spec.chain ?? "") && (x.asset ?? "") === (spec.asset ?? ""));
  const toggle = (spec: MethodSpec, on: boolean) =>
    edit("methods", (f) => ({
      ...f,
      methods: on ? [...f.methods, spec] : f.methods.filter((_, j) => j !== indexOf(spec)),
    }));
  const crypto = form.methods.map((m, i) => ({ m, i })).filter(({ m }) => m.method === "crypto");
  const unusedCrypto = cryptoOptions.filter((o) => indexOf({ method: "crypto", chain: o.chain, asset: o.asset }) < 0);

  return (
    <>
      {locked ? <LockedNote onDuplicate={onDuplicate} /> : null}
      <div className="space-y-2">
        <GroupLabel>{F.methods}</GroupLabel>
        {err("methods") ? (
          <p role="alert" className="text-label text-bad">
            {err("methods")}
          </p>
        ) : null}
        <ul className="divide-y divide-line rounded-sm border border-line">
          {FIAT_METHODS.map((m) => {
            const i = indexOf({ method: m });
            return (
              <MethodRow
                key={m}
                label={F.methodNames[m]}
                on={has(m)}
                disabled={locked}
                onToggle={(v) => toggle({ method: m }, v)}
                fee={i >= 0 ? fees[i] : undefined}
                error={i >= 0 ? err(`methods[${i}]`, true) : undefined}
                currency={form.currency}
              />
            );
          })}
          {crypto.map(({ m, i }) => (
            <MethodRow
              key={`c-${i}`}
              label={methodLabel(m)}
              on
              disabled={locked}
              onToggle={(v) => toggle(m, v)}
              fee={fees[i]}
              error={err(`methods[${i}]`, true)}
              currency={form.currency}
            />
          ))}
        </ul>
        {!locked && unusedCrypto.length > 0 ? (
          <NativeSelect
            aria-label={F.addCrypto}
            value=""
            className="max-w-64"
            onChange={(e) => {
              const o = unusedCrypto[Number(e.target.value)];
              if (o) toggle({ method: "crypto", chain: o.chain, asset: o.asset }, true);
            }}
          >
            <option value="">{F.addCrypto}</option>
            {unusedCrypto.map((o, i) => (
              <option key={`${o.chain}-${o.asset}`} value={i}>
                {o.asset} on {o.chain}
              </option>
            ))}
          </NativeSelect>
        ) : null}
      </div>

      <Field label={F.feeBearer} error={err("fee_bearer")}>
        <Segmented
          label={F.feeBearer}
          value={form.fee_bearer}
          invalid={Boolean(err("fee_bearer"))}
          options={(["merchant", "customer"] as const).map((v) => ({ value: v, label: F.bearers[v] }))}
          onChange={(v) => edit("fee_bearer", set("fee_bearer", v))}
        />
      </Field>

      {has("card") ? (
        <div className="grid gap-3 border-t border-line pt-4 sm:grid-cols-2">
          <Field label={F.capture} error={err("capture_mode")}>
            <Segmented label={F.capture} value={form.capture_mode} className="w-full" options={(["automatic", "manual"] as const).map((v) => ({ value: v, label: F.captureModes[v] }))} onChange={(v) => edit("capture_mode", set("capture_mode", v))} />
          </Field>
          <Field label={F.threeDS} error={err("three_ds_policy")}>
            <Segmented label={F.threeDS} value={form.three_ds_policy} className="w-full" options={(["inherit", "force"] as const).map((v) => ({ value: v, label: F.threeDSModes[v] }))} onChange={(v) => edit("three_ds_policy", set("three_ds_policy", v))} />
          </Field>
        </div>
      ) : err("capture_mode") ? (
        <p role="alert" className="text-label text-bad">
          {err("capture_mode")}
        </p>
      ) : null}

      {crypto.length > 0 ? (
        <div className="grid grid-cols-2 gap-3 border-t border-line pt-4">
          <Field label={F.tolerance} htmlFor="lf-tol" error={err("chain_tolerance_bps")}>
            <Input id="lf-tol" inputMode="numeric" className="num" value={form.chain_tolerance_bps} aria-invalid={Boolean(err("chain_tolerance_bps")) || undefined} onChange={(e) => edit("chain_tolerance_bps", set("chain_tolerance_bps", e.target.value))} />
          </Field>
          <Field label={F.quoteExpiry} htmlFor="lf-quote" error={err("quote_expiry_seconds")}>
            <Input id="lf-quote" inputMode="numeric" className="num" value={form.quote_expiry_seconds} aria-invalid={Boolean(err("quote_expiry_seconds")) || undefined} onChange={(e) => edit("quote_expiry_seconds", set("quote_expiry_seconds", e.target.value))} />
          </Field>
        </div>
      ) : null}
    </>
  );
}

function MethodRow({
  label,
  on,
  disabled,
  onToggle,
  fee,
  error,
  currency,
}: {
  label: string;
  on: boolean;
  disabled: boolean;
  onToggle: (v: boolean) => void;
  fee?: MethodFee;
  error?: string;
  currency: string;
}) {
  return (
    <li className="px-3 py-2.5">
      <label className={cn("flex items-center gap-3", disabled ? "cursor-not-allowed" : "cursor-pointer")}>
        <input type="checkbox" checked={on} disabled={disabled} onChange={(e) => onToggle(e.target.checked)} className="size-4 shrink-0 accent-[var(--ink)]" />
        <span className="flex-1 text-body-sm font-medium text-ink">{label}</span>
        {on && fee ? <FeeBadge fee={fee} /> : null}
      </label>
      {on && fee?.state === "ok" ? <FeeLines fee={fee} currency={currency} /> : null}
      {error ? (
        <p role="alert" className="mt-1 pl-7 text-label text-bad">
          {error}
        </p>
      ) : null}
    </li>
  );
}

function FeeBadge({ fee }: { fee: MethodFee }) {
  const C = LINKS_COPY.fee;
  switch (fee.state) {
    case "ok":
      return <span className="num shrink-0 rounded-xs bg-surface-sunken px-1.5 py-0.5 font-mono text-caption text-ink-soft">{C.rule(fee.preview.rule_id, fee.preview.version)}</span>;
    case "loading":
      return <span className="shrink-0 text-caption text-ink-faint">{C.loading}</span>;
    case "no_amount":
      return <span className="shrink-0 text-caption text-ink-faint">{C.noAmount}</span>;
    case "at_pay_time":
      return <span className="shrink-0 text-caption text-ink-soft">{C.atPayTime}</span>;
    case "refused":
      return <span className="shrink-0 text-caption text-wait">{C.refused[fee.code] ?? fee.message}</span>;
    case "error":
      return <span className="max-w-[50%] shrink-0 truncate text-caption text-bad" title={fee.message}>{fee.message}</span>;
  }
}

function FeeLines({ fee, currency }: { fee: Extract<MethodFee, { state: "ok" }>; currency: string }) {
  const C = LINKS_COPY.fee;
  const p = fee.preview;
  const cur = p.currency || currency;
  const rows: [string, string][] = [
    [C.fee, p.fee],
    ...(Number(p.tax) > 0 ? ([[C.tax, p.tax]] as [string, string][]) : []),
    [C.total, p.customer_total],
    [C.net, p.merchant_net],
  ];
  return (
    <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 pl-7 sm:grid-cols-4">
      {rows.map(([k, v]) => (
        <div key={k} className="min-w-0">
          <dt className="text-caption text-ink-soft">{k}</dt>
          <dd>
            <CurrencyDisplay amount={v} currency={cur} size="sm" />
          </dd>
        </div>
      ))}
    </dl>
  );
}

export function AfterStep({ form, edit, err, webhooks }: StepProps & { webhooks: { id: number; url: string }[] }) {
  return (
    <>
      <Field label={F.success} error={err("success_mode")}>
        <Segmented label={F.success} value={form.success_mode} options={(["message", "redirect"] as const).map((v) => ({ value: v, label: F.successModes[v] }))} onChange={(v) => edit("success_mode", set("success_mode", v))} />
      </Field>
      {form.success_mode === "message" ? (
        <Field label={F.successMessage} htmlFor="lf-succ" error={err("success_message")}>
          <Textarea id="lf-succ" rows={2} maxLength={1000} value={form.success_message} aria-invalid={Boolean(err("success_message")) || undefined} onChange={(e) => edit("success_message", set("success_message", e.target.value))} />
        </Field>
      ) : (
        <Field label={F.successUrl} htmlFor="lf-surl" error={err("success_url")} hint="?reference_id= is appended">
          <Input id="lf-surl" type="url" placeholder="https://example.com/thanks" value={form.success_url} aria-invalid={Boolean(err("success_url")) || undefined} onChange={(e) => edit("success_url", set("success_url", e.target.value))} />
        </Field>
      )}

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.receipt} checked={form.receipt_email} onChange={(v) => edit("receipt_email", set("receipt_email", v))} error={err("receipt_email")} />
        {form.receipt_email ? (
          <Field label={F.receiptNote} htmlFor="lf-rnote" error={err("receipt_note")}>
            <Textarea id="lf-rnote" rows={2} maxLength={1000} value={form.receipt_note} onChange={(e) => edit("receipt_note", set("receipt_note", e.target.value))} />
          </Field>
        ) : null}
      </div>

      <Field label={F.webhook} htmlFor="lf-webhook" error={err("webhook_id")} className="border-t border-line pt-4">
        <NativeSelect id="lf-webhook" value={form.webhook_id} aria-invalid={Boolean(err("webhook_id")) || undefined} onChange={(e) => edit("webhook_id", set("webhook_id", e.target.value))}>
          <option value="">{F.noWebhook}</option>
          {webhooks.map((w) => (
            <option key={w.id} value={String(w.id)}>
              {w.url}
            </option>
          ))}
          {form.webhook_id && !webhooks.some((w) => String(w.id) === form.webhook_id) ? (
            <option value={form.webhook_id}>#{form.webhook_id}</option>
          ) : null}
        </NativeSelect>
      </Field>

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.failureRetry} checked={form.failure_retry} onChange={(v) => edit("failure_retry", set("failure_retry", v))} error={err("failure_retry")} />
        <Field label={F.failureMessage} htmlFor="lf-fail" error={err("failure_message")}>
          <Textarea id="lf-fail" rows={2} maxLength={1000} value={form.failure_message} onChange={(e) => edit("failure_message", set("failure_message", e.target.value))} />
        </Field>
      </div>
    </>
  );
}

export function SettlementStep({ form, edit, err }: StepProps) {
  const s = form.settlement;
  const setS = (patch: Partial<LinkForm["settlement"]>) => (f: LinkForm): LinkForm => ({ ...f, settlement: { ...f.settlement, ...patch } });
  return (
    <>
      <Field label={F.settlement} error={err("settlement_override.kind") ?? err("settlement_override")}>
        <Segmented label={F.settlement} value={s.mode} options={(["default", "fiat", "crypto"] as const).map((v) => ({ value: v, label: F.settlementModes[v] }))} onChange={(v) => edit("settlement_override", setS({ mode: v }))} />
      </Field>
      {s.mode !== "default" ? (
        <div className="grid grid-cols-2 gap-3">
          <Field label={F.destination} htmlFor="lf-dest" error={err("settlement_override.destination_id")} className="col-span-2">
            <Input id="lf-dest" className="font-mono" value={s.destination_id} aria-invalid={Boolean(err("settlement_override.destination_id")) || undefined} onChange={(e) => edit("settlement_override.destination_id", setS({ destination_id: e.target.value }))} />
          </Field>
          {s.mode === "crypto" ? (
            <>
              <Field label={F.chain} htmlFor="lf-schain" error={err("settlement_override.chain")}>
                <Input id="lf-schain" className="uppercase" value={s.chain} onChange={(e) => edit("settlement_override.chain", setS({ chain: e.target.value.toUpperCase() }))} />
              </Field>
              <Field label={F.asset} htmlFor="lf-sasset" error={err("settlement_override.asset")}>
                <Input id="lf-sasset" className="uppercase" value={s.asset} onChange={(e) => edit("settlement_override.asset", setS({ asset: e.target.value.toUpperCase() }))} />
              </Field>
            </>
          ) : null}
        </div>
      ) : null}
      <div className="border-t border-line pt-4">
        <ToggleRow label={F.holdInAsset} hint={F.holdNote} checked={form.hold_in_asset} onChange={(v) => edit("hold_in_asset", set("hold_in_asset", v))} error={err("hold_in_asset")} />
      </div>
      <Field label={F.timing} error={err("settlement_timing")}>
        <Segmented label={F.timing} value={form.settlement_timing} options={(["cycle", "immediate"] as const).map((v) => ({ value: v, label: F.timingModes[v] }))} onChange={(v) => edit("settlement_timing", set("settlement_timing", v))} />
      </Field>
    </>
  );
}

export function LifecycleStep({ form, edit, err }: StepProps) {
  const accentOk = /^#[0-9a-fA-F]{6}$/.test(form.accent_color);
  return (
    <>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label={F.expiresAt} htmlFor="lf-exp" error={err("expires_at")}>
          <Input id="lf-exp" type="datetime-local" className="num" value={form.expires_at} aria-invalid={Boolean(err("expires_at")) || undefined} onChange={(e) => edit("expires_at", set("expires_at", e.target.value))} />
        </Field>
        <Field label={F.expiresAfter} htmlFor="lf-expn" error={err("expires_after_payments")} hint={form.multi_use ? undefined : "Single-use links end after one"}>
          <Input id="lf-expn" inputMode="numeric" className="num" disabled={!form.multi_use} value={form.multi_use ? form.expires_after_payments : ""} aria-invalid={Boolean(err("expires_after_payments")) || undefined} onChange={(e) => edit("expires_after_payments", set("expires_after_payments", e.target.value))} />
        </Field>
      </div>
      <div className="grid gap-3 border-t border-line pt-4 sm:grid-cols-2">
        <Field label={F.logo} htmlFor="lf-logo" error={err("logo_url")} className="sm:col-span-2">
          <Input id="lf-logo" type="url" placeholder="https://" value={form.logo_url} aria-invalid={Boolean(err("logo_url")) || undefined} onChange={(e) => edit("logo_url", set("logo_url", e.target.value))} />
        </Field>
        <Field label={F.accent} htmlFor="lf-accent" error={err("accent_color")}>
          <div className="flex items-center gap-2">
            <input
              type="color"
              aria-label={`${F.accent} picker`}
              value={accentOk ? form.accent_color : "#15181d"}
              onChange={(e) => edit("accent_color", set("accent_color", e.target.value.toUpperCase()))}
              className="h-9 pointer-coarse:h-11 w-10 pointer-coarse:w-11 shrink-0 cursor-pointer rounded-sm border border-line-strong bg-surface p-1"
            />
            <Input id="lf-accent" className="font-mono uppercase" placeholder="#0B7285" maxLength={7} value={form.accent_color} aria-invalid={Boolean(err("accent_color")) || undefined} onChange={(e) => edit("accent_color", set("accent_color", e.target.value))} />
          </div>
        </Field>
        <Field label={F.language} htmlFor="lf-lang" error={err("language")}>
          <Input id="lf-lang" placeholder="en" maxLength={35} value={form.language} aria-invalid={Boolean(err("language")) || undefined} onChange={(e) => edit("language", set("language", e.target.value))} />
        </Field>
      </div>
    </>
  );
}
