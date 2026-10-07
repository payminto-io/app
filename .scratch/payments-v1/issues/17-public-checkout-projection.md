# 17 Public checkout projection: fields the redesigned checkout needs

Status: ready-for-agent
Owner: Backend engineer (Opus)
Blocked by: 03, 05, 09

## Goal
The redesigned checkout (`checkout/`, see `.superpowers/checkout-ui-report.md` on branch `checkout-ui`) renders only what `GET /public/payment/:reference_id` returns. Add the missing fields from real data, never inferred: `merchant.logo_url`; `line_items[]`, `description`; `chain.received` (amount + asset), `chain.confirmations`, `chain.required_confirmations`, `chain.tx_hash`, `chain.explorer_url`; a `confirming` state (deposit seen below threshold) and `failed` with `failure_reason`; `paid_at`; `success.message`, `success.redirect_url`; `retry_allowed` and an endpoint to re-quote an expired payment; `methods.card` when a card connector is enabled on the link; `settled` for the third rail segment; `environment`; `card.brand`, `card.last4`; a quoted asset amount for non-stable assets.

Also answer two questions from the checkout report: release an assigned deposit address when the payer changes method, and quote expiry separate from payment expiry.

## Acceptance
Every field sourced from a table or a chain receipt, covered by a handler test; checkout preview fixtures match the real projection; copy trimmed where the owner asked for less text (confirming and awaiting lines).
