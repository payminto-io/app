import { cn } from "@/lib/utils";

export type RailStep = 0 | 1 | 2 | 3;

const LABELS = ["Received", "Final", "Settled"] as const;

/**
 * The finality rail: Received -> Final -> Settled. Fiat and chain use the
 * same three words. `step` is how many segments are filled (0 to 3).
 * `failed` paints the first unfilled segment bad. docs/design/DESIGN.md section 1.
 */
export function Rail({
  step,
  failed = false,
  size = "sm",
  timestamps,
  className,
}: {
  step: RailStep;
  failed?: boolean;
  size?: "sm" | "lg";
  /** Optional ISO or preformatted times under each segment (lg only). */
  timestamps?: [string?, string?, string?];
  className?: string;
}) {
  return (
    <div
      className={cn("flex", size === "sm" ? "w-9 gap-0.5" : "w-full gap-1.5", className)}
      role="img"
      aria-label={`${LABELS[Math.max(0, step - 1)] ?? "Not received"}${failed ? ", failed" : ""}`}
    >
      {LABELS.map((label, i) => {
        const filled = i < step;
        const isFailed = failed && i === step;
        return (
          <div key={label} className={cn("min-w-0 flex-1", size === "lg" && "flex flex-col gap-1.5")}>
            <div
              className={cn(
                "h-[3px] rounded-full transition-colors duration-[400ms] ease-in-out",
                filled ? "bg-ok" : isFailed ? "bg-bad" : "bg-line-strong"
              )}
            />
            {size === "lg" ? (
              <div className="flex flex-col">
                <span className={cn("text-label font-medium", filled ? "text-ink" : "text-ink-faint")}>
                  {label}
                </span>
                {timestamps?.[i] ? (
                  <span className="num text-caption text-ink-soft">{timestamps[i]}</span>
                ) : null}
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
