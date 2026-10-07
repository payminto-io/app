/**
 * Maps the links API error body `{error, code, field?, errors?[]}` (docs/API_SPECIFICATION.md 4.44)
 * onto form fields and steps. The message shown is the server's; this module only routes it.
 */
import { isApiError } from "@/lib/api/errors";

export interface ApiFieldError {
  code: string;
  field?: string;
  message: string;
}

export interface MappedErrors {
  /** Exact API field path to message, e.g. `line_items[0].name`. */
  fields: Record<string, string>;
  /** Errors with no field, or with a field no input renders. */
  general: ApiFieldError[];
}

export const NO_ERRORS: MappedErrors = { fields: {}, general: [] };

export type StepId = "item" | "customer" | "payment" | "after" | "settlement" | "lifecycle";

export const STEP_ORDER: StepId[] = ["item", "customer", "payment", "after", "settlement", "lifecycle"];

/** Top-level field (before any `.` or `[`) to the step that renders it. */
const FIELD_STEP: Record<string, StepId> = {
  title: "item",
  description: "item",
  amount_mode: "item",
  amount: "item",
  amount_min: "item",
  amount_max: "item",
  currency: "item",
  line_items: "item",
  reference_id: "item",
  metadata: "item",
  category: "item",
  customer_field_policy: "customer",
  billing_required: "customer",
  shipping_required: "customer",
  questions: "customer",
  multi_use: "customer",
  use_limit: "customer",
  methods: "payment",
  capture_mode: "payment",
  three_ds_policy: "payment",
  chain_tolerance_bps: "payment",
  quote_expiry_seconds: "payment",
  fee_bearer: "payment",
  success_mode: "after",
  success_url: "after",
  success_message: "after",
  receipt_email: "after",
  receipt_note: "after",
  webhook_id: "after",
  failure_retry: "after",
  failure_message: "after",
  settlement_override: "settlement",
  hold_in_asset: "settlement",
  settlement_timing: "settlement",
  expires_at: "lifecycle",
  expires_after_payments: "lifecycle",
  logo_url: "lifecycle",
  accent_color: "lifecycle",
  language: "lifecycle",
};

export function rootField(path: string): string {
  return path.split(/[.[]/, 1)[0];
}

export function stepForField(path: string | undefined): StepId | null {
  if (!path) return null;
  return FIELD_STEP[rootField(path)] ?? null;
}

function asItems(body: unknown): ApiFieldError[] {
  if (typeof body !== "object" || body === null) return [];
  const b = body as { error?: unknown; code?: unknown; field?: unknown; errors?: unknown };
  if (Array.isArray(b.errors) && b.errors.length > 0) {
    return b.errors
      .filter((e): e is Record<string, unknown> => typeof e === "object" && e !== null)
      .map((e) => ({
        code: typeof e.code === "string" ? e.code : "",
        field: typeof e.field === "string" && e.field ? e.field : undefined,
        message: typeof e.message === "string" ? e.message : "",
      }));
  }
  if (typeof b.error === "string") {
    return [
      {
        code: typeof b.code === "string" ? b.code : "",
        field: typeof b.field === "string" && b.field ? b.field : undefined,
        message: b.error,
      },
    ];
  }
  return [];
}

/**
 * Route a thrown save or publish error. A field the form renders gets its message inline;
 * the first message wins when two rules hit one field. Everything else is general.
 */
export function mapApiErrors(err: unknown): MappedErrors {
  if (!isApiError(err)) {
    return { fields: {}, general: [{ code: "", message: err instanceof Error ? err.message : String(err) }] };
  }
  const items = asItems(err.body);
  if (items.length === 0) return { fields: {}, general: [{ code: "", message: err.message }] };
  const fields: Record<string, string> = {};
  const general: ApiFieldError[] = [];
  for (const it of items) {
    if (it.field && stepForField(it.field)) {
      fields[it.field] ??= sentence(it.message);
    } else {
      general.push({ ...it, message: sentence(it.message) });
    }
  }
  return { fields, general };
}

/** The server writes lower-case fragments ("must be 1-100000"); show them as a sentence. */
export function sentence(m: string): string {
  const t = m.trim();
  return t ? t[0].toUpperCase() + t.slice(1) : t;
}

/** Message for an exact path, or for any nested path under it (`methods[0]` covers `methods[0].chain`). */
export function errorAt(fields: Record<string, string>, path: string, nested = false): string | undefined {
  if (fields[path]) return fields[path];
  if (!nested) return undefined;
  const hit = Object.keys(fields).find((k) => k.startsWith(path + ".") || k.startsWith(path + "["));
  return hit ? fields[hit] : undefined;
}

export function countByStep(fields: Record<string, string>): Partial<Record<StepId, number>> {
  const out: Partial<Record<StepId, number>> = {};
  for (const k of Object.keys(fields)) {
    const s = stepForField(k);
    if (s) out[s] = (out[s] ?? 0) + 1;
  }
  return out;
}

export function mergeErrors(local: Record<string, string>, api: MappedErrors): MappedErrors {
  return { fields: { ...api.fields, ...local }, general: api.general };
}
