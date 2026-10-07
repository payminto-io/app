# Ticket 09 fix round 3 - Solana sweeps as a reconciler

Branch `solana`, from `96512ad`, worktree `gateway-wt-solana`.
Inputs: `.superpowers/solana-rereview-2.md`, `.superpowers/solana-fix-2-report.md`, the previous agent's uncommitted work (8 files), the coordinator's rulings.
Date: 2026-10-07.

## Status

Every finding of re-review 2 is addressed: C1, I1-I4, M1-M2, L1-L5.
The reviewer's five probes are ported as tests (`backend/internal/service/solana_fix3_probes_test.go`), committed failing first, and all pass now.

## What the previous agent left, and what was kept

Kept: `SOLANA_SWEEPS.md` (rewritten to match the final code), `SignForSend` / `SendSigned` / `IsRejection` in `blockchain/solana/sweep.go`, the `solana_sweep_locks` table, `unresolved_attempts`, the probe file, and most of the reconciler in `solana_sweep_service.go`.
Fixed in what was kept:

- The lock row embedded `PaymintoModel`, so `Delete` was a soft delete and the deleted row kept holding the unique index: an account could never be swept again after its first sweep (`TestProbe_ResweepAfterFailedSweep` failed on this).
  The lock is now a plain row (id, created_at, token_account, sweep_id), always hard-deleted; migration matches.
- `SendSigned` treats a node returning a different signature as an error; existing tests scripted fake names (`SIG1`).
  Rather than weaken the check, the sweep fixture now goes through `aliasCaller` (`solana_sweep_alias_test.go`), which maps a scripted fake to the real signature the sweeper signed.
- `recoverUnrecorded` and the `solana_unrecorded_broadcast` anomaly are removed: with persist-before-send nothing writes that anomaly any more, and it was the only producer of height-0 attempts.

## Design

`backend/internal/service/SOLANA_SWEEPS.md`: one table each for account states, sweep states (allowed transitions and the evidence that moves each), attempt states, expiry evidence, and recovery after a restart from the database plus chain lookups alone.
Invariants the code now enforces:

1. Sweep rows, links and locks are written in one transaction before anything is signed.
2. Sign, persist the attempt (signature, blockhash, real `last_valid_block_height`, status `signed`), then send.
3. A JSON-RPC rejection fails the attempt; any other send error leaves it tracked.
4. One sweep in flight per token account: unique `solana_sweep_locks.token_account`; `skipAccount` checks it; orphan locks (sweep no longer in flight) are deleted at the start of `SweepConfirmed` and end of `TrackConfirmations`.
5. Deposits become `swept` only in `book`.
6. No attempt is written with height 0; a transaction recovered from history gets finalized height at recovery + 150, an upper bound that can only delay expiry evidence.
7. An attempt is persisted only while its sweep is `processing`/`pending` (same transaction) and under the next `attempt_no` (unique `(sweep_id, attempt_no)`): a sweep failed meanwhile is never revived by a late signer, and two workers rebuilding one sweep cannot both send.

## Findings

