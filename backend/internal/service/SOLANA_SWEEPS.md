# Solana sweeps: state model

The sweep path derives every decision from the database and the chain, never from memory.
One reconciler pass per in-flight sweep (`SolanaSweepService.reconcileSweep`, run by `TrackConfirmations`) reads the tracked attempts, the deposit links, the sweep transactions and the finalized token balances, and applies exactly one of the transitions below.
A restarted process recovers every state by running the same pass; nothing is carried between passes in memory.

Tables:

- `sweeps` - the sweep, its status and `version`, bumped by every transition.
- `sweep_transactions` - one per token account: the claim (amount), the source account, the hot ATA.
- `solana_sweep_deposits` - the deposits a sweep claimed, unique on `(sweep_id, deposit_id)`.
- `solana_sweep_attempts` - every signature ever signed for a sweep, unique on `signature` and on `(sweep_id, attempt_no)`, with `blockhash`, `last_valid_block_height` and `created_at`.
- `solana_sweep_locks` - one row per token account with a sweep in flight, unique on `token_account`, hard-deleted.
- `solana_deposit_accounts` - the account status (`watching`, `expired`, `drained`, `closed`).

## Invariants

1. The claim is the rows: the sweep (`processing`), its sweep transactions, links and locks are written in one transaction, after the balances were read and before anything is signed.
   A deposit is part of an in-flight sweep exactly when a link of a `processing`/`pending` sweep names it; there is no separate claim to strand.
2. Every signature is persisted (attempt row with signature, blockhash and the real `last_valid_block_height`) before it is sent.
   A send can therefore never produce a signature the database does not know.
3. A transport error on send leaves the attempt tracked; only a JSON-RPC error that proves the node refused the transaction (preflight -32002, signature -32003, -32013, -32015, malformed -32600/-32601/-32602) marks it failed. Any other error, including -32603, is treated as transport.
4. One sweep in flight per token account, enforced by the unique index on `solana_sweep_locks.token_account`.
5. Deposits stay `confirmed` while their sweep is in flight and become `swept` only in `book`, through the links of the sweep whose attempt finalized; `failSweep` touches no deposit.
   A Solana deposit found `swept` with no link to a completed sweep (a claim from an older build) is returned to `confirmed` with `solana_orphaned_claim` after `ValidityWindow`.
6. No attempt is ever written with `last_valid_block_height = 0`.
   A transaction recovered from an account's history gets the finalized height at recovery plus the validity span (150 blocks), an upper bound that can only delay expiry evidence.
7. Every sweep transition is a compare-and-set on the `(status, version)` it was decided from, bumping `version`, in the same transaction as its other writes (`casSweep`): persisting an attempt, failing, recovering an attempt from history, reviving, booking.
   A worker that decided from a stale read changes nothing: a signer whose sweep another worker failed (or rebuilt) meanwhile cannot persist and so cannot send, and a worker failing an attempt-less sweep loses to a signer that persisted first.
   Attempts are also unique on `(sweep_id, attempt_no)`.

## Account states

| State | Meaning | Derived from |
| --- | --- | --- |
| `watching` / `expired` | Polled by the watcher; sweepable when it has confirmed deposits and no lock | account row |
| in flight | A sweep holds its lock | `solana_sweep_locks` row whose sweep is `processing` or `pending` |
| `drained` | Set aside for an operator: balance below its claim with nothing of ours explaining it (`solana_unexplained_drain`), drained by an untracked transaction of our fee payer with no sweep to attach it to (`solana_untracked_sweep`), or a revive that could not take the lock (`solana_revive_conflict`); never claimed or re-read again | account row |
| `closed` | A booked sweep closed the account on chain | account row, `getAccountInfo` returns nothing |

An account enters "in flight" only through `SweepConfirmed`, which inserts the lock in the same transaction as the sweep rows.
A second sweep for the account cannot be inserted while the lock exists, and `skipAccount` does not even try.
A lock whose sweep is no longer `processing`/`pending` is an orphan and is deleted at the start of `SweepConfirmed` and the end of `TrackConfirmations`.

