# 20 CRE install-time option, configuration and dashboard settings

Status: done
Owner: DevOps (Opus) with Frontend engineer (Opus)
Blocked by: 19

## Goal
Make CRE an install-time choice that changes nothing unless chosen. Spec: `docs/cre/SPEC.md` sections 2 and 12.
- `backend/internal/config`: the `CRE_*` section (`CRE_ENABLED` default false, `CRE_PROVIDER` none|mock|chainlink, chain, RPC, consumer, forwarder, owner, gateway URL, workflow ids, `CRE_TRIGGER_SIGNER` reference, intervals, `CRE_PUBLIC_VERIFY_ENABLED`). `CRE_ENABLED=false` forces `none`. `mock` refuses to boot in live. `chainlink` with missing keys refuses in live and degrades to `mock` elsewhere, naming the missing keys.
- `docker-compose.yml`: a `cre` profile that adds a `cre-simulator` service and sets `CRE_PROVIDER=mock` on the backend; the default stack is untouched.
- `.env.example` entries with one-line comments; `docs/OPERATIONS.md` gains a CRE section.
- `GET /api/v1/cre/status` returning provider, chain, consumer, forwarder, owner, and per-workflow last run and last attestation (empty when `none`).
- Dashboard `/dashboard/settings/attestations`: provider state, status fields, and the explanatory copy for `none` (who this is for, and that direct-to-wallet merchants have nothing to attest). Copy in `frontend/lib/copy/attestations.ts`.

## Acceptance
Config tests for every rule above. Boot with no `CRE_*` set is byte-identical in routes and migrations applied (test compares the route table). `docker compose up` unchanged; `docker compose --profile cre up` starts the extra service. Screenshots of the settings page at 390 and 1440, light and dark, in `none` and `mock`.


## Comments

Ruling 2026-10-07 (coordinator): unblocked from 15. Add the `cre` profile to the existing compose files now; ticket 15 folds it into the new compose and CI.

Done 2026-10-07 (Chainlink engineer, branch `cre-module`): commits eda090a (config rules), fd599f5 (compose profile, .env.example, docs), 8f3f23c (settings page, preview, screenshots). `CRE_*` section in `config/cre_config.go` with every rule tested; `cre` registered in `environment.KnownSlots` and the slot carries the resolved provider only while enabled, so `GATEWAY_ENVIRONMENT=live` refuses mock through the same gate as other slots and an installation with CRE off is unchanged (`api.TestRouterUnchangedWhenCREIsOff` compares the route table and asserts no worker). `cre` profile in both compose files with `docker/cre-simulator` (runs `cre workflow simulate` only; idles until tickets 23 to 25 add workflows); `.env.example`; `docs/cre/OPERATIONS.md` and a section in `docs/OPERATIONS.md`. `GET /api/v1/cre/status` exists only when enabled; the dashboard treats 404 as off. Settings page `/dashboard/settings/attestations` with copy in `frontend/lib/copy/attestations.ts`; screenshots `docs/design/screens/pages/settings-attestations-{,off-}{390,1440}-{light,dark}.png`. Deviation: migrations are a fixed manifest, so the empty `cre_*` tables exist when the module is off; nothing reads them.
