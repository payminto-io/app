"use client";

import { useEffect, useState, type ReactNode } from "react";
import { Check, Copy, ExternalLink } from "lucide-react";
import { formatAmount } from "@/lib/money";
import { formatCountdown, secondsUntil, type Money } from "@/lib/model";

export function cx(...parts: Array<string | false | null | undefined>): string {
  return parts.filter(Boolean).join(" ");
}

/** Amount then code, tabular, code at 0.75em floored at 11px. DESIGN.md section 3. */
export function Amount({ money, size = "body", className }: { money: Money; size?: "display" | "h1" | "h2" | "body" | "body-sm"; className?: string }) {
  const sizes = {
    display: "text-display font-semibold",
    h1: "text-h1 font-semibold",
    h2: "text-h2 font-semibold",
    body: "text-body font-medium",
    "body-sm": "text-body-sm",
  } as const;
  return (
    <span className={cx("num inline-flex items-baseline whitespace-nowrap", sizes[size], className)}>
      <span>{formatAmount(money.amount, money.code)}</span>
      <span className="ml-[0.35em] font-normal text-ink-soft" style={{ fontSize: "max(11px, 0.75em)" }}>{money.code}</span>
    </span>
  );
}

export function useCopy(): [string, (value: string, key: string) => void] {
  const [copied, setCopied] = useState("");
  useEffect(() => {
    if (!copied) return;
    const t = window.setTimeout(() => setCopied(""), 1600);
    return () => window.clearTimeout(t);
  }, [copied]);
  const copy = (value: string, key: string) => {
    navigator.clipboard.writeText(value).then(() => setCopied(key)).catch(() => setCopied(""));
  };
  return [copied, copy];
}

export function CopyField({ label, value, display, copied, onCopy, mono = true }: {
  label: string;
  value: string;
  display?: ReactNode;
  copied: boolean;
  onCopy: () => void;
  mono?: boolean;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-label font-medium text-ink-soft">{label}</span>
      <button type="button" className="copyfield" onClick={onCopy} aria-label={`Copy ${label.toLowerCase()}`}>
        <span className={cx("min-w-0 flex-1 break-all text-body-sm text-ink", mono && "mono")}>{display ?? value}</span>
        <span className={cx("inline-flex items-center gap-1.5 text-label font-medium", copied ? "text-ok" : "text-tide")} aria-live="polite">
          {copied ? <Check size={14} strokeWidth={1.75} /> : <Copy size={14} strokeWidth={1.75} />}
          {copied ? "Copied" : "Copy"}
        </span>
      </button>
    </div>
  );
}

/** Mono timer next to the thing it governs, never a ring. references.md, pay states. */
export function Countdown({ until, label, onExpire }: { until?: string; label: string; onExpire?: () => void }) {
  const [now, setNow] = useState<number>();
  useEffect(() => {
    if (!until) return;
    const first = window.setTimeout(() => setNow(Date.now()), 0);
    const t = window.setInterval(() => setNow(Date.now()), 1000);
    return () => {
      window.clearTimeout(first);
      window.clearInterval(t);
    };
  }, [until]);
  const left = now === undefined ? undefined : secondsUntil(until, now);
  useEffect(() => {
    if (left === 0) onExpire?.();
  }, [left, onExpire]);
  if (left === undefined) return null;
  const urgent = left < 120;
  return (
    <span className={cx("inline-flex items-center gap-2 text-label", urgent ? "text-wait" : "text-ink-soft")}>
      <span>{label}</span>
      <span className="mono num font-medium">{formatCountdown(left)}</span>
    </span>
  );
}

const RAIL = ["Received", "Final", "Settled"] as const;

/** The finality rail. `step` is how many segments are filled. DESIGN.md section 1. */
export function Rail({ step, failed = false, timestamps }: { step: 0 | 1 | 2 | 3; failed?: boolean; timestamps?: [string?, string?, string?] }) {
  const label = step === 0 ? "Not received" : RAIL[step - 1];
  return (
    <div className="flex w-full gap-1.5" role="img" aria-label={`${label}${failed ? ", failed" : ""}`}>
      {RAIL.map((name, i) => (
        <div key={name} className="flex min-w-0 flex-1 flex-col gap-2">
          <div className="rail-seg" data-filled={i < step} data-failed={failed && i === step} />
          <div className="flex flex-col">
            <span className={cx("text-label font-medium", i < step ? "text-ink" : "text-ink-faint")}>{name}</span>
            {timestamps?.[i] ? <span className="num text-caption text-ink-soft">{timestamps[i]}</span> : null}
          </div>
        </div>
      ))}
    </div>
  );
}

export function Pulse() {
  return (
    <span className="pulse" aria-hidden>
      <i /><i /><i />
    </span>
  );
}

export function ExplorerLink({ href, children = "View on explorer" }: { href: string; children?: ReactNode }) {
  return (
    <a href={href} target="_blank" rel="noreferrer noopener" className="tap inline-flex items-center gap-1.5 text-body-sm font-medium text-tide hover:text-tide-strong">
      {children}
      <ExternalLink size={14} strokeWidth={1.75} />
    </a>
  );
}

export function Row({ label, children, mono }: { label: string; children: ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2.5">
      <span className="shrink-0 text-body-sm text-ink-soft">{label}</span>
      <span className={cx("min-w-0 text-right text-body-sm text-ink", mono && "mono num break-all")}>{children}</span>
    </div>
  );
}

export function formatWhen(iso: string): string {
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return iso;
  return d.toLocaleString("en-GB", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" });
}

export function truncateMiddle(value: string, keep = 6): string {
  return value.length > keep * 2 + 3 ? `${value.slice(0, keep)}…${value.slice(-keep)}` : value;
}

/** The gateway's own mark, owned by the brand branch at public/brand/mark.svg; a background image so a missing file renders nothing. */
export function PoweredBy() {
  return (
    <span className="inline-flex items-center gap-1.5 text-caption text-ink-faint">
      <span aria-hidden className="inline-block h-4 w-4 bg-contain bg-center bg-no-repeat" style={{ backgroundImage: "url(/brand/mark.svg)" }} />
      Powered by Payminto
    </span>
  );
}
