import { cn } from "@/lib/utils";

/**
 * The rail mark: a rounded square with three bars. Edit the SVG here and in
 * app/icon.svg together. docs/design/DESIGN.md section 1.
 */
export function LogoMark({ className, size = 28 }: { className?: string; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 28 28"
      fill="none"
      aria-hidden
      className={cn("shrink-0", className)}
    >
      <rect width="28" height="28" rx="7" className="fill-ink" />
      <rect x="7" y="8.5" width="14" height="2.5" rx="1.25" className="fill-ink-inverse" />
      <rect x="7" y="12.75" width="14" height="2.5" rx="1.25" className="fill-ink-inverse" />
      <rect x="7" y="17" width="8" height="2.5" rx="1.25" className="fill-tide" />
    </svg>
  );
}

export function Logo({ className, wordmark = true }: { className?: string; wordmark?: boolean }) {
  return (
    <span className={cn("inline-flex items-center gap-2.5", className)}>
      <LogoMark />
      {wordmark ? (
        <span className="text-[15px] font-semibold tracking-[-0.01em] text-ink">Payminto</span>
      ) : null}
    </span>
  );
}