- **NEW2-C1** - `pollAccount` carries `held_signature`/`held_attempts` unchanged on any non-hold error; only a completed poll clears it. A failed poll explains no balance movement, so a moved balance stays unexplained and keeps the next poll alive. Unresolved signatures are marked seen so they never re-hold the cursor. Tests: probe `TestProbe2_HeldSignatureSurvivesATransportError`, `TestSolanaDeposit_TransportErrorKeepsMovedBalanceUnexplained`.
- **NEW2-I1** - persist-before-send; a transport error leaves the attempt `signed`, and the reconciler resolves it by `getSignatureStatuses`. Tests: probe `TestProbe2_SendTransportErrorButLanded` (lands and is booked; a rejection fails and releases), `TestSolanaSweep_SignatureIsPersistedBeforeSend`, `TestSolanaSweep_SignedAttemptThatNeverLandedIsRebuilt`.
- **NEW2-I2** - `resolveNoAttempt`: a `processing` sweep with no attempt past `ValidityWindow` (2 min) reads its accounts. All holding their claim: `failSweep` with `solana_sweep_never_sent`, lock released, deposits back to confirmed. Below claim: history; a transaction our fee payer signed becomes the attempt; nothing of ours: account `drained`, `solana_unexplained_drain`, sweep failed. Also, a `pending` sweep waiting on a short balance with nothing finalized now gets a `DrainWait` (1 h) after which its history must explain the drain, so it cannot wait forever. Tests: probe `TestProbe2_ProcessingSweepWithoutAttemptIsStuck`, `TestSolanaSweep_ProcessingWithoutAttemptRecoversOurTransactionFromHistory`, `TestSolanaSweep_ProcessingWithoutAttemptAndForeignDrainFailsWithAnomaly`, `TestSolanaSweep_ShortBalanceWaitsThenNeedsAnExplanation`.
- **NEW2-I3** - no height-0 writes (see invariant 6); a legacy height-0 row still expires by age plus the two-endpoint absence and the balance guard. Test: probe `TestProbe2_RecoveredAttemptThatDroppedNeverRebuildsOrFails`.
- **NEW2-I4** - the lock table (database-enforced) plus `skipAccount`. Tests: probe `TestProbe2_SecondSweepBroadcastWhileFirstIsTracked` (the second deposit waits for sweep 1 to book, then is swept), `TestSolanaSweep_LockIsUniquePerAccount`, `TestSolanaSweep_ConcurrentRebuildAndFailedSweepSendNothing`.
- **NEW2-M1** - `requireProviderEndpoint`: live refuses boot with fewer than two distinct endpoints (URLs compared case-insensitively, trailing slash ignored), in addition to the provider rule. Test: `TestWireSolana_LiveRequiresTwoDistinctEndpoints`.
- **NEW2-M2** - `SOLANA_POST_DEPOSIT_JOURNALS` is removed. The watcher always has the ledger and posts the payment journal in the finalize transaction unless the switch owns the payment (`payment_requests.invoice_id` is a `switch_payment_attempts.id`; `switchOwnsPayment`). Tests: `TestSolanaDeposit_SwitchPathPostsExactlyOnePaymentJournal` (now with the ledger wired: watcher posts zero, switch posts one) and `TestSolanaDeposit_LegacyPaymentGetsExactlyOneWatcherJournal` (switch tables present, merchant invoice id, one journal, replay adds none).
- **NEW2-L1** - `TrackConfirmations` queries only `pending`/`processing` sweeps, paging by id in pages of 200.
- **NEW2-L2** - `history` pages up to `HistoryPages` (5) pages of 100 with a `before` cursor.
- **NEW2-L3** - `unresolved_attempts` counts polls with unresolved signatures outstanding; past `MaxUnresolvedPolls` (36) the account expires with `solana_unresolved_at_expiry` and the expired scan keeps retrying the signatures. Test: `TestSolanaDeposit_UnresolvedSignatureDoesNotBlockExpiryForever`.
- **NEW2-L4** - the expired scan reads only accounts within `ExpiredScanFor` (90 days) of `watch_until`; `ListExpiring` filters held and in-budget unresolved accounts in SQL so they cannot starve the 200-per-tick window. Cost table in the README.
- **NEW2-L5** - the sweeper posts through `r.journal`.

One more defect found on the way: a re-sweep signed against the same blockhash produces the identical signature and hit the attempts' unique index after rows were written. `signAndSend` now checks for an existing signature first and does not send (first attempt: the sweep fails cleanly, next round signs against a fresh blockhash).

## Checks

- `cd backend && go build ./... && go vet ./... && go test ./...` - pass.
- Integration (`-tags=integration -count=1 -p 2 -timeout 45m`, Colima socket) over `./internal/blockchain/... ./internal/worker/... ./internal/service/... ./internal/modules/... ./internal/paymentswitch/...` - pass. `TestSolanaEndToEnd` runs on the installed `solana-test-validator` and passes; `TestSolanaDevnet` skips (no funded key).
- Postgres: every `TestProbe*`, `TestSolanaDeposit_*` and `TestSolanaSweep_*` test re-run with the fixture on a `postgres:16-alpine` testcontainer (`database.NewTestDB` plus the `2026100708` migration SQL, via `-overlay`, not committed) - 46 of 46 pass, 240 s.

## Commits

- `ab29b02` state model, lock table, the five probes failing
- `938906d` sweep reconciler
- `add9d49` NEW2-C1
- `164ed77` NEW2-L3, NEW2-L4
- `af22a29` NEW2-M1, NEW2-M2, NEW2-L5, README
- `51ff05d` reconciler path tests
- `fa2df96` attempt numbering and in-flight guard

## Notes for the reviewer

- `DrainWait`, `ValidityWindow`, `HistoryPages`, `MaxUnresolvedPolls`, `ExpiredScanFor` are config defaults, not env variables.
- `switchOwnsPayment` checks `HasTable("switch_payment_attempts")` inside the finalize transaction; a deployment without the switch tables posts from the watcher.
- An `explainDrain` revival of a failed sweep inserts the lock in the same transaction as the revive; if another sweep holds the lock the revive is rolled back and logged.
