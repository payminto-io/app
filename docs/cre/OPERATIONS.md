# CRE attestations: turning the module on

The attestation module is off by default and the gateway is complete without it.
Turn it on only when you hold balances or settle at size and want a second, operator-independent signature on your numbers.
A direct-to-wallet merchant has nothing to attest; leave it off.
Design: `docs/cre/SPEC.md`. Code: `backend/internal/cre/README.md`.

## Tiers

| Tier | What you need | What you get |
| --- | --- | --- |
| `none` (default) | nothing | no routes, no worker, no tables read; the settings page says the module is off |
| `mock` | `CRE_ENABLED=true CRE_PROVIDER=mock` | records with `provider=mock`, signed by an in-process dev key; every badge says Mock; refused in live |
| simulate | the CRE CLI and the workflow code (tickets 23 to 25) writing through a mock forwarder, `CRE_PROVIDER=chainlink` with `CRE_FORWARDER_SIMULATED=true` | the real workflows run locally with `cre workflow simulate`; every record is `simulated=true` (the simulator's fixed identity, workflow id `0x11..11` and owner `0xaa..aa`, is recognised too) and is never presented as a production attestation |
| `chainlink` | Chainlink Early Access, deployed workflows, a consumer contract, a signer-held trigger key | records read from your own RPC and verified before they are stored or shown |

## Mock, locally

```bash
CRE_ENABLED=true CRE_PROVIDER=mock \
CRE_READ_TOKEN_SOLVENCY=$(openssl rand -hex 32) CRE_READ_TOKEN_DEPOSIT_FINALITY=$(openssl rand -hex 32) CRE_READ_TOKEN_CONVERSION_REFERENCE=$(openssl rand -hex 32) \
docker compose --profile cre up
```

Tokens are at least 32 characters; the gateway keeps only their SHA-256 and compares in constant time.
Until custody (ticket 08) supplies a reserve source, the mock refuses solvency runs rather than attesting reserves it never observed; deposit-finality and conversion-reference runs work.

The `cre` profile adds `cre-simulator`; the backend reads the same variables and starts the mock provider.
Without the variables, `docker compose up` is unchanged (`CRE_ENABLED` defaults to `false`).
The simulator idles until `cre/workflows/` exists; it never logs in, deploys or broadcasts.

Open `/dashboard/settings/attestations`: provider Mock, health ok, the three workflows and their last run and last record.
`POST /api/v1/cre/runs/solvency` (owner role) forces a run; the worker polls the mock every `CRE_POLL_INTERVAL`.

## Chainlink, step by step

Every step is yours. No agent or script in this repository logs in, deploys, creates secrets or spends funds.

1. **Access.** `cre account access` or app.chain.link/cre/request-access; 2FA on the account.
2. **Consumer contract.** Deploy `contracts/src/cre/GatewayAttestations.sol` (ticket 22) on the attestation chain with your multisig as owner, `setForwarder` to the chain's KeystoneForwarder, and `setWorkflow` for each kind with the workflow id, owner and name from `cre workflow deploy`.
3. **Workflows.** Deploy the three workflows (tickets 23 to 25) and activate them; note each 64-hex workflow id.
4. **Trigger key.** Create an EVM key with no funds and no custody role in the signer service and reference it as `CRE_TRIGGER_SIGNER=keyring://cre-trigger`.
   The API process never holds the key; config refuses a raw private key in that variable.
5. **Credentials.** Generate one token per workflow (`openssl rand -hex 32`, at least 32 characters), set them as `CRE_READ_TOKEN_*`, and upload them to the Vault DON with `cre secrets create` under the names in `cre/secrets.yaml`.
   The workflow sends its token as `Authorization: Bearer` and can read only its own route; the gateway stores only a hash.
6. **Configure the gateway.**

   ```
   CRE_ENABLED=true
   CRE_PROVIDER=chainlink
   CRE_CHAIN=ethereum-mainnet-base-1
   CRE_CHAIN_RPC_URL=https://<your own RPC>
   CRE_CONSUMER_ADDRESS=0x...
   CRE_FORWARDER_ADDRESS=0x...
   CRE_WORKFLOW_OWNER=0x...
   CRE_WORKFLOW_ID_SOLVENCY=...
   CRE_WORKFLOW_ID_DEPOSIT_FINALITY=...
   CRE_WORKFLOW_ID_CONVERSION_REFERENCE=...
   CRE_WORKFLOW_NAME_SOLVENCY=solvency                 # the workflow.yaml names; Keystone's name is derived
   CRE_WORKFLOW_NAME_DEPOSIT_FINALITY=deposit-finality
   CRE_WORKFLOW_NAME_CONVERSION_REFERENCE=conversion-reference
   CRE_VERIFY_CONFIRMATIONS=12                         # fallback when the RPC lacks the finalized tag; 0 is refused in live
   CRE_TRIGGER_SIGNER=keyring://cre-trigger
   CRE_PUBLIC_BASE_URL=https://pay.example.com
   CRE_READ_TOKEN_SOLVENCY=...
   CRE_READ_TOKEN_DEPOSIT_FINALITY=...
   CRE_READ_TOKEN_CONVERSION_REFERENCE=...
   ```

   In staging and production a missing key refuses to boot and names the key.
   In development a missing key degrades to `mock` and the settings page shows what was missing.
7. **Verify.** The status page shows health `ok`, the signer address, and after the first cron run a record per workflow with its transaction link.
   `GET /api/v1/public/attestations/<id>` renders the stored row and says `independently_signed: true` only for `chainlink` records.

## What the gateway checks before it stores a record

Every record, mock or chainlink, passes `verify.go` and mirrors the audited contract (SPEC section 5 and 6):

- the report decodes to version 1 with the exact byte length the contract requires (padding is malformed);
- the Keystone metadata `(workflowId, workflowOwner, workflowName)` equals the configured binding for that kind, the name being Keystone's truncation (`sha256`, first ten hex characters) of `CRE_WORKFLOW_NAME_*`;
- the gateway id is this deployment's;
- for `chainlink`: the `ReportAccepted` log came from the consumer contract over your own RPC, the report bytes are taken from the forwarder transaction's calldata (the receiver slice, `rawReport[109:]`) and hash to the `reportHash` the contract logged, and the log sits at or below the finality bound (the RPC's finalized tag, or latest minus `CRE_VERIFY_CONFIRMATIONS`); a log above the bound is read again on the next poll, never refused;
- for `mock`: the dev-key signature verifies;
- the observation is not in the future; its age never blocks recording (a poller outage must not lose reports; staleness is a display rule);
- replay is the contract's rule: `keccak256(report)` recorded once per provider; a second delivery of the same bytes is `409 replayed`, and distinct reports are not ordered;
- every item is matched to a subject the gateway served: a deposit's token, amount and destination must equal what was credited; a solvency item's liabilities and decimals must equal the checkpoint's figures for that asset; a conversion's pair must be the trade's base/quote. A difference is stored as `mismatch` and raised as an anomaly, never shown as attested. A solvency item the contract recorded as superseded (`SolvencyIgnored`) is stored as `ignored`.

