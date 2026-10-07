"use client";

import { useId, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

/**
 * Segmented choice built on native radios, so arrow keys, focus and form semantics come free.
 * Used for every one-of-few choice in the builder (amount mode, field mode, fee bearer).
 */
export function Segmented<T extends string>({
  label,
  value,
  options,
  onChange,
  disabled,
  size = "default",
  className,
  invalid,
}: {
  label: string;
  value: T;
  options: { value: T; label: string }[];
  onChange: (v: T) => void;
  disabled?: boolean;
  size?: "default" | "sm";
  className?: string;
  invalid?: boolean;
}) {
  const name = useId();
  return (
    <div
      role="radiogroup"
      aria-label={label}
      aria-invalid={invalid || undefined}
      className={cn(
        "inline-flex max-w-full items-center rounded-sm border border-line-strong bg-surface p-0.5",
        invalid && "border-bad",
        disabled && "bg-surface-sunken",
        className
      )}
    >
      {options.map((o) => {
        const checked = o.value === value;
        return (
          <label
            key={o.value}
            className={cn(
              "relative inline-flex flex-1 cursor-pointer items-center justify-center rounded-xs px-2.5 text-label font-medium whitespace-nowrap transition-colors duration-120",
              "has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-offset-1 has-[:focus-visible]:outline-tide",
              size === "sm" ? "h-7" : "h-8",
              "pointer-coarse:h-11 pointer-coarse:px-3.5",
              checked ? "bg-ink text-ink-inverse" : "text-ink-soft hover:text-ink",
              disabled && "cursor-not-allowed",
              disabled && checked && "bg-ink-faint text-ink-inverse",
              disabled && !checked && "hover:text-ink-soft"
            )}
          >
            <input
              type="radio"
              name={name}
              value={o.value}
              checked={checked}
              disabled={disabled}
              onChange={() => onChange(o.value)}
              className="absolute inset-0 cursor-[inherit] opacity-0"
            />
            {o.label}
          </label>
        );
      })}
    </div>
  );
}

/** Title on the left, switch on the right, optional one-line consequence under the title. */
export function ToggleRow({
  label,
  hint,
  checked,
  onChange,
  disabled,
  error,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  error?: string;
}) {
  const id = useId();
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <Label htmlFor={id} className="text-body-sm font-medium text-ink">
            {label}
          </Label>
          {hint ? <p className="text-caption text-ink-soft">{hint}</p> : null}
        </div>
        <Switch id={id} checked={checked} onCheckedChange={(v) => onChange(Boolean(v))} disabled={disabled} aria-invalid={Boolean(error) || undefined} />
      </div>
      {error ? (
        <p role="alert" className="text-label text-bad">
          {error}
        </p>
      ) : null}
    </div>
  );
}

/** A labelled control with the API's message under it. */
export function Field({
  label,
  htmlFor,
  error,
  hint,
  children,
  className,
}: {
  label: string;
  htmlFor?: string;
  error?: string;
  hint?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("min-w-0 space-y-1.5", className)}>
      {htmlFor ? (
        <Label htmlFor={htmlFor}>{label}</Label>
      ) : (
        <p className="text-label font-medium text-ink">{label}</p>
      )}
      {children}
      {hint && !error ? <p className="text-caption text-ink-soft">{hint}</p> : null}
      {error ? (
        <p role="alert" className="text-label text-bad">
          {error}
        </p>
      ) : null}
    </div>
  );
}

/** Native select in the Input's chrome; keyboard and mobile pickers behave as the platform expects. */
export function NativeSelect({
  className,
  children,
  ...props
}: React.ComponentProps<"select">) {
  return (
    <div className={cn("relative", className)}>
      <select
        {...props}
        className={cn(
          "h-9 pointer-coarse:h-11 w-full appearance-none rounded-sm border border-line-strong bg-surface pr-8 pl-3 text-body text-ink transition-[border-color] duration-120 outline-none hover:border-ink-faint",
          "focus-visible:border-tide focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide",
          "disabled:cursor-not-allowed disabled:bg-surface-sunken disabled:text-ink-faint aria-invalid:border-bad"
        )}
      >
        {children}
      </select>
      <ChevronDown aria-hidden className="pointer-events-none absolute top-1/2 right-2.5 size-4 -translate-y-1/2 text-ink-faint" />
    </div>
  );
}

/** One collapsible builder step: number, title, a summary of its values, and its error count. */
export function Step({
  index,
  title,
  summary,
  open,
  onToggle,
  errors,
  children,
}: {
  index: number;
  title: string;
  summary: string;
  open: boolean;
  onToggle: () => void;
  errors: number;
  children: ReactNode;
}) {
  const id = useId();
  return (
    <section className="rounded-md border border-line bg-surface">
      <h2>
        <button
          type="button"
          aria-expanded={open}
          aria-controls={id}
          onClick={onToggle}
          className="flex w-full items-center gap-3 rounded-md px-4 py-3.5 text-left outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide sm:px-5"
        >
          <span
            aria-hidden
            className={cn(
              "num flex size-6 shrink-0 items-center justify-center rounded-xs border text-label font-medium",
              errors > 0 ? "border-bad bg-bad-tint text-bad" : "border-line-strong text-ink-soft"
            )}
          >
            {index}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block text-body font-semibold text-ink">{title}</span>
            {!open ? <span className="block truncate text-body-sm font-normal text-ink-soft">{summary}</span> : null}
          </span>
          {errors > 0 ? (
            <span className="num shrink-0 rounded-xs bg-bad-tint px-1.5 text-label font-medium text-bad">
              {errors}
              <span className="sr-only"> {errors === 1 ? "error" : "errors"}</span>
            </span>
          ) : null}
          <ChevronDown aria-hidden className={cn("size-4 shrink-0 text-ink-faint transition-transform duration-120", open && "rotate-180")} />
        </button>
      </h2>
      <div id={id} hidden={!open} className="space-y-4 border-t border-line px-4 pt-4 pb-5 sm:px-5">
        {children}
      </div>
    </section>
  );
}

/** A quiet group heading inside a step. */
export function GroupLabel({ children }: { children: ReactNode }) {
  return <p className="text-label font-medium text-ink-soft">{children}</p>;
}
