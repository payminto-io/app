"use client";

import Link from "next/link";
import { Lock, Plus, Trash2 } from "lucide-react";
import type { DroppedMethod, LinkOptions, MethodPreview, MethodSpec } from "@/lib/api/links";
import { CurrencyDisplay } from "@/components/currency-display";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { formatDecimal } from "@/lib/money";
import { cn } from "@/lib/utils";
import { LINKS_COPY, methodLabel } from "../copy";
import { emptyLineItem, emptyQuestion, methodKey, type FieldForm, type LinkForm } from "../model";
import { a11y, Field, FieldError, Group, GroupLabel, NativeSelect, ReadOnly, Segmented, ToggleRow } from "./controls";

const F = LINKS_COPY.fields;
const FEE = LINKS_COPY.fee;

/** The options request as the steps need it: the data, or why there is none yet. */
export interface OptionsState {
  data: LinkOptions | undefined;
  loading: boolean;
  error: string | null;
  retry: () => void;
}

function OptionsProblem({ options }: { options: OptionsState }) {
  if (options.loading) {
    return (
      <p role="status" className="text-caption text-ink-soft">
        {F.optionsLoading}
      </p>
    );
  }
  if (!options.error) return null;
  return (
    <div role="alert" className="flex flex-wrap items-center gap-2 text-label text-bad">
      <span>{F.optionsFailed}</span>
      <Button type="button" variant="outline" size="sm" className="tap" onClick={options.retry}>
        {F.retry}
      </Button>
    </div>
  );
}

export interface StepProps {
  form: LinkForm;
  /** Apply a change and clear the API's message for `path` (and anything under it). */
  edit: (path: string, fn: (f: LinkForm) => LinkForm) => void;
  err: (path: string, nested?: boolean) => string | undefined;
  /** Amount, currency, methods, line items and fee bearer are fixed on a published link. */
  locked: boolean;
}

const set =
  <K extends keyof LinkForm>(k: K, v: LinkForm[K]) =>
  (f: LinkForm): LinkForm => ({ ...f, [k]: v });

const money = (v: string, cur: string) => (cur ? `${formatDecimal(v, cur)} ${cur}` : v);

export function LockedNote({ onDuplicate }: { onDuplicate?: () => void }) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 rounded-sm border border-line bg-surface px-3 py-2">
      <span className="flex items-center gap-2 text-body-sm text-ink">
        <Lock className="size-3.5 shrink-0 text-ink-soft" aria-hidden />
        {LINKS_COPY.builder.locked}
      </span>
      {onDuplicate ? (
        <Button type="button" size="sm" variant="outline" className="tap" onClick={onDuplicate}>
          {LINKS_COPY.builder.duplicateToChange}
        </Button>
      ) : null}
    </div>
  );
}

function CurrencyControl({ form, edit, err, options }: StepProps & { options: OptionsState }) {
  const e = err("currency");
  const list = options.data?.currencies ?? [];
  if (!options.data) {
    return (
      <Field label={F.currency} htmlFor="lf-currency" error={e}>
        <NativeSelect id="lf-currency" value={form.currency} disabled {...a11y("lf-currency", e)}>
          <option value={form.currency}>{form.currency || F.loadingShort}</option>
        </NativeSelect>
        <OptionsProblem options={options} />
      </Field>
    );
  }
  if (list.length === 0) {
    return (
      <Field label={F.currency} htmlFor="lf-currency" error={e}>
        <Input
          id="lf-currency"
          autoCapitalize="characters"
          className="uppercase"
          maxLength={12}
          value={form.currency}
          {...a11y("lf-currency", e)}
          onChange={(ev) => edit("currency", set("currency", ev.target.value.toUpperCase()))}
        />
      </Field>
    );
  }
  const extra = form.currency && !list.includes(form.currency) ? form.currency : null;
  return (
    <Field label={F.currency} htmlFor="lf-currency" error={e}>
      <NativeSelect id="lf-currency" value={form.currency} {...a11y("lf-currency", e)} onChange={(ev) => edit("currency", set("currency", ev.target.value))}>
        <option value="">{F.chooseCurrency}</option>
        {list.map((c) => (
          <option key={c} value={c}>
            {c}
          </option>
        ))}
        {extra ? <option value={extra}>{`${extra} (${F.notEnabled})`}</option> : null}
      </NativeSelect>
    </Field>
  );
}

