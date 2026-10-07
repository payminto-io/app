# 03 Payment link model and API

Status: ready-for-agent
Owner: Backend engineer (Opus)
Blocked by: 02

## Goal
The payment link as a product object, per the six-step form in `reports/Payments V1 on Kuberopay.md`. Extends Payminto's `payment_requests` rather than replacing it.

## Tables (additive)
`payment_links` (id, member_id, external_platform_id, title, description, amount_mode fixed|customer|line_items, amount, amount_min, amount_max, currency, reference_id, metadata jsonb, category, customer_field_policy jsonb, billing_required, shipping_required, multi_use bool, use_limit, methods jsonb, capture_mode, three_ds_policy, chain_tolerance_bps, quote_expiry_seconds, fee_bearer, success_mode message|redirect, success_url, success_message, receipt_email bool, receipt_note, webhook_id, failure_retry bool, settlement_override jsonb nullable, hold_in_asset bool, settlement_timing cycle|immediate, expires_at, expires_after_payments, status draft|active|paused|archived, logo_url, accent_color, language, short_code unique, created_at, updated_at). `payment_link_line_items`, `payment_link_questions`, `payment_link_answers`.

## API
CRUD under `/links`, `POST /links/:id/publish`, `/pause`, `/duplicate`; public `GET /pay/:short_code` returns the render model. Validation from the spec: method needs fee rule and connector in the environment; settlement override needs verified destination; surcharge blocked where disallowed; live links cannot change amount, currency or methods.

## Acceptance
Handler tests for every validation rule; short code and QR generated on publish; a paid link creates a `payment_request` with fee snapshot.
