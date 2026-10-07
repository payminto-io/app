# Fix round 3: tickets 20 and 21 (branch `cre-module`)

Inputs: `.superpowers/cre-module-rereview-2.md` (R1, R2, R3, F1) and the coordinator's rulings.
Each finding was reproduced by a failing test (or a failing build) before the fix.
No `cre` command ran other than `cre version` inside the freshly built simulator image (no `~/.cre` mount); nothing was deployed, logged into or paid for; Docker was not pruned.

## Commits

- `792044a` R1: solvency outcomes come from the contract's own events.
- `75e18d1` R2: checkpoint head is a commit watermark; the audit recipe reproduces the hash.
- `d1763f3` R3: simulator image installs the binary the v1.37.0 tarball ships.
- `2dcb7ad` F1: `TestTriggerJWT` tampers inside `r`, not the base64 tail.

## R1: duplicate assets mirror the contract exactly

- Red: the reviewer's probes as table cases in `TestVerify_DuplicateAssetInOneBatchMatchesTheContract` (compile-red against the old `Ignored` marker API; the reviewer had shown `mismatch/attested` and `failed/attested` on the old code).
- The poller now filters `SolvencyAttested` as well as `SolvencyIgnored` and hands the verifier `RawAttestation.Outcomes`: one `{asset, stored}` per item, in log order, collected between `ReportAccepted` events of the transaction.
- The verifier takes "ignored by the contract" only from those events (matched by position, asset checked); its own fact comparison is a separate field.
  Rows now carry `on_chain` (`stored` / `ignored` / `emitted`) and `fact_check` (`attested` / `mismatch` / `failed`); `status` is `ignored` whenever `on_chain` is, else `fact_check`.
  So an item the contract ignored is never attested, and a stored item whose facts differ is `mismatch`.
- Probe results now: `[USDC 101, USDC 100]` with events stored/ignored gives `mismatch/ignored`; unknown checkpoint first gives `failed/ignored`.
- Events that do not cover every item in order (none, short, reordered) are refused as `ErrForged` (`TestVerify_SolvencyOutcomesMustCoverEveryItem`).
- The mock provider plays the contract's per-asset rule (`cre.SimulateSolvencyOutcomes` with its own latest-observed map; `TestMockPlaysTheContractForSolvencyOutcomes`); a pushed mock report without outcomes gets the in-batch rule.
- `LogsForReport` takes ignored item indexes; the real-forwarder fixture test checks stored/stored and ignored/ignored.
- Schema: `on_chain` and `fact_check` added with `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (so dev databases converge), migration 2026100707 repeats it verbatim; store round trip asserted in `TestIntegration_SchemaConvergesAndStoreRoundTrips` (red first). API JSON and the dashboard type gain both fields (additive).

## R2: commit watermark and a reproducible audit recipe

- Red: `TestIntegration_CheckpointHashReproducesFromTheAuditRecipe`. Transaction A takes a journal id and stays open, B posts a higher id and commits, the checkpoint starts, A commits 300 ms later. Old code: `recipe USDC.SOLANA 15000000, checkpoint 10000000 (head 8)`.
- Fix (`internal/ledger/liabilities.go`, `service.go`): every `PostIn` takes `pg_advisory_xact_lock_shared(hashtextextended('ledger:posting_barrier:' || current_schema(), 0))` before the journal insert allocates an id. `LiabilityTotals` takes that lock exclusively (session lock on a pinned connection; the connection is discarded unless the unlock is confirmed), then in one `REPEATABLE READ` read-only transaction reads `MAX(id)` and `statement_timestamp()` in the statement that freezes the snapshot, and the sums bounded by it.
  Every journal at or below the head is therefore committed and counted.
  An advisory lock, not `LOCK TABLE`, because the hardened app role holds only SELECT/INSERT on the ledger tables.
- The sum is an inner join (an account opened after the snapshot with no counted lines cannot add an asset), ordered `COLLATE "C"`.
- `LiabilitySource` returns a `LedgerSnapshot{Totals, Head, TakenAt}`; the checkpoint's `taken_at` is the snapshot's database time, and `cre_subjects.facts` now stores `taken_at` (as hashed) beside `max_journal_id`.
- `docs/cre/OPERATIONS.md` "The checkpoint watermark" rewritten; new "Recomputing a checkpoint hash" section with two SQL blocks, scaling rules and the exact canonical JSON. The integration test extracts and runs both SQL blocks verbatim, for the held-transaction case and for 8 checkpoints taken while 6 workers post concurrently; all reproduce the hash. Ran green 3 times in a row.

## R3: simulator image builds

- Red: `docker build` of `docker/cre-simulator/Dockerfile` failed at `test -n "$bin"`.
- Fix: install exactly `/opt/cre/cre_${CRE_CLI_VERSION}_linux_${arch}` as `/usr/local/bin/cre` (fail with a listing if absent), pinned version and SHA-256 check kept, extraction dir removed.
- Proof: `docker build -t gateway-cre-simulator-r3:test docker/cre-simulator` succeeded (arm64); `cre version` in the image printed `CRE CLI version v1.37.0`. The amd64 tarball was downloaded separately: SHA-256 matches the pinned value and it contains `cre_v1.37.0_linux_amd64` (amd64 not built: no emulation on this host).
- Cleanup: not completed. `docker rmi gateway-cre-simulator-r3:test` (the only Docker removal command run) hangs on this Colima daemon: three attempts over about 30 minutes never returned, while `docker version`, `docker ps` and the testcontainers integration run worked; no container uses the image. The tag `gateway-cre-simulator-r3:test` (about 200 MB of CLI on `oven/bun:1-debian`) is still present and should be removed with that same command once the daemon responds. Nothing else was removed or pruned.

## F1: flaky `TestTriggerJWT`

- Cause: randomness. Overwriting the last two base64url characters with `AA` sets only `v` and the two low bits of `s` to zero; with a fresh random key that is already the case about 1 in 8 runs, so the "tampered" token was the original.
- Red: `go test -run TestTriggerJWT -count=300 ./internal/cre/chainlink/` gave 45 failures.
- Fix: decode the signature, flip a bit of `r`, re-encode. Same command: 300 of 300 pass.

## Checks

- `cd backend && go build ./... && go vet ./... && go test ./...`: green, 30 packages ok.
- Integration (`-tags=integration -count=1 -p 1 -timeout 45m ./internal/cre/... ./internal/ledger/... ./internal/api/...`, Colima): green: `cre` 86 s, `cre/chainlink`, `cre/mock`, `cre/none`, `ledger` 228 s, `api`, `api/dto`, `api/handler`, `api/middleware`, `api/paymentlifecycle/v1` ok (exit 0).
- Frontend: `npx tsc --noEmit` clean, `npm run lint` clean, `npm test` 33 passed in 7 files.

## Not addressed this round

The reviewer's minors M1 to M5 were not in the rulings and are unchanged.
