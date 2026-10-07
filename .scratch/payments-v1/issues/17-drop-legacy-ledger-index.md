# 17 Drop the four-column ledger account index

Status: ready-for-agent
Owner: Principal engineer (Fable)
Blocked by: 13

## Goal
Migration `2026100705_environment_isolation` added `ledger_accounts_env_owner_asset_kind_key` beside the pre-ticket `ledger_accounts_owner_asset_kind_key` so binaries that still infer the four-column arbiter keep posting during a rolling deploy (review of ticket 13, I1). Once no such binary runs anywhere, a new additive-numbered migration drops `ledger_accounts_owner_asset_kind_key` and `constraints.sql` stops creating it.

## Acceptance
Migration applies on a database that has both indexes and on one that already lacks the old one; `TestIntegration_OldBinariesStillPostDuringARollingDeploy` is retired in the same change; the ledger integration suite passes.
