/**
 * Payment links (v2) domain API. Contract: docs/API_SPECIFICATION.md 4.44.
 * Money is a decimal string; JSON is snake_case and mirrors the backend's `links.Input`.
 */
import { apiFetch } from "./client";
import { API_V2_BASE_URL } from "@/lib/constants";

export type LinkStatus = "draft" | "active" | "paused" | "archived";
export type AmountMode = "fixed" | "customer" | "line_items";
export type FieldMode = "required" | "optional" | "hidden";
export type PayMethod = "card" | "upi" | "bank" | "crypto";
export type QuestionType = "text" | "select" | "checkbox";
export type FeeBearer = "merchant" | "customer";

export interface MethodSpec {
  method: PayMethod;
  chain?: string;
  asset?: string;
}

export interface FieldRule {
  mode: FieldMode;
  prefill?: string;
}

export interface CustomerFieldPolicy {
  name: FieldRule;
  email: FieldRule;
  phone: FieldRule;
}

export interface LineItemInput {
  name: string;
  quantity: number;
  unit_price: string;
  tax_rate: string;
}

export interface QuestionInput {
  key: string;
  label: string;
  type: QuestionType;
  options?: string[];
  required: boolean;
  per_order: boolean;
}

export interface SettlementOverride {
  kind: "fiat" | "crypto";
  destination_id: string;
  chain?: string;
  asset?: string;
}

/** The create body and the editable half of a link. */
export interface LinkInput {
  title: string;
  description: string;
  amount_mode: AmountMode;
  amount: string | null;
  amount_min: string | null;
  amount_max: string | null;
  currency: string;
  reference_id: string;
  metadata: Record<string, string>;
  category: string;
  customer_field_policy: CustomerFieldPolicy;
  billing_required: boolean;
  shipping_required: boolean;
  multi_use: boolean;
  use_limit: number | null;
  methods: MethodSpec[];
  capture_mode: "automatic" | "manual";
  three_ds_policy: "inherit" | "force";
  chain_tolerance_bps: number;
  quote_expiry_seconds: number;
  fee_bearer: FeeBearer;
  success_mode: "message" | "redirect";
  success_url: string;
  success_message: string;
  receipt_email: boolean;
  receipt_note: string;
  webhook_id: number | null;
  failure_retry: boolean;
  failure_message: string;
  settlement_override: SettlementOverride | null;
  hold_in_asset: boolean;
  settlement_timing: "cycle" | "immediate";
  expires_at: string | null;
  expires_after_payments: number | null;
  logo_url: string;
  accent_color: string;
  language: string;
  line_items: LineItemInput[];
  questions: QuestionInput[];
}

export interface PaymentLink extends LinkInput {
  id: string;
  status: LinkStatus;
  environment: "live" | "test";
  short_code: string | null;
  /** `<CHECKOUT_BASE_URL>/l/<short_code>`, the QR payload; null on a draft. */
  url: string | null;
  /** The fixed amount or the server's line-item sum; null for customer-entered. */
  total: string | null;
  uses_count: number;
  revision: number;
  published_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface LinkList {
  links: PaymentLink[];
  total: number;
}

export interface LinkListParams {
  status?: LinkStatus;
  limit?: number;
  offset?: number;
}

/** Keys a published link may not change (409 `link_published_immutable`). */
export const IMMUTABLE_WHEN_PUBLISHED = [
  "amount_mode",
  "amount",
  "amount_min",
  "amount_max",
  "currency",
  "methods",
  "line_items",
] as const satisfies readonly (keyof LinkInput)[];

/** The public render model checkout reads (`GET /api/v2/public/links/:short_code`). */
export interface RenderLineItem {
  name: string;
  quantity: number;
  unit_price: string;
  tax_rate: string;
  /** Always set by the public endpoint; the builder preview leaves them null until the server computes them. */
  subtotal: string | null;
  tax: string | null;
  total: string | null;
}

export interface RenderField {
  mode: FieldMode;
  prefill: string | null;
}

export interface RenderMethod {
  method: PayMethod;
  chain: string | null;
  asset: string | null;
  fee: string | null;
  tax: string | null;
  customer_total: string | null;
}

export type UnavailableReason = "paused" | "expired" | "use_limit_reached" | "no_methods_available";

export interface RenderModel {
  short_code: string;
  url: string;
  available: boolean;
  unavailable_reason: UnavailableReason | null;
  merchant_name: string | null;
  title: string;
  description: string | null;
  amount_mode: AmountMode;
  amount: string | null;
  amount_min: string | null;
  amount_max: string | null;
  currency: string;
  line_items: RenderLineItem[];
  subtotal: string | null;
  tax_total: string | null;
  customer_fields: { name: RenderField; email: RenderField; phone: RenderField };
  billing_required: boolean;
  shipping_required: boolean;
  questions: { key: string; label: string; type: QuestionType; options: string[]; required: boolean; per_order: boolean }[];
  methods: RenderMethod[];
  fee_bearer: FeeBearer;
  chain_tolerance_bps: number;
  quote_expiry_seconds: number;
  success_mode: "message" | "redirect";
  success_message: string | null;
  failure_retry: boolean;
  failure_message: string | null;
  receipt_email: boolean;
  expires_at: string | null;
  branding: { logo_url: string | null; accent_color: string | null; language: string };
}

const v2 = { baseUrl: API_V2_BASE_URL };

function listQuery(p: LinkListParams): string {
  const q = new URLSearchParams();
  if (p.status) q.set("status", p.status);
  if (p.limit) q.set("limit", String(p.limit));
  if (p.offset) q.set("offset", String(p.offset));
  const s = q.toString();
  return s ? `?${s}` : "";
}

const path = (id: string, action = "") => `/links/${encodeURIComponent(id)}${action}`;

export const linksApi = {
  list: (p: LinkListParams = {}) => apiFetch<LinkList>(`/links${listQuery(p)}`, v2),
  get: (id: string) => apiFetch<PaymentLink>(path(id), v2),
  create: (input: Partial<LinkInput>) => apiFetch<PaymentLink>("/links", { ...v2, method: "POST", body: input }),
  update: (id: string, patch: Partial<LinkInput>) =>
    apiFetch<PaymentLink>(path(id), { ...v2, method: "PATCH", body: patch }),
  remove: (id: string) => apiFetch<void>(path(id), { ...v2, method: "DELETE" }),
  publish: (id: string) => apiFetch<PaymentLink>(path(id, "/publish"), { ...v2, method: "POST" }),
  pause: (id: string) => apiFetch<PaymentLink>(path(id, "/pause"), { ...v2, method: "POST" }),
  archive: (id: string) => apiFetch<PaymentLink>(path(id, "/archive"), { ...v2, method: "POST" }),
  duplicate: (id: string) => apiFetch<PaymentLink>(path(id, "/duplicate"), { ...v2, method: "POST" }),
};