export function ItemStep({ form, edit, err, locked, options }: StepProps & { options: OptionsState }) {
  const mode = form.amount_mode;
  const cur = form.currency;
  return (
    <>
      <Field label={F.title} htmlFor="lf-title" error={err("title")}>
        <Input id="lf-title" value={form.title} maxLength={200} {...a11y("lf-title", err("title"))} onChange={(e) => edit("title", set("title", e.target.value))} />
      </Field>
      <Field label={F.description} htmlFor="lf-desc" error={err("description")}>
        <Textarea id="lf-desc" rows={2} value={form.description} maxLength={5000} {...a11y("lf-desc", err("description"))} onChange={(e) => edit("description", set("description", e.target.value))} />
      </Field>

      {locked ? (
        <div className="grid grid-cols-2 gap-3 rounded-sm border border-line px-3 py-3">
          <ReadOnly label={F.amountMode}>{F.modes[mode]}</ReadOnly>
          <ReadOnly label={F.currency}>{cur}</ReadOnly>
          {mode === "fixed" && form.amount ? (
            <ReadOnly label={F.amount} className="col-span-2">
              <CurrencyDisplay amount={form.amount} currency={cur} />
            </ReadOnly>
          ) : null}
          {mode === "customer" ? (
            <ReadOnly label={`${F.min} / ${F.max}`} className="col-span-2">
              <span className="num">
                {[form.amount_min, form.amount_max].map((v) => (v ? money(v, cur) : "-")).join(" / ")}
              </span>
            </ReadOnly>
          ) : null}
          {mode === "line_items" ? (
            <ReadOnly label={F.lineItems} className="col-span-2">
              <ul className="space-y-1 text-body-sm">
                {form.line_items.map((li, i) => (
                  <li key={i} className="num flex justify-between gap-3">
                    <span className="min-w-0 truncate">{li.name}</span>
                    <span className="shrink-0">
                      {li.quantity} x {money(li.unit_price, cur)}
                      {Number(li.tax_rate) > 0 ? `, ${li.tax_rate}%` : ""}
                    </span>
                  </li>
                ))}
              </ul>
            </ReadOnly>
          ) : null}
        </div>
      ) : (
        <>
          <Group label={F.amountMode} id="lf-mode" error={err("amount_mode")}>
            <Segmented
              id="lf-mode"
              label={F.amountMode}
              value={mode}
              error={err("amount_mode")}
              className="w-full sm:w-auto"
              options={(["fixed", "customer", "line_items"] as const).map((v) => ({ value: v, label: F.modes[v] }))}
              onChange={(v) => edit("amount_mode", set("amount_mode", v))}
            />
          </Group>

          <div className="grid grid-cols-2 gap-3">
            {mode === "fixed" ? (
              <Field label={F.amount} htmlFor="lf-amount" error={err("amount")}>
                <Input id="lf-amount" inputMode="decimal" className="num" value={form.amount} {...a11y("lf-amount", err("amount"))} onChange={(e) => edit("amount", set("amount", e.target.value))} />
              </Field>
            ) : null}
            <div className={mode === "fixed" ? "" : "col-span-2 sm:col-span-1"}>
              <CurrencyControl form={form} edit={edit} err={err} locked={locked} options={options} />
            </div>
          </div>

          {mode === "customer" ? (
            <div className="grid grid-cols-2 gap-3">
              <Field label={F.min} htmlFor="lf-min" error={err("amount_min")}>
                <Input id="lf-min" inputMode="decimal" className="num" value={form.amount_min} {...a11y("lf-min", err("amount_min"))} onChange={(e) => edit("amount_min", set("amount_min", e.target.value))} />
              </Field>
              <Field label={F.max} htmlFor="lf-max" error={err("amount_max")}>
                <Input id="lf-max" inputMode="decimal" className="num" value={form.amount_max} {...a11y("lf-max", err("amount_max"))} onChange={(e) => edit("amount_max", set("amount_max", e.target.value))} />
              </Field>
            </div>
          ) : null}

          {mode === "line_items" ? <LineItems form={form} edit={edit} err={err} locked={locked} /> : null}
        </>
      )}

      <div className="grid grid-cols-2 gap-3">
        <Field label={F.reference} htmlFor="lf-ref" error={err("reference_id")}>
          <Input id="lf-ref" className="font-mono" maxLength={100} value={form.reference_id} {...a11y("lf-ref", err("reference_id"))} onChange={(e) => edit("reference_id", set("reference_id", e.target.value))} />
        </Field>
        <Field label={F.category} htmlFor="lf-cat" error={err("category")}>
          <Input id="lf-cat" maxLength={64} value={form.category} {...a11y("lf-cat", err("category"))} onChange={(e) => edit("category", set("category", e.target.value))} />
        </Field>
      </div>

      <Metadata form={form} edit={edit} err={err} locked={locked} />
    </>
  );
}

/** One row at sm and up under a shared header; below that each item stacks with its own visible labels. */
const ITEM_GRID = "grid grid-cols-[4rem_minmax(0,1fr)_4.5rem] gap-2 sm:grid-cols-[minmax(0,1fr)_4rem_6rem_4.5rem_2.25rem]";

