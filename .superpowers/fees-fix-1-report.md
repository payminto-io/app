# Ticket 02 fee rules - fix round 1

Branch `fees`. Review: `.superpowers/fees-review.md` (1 critical, 4 important, 8 minor).

Commits this round (oldest first):

- `933e0f5` merge of `ledger` (fix round: sealed journals, chain-qualified assets, `Record*In`, ledger roles)
- `b8ae98c` config: `FEES_*` keys in `config.FeesConfig` (M7)
- `dbd513d` fees: C1, I1, I2, I3, I4, M1-M6, M8
- `39a1349` docs: `docs/OPERATIONS.md` "Fee rules"
- the commit that adds this report

## Step 0: merge

`git merge ledger` conflicted only in `service/registry.go` (imports, and the ledger now builds its asset resolver from `blockchain_currencies`); resolved by keeping both sides.
`make test-integration` right after the merge: every package passed except one fees assertion, which expected the old signed `Balances` (-2 for fee income) where the merged ledger now returns natural balances (+2).
That is the API change this round adapts to; the rewritten tests read `AccountBalances` (signed and natural).

Adaptations to the new ledger:

- Assets: the fee journal posts in the snapshot's ledger asset, the bare ISO code for fiat (`USD`) and `CODE.CHAIN` for on-chain assets (`USDC.BASE`), built by `ledgerAsset` with the ledger's 16-character limit. `Snapshot` takes the chain in `Query.Chain` for method `crypto` and refuses a chain on fiat.
- `PostFee` posts through `ledger.PostIn` inside the caller's transaction, so it composes with the single-transaction `Record*In` style; no second connection is taken (all reads go through `tx`).
- Amounts posted are always on the currency grid and below 1e20, inside the ledger's new scale and magnitude rules.
- Ledger role ownership does not touch fee tables; the grants the application role needs on them are documented in `docs/OPERATIONS.md`.

## Critical

**C1, superseding a scheduled version left the previous one open.** `NewVersion` now closes every version of the lineage whose window extends past the new start, `effective_to = GREATEST(effective_from, new start)`, in the same transaction as the insert.
Test `TestIntegration_SupersedingAScheduledVersionClosesTheWholeLineage` (v1 open, v2 in 10 days, v3 now) asserts exactly one active version and that `Resolve` returns v3 at five instants across the old gap. With the old single-row close put back, the test fails, and the new exclusion constraint itself refuses the write (`overlaps active rule(s) 1,2`).

## Important

**I1, unbounded exponents.** `checkDecimal` rejects an exponent outside plus or minus 40, more than 40 digits, or a magnitude of 1e20 or more before any arithmetic or comparison, and only then checks the scale. It guards preview amounts, `PostFee` captured amounts, and on rule writes every `percent`, `flat`, `min_fee`, `max_fee`, `tax_percent` and slab `up_to`/`percent`/`flat`. Fee route bodies are capped at 16 KiB.
`TestHugeExponentsAreRejectedBeforeArithmetic` sends `1e100000000`, `1e-100000000` and `-1e2000000000` as an amount and as a slab bound and requires each rejection in under 10 ms. With the guard removed the test does not finish within a minute.

**I2, overlapping same-scope rules.** Migration `2026100702` (regenerated in place) runs `CREATE EXTENSION IF NOT EXISTS btree_gist` and adds `fee_rules_no_overlap EXCLUDE USING gist (method, currency, coalesce(connector), coalesce(card_type), coalesce(region), tstzrange(effective_from, effective_to, '[)') &&)`. Empty windows never conflict.
A violation (`23P01`) becomes `*OverlapError` naming the rules it overlaps, HTTP 409 `overlapping_fee_rule`.
`TestIntegration_OverlappingRulesForOneScopeAreRefused` covers: a second create, a future rule in another lineage, an unrelated connector allowed, a direct SQL insert refused by the constraint, six concurrent creates of one new scope with exactly one winner, and the extension present.
`btree_gist` is trusted, so the migration role needs `CREATE` on the database, not superuser; `docs/OPERATIONS.md` "Fee rules" says what to run if it lacks that.

**I3 + I4, `ApplyToPayment` trusted the caller and fused snapshot with posting.** Replaced in the port by:

