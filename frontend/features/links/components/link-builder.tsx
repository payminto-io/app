"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Archive, Copy, Pause, Play, Trash2 } from "lucide-react";
import type { PaymentLink } from "@/lib/api/links";
import { isApiError } from "@/lib/api/errors";
import {
  useArchiveLink,
  useCreateLink,
  useDeleteLink,
  useDuplicateLink,
  useLinkOptions,
  useLinkPreview,
  usePauseLink,
  usePublishLink,
  useUpdateLink,
} from "@/lib/query/hooks/use-links";
import { useWebhooksList } from "@/lib/query/hooks/use-webhooks";
import { PageHeader } from "@/components/page-header";
import { StatusBadge } from "@/components/status-badge";
import { Notice } from "@/components/notice";
import { DateTime } from "@/components/date-time";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { formatDecimal } from "@/lib/money";
import { cn } from "@/lib/utils";
import { LINKS_COPY, methodLabel } from "../copy";
import { countByStep, errorAt, mapApiErrors, NO_ERRORS, STEP_ORDER, stepForField, type MappedErrors, type StepId } from "../errors";
import { defaultForm, formFromLink, formToInput, formToPatch, hasContent, isPublished, localErrors, type LinkForm } from "../model";
import { droppedByKey, savedPricing } from "../pricing";
import { CheckoutPreview, type PreviewState } from "./checkout-preview";
import { Segmented, Step } from "./controls";
import { ShortLink, QrButton } from "./link-share";
import { AfterStep, CustomerStep, ItemStep, LifecycleStep, LockedNote, PaymentStep, SettlementStep } from "./steps";

const B = LINKS_COPY.builder;
const F = LINKS_COPY.fields;
const S = LINKS_COPY.summary;

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

const amountText = (v: string, cur: string) => (cur ? `${formatDecimal(v, cur)} ${cur}` : "");