function LineItems({ form, edit, err }: StepProps) {
  const change = (i: number, k: keyof LinkForm["line_items"][number], v: string) =>
    edit(`line_items[${i}].${k}`, (f) => ({ ...f, line_items: f.line_items.map((li, j) => (j === i ? { ...li, [k]: v } : li)) }));
  const cols = { name: F.itemName, quantity: F.qty, unit_price: F.unitPrice, tax_rate: F.taxRate } as const;
  const keys = ["name", "quantity", "unit_price", "tax_rate"] as const;
  return (
    <div className="space-y-2" role="group" aria-labelledby="lf-items-label">
      <GroupLabel id="lf-items-label">{F.lineItems}</GroupLabel>
      <FieldError id="lf-items">{err("line_items")}</FieldError>
      {form.line_items.length > 0 ? (
        <div className={cn(ITEM_GRID, "hidden text-caption font-medium text-ink-soft sm:grid")} aria-hidden>
          {keys.map((k) => (
            <span key={k}>{cols[k]}</span>
          ))}
        </div>
      ) : null}
      {form.line_items.map((li, i) => {
        const id = (k: string) => `lf-li-${i}-${k}`;
        const rowErrs = keys.map((k) => ({ k, e: err(`line_items[${i}].${k}`) })).filter((x) => x.e);
        const input = (k: (typeof keys)[number]) => (
          <label key={k} className={cn("min-w-0 space-y-1", k === "name" && "col-span-2 sm:col-span-1")}>
            <span className="block text-caption font-medium text-ink-soft sm:sr-only">{cols[k]}</span>
            <Input
              id={id(k)}
              inputMode={k === "name" ? undefined : k === "quantity" ? "numeric" : "decimal"}
              className={k === "name" ? "" : "num px-2"}
              value={li[k]}
              {...a11y(id(k), err(`line_items[${i}].${k}`))}
              onChange={(ev) => change(i, k, ev.target.value)}
            />
          </label>
        );
        return (
          <div key={i} className="space-y-1 rounded-sm border border-line p-2.5 sm:border-0 sm:p-0">
            <div className={cn(ITEM_GRID, "items-end")}>
              {input("name")}
              <Button type="button" variant="ghost" size="icon" className="tap justify-self-end sm:order-last" aria-label={F.removeItem(i + 1)} onClick={() => edit("line_items", (f) => ({ ...f, line_items: f.line_items.filter((_, j) => j !== i) }))}>
                <Trash2 />
              </Button>
              {input("quantity")}
              {input("unit_price")}
              {input("tax_rate")}
            </div>
            {rowErrs.map((x) => (
              <FieldError key={x.k} id={id(x.k)}>
                {`${cols[x.k]}: ${x.e}`}
              </FieldError>
            ))}
          </div>
        );
      })}
      <Button type="button" size="sm" variant="outline" className="tap" onClick={() => edit("line_items", (f) => ({ ...f, line_items: [...f.line_items, emptyLineItem()] }))}>
        <Plus />
        {F.addItem}
      </Button>
    </div>
  );
}

function Metadata({ form, edit, err }: StepProps) {
  const change = (i: number, k: "key" | "value", v: string) =>
    edit("metadata", (f) => ({ ...f, metadata: f.metadata.map((r, j) => (j === i ? { ...r, [k]: v } : r)) }));
  const grid = "grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)_2.25rem] gap-2";
  return (
    <div className="space-y-2" role="group" aria-labelledby="lf-meta-label">
      <GroupLabel id="lf-meta-label">{F.metadata}</GroupLabel>
      {form.metadata.length > 0 ? (
        <div className={cn(grid, "text-caption font-medium text-ink-soft")} aria-hidden>
          <span>{F.key}</span>
          <span>{F.value}</span>
        </div>
      ) : null}
      {form.metadata.map((r, i) => (
        <div key={i} className={grid}>
          <Input aria-label={`${F.key} ${i + 1}`} className="font-mono" value={r.key} onChange={(e) => change(i, "key", e.target.value)} />
          <Input aria-label={`${F.value} ${i + 1}`} value={r.value} onChange={(e) => change(i, "value", e.target.value)} />
          <Button type="button" variant="ghost" size="icon" className="tap" aria-label={F.removeMetadata(i + 1)} onClick={() => edit("metadata", (f) => ({ ...f, metadata: f.metadata.filter((_, j) => j !== i) }))}>
            <Trash2 />
          </Button>
        </div>
      ))}
      <FieldError id="lf-meta">{err("metadata")}</FieldError>
      <Button type="button" size="sm" variant="ghost" className="tap -ml-2" onClick={() => edit("metadata", (f) => ({ ...f, metadata: [...f.metadata, { key: "", value: "" }] }))}>
        <Plus />
        {F.addMetadata}
      </Button>
    </div>
  );
}

