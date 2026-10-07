# Ticket 02 fee rules - fix round 2

Branch `fees`. Re-review: `.superpowers/fees-rereview.md` (all original findings fixed; new minors N1-N6).

Commits this round (oldest first):

- `d1aec39` fix round 1 report (with its full-suite result) and the re-review file
- `0686c83` merge of `ledger` (fix round 2: pinned trigger `search_path`, `varchar(32)` assets with a format CHECK, POL seeds, boot checks on function definitions)
- `f4d0b0c` fees: N1-N6
- the commit that adds this report

## Step 0: merge

`git merge ledger` merged without conflicts.
To keep the post-merge result clean while I edited, I ran `make test-integration` on an exported copy of the merge commit (`git archive` into my scratchpad; not a worktree).
Result: 23 packages ok, 2 failed, both by Go's 10-minute package timeout and not by an assertion: `database` (600.8 s, in `TestApplyMigrationsRejectsLegacyOrMixedSchemaWithoutMetadata/mixed`) and `paymentlifecycle/postgres` (601.4 s, in `TestStoreOpenEnforcesAssetDecimalDomain/decimals_0`).
That run overlapped with my fees integration run on the same Colima VM, which more than quadrupled container times (fees took 368 s instead of 85 s). The run that counts is the full suite below, alone, on the final code, which includes the merge.

Adaptations to the merged ledger: snapshot `ledger_asset` is `varchar(32)` to match; the asset now comes from the ledger's resolver (N1), so it always satisfies the ledger's format CHECK.

## Findings

Every new test below was written before its implementation and failed first (most failed to compile against the old API: `WithAssetResolver`, `ErrFeeAlreadyPosted`, `ErrPostingConflict`, `ErrGrossMismatch`, `fee_postings`, `fee_rules_close_lineage`; `TestBaseFromGrossInvertsTheCustomerSurcharge` failed on its own assertions until the slab rule below was added).

**N1, chain only shape-checked.** For an on-chain asset, `Snapshot` looks up the `blockchain_currencies` row for `(currency, chain)` that is not deleted, has `deposit_enabled`, and whose chain is not deleted and has `status = 'active'`. No row is a `chain` `ValidationError`; more than one is a configuration error. The asset is then built by the ledger's own resolver: `service.LedgerAssetResolver()` (a thin export of `blockchainCurrencyAssetResolver`) is injected through `modules.Deps.LedgerAsset` and `fees.WithAssetResolver`. Without a resolver an on-chain snapshot is refused rather than concatenated. Fiat stays the bare ISO code and refuses a chain. `fees` cannot import `service` (service imports modules imports fees), which is why it is injected.
Test: `TestIntegration_SnapshotChainMustBeAnActiveBlockchainCurrency` (unknown chain, inactive chain, real chain without the currency, the happy path giving `USDC.BASE`, and the missing-resolver refusal).

**N2, raw unique violation on a race.** The insert is `ON CONFLICT (attempt_id) DO NOTHING` followed by a reload and the N3 comparison, so the losing racer gets `ErrSnapshotConflict` without aborting the caller's transaction; a `23505` that still surfaces is mapped to `ErrSnapshotConflict` as well.
Test: two payments race one new attempt id; exactly one succeeds and one gets `ErrSnapshotConflict`.

**N3, replay with a different query.** `fee_snapshots` stores the normalized query (`method`, `connector`, `card_type`, `region`, `currency`, `chain`). A replay with the same query (case-insensitive) returns the original; a different connector, card type, method or currency, or another payment, is `ErrSnapshotConflict`.
Test: `TestIntegration_SnapshotReplayAndConcurrentReuse`.

**N4, more than one posted fee per payment.** New append-only table `fee_postings`, unique on `attempt_id` and on `(payment_request_id, environment)`, written by `PostFee` before the journal with `ON CONFLICT DO NOTHING`. A second attempt of a posted payment gets `ErrFeeAlreadyPosted` and an error-level log (`fees: second successful attempt on one payment refused`, with both attempt ids and the refused amount). A zero fee still takes the slot. The same attempt replays; the same attempt with another amount is `ErrPostingConflict`. The environment is the server's `SERVER` lowercased, via `fees.WithEnvironment`, until ticket 13 brings live/test modes. The README "Payment path" tells the switch to treat `ErrFeeAlreadyPosted` as a duplicate success to reconcile, not a retry.
Test: `TestIntegration_OneFeePostingPerPayment` (merchant debited once, replay ok, zero-fee slot, two posting rows).