function summary(step: StepId, f: LinkForm): string {
  const title = f.title || LINKS_COPY.untitled;
  switch (step) {
    case "item":
      if (f.amount_mode === "fixed") return [title, f.amount ? amountText(f.amount, f.currency) : ""].filter(Boolean).join(", ");
      if (f.amount_mode === "customer") return [title, F.modes.customer].join(", ");
      return [title, S.items(f.line_items.length)].join(", ");
    case "customer": {
      const asked = (["email", "name", "phone"] as const).filter((k) => f.customer[k].mode !== "hidden").map((k) => F.customerFields[k]);
      return [asked.length ? asked.join(", ") : S.noContact, f.multi_use ? F.multiUse : S.singleUse].join(", ");
    }
    case "payment":
      return f.methods.length ? `${f.methods.map(methodLabel).join(", ")}, ${S.fees(F.bearers[f.fee_bearer])}` : S.noMethods;
    case "after":
      return [F.successModes[f.success_mode], f.receipt_email ? F.receipt : ""].filter(Boolean).join(", ");
    case "settlement":
      return `${F.settlementModes[f.settlement.mode]}, ${F.timingModes[f.settlement_timing]}`;
    case "lifecycle": {
      const when = f.expires_at ? new Date(f.expires_at).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "";
      return [when ? S.ends(when) : S.noEnd, f.language].filter(Boolean).join(", ");
    }
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
  const [touched, setTouched] = useState<Set<string>>(() => new Set());
  const [showAll, setShowAll] = useState(false);
  const [focusNonce, setFocusNonce] = useState(0);
  const formEl = useRef<HTMLFieldSetElement>(null);

  const create = useCreateLink();
  const update = useUpdateLink();
  const publishM = usePublishLink();
  const pauseM = usePauseLink();
  const archiveM = useArchiveLink();
  const duplicateM = useDuplicateLink();
  const deleteM = useDeleteLink();
  const webhooks = useWebhooksList();
  const options = useLinkOptions();

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
      const saved = cur ? await update.mutateAsync({ id: cur.id, patch: formToPatch(f, cur.status) }) : await create.mutateAsync(formToInput(f));
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
    setTouched((t) => (t.has(path) ? t : new Set(t).add(path)));
    setSaveErrors((e) => without(e, path));
    setPublishErrors((e) => without(e, path));
  }, []);

  // Local messages show once a field was edited or a publish was tried; they always block saving.
  const shownLocal = useMemo(
    () => Object.fromEntries(Object.entries(local).filter(([k]) => showAll || touched.has(k))),
    [local, showAll, touched]
  );
  const fields = useMemo(() => ({ ...publishErrors.fields, ...saveErrors.fields, ...shownLocal }), [publishErrors.fields, saveErrors.fields, shownLocal]);
  const general = [...saveErrors.general, ...publishErrors.general];
  const err = useCallback((path: string, nested = false) => errorAt(fields, path, nested), [fields]);
  const counts = countByStep(fields);
  const errorTotal = Object.keys(fields).length;

  const openStepsFor = useCallback((paths: string[]) => {
    setOpen((o) => {
      const next = new Set(o);
      for (const k of paths) {
        const s = stepForField(k);
        if (s) next.add(s);
      }
      return next;
    });
    setFocusNonce((n) => n + 1);
  }, []);

  // After a failed publish the steps open, then focus goes to the first field that needs fixing.
  useEffect(() => {
    if (focusNonce === 0) return;
    const id = requestAnimationFrame(() => {
      formEl.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
    });
    return () => cancelAnimationFrame(id);
  }, [focusNonce]);

  // The checkout view is the server's render of the settled form; it is never assembled here.
  const settled = useDebounced(form, 400);
  const settledLocal = useMemo(() => Object.keys(localErrors(settled)).length > 0, [settled]);
  const previewInput = useMemo(() => (settledLocal ? null : formToInput(settled)), [settled, settledLocal]);
  const preview = useLinkPreview(previewInput, current?.id ?? null);
  const inSync = settled === form && !preview.isPlaceholderData && !preview.isFetching;
  const previewState: PreviewState = hasLocal
    ? "paused"
    : preview.error
      ? isApiError(preview.error) && preview.error.status === 422
        ? "paused"
        : "failed"
      : !preview.data
        ? "loading"
        : inSync
          ? "ready"
          : "updating";
  const dropped = useMemo(() => droppedByKey(inSync ? preview.data?.dropped_methods : undefined), [inSync, preview.data]);
  const pricing = useMemo(() => savedPricing(current, form), [current, form]);

  async function publish() {
    setShowAll(true);
    if (hasLocal) {
      openStepsFor(Object.keys(local));
      return;
    }
    const saved = await enqueueSave();
    if (!saved) {
      openStepsFor(Object.keys(saveErrRef.current.fields));
      return;
    }
    try {
      const l = await publishM.mutateAsync(saved.id);
      setCurrent(l);
      currentRef.current = l;
      setPublishErrors(NO_ERRORS);
      toast.success(status === "paused" ? B.resumed : B.published);
    } catch (e) {
      const m = mapApiErrors(e);
      setPublishErrors(m);
      openStepsFor(Object.keys(m.fields));
    }
  }

  async function transition(kind: "pause" | "archive") {
    if (!current) return;
    try {
      const l = await (kind === "pause" ? pauseM : archiveM).mutateAsync(current.id);
      setCurrent(l);
      currentRef.current = l;
    } catch (e) {
      setPublishErrors(mapApiErrors(e));
    }
    setConfirm(null);
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

  const stepProps = { form, edit, err, locked };
  const stepBody: Record<StepId, React.ReactNode> = {
    item: <ItemStep {...stepProps} options={options.data} />,
    customer: <CustomerStep {...stepProps} />,
    payment: <PaymentStep {...stepProps} options={options.data} pricing={pricing} dropped={dropped} />,
    after: <AfterStep {...stepProps} webhooks={(webhooks.data ?? []).map((w) => ({ id: w.id, url: w.url }))} />,
    settlement: <SettlementStep {...stepProps} />,
    lifecycle: <LifecycleStep {...stepProps} />,
  };

  const saveLabel = readOnly
    ? ""
    : hasLocal || saveState === "error"
      ? B.notSaved
      : saveState === "saving"
        ? B.saving
        : locked && dirty
          ? B.unsaved
          : current
            ? dirty
              ? B.saving
              : B.saved
            : B.notStarted;
  const showSummary = errorTotal > 0 && (showAll || saveState === "error" || publishErrors !== NO_ERRORS);
  const title = form.title || (current ? LINKS_COPY.untitled : B.newTitle);

  return (
    <div className="space-y-5">
      <div className="space-y-1">
      <PageHeader
        breadcrumbs={[{ label: LINKS_COPY.listTitle, href: basePath }, { label: title }]}
        title={
          <span className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
            <span className="min-w-0 truncate">{title}</span>
            {status ? <StatusBadge status={status} /> : null}
          </span>
        }
        className="mb-0"
      >
        <div className="flex flex-wrap items-center gap-2">
          {status === "draft" ? (
            <Button variant="ghost" onClick={() => setConfirm("delete")}>
              <Trash2 />
              {B.deleteDraft}
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
              {B.duplicate}
            </Button>
          ) : null}
          {locked ? (
            <Button variant="outline" disabled={busy} onClick={() => setConfirm("archive")}>
              <Archive />
              {B.archive}
            </Button>
          ) : null}
          {locked && dirty && !readOnly ? (
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
        </div>
      </PageHeader>
      <p role="status" aria-live="polite" className={cn("min-h-4 text-label", saveLabel === B.notSaved ? "text-bad" : "text-ink-soft")}>
        {saveLabel}
      </p>
      </div>

      {current?.url ? (
        <div className="flex flex-col gap-3 rounded-md border border-line bg-surface p-4 sm:flex-row sm:items-center sm:p-5">
          <ShortLink url={current.url} title={current.title} qr={false} className="min-w-0 flex-1" />
          <div className="flex items-center gap-4">
            <QrButton url={current.url} title={current.title} labelled />
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

      {general.length > 0 || showSummary ? (
        <Notice tone="bad">
          {general.map((g, i) => (
            <p key={i}>{g.message}</p>
          ))}
          {showSummary ? <p>{B.fixErrors(errorTotal)}</p> : null}
        </Notice>
      ) : null}

      <div className="lg:hidden">
        <Segmented
          label={B.view}
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
        <fieldset ref={formEl} disabled={readOnly} className={cn("min-w-0 space-y-3", view === "preview" && "max-lg:hidden")}>
          <legend className="sr-only">{B.settings}</legend>
          {locked ? <LockedNote onDuplicate={() => void duplicate()} /> : null}
          <div className="flex justify-end">
            <Button type="button" variant="ghost" size="sm" className="tap" onClick={() => setOpen(allOpen ? new Set() : new Set(STEP_ORDER))}>
              {allOpen ? B.collapseAll : B.expandAll}
            </Button>
          </div>
          {STEP_ORDER.map((s, i) => (
            <Step key={s} index={i + 1} title={LINKS_COPY.steps[s].title} summary={summary(s, form)} open={open.has(s)} onToggle={() => toggle(s)} errors={counts[s] ?? 0}>
              {stepBody[s]}
            </Step>
          ))}
        </fieldset>
        <div className={cn("min-w-0", view === "edit" && "max-lg:hidden")}>
          <div className="lg:sticky lg:top-20">
            <CheckoutPreview model={preview.data?.model ?? null} state={previewState} />
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
              {B.cancel}
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