export function CustomerStep({ form, edit, err, locked }: StepProps) {
  const fields = ["name", "email", "phone"] as const;
  const setField = (k: (typeof fields)[number], v: Partial<FieldForm>) => (f: LinkForm): LinkForm => ({
    ...f,
    customer: { ...f.customer, [k]: { ...f.customer[k], ...v } },
  });
  return (
    <>
      <div className="space-y-4">
        {fields.map((k) => {
          const base = `customer_field_policy.${k}`;
          const modeErr = err(`${base}.mode`);
          const prefillErr = err(`${base}.prefill`);
          const id = `lf-cf-${k}`;
          return (
            <div key={k} className="space-y-1.5">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span id={`${id}-label`} className="text-body-sm font-medium text-ink">
                  {F.customerFields[k]}
                </span>
                <Segmented
                  id={`${id}-mode`}
                  error={modeErr}
                  label={F.customerFields[k]}
                  value={form.customer[k].mode}
                  options={(["required", "optional", "hidden"] as const).map((v) => ({ value: v, label: F.fieldModes[v] }))}
                  onChange={(v) => edit(`${base}.mode`, setField(k, { mode: v }))}
                />
              </div>
              <FieldError id={`${id}-mode`}>{modeErr}</FieldError>
              <Field label={`${F.customerFields[k]} ${F.prefill.toLowerCase()}`} htmlFor={`${id}-prefill`} error={prefillErr}>
                <Input
                  id={`${id}-prefill`}
                  placeholder={F.prefillExamples[k]}
                  type={k === "email" ? "email" : k === "phone" ? "tel" : "text"}
                  value={form.customer[k].prefill}
                  {...a11y(`${id}-prefill`, prefillErr)}
                  onChange={(e) => edit(`${base}.prefill`, setField(k, { prefill: e.target.value }))}
                />
              </Field>
            </div>
          );
        })}
      </div>

      <div className="space-y-2 border-t border-line pt-4">
        <ToggleRow label={F.billing} checked={form.billing_required} onChange={(v) => edit("billing_required", set("billing_required", v))} error={err("billing_required")} />
        <ToggleRow label={F.shipping} checked={form.shipping_required} onChange={(v) => edit("shipping_required", set("shipping_required", v))} error={err("shipping_required")} />
      </div>

      <Questions form={form} edit={edit} err={err} locked={locked} />

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.multiUse} checked={form.multi_use} onChange={(v) => edit("multi_use", set("multi_use", v))} error={err("multi_use")} />
        {form.multi_use ? (
          <>
            <div className="grid grid-cols-2 gap-3">
              <Field label={F.useLimit} htmlFor="lf-uselimit" error={err("use_limit")}>
                <Input id="lf-uselimit" inputMode="numeric" className="num" value={form.use_limit} {...a11y("lf-uselimit", err("use_limit"))} onChange={(e) => edit("use_limit", set("use_limit", e.target.value))} />
              </Field>
              <Field label={F.expiresAfter} htmlFor="lf-expn" error={err("expires_after_payments")}>
                <Input id="lf-expn" inputMode="numeric" className="num" value={form.expires_after_payments} {...a11y("lf-expn", err("expires_after_payments"))} onChange={(e) => edit("expires_after_payments", set("expires_after_payments", e.target.value))} />
              </Field>
            </div>
            <p className="text-caption text-ink-soft">{F.limitsHint}</p>
          </>
        ) : null}
      </div>
    </>
  );
}

function Questions({ form, edit, err }: StepProps) {
  const change = (i: number, patch: Partial<LinkForm["questions"][number]>, key: string) =>
    edit(`questions[${i}].${key}`, (f) => ({ ...f, questions: f.questions.map((q, j) => (j === i ? { ...q, ...patch } : q)) }));
  return (
    <div className="space-y-2 border-t border-line pt-4" role="group" aria-labelledby="lf-q-label">
      <GroupLabel id="lf-q-label">{F.questions}</GroupLabel>
      <FieldError id="lf-q">{err("questions")}</FieldError>
      {form.questions.map((q, i) => {
        const id = (k: string) => `lf-q-${i}-${k}`;
        const e = (k: string) => err(`questions[${i}].${k}`);
        return (
          <div key={i} className="space-y-3 rounded-sm border border-line p-3">
            <div className="grid grid-cols-[minmax(0,1fr)_2.25rem] items-end gap-2">
              <Field label={F.questionLabel} htmlFor={id("label")} error={e("label")}>
                <Input id={id("label")} value={q.label} {...a11y(id("label"), e("label"))} onChange={(ev) => change(i, { label: ev.target.value }, "label")} />
              </Field>
              <Button type="button" variant="ghost" size="icon" className={cn("tap", e("label") && "mb-6")} aria-label={F.removeQuestion(i + 1)} onClick={() => edit("questions", (f) => ({ ...f, questions: f.questions.filter((_, j) => j !== i) }))}>
                <Trash2 />
              </Button>
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Field label={F.questionKey} htmlFor={id("key")} error={e("key")}>
                <Input id={id("key")} placeholder="size" className="font-mono" value={q.key} {...a11y(id("key"), e("key"))} onChange={(ev) => change(i, { key: ev.target.value }, "key")} />
              </Field>
              <Field label={F.questionType} htmlFor={id("type")} error={e("type")}>
                <NativeSelect id={id("type")} value={q.type} {...a11y(id("type"), e("type"))} onChange={(ev) => change(i, { type: ev.target.value as typeof q.type }, "type")}>
                  {(["text", "select", "checkbox"] as const).map((t) => (
                    <option key={t} value={t}>
                      {F.questionTypes[t]}
                    </option>
                  ))}
                </NativeSelect>
              </Field>
            </div>
            {q.type === "select" ? (
              <Field label={F.options} htmlFor={id("options")} error={e("options")}>
                <Textarea id={id("options")} rows={3} value={q.options} {...a11y(id("options"), e("options"))} onChange={(ev) => change(i, { options: ev.target.value }, "options")} />
              </Field>
            ) : null}
            <div className="flex flex-wrap gap-x-5 gap-y-1">
              <CheckLabel label={F.required} checked={q.required} onChange={(v) => change(i, { required: v }, "required")} />
              <CheckLabel label={F.perOrder} checked={q.per_order} onChange={(v) => change(i, { per_order: v }, "per_order")} />
            </div>
          </div>
        );
      })}
      <Button type="button" size="sm" variant="ghost" className="tap -ml-2" onClick={() => edit("questions", (f) => ({ ...f, questions: [...f.questions, emptyQuestion()] }))}>
        <Plus />
        {F.addQuestion}
      </Button>
    </div>
  );
}

