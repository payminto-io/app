# Solana sweeps: state model

The sweep path derives every decision from the database and the chain, never from memory.
One reconciler pass per in-flight sweep (`SolanaSweepService.reconcileSweep`, run by `TrackConfirmations`) reads the tracked attempts, the deposit links, the sweep transactions and the finalized token balances, and applies exactly one of the transitions below.
A restarted process recovers every state by running the same pass; nothing is carried between passes in memory.

Tables:

- `sweeps` - the sweep and its status.
- `sweep_transactions` - one per token account: the claim (amount), the source account, the hot ATA.
- `solana_sweep_deposits` - the deposits a sweep claimed, unique on `(sweep_id, deposit_id)`.
- `solana_sweep_attempts` - every signature ever signed for a sweep, unique on `signature` and on `(sweep_id, attempt_no)`, with `blockhash`, `last_valid_block_height` and `created_at`.
- `solana_sweep_locks` - one row per token account with a sweep in flight, unique on `token_account`, hard-deleted.
- `solana_deposit_accounts` - the account status (`watching`, `expired`, `drained`, `closed`).

## Invariants

1. A sweep's rows, links and locks are written in one transaction before anything is signed.
2. Every signature is persisted (attempt row with signature, blockhash and the real `last_valid_block_height`) before it is sent.
   A send can therefore never produce a signature the database does not know.
3. A transport error on send leaves the attempt tracked; only a JSON-RPC rejection (`*RPCError`: the node answered and refused) marks it failed.
4. One sweep in flight per token account, enforced by the unique index on `solana_sweep_locks.token_account`.
5. Deposits become `swept` only in `book`, through the links of the sweep whose attempt finalized; `failSweep` returns exactly the sweep's own deposits to `confirmed`.
6. No attempt is ever written with `last_valid_block_height = 0`.
   A transaction recovered from an account's history gets the finalized height at recovery plus the validity span (150 blocks), an upper bound that can only delay expiry evidence.
7. An attempt is persisted only while its sweep is `processing` or `pending` (checked in the same transaction), and under the next `attempt_no`: a sweep the reconciler failed meanwhile is never revived by a late signer, and two workers rebuilding one sweep cannot both send.

## Account states

| State | Meaning | Derived from |
| --- | --- | --- |
| `watching` / `expired` | Polled by the watcher; sweepable when it has confirmed deposits and no lock | account row |
| in flight | A sweep holds its lock | `solana_sweep_locks` row whose sweep is `processing` or `pending` |
| `drained` | Balance below its claim with nothing of ours explaining it; anomaly `solana_unexplained_drain` written; never claimed again until an operator resolves it | account row |
| `closed` | A booked sweep closed the account on chain | account row, `getAccountInfo` returns nothing |

An account enters "in flight" only through `SweepConfirmed`, which inserts the lock in the same transaction as the sweep rows.
A second sweep for the account cannot be inserted while the lock exists, and `skipAccount` does not even claim its deposits.
A lock whose sweep is no longer `processing`/`pending` is an orphan and is deleted at the start of `SweepConfirmed` and the end of `TrackConfirmations`.

## Sweep states and transitions

| State | Allowed transitions | Evidence that moves it |
| --- | --- | --- |
| `processing` (rows, links, locks; no attempt yet) | to `pending` when the first attempt row is persisted; to `failed` when no attempt exists past `ValidityWindow` and every account still holds its claim; to `pending` when an account is below its claim and its history holds a transaction our fee payer signed (recorded as the attempt); to `failed` with the account `drained` when it is below its claim and nothing of ours explains it | attempt rows; `getTokenAccountBalance` (finalized); `getSignaturesForAddress` and `getTransaction` fee payer |
| `pending` (at least one attempt) | to `completed` when any attempt is finalized; a new attempt when the latest is expired or rejected, every account holds its claim and the budget (`MaxAttempts`) allows; to `failed` when the budget is spent with every account holding its claim; waits while an account is below its claim and nothing is finalized; after `DrainWait` of that, the account's history must explain the drain (our signature: keep waiting for it to finalize) or the sweep fails and the account is `drained` | `getSignatureStatuses` for every attempt; finalized `getBlockHeight` past the attempt's `last_valid_block_height` and the signature absent at finalized on two distinct endpoints |
| `completed` | terminal | the finalized transaction of the landed attempt |
| `failed` | to `pending` when a later drain is explained by one of its signatures (revival, anomaly `solana_failed_sweep_landed`) | account history |

## Attempt states

| State | Meaning | Set by |
| --- | --- | --- |
| `signed` | Persisted; the send was not acknowledged (transport error, or a crash before the send) | `signAndSend`, before `sendTransaction` |
| `sent` | The node acknowledged the send | `signAndSend`, after `sendTransaction` |
| `landed` | Finalized and booked | `book` |
| `expired` | Evidenced expired, superseded by the landed attempt, or closed out by `failSweep` | reconciler |
| `failed` | The node rejected it (JSON-RPC error) or it failed on chain | `signAndSend` / reconciler |

A `signed` attempt is tracked exactly like a `sent` one: its signature is known and a node may have forwarded it.
An attempt whose signature already exists (same message against the same blockhash) is not sent; the next pass signs against a fresh blockhash.

## Expiry evidence

An attempt is expired only when both hold:

1. The finalized block height is past its `last_valid_block_height` (for a row with no height, which no current code writes: the attempt is older than `ValidityWindow`).
2. `getTransaction` at finalized returns nothing on two distinct RPC endpoints.

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
