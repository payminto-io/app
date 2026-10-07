# 18 Dashboard follow-ups: per-asset totals and mobile tables

Status: ready-for-agent
Owner: Frontend engineer (Opus) with UI/UX
Blocked by: 01

## Goal
- Totals that sum across assets (home "Deposit volume", sweeps, gas) are meaningless; show per-asset totals, or a USD value only once the ledger provides one from executed rates. Never add BTC to USDC.
- Tables at 390px truncate columns; use a stacked row layout (amount + status on line one, reference and customer on line two) below the tablet breakpoint, shared through `data-table`.
- Leftovers from the restyle report: worker controls call endpoints that return 501 (hide until implemented); referrals shows fallback 0 when stats 404 (show nothing instead); hot wallet low-balance floors hard-coded in the page (move to configuration or remove); sticky first column in wide tables; faint outline-button border in dark mode.

## Acceptance
Screenshots at 390 and 1440, light and dark; no figure shown that is not a single-asset sum or a ledger-backed value.
