"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Archive, Copy, Pause, Play, Trash2 } from "lucide-react";
import type { PaymentLink } from "@/lib/api/links";
import {
  useArchiveLink,
  useCreateLink,
  useDeleteLink,
  useDuplicateLink,
  useMethodFees,
  usePauseLink,
  usePublishLink,
  useUpdateLink,
} from "@/lib/query/hooks/use-links";
import { useWebhooksList } from "@/lib/query/hooks/use-webhooks";
import { useBlockchainCurrencies } from "@/lib/query/hooks/use-public";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Notice } from "@/components/notice";
import { CopyField } from "@/components/copy-field";
import { DateTime } from "@/components/date-time";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { LINKS_COPY, methodLabel } from "../copy";
import {
  countByStep,
  errorAt,
  mapApiErrors,
  NO_ERRORS,
  STEP_ORDER,
  stepForField,
  type MappedErrors,
  type StepId,
} from "../errors";
import {
  defaultForm,
  formFromLink,
  formToInput,
  formToPatch,
  hasContent,
  isPublished,
  localErrors,
  pricingFingerprint,
  type LinkForm,
} from "../model";
import { buildRenderModel, feeRequestFor } from "../preview";
import { CheckoutPreview } from "./checkout-preview";
import { Segmented, Step } from "./controls";
import { QrButton } from "./link-share";
import { AfterStep, CustomerStep, ItemStep, LifecycleStep, PaymentStep, SettlementStep } from "./steps";

const B = LINKS_COPY.builder;
const F = LINKS_COPY.fields;

type SaveState = "idle" | "saving" | "saved" | "error";

function useDebounced<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

/** Drop messages for `path` and anything under it: the merchant has changed that value. */
function without(e: MappedErrors, path: string): MappedErrors {
  const keys = Object.keys(e.fields).filter((k) => k === path || k.startsWith(path + ".") || k.startsWith(path + "["));
  if (keys.length === 0) return e;
  const fields = { ...e.fields };
  for (const k of keys) delete fields[k];
  return { ...e, fields };
}

function summary(step: StepId, f: LinkForm, total: string | null): string {
  const cur = f.currency || "";
  switch (step) {
    case "item":
      if (f.amount_mode === "fixed") return [f.title || LINKS_COPY.untitled, f.amount ? `${f.amount} ${cur}` : ""].filter(Boolean).join(", ");
      if (f.amount_mode === "customer") return [f.title || LINKS_COPY.untitled, F.modes.customer].join(", ");
      return [f.title || LINKS_COPY.untitled, `${f.line_items.length} ${f.line_items.length === 1 ? "item" : "items"}`, total ? `${total} ${cur}` : ""].filter(Boolean).join(", ");
    case "customer": {
      const asked = (["email", "name", "phone"] as const).filter((k) => f.customer[k].mode !== "hidden").map((k) => F.customerFields[k]);
      return [asked.length ? asked.join(", ") : "No contact fields", f.multi_use ? F.multiUse : "Single use"].join(", ");
    }
    case "payment":
      return f.methods.length ? `${f.methods.map(methodLabel).join(", ")}, fees: ${F.bearers[f.fee_bearer]}` : "No methods";
    case "after":
      return [F.successModes[f.success_mode], f.receipt_email ? F.receipt : ""].filter(Boolean).join(", ");
    case "settlement":
      return `${F.settlementModes[f.settlement.mode]}, ${F.timingModes[f.settlement_timing]}`;
    case "lifecycle":
      return [f.expires_at ? `${F.expiresAt} ${f.expires_at.replace("T", " ")}` : "No end date", f.language].filter(Boolean).join(", ");
  }
}

