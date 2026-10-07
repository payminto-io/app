# 04 Payment link form with live preview

Status: ready-for-agent
Owner: Frontend engineer + UI/UX (Opus)
Blocked by: 03

## Goal
Six collapsible steps on one page in `frontend/` under `/dashboard/links/new` and `/dashboard/links/[id]`, with the checkout preview beside it. Fee preview per method shows rule and version. Reference: Kuberopay `web/src/pages/payment-link-builder` for structure (design reference only; reuse code only if the owner confirms those files are theirs).

## Acceptance
Every field maps to a column; validation messages match the API; preview updates on every change; keyboard accessible; works at 390px; no placeholder numbers.
