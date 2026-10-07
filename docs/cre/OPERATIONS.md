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
| simulate | mock plus the CRE CLI and the workflow code (tickets 23 to 25) | the real workflows run locally with `cre workflow simulate`; records still carry `provider=mock`, `simulated=true` |
| `chainlink` | Chainlink Early Access, deployed workflows, a consumer contract, a signer-held trigger key | records read from your own RPC and verified before they are stored or shown |

## Mock, locally

```bash
CRE_ENABLED=true CRE_PROVIDER=mock \
CRE_READ_TOKEN_SOLVENCY=dev-solvency CRE_READ_TOKEN_DEPOSIT_FINALITY=dev-deposit CRE_READ_TOKEN_CONVERSION_REFERENCE=dev-conversion \
docker compose --profile cre up
```

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
5. **Credentials.** Generate one token per workflow (`openssl rand -hex 32`), set them as `CRE_READ_TOKEN_*`, and upload them to the Vault DON with `cre secrets create` under the names in `cre/secrets.yaml`.
   The workflow sends its token as `Authorization: Bearer` and can read only its own route.
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

Every record, mock or chainlink, passes `verify.go` (SPEC section 6): the report decodes to version 1; the workflow id and owner match the configuration for that kind; the gateway id is this deployment's; for `chainlink`, the log came from the consumer contract over your own RPC, the rebuilt report hashes to the `reportHash` the contract logged, and the block is `CRE_VERIFY_CONFIRMATIONS` behind the head; for `mock`, the dev-key signature verifies; the observation is younger than `CRE_MAX_REPORT_AGE` and strictly newer than the last accepted one for that kind; the payload hash was never recorded; and every subject is one the gateway asked about, with a deposit's token, amount and destination equal to what was credited.
A subject that fails its check is stored `failed` with the reason and raises `cre.attestation.failed.v1`; everything else is refused and stored nowhere.

## Turning it off

Set `CRE_ENABLED=false` and restart.
Routes, worker and badges disappear; the `cre_*` tables keep their rows and are not read.
The settlement policy line `require_attestation_above` (ticket 21b) refuses validation while the provider is `none`, so nothing can wait on a module that is off.

## Events and metrics

Events: `cre.attestation.recorded.v1`, `cre.attestation.failed.v1`, `cre.workflow.stale.v1` on the `ee_events` queue.
Metrics: `payminto_cre_runs_total{kind,provider,result}`, `payminto_cre_attestation_age_seconds{kind}`, `payminto_cre_verify_failures_total{reason}`, `payminto_cre_trigger_latency_seconds`.
Logs carry the workflow kind, execution id and transaction hash; never a token, a JWT or a key.
