# cre

Optional slot module `attestation`: an operator-independent signature on numbers the gateway already holds, obtained through Chainlink CRE workflows, verified by the gateway before it is stored or shown.
It never computes a balance, moves money or decides a settlement.
Design: `docs/cre/SPEC.md`. Turning it on: `docs/cre/OPERATIONS.md`. Tickets 20 and 21.

## Port (`port.go`)

- `Attester`: `Name`, `Trigger(ctx, kind, input) (executionID, error)`, `Poll(ctx, kind, cursor) ([]RawAttestation, Cursor, error)`, `Health(ctx)`.
- `RawAttestation`: the forwarder metadata bytes, the report bytes and `Evidence` (own-RPC log: emitter, tx, block, head, `reportHash`; or the mock dev-key signature).
- `Attestation`: one verified (or refused) row per report item.
- Facts the module reads, each through a narrow port with an honest empty default: `LiabilitySource` (the ledger), `ReserveSource` (custody, ticket 08), `DepositSource` (the switch), `ConversionSource`.
- `SettlementGate` with `NoopGate`; ticket 21b ships the attestation-aware gate.
- Events: `cre.attestation.recorded.v1`, `cre.attestation.failed.v1`, `cre.workflow.stale.v1` through `EventSink`.

## Providers

| Provider | Folder | Trigger | Poll | Evidence |
| --- | --- | --- | --- | --- |
| `none` | `none/` | `ErrDisabled` | nothing | none |
| `mock` | `mock/` | plays the DON: builds the report from the same JSON the routes serve, signs with an in-process dev key, queues it; scriptable delays, failures and verdicts | the queue | 65-byte secp256k1 signature over `keccak256(metadata || report)` |
| `chainlink` | `chainlink/` | `workflows.execute` over the CRE gateway with an `alg: ETH` JWT (digest of the key-sorted body, iss, iat, exp <= 5 min, jti) signed by the signer service through a key reference | `ReportAccepted` plus item events from the consumer contract over `CRE_CHAIN_RPC_URL`, rebuilt into the report | emitter, tx, block, head, the contract's `reportHash` |

`conformance/` is the suite every provider passes (`none_test.go`, `mock_test.go`, `chainlink_test.go` with a fake chain and a fake CRE gateway).

## Wire format (`codec.go`, `metadata.go`)

`abi.encode(uint8 version=1, uint8 kind, bytes32 gatewayId, uint64 observedAt, Item[] items)`; items per kind as in `contracts/test/cre/ReportEncoder.sol`.
Metadata is the 64-byte Keystone header `workflowId(32) | workflowName(10) | owner(20) | reportId(2)`.
`gatewayId = keccak256(CRE_PUBLIC_BASE_URL)`; subject keys are `keccak256(id)`; short labels (asset, chain, token, pair) are right-padded bytes32.
The consumer ABI is embedded from `abi/GatewayAttestations.json` (byte-identical with the contract track).

## Verifier (`verify.go`)

In order: version and metadata decode; workflow id for the kind; owner; gateway id; evidence (chainlink: tx present, emitter is the consumer, rebuilt report hashes to `reportHash`, confirmations; mock: signature recovers to the dev key); age under `CRE_MAX_REPORT_AGE` and not in the future; payload hash never recorded; `observedAt` strictly newer than the last accepted for the kind; then per item the subject lookup.
A subject the gateway never asked about, a non-confirmed verdict, or a deposit whose token, amount or destination differ from what was credited becomes a `failed` row with a reason; whole-report failures store nothing.

## Service (`service.go`, `worker.go`)

- Pull inputs: `Liabilities` (publishes a checkpoint when none is younger than the interval), `PendingDeposits(limit<=12)`, `Conversions(since, limit<=10)`; each remembers the subjects it served in `cre_subjects`.
- `Submit(raw)`: verify, store, emit. `Poll(kind)`: provider cursor to `Submit`. `Run(kind)`: build the input and `Trigger`.
- `Status`: provider, degradation, environment (from the environment module's guard), addresses, signer address, health, per-workflow last run, last record and `never|fresh|stale` (stale after two intervals).
- `Worker()`: nil when off; otherwise polls every `CRE_POLL_INTERVAL`, refreshes the checkpoint every `CRE_SOLVENCY_INTERVAL`, triggers the mock on its schedules (deployed workflows pull on their own cron), sweeps staleness.
- Credentials: per-workflow bearer tokens compared as SHA-256 in constant time; an unset token refuses that workflow's routes.

## Tables (`schema.sql`, migration `2026100707_cre_attestations`)

`cre_attestations` (one row per item; unique `(payload_hash, item_index)`; indexes `(kind, subject_id)`, `(status, recorded_at)`, `(kind, recorded_at desc)`), `cre_subjects` (what was served, first facts win), `cre_runs`, `cre_cursors`.
`Migrate` installs them in dev/test; production applies the checksummed migration.
With the provider `none` nothing reads them.

## HTTP (`internal/api/routes_cre.go`, mounted only when enabled)

| Route | Auth | Body or query |
| --- | --- | --- |
| `GET /api/v1/cre/liabilities` | solvency token, or none when `CRE_PUBLIC_VERIFY_ENABLED` | `{checkpoint_id, checkpoint_hash, taken_at, max_journal_id, assets:[{asset, liabilities_minor, decimals}]}` |
| `GET /api/v1/cre/pending-deposits?limit=` | deposit-finality token | `{deposits:[{deposit_id, chain, tx, log_index_or_signature, token, expected_amount_minor, destination}]}` |
| `GET /api/v1/cre/conversions?since=&limit=` | conversion-reference token | `{conversions:[{conversion_id, executed_at, base, quote, executed_rate_decimal, amount_minor}], next_since}` |
| `POST /api/v1/cre/reports` | token of the body's `kind` | `{kind, metadata, report, signature, tx_hash, execution_id, simulated}`; mock verifies the signature; chainlink ignores the bytes and polls its own RPC; `202 {attestation_ids, recorded, attested}`, `409 replayed`, `422 report_rejected` |
| `GET /api/v1/cre/status` | dashboard session | status report |
| `GET /api/v1/cre/attestations?kind=&limit=`, `/:id` | dashboard session | rows |
| `POST /api/v1/cre/runs/:kind` | session + `system.admin` | `202` run or `502` failed run |
| `GET /api/v1/public/attestations/:id` | none, when `CRE_PUBLIC_VERIFY_ENABLED` | row plus consumer, forwarder, gateway id, `independently_signed` |

Errors are `{error, code}`.

## Configuration

`config.CREConfig`, keys in `.env.example`; rules in `config/cre_config.go`: `CRE_ENABLED=false` forces `none`; `mock` refuses staging and production and `GATEWAY_ENVIRONMENT=live` (the `cre` slot carries the resolved provider while enabled); `chainlink` with missing keys refuses in live and degrades to `mock` elsewhere naming the keys; `CRE_TRIGGER_SIGNER` must be a reference, never a key.

## Tests

Unit next to the code (codec, metadata, verifier with valid, forged, replayed, wrong workflow, wrong owner, wrong gateway, wrong emitter, unconfirmed, stale, future, unknown and mismatched subjects, report hash, ordering; checkpoint; decimals; store double), provider tests and conformance, `modules/cre_test.go` (none is a no-op, mock refused in live, round trip, no key material), `api/routes_cre_test.go` (route table unchanged when off, credentials scoped per workflow, submit, replay, forgery, status, run), and `integration_test.go` (`-tags=integration`: schema convergence, Postgres store, ledger liabilities to checkpoint to mock attestation).