function CheckLabel({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="inline-flex min-h-8 cursor-pointer items-center gap-2 text-body-sm text-ink pointer-coarse:min-h-11">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="size-4 accent-[var(--ink)]" />
      {label}
    </label>
  );
}

/** What step 3 shows for one method: the server's pricing of the saved link, or why there is none. */
export type MethodFeeView =
  | { kind: "priced"; preview: MethodPreview }
  | { kind: "refused"; code: string; message: string }
  | { kind: "pending" }
  | { kind: "pay_time" }
  | { kind: "no_amount" };

export function feeView(spec: MethodSpec, currency: string, pricing: Map<string, MethodPreview> | null, dropped: Map<string, DroppedMethod>): MethodFeeView {
  const key = methodKey(spec);
  const d = dropped.get(key);
  if (d) return { kind: "refused", code: d.code, message: d.message };
  if (!pricing) return { kind: "pending" };
  const p = pricing.get(key);
  if (!p) return { kind: "pending" };
  if (p.unavailable) return { kind: "refused", code: p.unavailable, message: p.unavailable };
  if (p.fee !== null) return { kind: "priced", preview: p };
  if (p.fee_currency && p.fee_currency !== currency) return { kind: "pay_time" };
  return { kind: "no_amount" };
}

interface Offer {
  spec: MethodSpec;
  enabled: boolean;
}

/** Offers in the server's order: enabled ones for the currency, then any chosen method that is not enabled here. */
function offersFor(form: LinkForm, options: LinkOptions | undefined): Offer[] {
  const enabled: Offer[] = (options?.methods ?? [])
    .filter((o) => !form.currency || o.currencies.includes(form.currency))
    .map((o) => ({
      spec: o.method === "crypto" ? { method: o.method, chain: o.chain ?? "", asset: o.asset ?? "" } : { method: o.method },
      enabled: Boolean(form.currency),
    }));
  const keys = new Set(enabled.map((o) => methodKey(o.spec)));
  const stray = form.methods.filter((m) => !keys.has(methodKey(m))).map((spec) => ({ spec, enabled: false }));
  return [...enabled.filter((o) => o.enabled), ...stray];
}

