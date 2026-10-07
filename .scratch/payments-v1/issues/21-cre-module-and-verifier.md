# 21 CRE module: port, service, none and mock providers, verifier, storage, routes

Status: done
Owner: Chainlink engineer (Fable)
Blocked by: 01, 20

## Goal
`backend/internal/cre/` per `docs/cre/SPEC.md` sections 3, 4, 6, 7 and 9.
- `port.go`: `Attester`, `Attestation`, `RawAttestation`, `Cursor`, `Health`, errors.
- `service.go`: liabilities snapshot from the ledger checkpoint (ledger port only), pending-deposit batch from the switch port, conversion batch from the conversion port, poller, staleness sweep, events `cre.attestation.recorded.v1`, `cre.attestation.failed.v1`, `cre.workflow.stale.v1`.
- `verify.go`: the six checks in section 6, including metadata decode (workflow id, name hash, owner, report id) and the confirmations rule.
- Providers `none/` and `mock/` (dev signing key, scripted delays and failures, recorded logs or a local Anvil for the verifier path).
- `chainlink/`: HTTP trigger client building the `workflows.execute` JSON-RPC body and the `alg: ETH` JWT (digest of the sorted body, iss, iat, exp <= 5 min, jti) with the signature obtained from the signer service by key reference; consumer-contract event reader over `CRE_CHAIN_RPC_URL`.
- Migration `cre_attestations` (additive) with indexes on `(kind, subject_id)` and `(status, recorded_at)`.
- `routes_cre.go`: the read endpoints in section 7 with `CRE_READ_TOKEN` bearer where specified, the owner-role force-run endpoint, and the public read behind `CRE_PUBLIC_VERIFY_ENABLED`.
- `modules/cre.go` wiring and one registry line.
- Settlement integration: `require_attestation_above` policy line (validator rejects it when provider is `none`), `awaiting_attestation` state, freeze on verdict 2 or 3, owner release with hash-chained audit entry.
- `conformance/`: suite run against `none`, `mock` and `chainlink` (recorded fixtures).

## Acceptance
Unit tests for the verifier reject: wrong emitter, wrong forwarder path (contract), wrong owner, wrong workflow id, wrong gateway id, unknown checkpoint, mismatched deposit amount, insufficient confirmations. Integration test: a settlement above threshold waits, proceeds on verdict 1, freezes on verdict 3; with provider `none` the policy line fails validation and nothing waits. The API process has no CRE signing key material (test: config holds only a key reference). `make test-integration` green.


## Comments

Ruling 2026-10-07 (coordinator): unblocked from 11 by moving the settlement gate (`require_attestation_above` enforcement inside settlement) to ticket 21b, blocked by 11. This ticket ships the module, providers none/mock/chainlink, verifier, storage, routes and a `SettlementGate` port with a no-op default.

Done 2026-10-07 (Chainlink engineer, branch `cre-module`): commits c6088c3 (port, codec, verifier), 06a8dd4 (storage, service, providers, conformance), e6f5c8e (merge main, cre as an environment slot), c5346ea (wiring, routes, worker, migration, ledger read), fd599f5 (contract track encoding, ReportAccepted reader, item index). Module per SPEC under `backend/internal/cre/` (README there): port, service, verifier with the six checks plus report-hash and strictly-newer `observedAt` from the contract track, storage (`2026100707_cre_attestations`, one row per report item), providers none/mock/chainlink, conformance suite, routes with per-workflow bearer credentials and payload-hash replay refusal, `SettlementGate` with `NoopGate`, `PolicyMayRequireAttestation()` for 21b. Encoding and ABI follow ticket 22 (`abi/GatewayAttestations.json` byte-identical with `cre-contract`). Signer is a port (`chainlink.Signer`) resolved by key reference; the API process holds no key (config refuses a raw key). Settlement integration moved to 21b per the ruling. Not in this ticket: the signer service implementation (defaults to `UnavailableSigner`, health degrades), custody reserves (`ReserveSource` port, custody implements on merge), switch and conversion sources (ports with empty defaults).