export function LinkBuilder({ link: initial, basePath = "/dashboard/links" }: { link: PaymentLink | null; basePath?: string }) {
  const router = useRouter();
  const [form, setForm] = useState<LinkForm>(() => (initial ? formFromLink(initial) : defaultForm()));
  const [current, setCurrent] = useState<PaymentLink | null>(initial);
  const [saveErrors, setSaveErrors] = useState<MappedErrors>(NO_ERRORS);
  const [publishErrors, setPublishErrors] = useState<MappedErrors>(NO_ERRORS);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [open, setOpen] = useState<Set<StepId>>(() => new Set(initial && isPublished(initial.status) ? [] : ["item"]));
  const [view, setView] = useState<"edit" | "preview">("edit");
  const [confirm, setConfirm] = useState<null | "archive" | "delete">(null);

  const create = useCreateLink();
  const update = useUpdateLink();
  const publishM = usePublishLink();
  const pauseM = usePauseLink();
  const archiveM = useArchiveLink();
  const duplicateM = useDuplicateLink();
  const deleteM = useDeleteLink();
  const webhooks = useWebhooksList();
  const chains = useBlockchainCurrencies();

  const status = current?.status ?? null;
  const locked = status !== null && isPublished(status);
  const readOnly = status === "archived";

  const local = useMemo(() => localErrors(form), [form]);
  const hasLocal = Object.keys(local).length > 0;
  const sendable = useMemo(() => (hasLocal ? null : JSON.stringify(formToInput(form))), [form, hasLocal]);

  const formRef = useRef(form);
  const currentRef = useRef(current);
  const [savedFp, setSavedFp] = useState(() => JSON.stringify(formToInput(initial ? formFromLink(initial) : defaultForm())));
  const savedRef = useRef(savedFp);
  const saveErrRef = useRef<MappedErrors>(NO_ERRORS);
  const queue = useRef<Promise<PaymentLink | null>>(Promise.resolve(initial));
  useEffect(() => {
    formRef.current = form;
    currentRef.current = current;
  });

  const dirty = sendable !== null ? sendable !== savedFp : true;

  const doSave = useCallback(async (): Promise<PaymentLink | null> => {
    const f = formRef.current;
    const cur = currentRef.current;
    if (Object.keys(localErrors(f)).length > 0) return null;
    const fp = JSON.stringify(formToInput(f));
    if (cur && fp === savedRef.current) return cur;
    if (!cur && !hasContent(f)) return null;
    setSaveState("saving");
    try {
      const saved = cur
        ? await update.mutateAsync({ id: cur.id, patch: formToPatch(f, cur.status) })
        : await create.mutateAsync(formToInput(f));
      savedRef.current = fp;
      setSavedFp(fp);
      saveErrRef.current = NO_ERRORS;
      currentRef.current = saved;
      setCurrent(saved);
      setSaveErrors(NO_ERRORS);
      setSaveState("saved");
      if (!cur) window.history.replaceState(null, "", `${basePath}/${saved.id}`);
      return saved;
    } catch (e) {
      const mapped = mapApiErrors(e);
      saveErrRef.current = mapped;
      setSaveErrors(mapped);
      setSaveState("error");
      return null;
    }
  }, [basePath, create, update]);

  const enqueueSave = useCallback(() => {
    queue.current = queue.current.then(doSave, doSave);
    return queue.current;
  }, [doSave]);

  // Drafts save themselves; a published link waits for "Save changes" so nothing goes live by accident.
  useEffect(() => {
    if (locked || readOnly || sendable === null || sendable === savedFp) return;
    if (!currentRef.current && !hasContent(formRef.current)) return;
    const t = setTimeout(() => void enqueueSave(), 800);
    return () => clearTimeout(t);
  }, [sendable, savedFp, locked, readOnly, enqueueSave]);

  const edit = useCallback((path: string, fn: (f: LinkForm) => LinkForm) => {
    setForm(fn);
    setSaveErrors((e) => without(e, path));
    setPublishErrors((e) => without(e, path));
  }, []);

  const fields = useMemo(
    () => ({ ...publishErrors.fields, ...saveErrors.fields, ...local }),
    [publishErrors.fields, saveErrors.fields, local]
  );
  const general = [...saveErrors.general, ...publishErrors.general];
  const err = useCallback((path: string, nested = false) => errorAt(fields, path, nested), [fields]);
  const counts = countByStep(fields);
  const errorTotal = Object.keys(fields).length;

  const openStepsFor = (m: MappedErrors) =>
    setOpen((o) => {
      const next = new Set(o);
      for (const k of Object.keys(m.fields)) {
        const s = stepForField(k);
        if (s) next.add(s);
      }
      return next;
    });

  // Fees and the line-item total come from the API; the total is shown only while it matches the form.
  const serverTotal = current && pricingFingerprint(formFromLink(current)) === pricingFingerprint(form) ? current.total : null;
  const pricingForm = useDebounced(form, 400);
  const pricingTotal = current && pricingFingerprint(formFromLink(current)) === pricingFingerprint(pricingForm) ? current.total : null;
  const plans = useMemo(() => pricingForm.methods.map((m) => feeRequestFor(pricingForm, m, pricingTotal)), [pricingForm, pricingTotal]);
  const fees = useMethodFees(plans);
  const alignedFees = form.methods.length === pricingForm.methods.length ? fees : form.methods.map(() => ({ state: "loading" as const }));
  const model = buildRenderModel(form, { link: current, serverTotal, fees: alignedFees, merchantName: null });

  const cryptoOptions = useMemo(() => {
    const seen = new Set<string>();
    const out: { chain: string; asset: string }[] = [];
    for (const c of chains.data?.currencies ?? []) {
      const chain = c.blockchainCode.toUpperCase();
      const asset = c.currencyCode.toUpperCase();
      if (seen.has(`${chain}:${asset}`)) continue;
      seen.add(`${chain}:${asset}`);
      out.push({ chain, asset });
    }
    return out;
  }, [chains.data]);

  async function publish() {
    if (hasLocal) {
      openStepsFor({ fields: local, general: [] });
      return;
    }
    const saved = await enqueueSave();
    if (!saved) {
      openStepsFor(saveErrRef.current);
      return;
    }
    try {
      const l = await publishM.mutateAsync(saved.id);
      setCurrent(l);
      currentRef.current = l;
      setPublishErrors(NO_ERRORS);
      toast.success(status === "paused" ? "Link resumed" : "Link published");
    } catch (e) {
      const m = mapApiErrors(e);
      setPublishErrors(m);
      openStepsFor(m);
    }
  }

  async function transition(kind: "pause" | "archive") {
    if (!current) return;
    try {
      const l = await (kind === "pause" ? pauseM : archiveM).mutateAsync(current.id);
      setCurrent(l);
      currentRef.current = l;
      setConfirm(null);
    } catch (e) {
      setPublishErrors(mapApiErrors(e));
      setConfirm(null);
    }
  }

  async function duplicate() {
    if (!current) return;
    try {
      const l = await duplicateM.mutateAsync(current.id);
      router.push(`${basePath}/${l.id}`);
    } catch (e) {
      setPublishErrors(mapApiErrors(e));
    }
  }

  async function remove() {
    if (!current) return;
    try {
      await deleteM.mutateAsync(current.id);
      router.push(basePath);
    } catch (e) {
      setPublishErrors(mapApiErrors(e));
      setConfirm(null);
    }
  }

  const busy = publishM.isPending || pauseM.isPending || archiveM.isPending || duplicateM.isPending;
  const toggle = (s: StepId) =>
    setOpen((o) => {
      const n = new Set(o);
      if (n.has(s)) n.delete(s);
      else n.add(s);
      return n;
    });
  const allOpen = open.size === STEP_ORDER.length;
  const onDuplicate = current ? () => void duplicate() : undefined;

  const stepProps = { form, edit, err, locked };
  const stepBody: Record<StepId, React.ReactNode> = {
    item: <ItemStep {...stepProps} onDuplicate={onDuplicate} />,
    customer: <CustomerStep {...stepProps} />,
    payment: <PaymentStep {...stepProps} fees={alignedFees} cryptoOptions={cryptoOptions} onDuplicate={onDuplicate} />,
    after: <AfterStep {...stepProps} webhooks={(webhooks.data ?? []).map((w) => ({ id: w.id, url: w.url }))} />,
    settlement: <SettlementStep {...stepProps} />,
    lifecycle: <LifecycleStep {...stepProps} />,
  };

  const saveLabel =
    hasLocal || saveState === "error"
      ? B.notSaved
      : saveState === "saving"
        ? B.saving
        : locked && dirty
          ? B.unsaved
          : current && !dirty
            ? B.saved
            : "";

  return (
    <div className="space-y-5">
      <PageHeader
        breadcrumbs={[{ label: LINKS_COPY.listTitle, href: basePath }, { label: form.title || (current ? LINKS_COPY.untitled : B.newTitle) }]}
        title={
          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <span className="min-w-0 truncate">{form.title || (current ? LINKS_COPY.untitled : B.newTitle)}</span>
            {status ? <StatusBadge status={status} /> : null}
          </span>
        }
      >
        {saveLabel ? (
          <span role="status" aria-live="polite" className={cn("hidden text-label sm:inline", saveLabel === B.notSaved ? "text-bad" : "text-ink-soft")}>
            {saveLabel}
          </span>
        ) : null}
        {status === "draft" ? (
          <Button variant="ghost" size="icon" aria-label={B.deleteDraft} onClick={() => setConfirm("delete")}>
            <Trash2 />
          </Button>
        ) : null}
        {status === "active" ? (
          <Button variant="outline" disabled={busy} onClick={() => void transition("pause")}>
            <Pause />
            {B.pause}
          </Button>
        ) : null}
        {status && status !== "draft" ? (
          <Button variant="outline" disabled={busy} onClick={() => void duplicate()}>
            <Copy />
            <span className="max-sm:sr-only">{B.duplicate}</span>
          </Button>
        ) : null}
        {locked ? (
          <Button variant="outline" size="icon" aria-label={B.archive} disabled={busy} onClick={() => setConfirm("archive")}>
            <Archive />
          </Button>
        ) : null}
        {locked && dirty ? (
          <Button disabled={hasLocal || saveState === "saving"} onClick={() => void enqueueSave()}>
            {B.save}
          </Button>
        ) : null}
        {status === null || status === "draft" ? (
          <Button disabled={busy || (!current && !hasContent(form))} onClick={() => void publish()}>
            {B.publish}
          </Button>
        ) : null}
        {status === "paused" ? (
          <Button disabled={busy} onClick={() => void publish()}>
            <Play />
            {B.resume}
          </Button>
        ) : null}
      </PageHeader>

      {current?.url ? (
        <div className="flex flex-col gap-3 rounded-md border border-line bg-surface p-4 sm:flex-row sm:items-center sm:p-5">
          <CopyField value={current.url} className="min-w-0 flex-1" />
          <div className="flex items-center gap-4">
            <QrButton url={current.url} title={current.title} size="sm">
              {LINKS_COPY.detail.qr}
            </QrButton>
            <dl className="flex items-center gap-4 text-body-sm">
              <div>
                <dt className="text-caption text-ink-soft">{LINKS_COPY.detail.uses}</dt>
                <dd className="num text-ink">{current.uses_count}</dd>
              </div>
              {current.published_at ? (
                <div>
                  <dt className="text-caption text-ink-soft">{LINKS_COPY.detail.published}</dt>
                  <dd className="text-ink">
                    <DateTime value={current.published_at} format="date" />
                  </dd>
                </div>
              ) : null}
            </dl>
          </div>
        </div>
      ) : null}

      {general.length > 0 || (errorTotal > 0 && (publishErrors !== NO_ERRORS || saveState === "error")) ? (
        <Notice tone="bad">
          {general.length > 0 ? general.map((g, i) => <p key={i}>{g.message}</p>) : B.fixErrors(errorTotal)}
        </Notice>
      ) : null}

      <div className="lg:hidden">
        <Segmented
          label={B.preview}
          value={view}
          className="w-full"
          options={[
            { value: "edit", label: B.edit },
            { value: "preview", label: B.preview },
          ]}
          onChange={setView}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,34rem)_minmax(0,1fr)]">
        <fieldset disabled={readOnly} className={cn("min-w-0 space-y-3", view === "preview" && "max-lg:hidden")}>
          <legend className="sr-only">{B.settings}</legend>
          <div className="flex justify-end">
            <Button type="button" variant="ghost" size="xs" onClick={() => setOpen(allOpen ? new Set() : new Set(STEP_ORDER))}>
              {allOpen ? "Collapse all" : "Expand all"}
            </Button>
          </div>
          {STEP_ORDER.map((s, i) => (
            <Step key={s} index={i + 1} title={LINKS_COPY.steps[s].title} summary={summary(s, form, serverTotal)} open={open.has(s)} onToggle={() => toggle(s)} errors={counts[s] ?? 0}>
              {stepBody[s]}
            </Step>
          ))}
        </fieldset>
        <div className={cn("min-w-0", view === "edit" && "max-lg:hidden")}>
          <div className="lg:sticky lg:top-20">
            <CheckoutPreview model={model} />
          </div>
        </div>
      </div>

      <Dialog open={confirm !== null} onOpenChange={(v) => !v && setConfirm(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{confirm === "archive" ? B.archive : B.deleteDraft}</DialogTitle>
            <DialogDescription>{confirm === "archive" ? B.archiveConfirm : B.deleteConfirm}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirm(null)}>
              Cancel
            </Button>
            <Button variant="destructive" disabled={archiveM.isPending || deleteM.isPending} onClick={() => void (confirm === "archive" ? transition("archive") : remove())}>
              {confirm === "archive" ? B.archive : B.deleteDraft}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
