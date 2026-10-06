export interface RedactedWebhook {
  id: number;
  externalPlatformID: number;
  url: string;
  events: string[];
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreatedWebhook extends RedactedWebhook {
  secret: string;
}

type JsonRecord = Record<string, unknown>;

function record(value: unknown, label: string): JsonRecord {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new Error(`Invalid ${label} response`);
  }
  return value as JsonRecord;
}

function normalizeWebhookFields(wire: JsonRecord): RedactedWebhook {
  if (
    typeof wire.id !== "number" ||
    typeof wire.externalPlatformID !== "number" ||
    typeof wire.url !== "string" ||
    !Array.isArray(wire.events) ||
    typeof wire.active !== "boolean" ||
    typeof wire.createdAt !== "string" ||
    typeof wire.updatedAt !== "string"
  ) {
    throw new Error("Invalid webhook response");
  }
  const events = wire.events as unknown[];
  if (
    events.length === 0 ||
    !events.every(
      (event) =>
        typeof event === "string" &&
        event.length > 0 &&
        event.trim() === event &&
        !event.includes(",")
    ) ||
    new Set(events).size !== events.length
  ) {
    throw new Error("Invalid webhook events");
  }
  return {
    id: wire.id,
    externalPlatformID: wire.externalPlatformID,
    url: wire.url,
    events: events as string[],
    active: wire.active,
    createdAt: wire.createdAt,
    updatedAt: wire.updatedAt,
  };
}

export function normalizeWebhook(value: unknown): RedactedWebhook {
  const wire = record(value, "webhook");
  if ("secret" in wire) {
    throw new Error("Unexpected webhook secret in redacted response");
  }
  return normalizeWebhookFields(wire);
}

export function normalizeCreatedWebhook(value: unknown): CreatedWebhook {
  const wire = record(value, "created webhook");
  if (typeof wire.secret !== "string" || wire.secret.length === 0) {
    throw new Error("Invalid created webhook secret");
  }
  return { ...normalizeWebhookFields(wire), secret: wire.secret };
}

export function unwrapWebhook(value: unknown): RedactedWebhook {
  return normalizeWebhook(record(value, "webhook envelope").webhook);
}

export function unwrapCreatedWebhook(value: unknown): CreatedWebhook {
  return normalizeCreatedWebhook(
    record(value, "created webhook envelope").webhook
  );
}

export function unwrapWebhookList(value: unknown): RedactedWebhook[] {
  const envelope = record(value, "webhook list envelope");
  if (!Array.isArray(envelope.webhooks)) {
    throw new Error("Invalid webhook list");
  }
  return envelope.webhooks.map(normalizeWebhook);
}
