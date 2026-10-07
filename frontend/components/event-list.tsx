import { cn } from "@/lib/utils"

/** Event names as mono chips; `max` collapses the rest into "+n". */
export function EventList({ events, max, className }: { events: string[]; max?: number; className?: string }) {
  const shown = max ? events.slice(0, max) : events
  const rest = events.length - shown.length
  return (
    <span className={cn("flex flex-wrap items-center gap-1", className)}>
      {shown.map((ev) => (
        <span
          key={ev}
          className="inline-flex h-5 items-center rounded-xs bg-surface-sunken px-1.5 font-mono text-[11px] text-ink-soft"
        >
          {ev}
        </span>
      ))}
      {rest > 0 ? <span className="num text-caption text-ink-soft">+{rest}</span> : null}
    </span>
  )
}