**N5, GUC bypass.** Removed. Versions are closed only by `fee_rules_close_lineage(lineage uuid, at timestamptz)`, `SECURITY DEFINER`, `SET search_path = pg_catalog, pg_temp`, with schema-qualified table references baked in through `format()` like the ledger's functions. It refuses `at` earlier than `transaction_timestamp()`. The `fee_rules` trigger (also pinned) accepts an `effective_to` change only when `current_user` equals the close function's owner, and never into the past for anyone.
The migration hands the function to `ledger_owner` (and grants it `SELECT, UPDATE` on `fee_rules`) under the same conditions the ledger uses: superuser or able to `SET ROLE ledger_owner`, and `ledger_owner` has `CREATE` on the schema; otherwise it raises a NOTICE. `NewVersion` now serialises editors with `pg_advisory_xact_lock` on the lineage instead of `SELECT ... FOR UPDATE`, so the application role needs no `UPDATE` on `fee_rules` at all.
Test: `TestIntegration_VersionCloseIsReservedToTheCloseFunction` applies the real migrations, creates a narrowed `fees_app` role (with `UPDATE` deliberately granted, to prove the trigger and not just privileges), and shows: the function is owned by `ledger_owner`; the app role can create and version a rule; a direct close and the old `SET LOCAL fees.closing_version` bypass are both refused; the function refuses a past instant; even the superuser table owner's direct close is refused.

Is it airtight? Under the production topology the ledger already enforces, yes. The app role cannot become the function owner (the ledger boot check refuses an app role that is a member of `ledger_owner`), and nothing it can set changes `current_user`.
It is not airtight in two topologies, both documented in the README and `docs/OPERATIONS.md`:
1. **The server runs as the migrator and the hand-over could not happen.** The owner check passes for that role, and only the never-in-the-past rule holds.
2. **The table owner or a superuser.** They can disable triggers anyway, the same limit the ledger documents.

**N6, caller time and customer-borne capture.**
- **Database time.** `Snapshot` resolves at `transaction_timestamp()` and ignores `Query.At`. `CreateRule` and `NewVersion` default and check `effective_from` against the same clock, which also removes app/DB clock skew from the close function's past check. `Preview` uses database time too.
- **Customer-borne capture.** For a customer-borne rule, `captured` is the gross the customer paid. `baseFromGross` recovers the base from estimates per pricing regime (each slab, the min clamp, the max clamp), each checked exactly with `Compute` on nearby grid points. Exactly one base must match, otherwise `ErrGrossMismatch`; a gross can be unreachable because fee rounding skips a cent. The fee is computed on that base, so there is no fee on the surcharge and the result matches the preview.
- **Slab rule.** Tiers that lower the rate above a bound would make some totals reachable from two bases. Customer-borne rules whose total does not strictly increase across every slab bound are therefore refused at write time (`slabs` `ValidationError`); merchant-borne rules are unaffected.

Tests: `TestBaseFromGrossInvertsTheCustomerSurcharge` (linear with tax, odd cents, min and max clamp, slab bound, JPY, a computed unreachable gross, an ambiguous lowering-tier gross), `TestCustomerBorneSlabsMustNotLowerTheTotal`, and `TestIntegration_SnapshotTimeAndCustomerBorneGross` (a caller `At` two days ahead still gets the current version; gross 102.20 under 2% + 10% tax posts fee 2.00 and tax 0.20 on base 100).

The re-review's note about developer databases built with the pre-fix schema applies again: `CREATE TABLE IF NOT EXISTS` will not add the new columns to an existing local `fee_snapshots`; recreate local databases once. `2026100702` is still not on `main`, so regenerating it in place is safe.

## Test summary

- `go build ./...`, `go vet ./...`, `go vet -tags=integration ./...`, `go test ./...`: 24 packages ok, none failing.
- Fees integration package alone: 16/16 integration tests pass.
- Full `make test-integration` on the final commit, run alone: see "Full suite" below.

## Not fixed, with reason

- Nothing from N1-N6 is left open.
- N5 has the two limits stated above. Both are the same limits the ledger accepts for its own triggers.
- The pre-existing `gofmt` findings outside this module are still untouched, for the reason given in round 1.

## Full suite

`make test-integration` on `f4d0b0c` (which includes the `ledger` merge), run with nothing else using Docker: exit 0, all 25 packages ok.
Slowest packages: `paymentlifecycle/postgres` 405.9 s, `database` 400.3 s, `fees` 342.4 s, `ledger` 280.9 s, `service` 181.8 s.
This also clears the two timeouts from the contended post-merge run above: `database` and `paymentlifecycle/postgres` pass well under the 10-minute limit.