export function PaymentStep({
  form,
  edit,
  err,
  locked,
  options,
  pricing,
  dropped,
}: StepProps & { options: OptionsState; pricing: Map<string, MethodPreview> | null; dropped: Map<string, DroppedMethod> }) {
  const chosen = new Map(form.methods.map((m, i) => [methodKey(m), i]));
  const offers = locked ? form.methods.map((spec) => ({ spec, enabled: true })) : offersFor(form, options.data);
  const order = offers.map((o) => methodKey(o.spec));
  const toggle = (spec: MethodSpec, on: boolean) =>
    edit("methods", (f) => {
      const key = methodKey(spec);
      const next = on ? [...f.methods.filter((m) => methodKey(m) !== key), spec] : f.methods.filter((m) => methodKey(m) !== key);
      next.sort((a, b) => order.indexOf(methodKey(a)) - order.indexOf(methodKey(b)));
      return { ...f, methods: next };
    });
  const has = (m: string) => form.methods.some((x) => x.method === m);
  const data = options.data;
  const empty = !locked && data && offers.length === 0;
  const noCatalog = !locked && data && data.methods.length === 0;

  return (
    <>
      <div className="space-y-2" role="group" aria-labelledby="lf-methods-label">
        <GroupLabel id="lf-methods-label">{F.methods}</GroupLabel>
        <FieldError id="lf-methods">{err("methods")}</FieldError>
        {!locked && !data ? <OptionsProblem options={options} /> : null}
        {noCatalog && offers.length === 0 ? (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-sm border border-dashed border-line-strong px-3 py-2.5 text-body-sm text-ink-soft">
            <span>{F.noOptions}</span>
            <Link href="/dashboard/settings" className="tap rounded-xs font-medium text-tide hover:text-tide-strong">
              {F.openSettings}
            </Link>
          </div>
        ) : !locked && !data && offers.length === 0 ? null : empty ? (
          <p className="rounded-sm border border-dashed border-line-strong px-3 py-2.5 text-body-sm text-ink-soft">
            {!form.currency ? F.chooseCurrencyFirst : F.noOptionsForCurrency(form.currency)}
          </p>
        ) : (
          <ul className="divide-y divide-line rounded-sm border border-line">
            {offers.map(({ spec, enabled }) => {
              const key = methodKey(spec);
              const i = chosen.get(key);
              const on = i !== undefined;
              return (
                <MethodRow
                  key={key}
                  id={`lf-m-${key.replace(/[^a-z0-9]/gi, "-")}`}
                  label={methodLabel(spec)}
                  on={on}
                  locked={locked}
                  notEnabled={!enabled && !locked}
                  onToggle={(v) => toggle(spec, v)}
                  fee={on ? feeView(spec, form.currency, pricing, dropped) : undefined}
                  customerMode={form.amount_mode === "customer"}
                  error={i !== undefined ? err(`methods[${i}]`, true) : undefined}
                  currency={form.currency}
                />
              );
            })}
          </ul>
        )}
      </div>

      {locked ? (
        <ReadOnly label={F.feeBearer}>{F.bearers[form.fee_bearer]}</ReadOnly>
      ) : (
        <Group label={F.feeBearer} id="lf-bearer" error={err("fee_bearer")}>
          <Segmented
            id="lf-bearer"
            error={err("fee_bearer")}
            label={F.feeBearer}
            value={form.fee_bearer}
            options={(["merchant", "customer"] as const).map((v) => ({ value: v, label: F.bearers[v] }))}
            onChange={(v) => edit("fee_bearer", set("fee_bearer", v))}
          />
        </Group>
      )}

      {has("card") ? (
        <div className="grid gap-3 border-t border-line pt-4 sm:grid-cols-2">
          <Group label={F.capture} id="lf-capture" error={err("capture_mode")}>
            <Segmented id="lf-capture" error={err("capture_mode")} label={F.capture} value={form.capture_mode} className="w-full" options={(["automatic", "manual"] as const).map((v) => ({ value: v, label: F.captureModes[v] }))} onChange={(v) => edit("capture_mode", set("capture_mode", v))} />
          </Group>
          <Group label={F.threeDS} id="lf-3ds" error={err("three_ds_policy")}>
            <Segmented id="lf-3ds" error={err("three_ds_policy")} label={F.threeDS} value={form.three_ds_policy} className="w-full" options={(["inherit", "force"] as const).map((v) => ({ value: v, label: F.threeDSModes[v] }))} onChange={(v) => edit("three_ds_policy", set("three_ds_policy", v))} />
          </Group>
        </div>
      ) : (
        <FieldError id="lf-capture">{err("capture_mode")}</FieldError>
      )}

      {has("crypto") ? (
        <div className="grid grid-cols-2 gap-3 border-t border-line pt-4">
          <Field label={F.tolerance} htmlFor="lf-tol" error={err("chain_tolerance_bps")}>
            <Input id="lf-tol" inputMode="numeric" className="num" value={form.chain_tolerance_bps} {...a11y("lf-tol", err("chain_tolerance_bps"))} onChange={(e) => edit("chain_tolerance_bps", set("chain_tolerance_bps", e.target.value))} />
          </Field>
          <Field label={F.quoteExpiry} htmlFor="lf-quote" error={err("quote_expiry_seconds")}>
            <Input id="lf-quote" inputMode="numeric" className="num" value={form.quote_expiry_seconds} {...a11y("lf-quote", err("quote_expiry_seconds"))} onChange={(e) => edit("quote_expiry_seconds", set("quote_expiry_seconds", e.target.value))} />
          </Field>
        </div>
      ) : null}
    </>
  );
}

function MethodRow({
  id,
  label,
  on,
  locked,
  notEnabled,
  onToggle,
  fee,
  customerMode,
  error,
  currency,
}: {
  id: string;
  label: string;
  on: boolean;
  locked: boolean;
  notEnabled: boolean;
  onToggle: (v: boolean) => void;
  fee?: MethodFeeView;
  customerMode: boolean;
  error?: string;
  currency: string;
}) {
  return (
    <li className="px-3 py-1.5">
      <div className="flex min-h-8 items-center gap-3 pointer-coarse:min-h-11">
        {locked ? (
          <span className="flex-1 text-body-sm font-medium text-ink">{label}</span>
        ) : (
          <label className="flex min-h-8 flex-1 cursor-pointer items-center gap-3 pointer-coarse:min-h-11">
            <input type="checkbox" id={id} checked={on} onChange={(e) => onToggle(e.target.checked)} {...a11y(id, error)} className="size-4 shrink-0 accent-[var(--ink)]" />
            <span className="text-body-sm font-medium text-ink">
              {label}
              {notEnabled ? <span className="font-normal text-wait"> ({F.notEnabled})</span> : null}
            </span>
          </label>
        )}
        {on && fee ? <FeeBadge fee={fee} customerMode={customerMode} /> : null}
      </div>
      {on && fee?.kind === "priced" ? <FeeGrid preview={fee.preview} currency={currency} indent={!locked} /> : null}
      {error ? (
        <div className={cn("pb-1", !locked && "pl-7")}>
          <FieldError id={id}>{error}</FieldError>
        </div>
      ) : null}
    </li>
  );
}

