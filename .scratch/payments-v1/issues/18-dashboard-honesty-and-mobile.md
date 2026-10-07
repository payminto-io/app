# 18 Dashboard follow-ups: per-asset totals and mobile tables

Status: done
Owner: Frontend engineer (Opus) with UI/UX
Blocked by: 01

## Goal
- Totals that sum across assets (home "Deposit volume", sweeps, gas) are meaningless; show per-asset totals, or a USD value only once the ledger provides one from executed rates. Never add BTC to USDC.
- Tables at 390px truncate columns; use a stacked row layout (amount + status on line one, reference and customer on line two) below the tablet breakpoint, shared through `data-table`.
- Leftovers from the restyle report: worker controls call endpoints that return 501 (hide until implemented); referrals shows fallback 0 when stats 404 (show nothing instead); hot wallet low-balance floors hard-coded in the page (move to configuration or remove); sticky first column in wide tables; faint outline-button border in dark mode.

## Acceptance
Screenshots at 390 and 1440, light and dark; no figure shown that is not a single-asset sum or a ledger-backed value.

## Comments

Done on branch `dash-followups`.
Commits: `0f0bf83` (stacked rows below 1024px, sticky first column), `a1d4cd6` (per-asset totals, summed figures removed), `f6ff14a` (worker controls, gas floors, dark borders), `c811ef9` (polish after visual review), `4422991` (captures).
Checks: `npx tsc --noEmit`, `npm run lint`, `npm test` (4 files, 25 tests) pass.
Screens: `docs/design/screens/pages/` at 390 and 1440, light and dark; DESIGN.md section 15 lists which.
Home and analytics show paid deposits per asset from `/analytics/revenue`; sweeps shows no figures, because `/analytics/sweeps` is instance-wide and summed across assets.
API fields needed and the cross-tenant sweep finding: `.superpowers/dash-followups-report.md`.

