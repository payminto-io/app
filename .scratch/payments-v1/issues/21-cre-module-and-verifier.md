# 21 CRE module: port, service, none and mock providers, verifier, storage, routes

Status: ready-for-agent
Owner: Chainlink engineer (Fable)
Blocked by: 01, 11, 20

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
