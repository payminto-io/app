# 07 Demo fiat connectors: Keypay and Payvang

Status: ready-for-agent
Owner: Backend engineer (Fable)
Blocked by: 05

## Goal
Owner decision 2026-10-07: no Stripe connector. Port Kuberopay's two hosted-redirect fiat processors into the gateway's connector slot for demo use, in Go: Kuberpayss under the provider name **Keypay**, and **Payvang** (UPI pay-in). Reference: Kuberopay `backend/libs/shared/src/processors/kuberpayss.adapter.ts`, `payvang.adapter.ts`, `base.adapter.ts`, `types.ts`, and the connector credential and config schema next to them. Behaviour to keep: hosted redirect at authorize (payment URL returned as `next_action`), the provider's status call as the only trusted outcome, our reference as the idempotency and sync key, per-environment base URLs overridable per merchant, no default base URL where the provider has no sandbox.

## Acceptance
Both implement `connectors.Connector` and pass the conformance suite against a local fake server; status maps cover every provider status; credentials only from configuration or the secrets vault, never logged; live refuses test base URLs; the switch drives an end-to-end demo payment through each in test with the fake server; checkout can render the redirect.
