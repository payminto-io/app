# 04 Payment link form with live preview

Status: done
Owner: Frontend engineer + UI/UX (Opus)
Blocked by: 03

## Goal
Six collapsible steps on one page in `frontend/` under `/dashboard/links/new` and `/dashboard/links/[id]`, with the checkout preview beside it. Fee preview per method shows rule and version. Reference: Kuberopay `web/src/pages/payment-link-builder` for structure (design reference only; reuse code only if the owner confirms those files are theirs).

## Acceptance
Every field maps to a column; validation messages match the API; preview updates on every change; keyboard accessible; works at 390px; no placeholder numbers.

## Comments

- 2026-10-07 (Frontend + UI/UX, Opus): done on branch `link-form`. Commits `181a226` (typed v2 client, fee preview client, form model, error and preview mapping, tests), `96db405` (list, builder with live preview, detail actions, nav, preview fixtures), `4ad2a4b` (touch sizes, preview polish, screenshots in `docs/design/screens/links/`). Full report: `.superpowers/link-form-report.md`. Verified on the dev preview with sample fixtures, not the real backend. API gaps listed in the report (no draft render/pricing preview with connector, no enabled-methods or currencies endpoint, no destinations endpoint, no merchant name).
