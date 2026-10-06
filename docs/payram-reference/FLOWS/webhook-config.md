# Webhook configuration

PayRam fires merchant webhooks on every payment / sweep / withdrawal status
transition. The dashboard surfaces are split across **two** routes — confusing
but that's how it shipped:

- `/developers/webhook` — per-developer webhook **endpoint** management
- `/settings/webhook` — global signing key + retry policy

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/developers/webhook` | [developers-webhook.md](../SCREENS/developers-webhook.md) |
| 2 | `/settings/webhook` | [settings-webhook.md](../SCREENS/settings-webhook.md) |
| 3 | `/developers/apiKeys` (uses same signing key) | [developers-apiKeys.md](../SCREENS/developers-apiKeys.md) |

## Developers → Webhook

Page lists configured endpoints in a card grid:

- **Endpoint URL**, **Description**, **Events** (subscribed event types),
  **Status** (Active / Paused), **Secret** (reveal-once)
- Toolbar: **"Add Endpoint"** lime CTA, search input, status filter chip group

### Add endpoint modal

Wizard with three sections:

1. **URL + description**
2. **Event subscription** — checkbox list grouped by domain
   (`payment.created`, `payment.updated`, `payment.completed`,
   `payment.cancelled`, `payment.refunded`, `withdrawal.created`,
   `withdrawal.completed`, `sweep.completed`, `customer.created`, …)
3. **Test delivery** — sends a sample event to the URL and shows the
   response status / body in a small terminal-style panel

Submit → `POST /api/v1/webhook-endpoints` (mutating).

### Per-endpoint detail drawer

Clicking a row opens a side drawer with three tabs:

- **Overview** — URL, secret (reveal-once), edit, pause, delete
- **Deliveries** — history table with status, retry count, response time,
  payload preview button
- **Events** — list of subscribed events with toggle switches

Delivery rows expand inline to show the **request payload** and **response
body** in monospace pre-tags using `--pr-font-mono` (JetBrains Mono).

## Settings → Webhook

A simpler page with global controls:

- **Signing secret** — used to compute the `Payram-Signature` HMAC header
  on every delivery; rotate button + reveal-once
- **Retry policy** — max attempts (default 5), backoff curve
- **Timeout** — per-attempt timeout in seconds (default 30)
- **Failure threshold** — auto-pause endpoint after N consecutive failures

## API endpoints (read side observed)

The crawl did not visit the developer-webhook page after the deep route
because the safe-click filter avoids "Test", "Send", and "Save" buttons.
The list/get endpoints expected here:

- `GET /api/v1/webhook-endpoints` — list
- `GET /api/v1/webhook-endpoints/{id}` — detail
- `GET /api/v1/webhook-endpoints/{id}/deliveries` — delivery history
- `GET /api/v1/webhook-endpoints/{id}/events` — subscribed events

(These are documented in `research/PAYRAM_TECHNICAL_DOCUMENTATION.md`; we did
not see them fire in `network.json` because the table is gated behind a
secondary fetch.)

## Signature scheme

Each webhook delivery carries headers (from
`research/PAYRAM_TECHNICAL_DOCUMENTATION.md`):

```
Payram-Signature: t=<unix>,v1=<hex hmac-sha256>
Payram-Event: payment.completed
Payram-Delivery: <uuid>
```

The HMAC is computed over `<t>.<raw body>` using the endpoint's signing
secret. The receiver should verify within a 5-minute clock skew.

## Gaps vs. Payminto clone

- `/developers/webhook` exists in the clone as a single rule list — no
  add-endpoint wizard, no test delivery, no deliveries history, no payload
  viewer modal.
- `/settings/webhook` is missing entirely.
- The reveal-once secret pattern (used by both webhook and api-key flows)
  is a missing primitive — see [`COMPONENT_LIBRARY.md`](../COMPONENT_LIBRARY.md) §10.
