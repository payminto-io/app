# 07 Card connector through hosted fields

Status: ready-for-agent
Owner: Backend engineer (Opus)
Blocked by: 05

## Goal
One card connector (Stripe) implementing the ticket 05 interface, card data only via the provider's hosted fields or tokens, 3DS via the provider, authorize-and-capture and capture-now, refunds, webhook verification with replay protection. Secrets per environment.

## Acceptance
Recorded-fixture tests for every call; status map coverage; webhook replay rejected; no PAN ever logged (test asserts on log output).
