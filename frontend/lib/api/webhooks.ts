/**
 * Webhooks domain API.
 * Routes: /webhooks (CRUD) + /webhooks/:id/deliveries.
 *
 * Backend wraps list responses as { webhooks: [...] } and
 * { deliveries: [...] }. We unwrap so hook consumers work with plain arrays.
 */
import { apiFetch } from "./client";
import {
  unwrapCreatedWebhook,
  unwrapWebhook,
  unwrapWebhookList,
  type RedactedWebhook,
} from "./webhook-contract";

export type { CreatedWebhook, RedactedWebhook } from "./webhook-contract";

export interface WebhookDelivery {
  id: number;
  webhookID: number;
  event: string;
  status: "pending" | "delivered" | "failed" | "retrying";
  statusCode?: number;
  attempts: number;
  responseBody?: string;
  requestBody?: string;
  createdAt: string;
  deliveredAt?: string;
}

export interface CreateWebhookInput {
  url: string;
  events: string[];
}

export interface UpdateWebhookInput {
  url?: string;
  events?: string[];
  active?: boolean;
}

export const webhooksApi = {
  list: async (): Promise<RedactedWebhook[]> =>
    unwrapWebhookList(await apiFetch<unknown>("/webhooks")),
  get: async (id: number) =>
    unwrapWebhook(await apiFetch<unknown>(`/webhooks/${id}`)),
  create: async (input: CreateWebhookInput) =>
    unwrapCreatedWebhook(
      await apiFetch<unknown>("/webhooks", { method: "POST", body: input })
    ),
  update: async (id: number, input: UpdateWebhookInput) =>
    unwrapWebhook(
      await apiFetch<unknown>(`/webhooks/${id}`, { method: "PUT", body: input })
    ),
  remove: (id: number) =>
    apiFetch<void>(`/webhooks/${id}`, { method: "DELETE" }),
  deliveries: async (id: number): Promise<WebhookDelivery[]> => {
    const res = await apiFetch<{ deliveries: WebhookDelivery[] }>(
      `/webhooks/${id}/deliveries`
    );
    return res.deliveries ?? [];
  },
};