function FeeBadge({ fee, customerMode }: { fee: MethodFeeView; customerMode: boolean }) {
  switch (fee.kind) {
    case "priced":
      return (
        <span className="flex shrink-0 items-center gap-2">
          {customerMode ? <span className="text-caption text-ink-soft">{FEE.onMinimum}</span> : null}
          {fee.preview.rule_id !== null && fee.preview.rule_version !== null ? (
            <span className="num rounded-xs bg-surface-sunken px-1.5 py-0.5 font-mono text-caption text-ink-soft">{FEE.rule(fee.preview.rule_id, fee.preview.rule_version)}</span>
          ) : null}
        </span>
      );
    case "pending":
      return <span className="shrink-0 text-caption text-ink-faint">{FEE.pending}</span>;
    case "no_amount":
      return <span className="shrink-0 text-caption text-ink-faint">{FEE.noAmount}</span>;
    case "pay_time":
      return <span className="shrink-0 text-caption text-ink-soft">{FEE.atPayTime}</span>;
    case "refused":
      return <span className="max-w-[55%] shrink-0 text-right text-caption text-wait">{FEE.refused[fee.code] ?? fee.message}</span>;
  }
}

/** The same four columns for every method so rows line up; the tax cell stays empty when there is none. */
function FeeGrid({ preview, currency, indent }: { preview: MethodPreview; currency: string; indent: boolean }) {
  const cur = preview.fee_currency || currency;
  const cells: [string, string | null][] = [
    [FEE.fee, preview.fee],
    [FEE.tax, preview.tax && Number(preview.tax) > 0 ? preview.tax : null],
    [FEE.total, preview.customer_total],
    [FEE.net, preview.merchant_net],
  ];
  return (
    <dl className={cn("grid grid-cols-2 gap-x-3 gap-y-1 pb-1.5 sm:grid-cols-4", indent && "pl-7")}>
      {cells.map(([k, v]) => (
        <div key={k} className="min-w-0">
          <dt className="truncate text-caption text-ink-soft">{k}</dt>
          <dd className="truncate">{v ? <CurrencyDisplay amount={v} currency={cur} size="sm" /> : null}</dd>
        </div>
      ))}
    </dl>
  );
}

export function AfterStep({ form, edit, err, webhooks }: StepProps & { webhooks: { id: number; url: string }[] }) {
  return (
    <>
      <Group label={F.success} id="lf-success" error={err("success_mode")}>
        <Segmented id="lf-success" error={err("success_mode")} label={F.success} value={form.success_mode} options={(["message", "redirect"] as const).map((v) => ({ value: v, label: F.successModes[v] }))} onChange={(v) => edit("success_mode", set("success_mode", v))} />
      </Group>
      {form.success_mode === "message" ? (
        <Field label={F.successMessage} htmlFor="lf-succ" error={err("success_message")}>
          <Textarea id="lf-succ" rows={2} maxLength={1000} value={form.success_message} {...a11y("lf-succ", err("success_message"))} onChange={(e) => edit("success_message", set("success_message", e.target.value))} />
        </Field>
      ) : (
        <Field label={F.successUrl} htmlFor="lf-surl" error={err("success_url")} hint={F.successUrlHint}>
          <Input id="lf-surl" type="url" placeholder="https://example.com/thanks" value={form.success_url} {...a11y("lf-surl", err("success_url"))} onChange={(e) => edit("success_url", set("success_url", e.target.value))} />
        </Field>
      )}

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.receipt} checked={form.receipt_email} onChange={(v) => edit("receipt_email", set("receipt_email", v))} error={err("receipt_email")} />
        {form.receipt_email ? (
          <Field label={F.receiptNote} htmlFor="lf-rnote" error={err("receipt_note")}>
            <Textarea id="lf-rnote" rows={2} maxLength={1000} value={form.receipt_note} {...a11y("lf-rnote", err("receipt_note"))} onChange={(e) => edit("receipt_note", set("receipt_note", e.target.value))} />
          </Field>
        ) : null}
      </div>

      <Field label={F.webhook} htmlFor="lf-webhook" error={err("webhook_id")} className="border-t border-line pt-4">
        <NativeSelect id="lf-webhook" value={form.webhook_id} {...a11y("lf-webhook", err("webhook_id"))} onChange={(e) => edit("webhook_id", set("webhook_id", e.target.value))}>
          <option value="">{F.noWebhook}</option>
          {webhooks.map((w) => (
            <option key={w.id} value={String(w.id)}>
              {w.url}
            </option>
          ))}
          {form.webhook_id && !webhooks.some((w) => String(w.id) === form.webhook_id) ? <option value={form.webhook_id}>#{form.webhook_id}</option> : null}
        </NativeSelect>
      </Field>

      <div className="space-y-3 border-t border-line pt-4">
        <ToggleRow label={F.failureRetry} checked={form.failure_retry} onChange={(v) => edit("failure_retry", set("failure_retry", v))} error={err("failure_retry")} />
        <Field label={F.failureMessage} htmlFor="lf-fail" error={err("failure_message")}>
          <Textarea id="lf-fail" rows={2} maxLength={1000} value={form.failure_message} {...a11y("lf-fail", err("failure_message"))} onChange={(e) => edit("failure_message", set("failure_message", e.target.value))} />
        </Field>
      </div>
    </>
  );
}