## Sweep states and transitions

| State | Allowed transitions | Evidence that moves it |
| --- | --- | --- |
| `processing` (rows, links, locks; no attempt yet) | to `pending` when the first attempt row is persisted; to `failed` when no attempt exists past `ValidityWindow` and every account still holds its claim; to `pending` when an account is below its claim and its history holds a transaction our fee payer signed (recorded as the attempt); to `failed` with the account `drained` when it is below its claim and nothing of ours explains it | attempt rows; `getTokenAccountBalance` (finalized); `getSignaturesForAddress` and `getTransaction` fee payer |
| `pending` (at least one attempt) | to `completed` when any attempt is finalized; a new attempt when the latest is expired or rejected, every account holds its claim and the budget (`MaxAttempts`) allows; to `failed` when the budget is spent with every account holding its claim; waits while an account is below its claim and nothing is finalized; after `DrainWait` of that, the account's history must explain the drain (our signature: keep waiting for it to finalize) or the sweep fails and the account is `drained` | `getSignatureStatuses` for every attempt; finalized `getBlockHeight` past the attempt's `last_valid_block_height` and the signature absent at finalized on two distinct endpoints |
| `completed` | terminal | the finalized transaction of the landed attempt |
| `failed` | to `pending` when a later drain is explained by one of its signatures (revival, anomaly `solana_failed_sweep_landed`); the revive takes its accounts' locks in the same transaction, and when the lock holder is the sweep being reconciled (which has no live attempt: none, or all expired or failed) that sweep fails in the same transaction; any other holder sets the account aside | account history |

## Attempt states

| State | Meaning | Set by |
| --- | --- | --- |
| `signed` | Persisted; the send was not acknowledged (transport error, or a crash before the send) | `signAndSend`, before `sendTransaction` |
| `sent` | The node acknowledged the send | `signAndSend`, after `sendTransaction` |
| `landed` | Finalized and booked | `book` |
| `expired` | Evidenced expired, or superseded by the landed attempt | reconciler, `book` |
| `failed` | The node rejected it (JSON-RPC error) or it failed on chain | `signAndSend` / reconciler |

A `signed` attempt is tracked exactly like a `sent` one: its signature is known and a node may have forwarded it.
An attempt whose signature already exists (same message against the same blockhash) is not sent; the next pass signs against a fresh blockhash.

## Expiry evidence

An attempt is expired only when two distinct RPC endpoints each show both, from the same endpoint (`Client.ExpiredOnDistinctNodes`):

1. That endpoint's own finalized block height is past the attempt's `last_valid_block_height` (for a row with no height, which no current code writes: the attempt is also older than `ValidityWindow`).
2. That endpoint's `getTransaction` at finalized returns nothing.

A lagging endpoint therefore cannot evidence expiry for a transaction it has not finalized yet. The balance guard before a rebuild is read from the first evidencing endpoint.

With one endpoint the evidence is unavailable and the sweep waits with `solana_evidence_unavailable`.
Live boot refuses fewer than two endpoints, so this only happens while one is evicted.

## Recovery after a restart

| Found in the database | Chain lookups | Action |
| --- | --- | --- |
| `processing`, no attempt, younger than `ValidityWindow` | none | wait (a send may be in progress) |
| `processing`, no attempt, older | finalized balance per account | all at or above claim: `failSweep` with anomaly `solana_sweep_never_sent`; any below: history for a transaction our fee payer signed; found: record it as an attempt (`pending`); not found: account `drained`, anomaly, `failSweep` |
| `pending` | statuses of every attempt | as the `pending` row above |
| `signed` attempt | `getSignatureStatuses` | tracked like `sent`; expires only with the evidence above |
| `completed` / `failed` | none | nothing; a drain explained by a `failed` sweep's signature revives it |
| lock without an in-flight sweep | none | deleted |

Money beyond a claim (a deposit landing mid-sweep) is left in the account: the transfer is the claim, the account is not closed, and the next sweep, after the lock is released, takes the rest.