A record is `simulated` when its workflow identity is the CRE simulator's fixed identity or `CRE_FORWARDER_SIMULATED=true`; the dashboard labels it and the public page answers `independently_signed: false`. Live refuses both the flag and those identities at boot, and the verifier refuses such a report in live.
A report refused for a definitive reason (forged, wrong workflow, malformed) is kept as a `failed` row so an operator can see it; the poll cursor then passes it.
RPC and provider errors are sanitized (URLs and token-shaped strings stripped) before they reach health, the status page or a log line.

## Turning it off

Set `CRE_ENABLED=false` and restart.
Routes, worker and badges disappear; the `cre_*` tables keep their rows and are not read.
The settlement policy line `require_attestation_above` (ticket 21b) refuses validation while the provider is `none`, so nothing can wait on a module that is off.
Rows carry their provider: after switching from `mock` to `chainlink`, status and freshness read only `chainlink` rows and mock rows never block a chainlink report.

## Public reads

`GET /api/v1/cre/liabilities` without a credential serves the latest published checkpoint (`checkpoint_id`, `checkpoint_hash`, `taken_at`, `assets`) and answers `503 no_checkpoint` until the worker or an authenticated solvency read has published one; it never publishes.
The journal counter (`max_journal_id`) is served only under the solvency credential.
Every `/cre` and `/public` route sits behind the rate limiter (a no-op until Redis is configured).

## Events and metrics

Events: `cre.attestation.recorded.v1`, `cre.attestation.failed.v1`, `cre.workflow.stale.v1` on the `ee_events` queue.
Metrics: `payminto_cre_runs_total{kind,provider,result}`, `payminto_cre_attestation_age_seconds{kind}`, `payminto_cre_verify_failures_total{reason}`, `payminto_cre_trigger_latency_seconds`.
Logs carry the workflow kind, execution id and transaction hash; never a token, a JWT or a key.