export function SettlementStep({ form, edit, err }: StepProps) {
  const s = form.settlement;
  const setS = (patch: Partial<LinkForm["settlement"]>) => (f: LinkForm): LinkForm => ({ ...f, settlement: { ...f.settlement, ...patch } });
  const kindErr = err("settlement_override.kind") ?? err("settlement_override");
  return (
    <>
      <Group label={F.settlement} id="lf-settle" error={kindErr}>
        <Segmented id="lf-settle" error={kindErr} label={F.settlement} value={s.mode} options={(["default", "fiat", "crypto"] as const).map((v) => ({ value: v, label: F.settlementModes[v] }))} onChange={(v) => edit("settlement_override", setS({ mode: v }))} />
      </Group>
      {s.mode !== "default" ? (
        <div className="grid grid-cols-2 gap-3">
          <Field label={F.destination} htmlFor="lf-dest" error={err("settlement_override.destination_id")} className="col-span-2">
            <Input id="lf-dest" className="font-mono" value={s.destination_id} {...a11y("lf-dest", err("settlement_override.destination_id"))} onChange={(e) => edit("settlement_override.destination_id", setS({ destination_id: e.target.value }))} />
          </Field>
          {s.mode === "crypto" ? (
            <>
              <Field label={F.chain} htmlFor="lf-schain" error={err("settlement_override.chain")}>
                <Input id="lf-schain" className="uppercase" value={s.chain} {...a11y("lf-schain", err("settlement_override.chain"))} onChange={(e) => edit("settlement_override.chain", setS({ chain: e.target.value.toUpperCase() }))} />
              </Field>
              <Field label={F.asset} htmlFor="lf-sasset" error={err("settlement_override.asset")}>
                <Input id="lf-sasset" className="uppercase" value={s.asset} {...a11y("lf-sasset", err("settlement_override.asset"))} onChange={(e) => edit("settlement_override.asset", setS({ asset: e.target.value.toUpperCase() }))} />
              </Field>
            </>
          ) : null}
        </div>
      ) : null}
      <div className="border-t border-line pt-4">
        <ToggleRow label={F.holdInAsset} hint={F.holdNote} checked={form.hold_in_asset} onChange={(v) => edit("hold_in_asset", set("hold_in_asset", v))} error={err("hold_in_asset")} />
      </div>
      <Group label={F.timing} id="lf-timing" error={err("settlement_timing")}>
        <Segmented id="lf-timing" error={err("settlement_timing")} label={F.timing} value={form.settlement_timing} options={(["cycle", "immediate"] as const).map((v) => ({ value: v, label: F.timingModes[v] }))} onChange={(v) => edit("settlement_timing", set("settlement_timing", v))} />
      </Group>
    </>
  );
}

export function LifecycleStep({ form, edit, err }: StepProps) {
  const accentOk = /^#[0-9a-fA-F]{6}$/.test(form.accent_color);
  return (
    <>
      <Field label={F.expiresAt} htmlFor="lf-exp" error={err("expires_at")} className="sm:max-w-[50%]">
        <Input id="lf-exp" type="datetime-local" className="num" value={form.expires_at} {...a11y("lf-exp", err("expires_at"))} onChange={(e) => edit("expires_at", set("expires_at", e.target.value))} />
      </Field>
      <div className="grid gap-3 border-t border-line pt-4 sm:grid-cols-2">
        <Field label={F.logo} htmlFor="lf-logo" error={err("logo_url")} className="sm:col-span-2">
          <Input id="lf-logo" type="url" placeholder="https://" value={form.logo_url} {...a11y("lf-logo", err("logo_url"))} onChange={(e) => edit("logo_url", set("logo_url", e.target.value))} />
        </Field>
        <Field label={F.accent} htmlFor="lf-accent" error={err("accent_color")}>
          <div className="flex items-center gap-2">
            <label
              className={cn(
                "relative flex size-9 shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm border pointer-coarse:size-11",
                accentOk ? "border-line-strong" : "border-dashed border-line-strong text-caption text-ink-faint"
              )}
              style={accentOk ? { background: form.accent_color } : undefined}
            >
              {accentOk ? null : <span aria-hidden>-</span>}
              <input
                type="color"
                aria-label={F.accentPicker}
                value={accentOk ? form.accent_color.toLowerCase() : "#0b7285"}
                onChange={(e) => edit("accent_color", set("accent_color", e.target.value.toUpperCase()))}
                className="absolute inset-0 size-full cursor-pointer opacity-0"
              />
            </label>
            <Input id="lf-accent" className="font-mono uppercase placeholder:normal-case placeholder:font-sans" placeholder={F.noAccent} maxLength={7} value={form.accent_color} {...a11y("lf-accent", err("accent_color"))} onChange={(e) => edit("accent_color", set("accent_color", e.target.value))} />
          </div>
        </Field>
        <Field label={F.language} htmlFor="lf-lang" error={err("language")}>
          <Input id="lf-lang" placeholder="en" maxLength={35} value={form.language} {...a11y("lf-lang", err("language"))} onChange={(e) => edit("language", set("language", e.target.value))} />
        </Field>
      </div>
    </>
  );
}
