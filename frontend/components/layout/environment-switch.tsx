"use client";

import { cn } from "@/lib/utils";
import { LIVE_ENABLED, useEnvironment, type Environment } from "@/lib/environment/store";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

const OPTIONS: { value: Environment; label: string }[] = [
  { value: "test", label: "Test" },
  { value: "live", label: "Live" },
];

/**
 * Segmented Test / Live control. Test is amber, live is ink; the frame also
 * shows an amber rule along the top edge while in test (see AppShell).
 */
export function EnvironmentSwitch({ className }: { className?: string }) {
  const environment = useEnvironment((s) => s.environment);
  const setEnvironment = useEnvironment((s) => s.setEnvironment);

  return (
    <div
      role="radiogroup"
      aria-label="Environment"
      className={cn(
        "inline-flex h-8 items-center rounded-sm border border-line-strong bg-surface p-0.5",
        className
      )}
    >
      {OPTIONS.map((opt) => {
        const active = environment === opt.value;
        const disabled = opt.value === "live" && !LIVE_ENABLED;
        const button = (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-disabled={disabled || undefined}
            onClick={() => !disabled && setEnvironment(opt.value)}
            className={cn(
              "tap inline-flex h-7 min-w-14 items-center justify-center gap-1.5 rounded-xs px-2.5 text-label font-medium transition-colors duration-120",
              active && opt.value === "test" && "bg-env-test text-env-test-ink",
              active && opt.value === "live" && "bg-env-live text-ink-inverse",
              !active && "text-ink-soft hover:text-ink",
              disabled && "cursor-not-allowed opacity-50 hover:text-ink-soft"
            )}
          >
            {active ? (
              <span
                aria-hidden
                className={cn(
                  "size-1.5 rounded-full",
                  opt.value === "test" ? "bg-env-test-ink" : "bg-ink-inverse"
                )}
              />
            ) : null}
            {opt.label}
          </button>
        );
        if (!disabled) return button;
        return (
          <Tooltip key={opt.value}>
            <TooltipTrigger render={<span className="inline-flex" />}>{button}</TooltipTrigger>
            <TooltipContent>Live is not enabled on this deployment.</TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}
