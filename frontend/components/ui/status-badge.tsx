import { ArrowUp, Check, Circle, Clock, Minus, Undo2, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { statusMeta, type StatusGlyph, type StatusTone } from "@/lib/status";

const TONE: Record<StatusTone, string> = {
  ok: "bg-ok-tint text-ok",
  wait: "bg-wait-tint text-wait",
  bad: "bg-bad-tint text-bad",
  note: "bg-note-tint text-note",
  mute: "bg-mute-tint text-mute",
};

const GLYPH: Record<StatusGlyph, typeof Check> = {
  check: Check,
  clock: Clock,
  x: X,
  circle: Circle,
  "arrow-up": ArrowUp,
  undo: Undo2,
  minus: Minus,
};

/** Maps a backend status string to label + tone + glyph. Colour is never the only signal. */
export function StatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const meta = statusMeta(status);
  const Glyph = GLYPH[meta.glyph];
  return (
    <span
      className={cn(
        "inline-flex h-5 items-center gap-1 rounded-xs px-1.5 text-label font-medium whitespace-nowrap",
        TONE[meta.tone],
        className
      )}
    >
      <Glyph className="size-3" strokeWidth={2.25} aria-hidden />
      {meta.label}
    </span>
  );
}
