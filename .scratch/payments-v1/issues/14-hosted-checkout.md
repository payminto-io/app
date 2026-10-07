# 14 Hosted checkout rendering every link option

Status: ready-for-agent
Owner: Frontend engineer (Opus)
Blocked by: 04, 07, 09

## Goal
`checkout/` renders the link's render model: line items or customer amount, customer fields per policy, custom questions, method choice (card hosted fields, USDC on Solana with QR, amount and expiry countdown), surcharge display, success and failure behaviour, receipt. Branding from the link.

## Acceptance
Playwright flow per method against the mock connector and mock chain; 390px and 1440px audited; no number shown that is not from the API.