- `Snapshot(ctx, tx, AttemptRef{PaymentRequestID, AttemptID}, Query)` at attempt creation: fees resolves the rule itself, reads the merchant from `payment_requests.member_id` under `FOR UPDATE`, derives the ledger asset, and writes one append-only `fee_snapshots` row per attempt (unique `attempt_id`, composite FK to `fee_rules (id, version)`). A retry on another connector is a new attempt with its own snapshot. Same attempt and same payment is idempotent; same attempt id on another payment is `ErrSnapshotConflict`.
- `PostFee(ctx, tx, AttemptRef, captured)` on success or capture: loads the snapshot, re-locks the payment, checks the merchant has not changed, loads the rule by `(id, version)`, recomputes the fee on the captured amount, posts the journal keyed `fee:attempt:<id>`, and sets the legacy `payment_requests.fee_rule_id/fee_rule_version` from that attempt (the latest successful attempt wins). The old fixed-once trigger on those columns is dropped for that reason.

Nothing is posted for an attempt that never succeeds. `TestIntegration_SnapshotPerAttemptAndPostFeeOnCapture` covers a retry on another connector, a rule edited after the snapshot (the capture still prices under the snapshotted version), posting on a captured amount below the payment amount, idempotent replay, a replay with another amount refused by the ledger, attempt reuse across payments, and an unknown attempt. `TestIntegration_CryptoFeePostsInTheChainQualifiedAsset` covers the chain requirement, `USDC.BASE`, and the fee-above-amount refusal at capture. The README's "Payment path" names both call points for the switch.

## Minor

- **M1** Percentages with more than 6 decimals and money values finer than the currency's minor unit are rejected, not rounded by Postgres; created and versioned rows are re-read after insert so the response is what is stored. `minor_units` is stored per version so a later precision config change cannot change how an old version rounds. Tests in `TestValidateRejectsPrecisionBeyondTheColumn` and `TestIntegration_DecimalsRoundTripExactly`.
- **M2** `Precision` holds the ISO 4217 list with real minor units (0, 2, 3, 4) plus on-chain assets, extended by `FEES_ASSET_PRECISION`; `XRP`, `TON`, `ADA`, `USF` are rejected unless configured. Fiat methods take only ISO codes and `crypto` takes only assets.
- **M3** A fee plus tax above the amount is `ErrFeeExceedsAmount` (HTTP 422 `fee_exceeds_amount`) in preview and in `PostFee`, for either bearer.
- **M4** The trigger refuses `effective_to` earlier than `transaction_timestamp()` unless the transaction set `fees.closing_version`, which only `NewVersion` does (`SET LOCAL`).
- **M5** `Snapshot` and `PostFee` read the payment with `deleted_at IS NULL`; `TestIntegration_SoftDeletedPaymentsAreRefused`.
- **M6** `/admin/fee-rules` uses `JWTAuth` (dashboard session); preview keeps `JWTOrAPIKey`. `RegisterFeesRoutes` now takes a `FeesAuth{Merchant, Session, Admin}`; admin routes mount only with both Session and Admin.
- **M7** `config.FeesConfig` loaded by `config.Load`; `FEES_OPERATOR_PLATFORM_ID` parsed there and rejected unless a positive integer.
- **M8** All the listed test gaps are covered above. `TestNewRouterEnforcesSessionPermissionAndOperatorOnFeeRules` builds the real `AuthService` and RBAC services over SQLite and signs real JWTs: no credentials 401, API key 401, merchant without `system.admin` 403, `system.admin` on a non-operator platform 403 (operator gate), operator admin 200, merchant preview 200.

## Test summary

- `go build ./...`, `go vet ./...` (also with `-tags=integration`), `go test ./...`: all packages ok.
- `make test-integration`: exit 0, all 25 packages ok (database 136 s, fees 85 s, ledger 88 s).
- New or rewritten this round: 8 fees unit tests (precision, currency class, column scale, huge exponents, fee above amount, ledger asset, journal by attempt, policy precision), 5 new fees integration tests and 3 rewritten, 1 config test, 2 module tests reworked, 2 api tests (session-only admin guard, real RBAC router test).

## Not fixed, with reason

- Nothing from the review is left open.
- `gofmt -l` lists about 20 pre-existing files outside this module (handlers, models, blockchain, worker). They are not part of this ticket and other worktrees edit those packages; reformatting them here would create merge noise. Worth a dedicated formatting commit on `main`.
- `fees` does not yet add a boot-time schema check like `ledger.ValidateSchema`; the ledger round added that for money tables, and fees could follow when ticket 13 (environments) revisits boot checks.
