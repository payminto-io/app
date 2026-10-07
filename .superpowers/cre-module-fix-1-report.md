# Tickets 20 and 21, fix round 1

Branch `cre-module`, merged with main at d059967 (audited contract, ticket 22). Review: `.superpowers/cre-module-review.md`. Date: 2026-10-07.
Commits: d059967 (merge main, main's ABI taken), c6e7fa0 (the fixes), then the simulator identity rule and this report (see git log).

## Critical

### C1 lossy poll loop
- `cre.Service.Poll` sorts raws by block and log index and submits in order. The cursor advances only past blocks whose reports were recorded or definitively refused (`IsRejection`: forged, wrong workflow/owner/name/gateway/emitter, malformed, replayed). `ErrUnconfirmed`, `ErrNotFinal` (provider or RPC error), store errors and anything else retryable stop the cursor at that block (`service.go` `Poll`).
- The reader's upper bound is the finality bound: `EthLogReader.FinalizedHead` prefers the `finalized` tag and falls back to `latest - CRE_VERIFY_CONFIRMATIONS` only when confirmations are above zero, logging the fallback once; with zero it returns `ErrNoFinality` and nothing is read until the tag works. Every returned log carries `Evidence.Final`; the verifier refuses a non-final log as `ErrUnconfirmed` (retryable).
- Age no longer blocks recording: `CRE_MAX_REPORT_AGE` and the verifier's age check are gone; staleness is `never|fresh|stale` on the status page only. The future check (5 min) stays, like the contract.
- A definitively refused on-chain report is kept as a `failed` row (`subject_type=report`, sanitized reason) so operators see it.
- Tests: `service_test.go` `TestPollCursorWaitsForFinality` (polls twice around the bound, record on the second pass, cursor stops at 105 then moves to 111), `TestPollCursorOnErrorsAndRefusals` (RPC error leaves the cursor; forged report stored failed and passed), `ethreader_test.go` for the tag, the refused zero-confirmation fallback and `latest - n`; `config` `TestCRE_LiveConfirmationsDefaultAndZeroRefused` (default 12 in staging/production, 0 refused).

### C2 contract rules
- Replay is `keccak256(report)` per provider (`Store.Seen(provider, hash)`); the per-kind `LatestObservedAt` gate, the mock's strictly-increasing clock and the docs that described them are removed. `TestVerify_NoOrderingBetweenDistinctReports` records two distinct batches with the same `observedAt` and refuses the same bytes.
- Report bytes come from the forwarder calldata: `chainlink.DecodeForwarderCall` decodes `KeystoneForwarder.report(address,bytes,bytes,bytes[])`, splits `rawReport[45:109]` (metadata) and `rawReport[109:]` (report); the verifier requires `keccak256(report) == ReportAccepted.reportHash`. `RebuildReports` and the `ItemCount == len(items)` fragility are gone. `SolvencyIgnored` logs are read and their assets become rows with status `ignored` (recorded-but-superseded, never attested).
- Bindings: `Verifier.Bindings[kind] = {ID, Owner, Name}` from `CRE_WORKFLOW_ID_*`, `CRE_WORKFLOW_OWNER_*` (fallback `CRE_WORKFLOW_OWNER`) and `CRE_WORKFLOW_NAME_*` (defaults `solvency`, `deposit-finality`, `conversion-reference`). `cre.KeystoneName` is HashTruncateName in Go, pinned to `my_workflow -> 0x62373666336165316465` and to `solvency -> "58c66935b7"` (same vectors as `WorkflowName.t.sol`). The mock writes Keystone names (`Provider.Binding(kind)`). `ErrWrongName` is tested, and the real-forwarder fixture test shows the audit's wrong-derivation binding refusing a real delivery.
- `DecodeReport` mirrors `_decodeHeader`: canonical items offset, non-zero count, exact byte length (`TestVerify_PaddedReportIsMalformed`, codec test with a padded report).
- ABI: main's `backend/internal/cre/abi/GatewayAttestations.json`; `abi_test.go` compares it semantically with `contracts/out/.../GatewayAttestations.json` when built (it is, and matches). Main's `frontend/lib/cre/abi.ts` had a syntax error (`]\n as const`) that broke `tsc`; fixed in place.
- Real-contract fixture: `contracts/test/cre/GatewayAttestations.fixture.t.sol` delivers two solvency reports through the vendored real `KeystoneForwarder` (the second older, so both items are `SolvencyIgnored`) and writes logs plus calldata to `contracts/test/cre/fixtures/forwarder_logs.json` (`fs_permissions` added). `chainlink/fixture_test.go` feeds it to the reader: both report hashes match, calldata slices hash to them, delivery 1 attests, delivery 2 is `ignored`, the forwarder's own `ReportProcessed` log is filtered by address.

## Important

- I1: `checkSolvency` requires the checkpoint's liabilities and decimals for that asset (asset absent from the checkpoint is a mismatch; reserves are not compared); `checkConversion` requires the pair to equal `base/quote`; deposits as before. A difference is `StatusMismatch` with the reason, raised as `cre.attestation.failed.v1` and an error-level anomaly log; never `attested`. `TestVerify_SolvencyAndConversionFactsMustMatchWhatWasServed` covers attested, ignored, wrong liabilities, wrong decimals, unknown asset, unknown checkpoint, and a wrong pair.
- I2: `ledger.LiabilityTotals` runs one `REPEATABLE READ` transaction: head first, then the sum joined on `ledger_lines.journal_id <= head`. The integration test posts after the checkpoint and checks the head and totals move together. `BuildCheckpoint` returns anomalies (unknown decimals, negative total) and omits those assets; the clamp is gone (`checkpoint_test.go`).
- I3: `cre.Sanitize` strips URLs and token-shaped strings; applied to provider health, trigger and dial errors, poll errors, run detail, status JSON and log lines. `sanitize_test.go` uses a URL with a path secret and a `sk_live_` token; `TestPollCalldataGuards` and `TestPollCursorOnErrorsAndRefusals` assert an RPC key never reaches the error text.
- I4: `Liabilities(ctx, authenticated)`: a public read serves the latest stored checkpoint only and answers `503 no_checkpoint` when none exists; only the solvency credential (or the worker) publishes. `max_journal_id` is in the hashed facts and served only under the credential. `RouterConfig.RateLimit` (the existing Redis limiter, wired from the registry's client, a no-op without Redis) wraps every `/cre` and `/public/attestations` route; the route test asserts the limiter ran on both.
- I5: `LatestAttestation(provider, kind, status)`; freshness derives from the latest `attested` row (`LastVerified`), the status page also shows the latest row of any status; `Seen`, lists and status are scoped to the active provider (integration test: a mock row is not seen for chainlink). `TestStatusFreshnessUsesVerifiedRowsOnly`.

## Minor

- M1 exact length (above). M2 `SaveAttestations` returns inserted rows (`ON CONFLICT DO NOTHING ... RETURNING id`); `Submit` returns `ErrReplayed` when nothing was inserted, so the race answers `409` and emits nothing (`TestSubmitReportsOnlyInsertedRows`, integration duplicate-insert check). M3 the mock has no default reserve source: without one it refuses solvency with `ErrNoReserveSource`, and an asset with no observed reserve is left out, never zeroed (`TestMockRefusesSolvencyWithoutReserves`); the docs say so. M4 tokens must be at least 32 characters (`TestCRE_ShortTokenRefusedAndNamesDefault`); docs state the hash-only comparison. M5 `POST /cre/reports/:kind`: the credential is checked from the path before any body is read; body cap 64 KiB (route test: an unauthenticated 100 KiB body is `401`, an authenticated one `400`). M6 `LogsForReport` emits item events first and `ReportAccepted` last, supports ignored assets; tests cover ignored items, a `Removed` (reorged) log, two reports in one block, the finality fallback, and the Foundry-recorded real-forwarder logs; the lossy rebuild test is gone. M7 covered under C1. M8 the conversions handler uses `Service.Now()`. M9 frontend tests: `lib/api/attestations.test.ts` (404 means off, other errors rethrown, explorer links only for known chains), `lib/copy/attestations.test.ts` (two-sentence explanation, labels), `page.test.tsx` (off state, a mismatch row never shown as attested or fresh, empty addresses omitted, error state).

## Simulator identity (coordinator addition from ticket 23)

- `cre.SimulatorWorkflowID` (0x11..11), `cre.SimulatorOwner` (0xaa..aa), `IsSimulatorIdentity`, `IsSimulatorBinding`. The verifier marks a record `simulated` when the metadata carries that identity, when `CRE_FORWARDER_SIMULATED=true` (new flag, `Verifier.SimulatedForwarder`), or when the caller said so; in live (`Verifier.Live`, from the environment module) such a report is refused with `ErrSimulated` (a definitive rejection).
- Config refuses `CRE_FORWARDER_SIMULATED=true`, the simulator workflow id in `CRE_WORKFLOW_ID_*` and the simulator owner in `CRE_WORKFLOW_OWNER*` in staging and production; `WireCRE` refuses a simulator binding under `GATEWAY_ENVIRONMENT=live`. Development accepts them (the simulate tier).
- Presentation: the public page's `independently_signed` is true only for a non-simulated chainlink record; status carries `forwarder_simulated`; the dashboard shows a notice for a simulation forwarder and a "Simulated" badge on a simulated record.
- Tests: `TestVerify_SimulatorIdentityIsSimulatedAndRefusedInLive`, `config.TestCRE_LiveRefusesSimulatorIdentities`, the public route test (`simulated: true`, `independently_signed: false` for mock), and the page test for the simulated labels.

## Also

- `.env.example`, `docs/cre/OPERATIONS.md`, `backend/internal/cre/README.md` updated for the new rules and keys; `CRE_MAX_REPORT_AGE` removed everywhere.
- Frontend: statuses `mismatch` (bad) and `ignored` ("Superseded", mute) in the status map; the workflows table shows "Last verified" with the latest unverified row's status beside it; `last_verified` and `workflow_name` in the API type and preview fixtures.

## Checks

- Backend: `go build ./... && go vet ./... && go test ./...` green; `forge test` in `contracts/` 81 passed (including the new fixture test).
- Integration (Colima Docker, `./internal/cre/... ./internal/api/... ./internal/database/ ./internal/ledger/`): see the line below.
- Frontend: `npx tsc --noEmit`, `npm run lint`, `npm test` (33 tests in 7 files) green.
- Integration result: `chainlink`, `mock`, `none`, `api` (all packages), `database` and `ledger` green in the combined run; `internal/cre` failed once on a testcontainer start timeout (four Postgres containers starting at once, `wait until ready` matched 1 of 2) and passed on a rerun with `-p 1` (13.7 s). No test assertion failed.

## Not done or worth knowing

- The rate limiter is wired but a no-op until Redis is configured for the registry (`main.go` passes nil today, as before this ticket).
- The Foundry fixture is committed; regenerate it with `forge test --match-contract GatewayAttestationsFixtureTest` after a contract change and the Go test will tell if the reader drifted.
- `frontend/lib/cre/abi.ts` from main did not compile (`]\n as const`); fixed here with a one-line edit.
