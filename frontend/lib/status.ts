/**
 * Status vocabulary. One map for every backend status string, so a status is
 * always a label, a tone and a glyph. Rules: docs/design/DESIGN.md section 7.
 */
export type StatusTone = "ok" | "wait" | "bad" | "note" | "mute";
export type StatusGlyph = "check" | "clock" | "x" | "circle" | "arrow-up" | "undo" | "minus";

export interface StatusMeta {
  label: string;
  tone: StatusTone;
  glyph: StatusGlyph;
}

export const STATUS_MAP: Record<string, StatusMeta> = {
  // payment intents
  created: { label: "Created", tone: "note", glyph: "circle" },
  open: { label: "Awaiting payment", tone: "wait", glyph: "clock" },
  confirming: { label: "Confirming", tone: "wait", glyph: "clock" },
  partially_filled: { label: "Under paid", tone: "wait", glyph: "clock" },
  filled: { label: "Paid", tone: "ok", glyph: "check" },
  confirmed: { label: "Paid", tone: "ok", glyph: "check" },
  closed: { label: "Paid", tone: "ok", glyph: "check" },
  over_filled: { label: "Over paid", tone: "note", glyph: "arrow-up" },
  refunded: { label: "Refunded", tone: "mute", glyph: "undo" },
  cancelled: { label: "Cancelled", tone: "bad", glyph: "x" },
  expired: { label: "Expired", tone: "mute", glyph: "minus" },
  // attempts
  initiated: { label: "Initiated", tone: "note", glyph: "circle" },
  authorized: { label: "Authorised", tone: "wait", glyph: "clock" },
  captured: { label: "Captured", tone: "ok", glyph: "check" },
  declined: { label: "Declined", tone: "bad", glyph: "x" },
  errored: { label: "Error", tone: "bad", glyph: "x" },
  // settlement and withdrawals
  pending_approval: { label: "Awaiting approval", tone: "wait", glyph: "clock" },
  pending_otp: { label: "Awaiting code", tone: "wait", glyph: "clock" },
  approved: { label: "Approved", tone: "note", glyph: "circle" },
  sent: { label: "Sent", tone: "wait", glyph: "clock" },
  processed: { label: "Settled", tone: "ok", glyph: "check" },
  // chain finality
  seen: { label: "Seen", tone: "note", glyph: "circle" },
  final: { label: "Final", tone: "ok", glyph: "check" },
  reorged: { label: "Reorged", tone: "bad", glyph: "x" },
  // address pool
  available: { label: "Available", tone: "note", glyph: "circle" },
  used: { label: "Used", tone: "mute", glyph: "check" },
  locked: { label: "Locked", tone: "wait", glyph: "clock" },
  // system
  running: { label: "Running", tone: "ok", glyph: "check" },
  stopped: { label: "Stopped", tone: "mute", glyph: "minus" },
  healthy: { label: "Healthy", tone: "ok", glyph: "check" },
  ok: { label: "Healthy", tone: "ok", glyph: "check" },
  up: { label: "Up", tone: "ok", glyph: "check" },
  degraded: { label: "Degraded", tone: "wait", glyph: "clock" },
  down: { label: "Down", tone: "bad", glyph: "x" },
  // generic
  active: { label: "Active", tone: "ok", glyph: "check" },
  inactive: { label: "Inactive", tone: "mute", glyph: "minus" },
  pending: { label: "Pending", tone: "wait", glyph: "clock" },
  processing: { label: "Processing", tone: "note", glyph: "circle" },
  completed: { label: "Completed", tone: "ok", glyph: "check" },
  failed: { label: "Failed", tone: "bad", glyph: "x" },
  delivered: { label: "Delivered", tone: "ok", glyph: "check" },
  retrying: { label: "Retrying", tone: "wait", glyph: "clock" },
  success: { label: "Success", tone: "ok", glyph: "check" },
  failure: { label: "Failure", tone: "bad", glyph: "x" },
  resolved: { label: "Resolved", tone: "ok", glyph: "check" },
  dismissed: { label: "Dismissed", tone: "mute", glyph: "minus" },
};

export function statusMeta(status: string): StatusMeta {
  const key = status.toLowerCase();
  return (
    STATUS_MAP[key] ?? {
      label: status.replace(/_/g, " ").replace(/^\w/, (c) => c.toUpperCase()),
      tone: "mute",
      glyph: "circle",
    }
  );
}

/**
 * Rail position from a payment intent state alone. A paid intent is Received
 * and Final; Settled is never filled here because the payment API carries no
 * settlement record yet. Cancelled and failed paint the first open segment
 * bad; expired stays empty because nothing was lost.
 */
export function paymentRail(state: string): { step: 0 | 1 | 2 | 3; failed: boolean } {
  switch (state.toLowerCase()) {
    case "confirming":
    case "partially_filled":
      return { step: 1, failed: false };
    case "filled":
    case "confirmed":
    case "closed":
    case "over_filled":
      return { step: 2, failed: false };
    case "cancelled":
    case "failed":
      return { step: 0, failed: true };
    default:
      return { step: 0, failed: false };
  }
}
