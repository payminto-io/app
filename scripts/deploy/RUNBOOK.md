# Payminto — payminto.io deployment runbook

This host (EC2, `54.234.10.215`) serves all three public surfaces from one
nginx instance in front of four loopback processes.

| Hostname | Serves | Upstream |
| --- | --- | --- |
| `payminto.io`, `www.payminto.io` | marketing site (`landing/`) | `127.0.0.1:3001` |
| `app.payminto.io` | merchant dashboard (`frontend/`) | `127.0.0.1:3003` |
| `app.payminto.io/api/v1`, `/api/v2` | gateway API (`backend/`) | `127.0.0.1:8090` |
| `checkout.payminto.io` | hosted checkout (`checkout/`) | `127.0.0.1:3002` |

`/api/auth/*` on the dashboard is served by Next itself (it sets the
`payminto_rt` refresh cookie); only `/api/v1` and `/api/v2` are proxied to Go.
The checkout never calls the gateway from the browser — it proxies server-side
through `PAYMINTO_API_URL`, so the SSE stream at
`/api/payment/:reference/events` is a Next route.

## Runtime

Four systemd units, all enabled at boot, logging to `/var/log/payminto/`
(rotated daily, 14 days):

```sh
systemctl status payminto-api payminto-app payminto-checkout payminto-landing
sudo systemctl restart payminto-api
tail -f /var/log/payminto/api.log
```

Unit and nginx sources are mirrored in `systemd/` and `nginx/` beside this file.

## Configuration

- `backend/.env` (mode 600, gitignored) — the API's environment. `SERVER=staging`,
  `GATEWAY_ENVIRONMENT=test`, `POSTGRES_SCHEMA_MODE=validate`.
- `frontend/.env.production`, `checkout/.env.production`, `landing/.env.production` —
  build- and run-time values for each Next app. `NEXT_PUBLIC_*` values are baked
  in at build time, so **changing one requires a rebuild**, not just a restart.
- `backend/.env.migrate` (mode 600) — the privileged migration role's overrides.
  Sourced *after* `.env` only when running `cmd/migrate`; never by the server.

## Public URLs the backend mints

`CHECKOUT_BASE_URL` must be set in `backend/.env`. A published payment link's
URL is `<base>/l/<short code>`, and the variable defaults to
`http://localhost:3002`, so leaving it unset makes every link a merchant
shares unusable while the API still answers 200. It is set to
`https://checkout.payminto.io` here.

## Taking a crypto payment end to end

`POST /api/v2/public/links/:code/pay` fails with `payment_creation_failed`
(502) until the merchant has a wallet for the chain family: the handler looks
up `wallets` by `(member_id, blockchain_family_id)` and a crypto payment has no
deposit address without one. Wallet derivation lives behind
`CUSTODY_ENABLED=true`, which additionally needs a `VAULT_PASSPHRASE` of at
least 24 non-placeholder characters from a secrets manager. With custody off
the link renders and the checkout page loads, but no payment can be created.

## Database roles

The staging boot gate requires that the server's role cannot rewrite the ledger,
so there are two roles on database `payminto`:

- `payminto_mig` — owns the database and `public`, has `CREATEROLE`, runs
  `cmd/migrate`. Not used by any long-running process.
- `payminto` — the server's role. Ordinary DML everywhere, `SELECT, INSERT`
  only on `ledger_accounts`, `ledger_journals`, `ledger_lines`; no `CREATE` on
  the database or any schema; not a superuser.

The server refuses to boot if that separation is broken.

## Rebuild and redeploy

```sh
cd /home/ubuntu/payminto && git pull

# API
cd backend && go build -o bin/payminto-server ./cmd/server
set -a && . ./.env && . ./.env.migrate && set +a
go run ./cmd/migrate up --ledger-app-role payminto     # as the migration role
sudo systemctl restart payminto-api

# Web (NEXT_PUBLIC_* are compile-time; always rebuild)
cd ../frontend && npm ci && NODE_ENV=production npm run build
cd ../checkout && npm ci && NODE_ENV=production npm run build
cd ../landing  && npm ci && NODE_ENV=production npm run build
sudo systemctl restart payminto-app payminto-checkout payminto-landing
```

### Bootstrapping a *fresh* database

`cmd/migrate up` cannot create a schema from nothing — `ApplyMigrations`
fails closed on an empty history, and the base schema comes from GORM
AutoMigrate, which only runs under `SERVER=development|test`. The order is:

1. Start the server once with `SERVER=development` and
   `POSTGRES_SCHEMA_MODE=auto-migrate` as `payminto_mig`; stop it once
   `/healthz` answers.
2. `go run ./cmd/migrate up --ledger-app-role payminto` as `payminto_mig`.
3. Grant the app role ordinary rights, then re-run step 2 so the ledger
   narrowing is applied last:
   ```sql
   GRANT ALL ON ALL TABLES IN SCHEMA public TO payminto;
   GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO payminto;
   ```
4. Start the real unit (`SERVER=staging`, `validate`).

Order matters: a blanket `GRANT ALL` after the narrowing re-grants `UPDATE`
and `DELETE` on the ledger and the server will refuse to start.

## Demo media

The 88-second demo video is **not in the repository** (27 MB). It lives outside the
working tree and nginx serves it directly, so byte-range seeking works and nothing
is fetched from a third party:

```
/var/www/payminto-media/payminto-demo.mp4
/var/www/payminto-media/payminto-demo-poster.jpg
```

The `location /media/` block on the payminto.io vhost aliases that directory. A
`git pull` and rebuild never touch the file, but a fresh host must repopulate it
from the project's release asset or the Drive copy.

The demo **section** plays the YouTube upload (`2BUyh3VFF74`) behind a
click-to-play facade, so the page requests nothing from YouTube until a viewer
clicks; the mp4 above is the direct-download fallback. The poster deliberately
lives in `landing/public/demo/` and **not** in `/media/`: `next/image` optimises
local paths by fetching them from the Next server, which does not serve the nginx
alias, so a `/media/` poster would 404 through the optimiser.

## Chainlink CRE demo (local, proven)

`scripts/deploy/cre-demo-local.sh` reproduces the solvency attestation end to
end on a local Anvil. Verified 2026-10-13: tx
`0xa7ab8b4d28e3622c711b613fb53c8f4fc0eba472ff0dfcea8572a678e8fb62f1` at block
548, `status=attested`, `provider=chainlink`, USD 125000 owed against 130000
held. The consumer lands at `0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9`, the
address the track write-up quotes, because the deploy order is deterministic.

Three things that are easy to get wrong:

- **It cannot run on the live gateway.** `SERVER=staging` refuses a simulated
  forwarder, so the demo needs its own development-mode process (`:8097`,
  database `payminto_cre_test`, `backend/.env.cre-local`).
- **`CRE_START_BLOCK` is required for the chainlink provider.** Without it the
  module degrades to the mock provider; `GET /api/v1/cre/status` reports exactly
  that in `degraded_from` and `missing_keys`, which is how to diagnose it.
- **Decimals must match the ledger, not the token.** Switch intents price in
  USD and a USD liability carries 2 decimals, so the stack's 6-decimal mock
  token cannot back it; the script deploys a 2-decimal reserve token.

The attestation is labelled `simulated: true` and `independently_signed: false`
because the report goes through the mock forwarder. It proves the workflow's
bytes satisfy the audited consumer and the gateway's verifier; it does not prove
DON consensus. A genuinely signed attestation needs `cre account access`
granted — `cre whoami` currently reports `Deploy Access: Not enabled`.

## TLS

**Issued 2026-10-07** for `payminto.io`, `www`, `app`, `checkout` (one cert,
all four SANs, expires 2027-01-05, auto-renewed by the certbot systemd timer).

`./issue-tls.sh` checks that all four names resolve to this host, then runs
`certbot --nginx --redirect`. It queries the zone's **authoritative**
nameserver, not a public resolver: a resolver caches the absence of a
just-added name for up to the SOA minimum (3600s here), so `1.1.1.1` can still
say NXDOMAIN long after the record is live and Let's Encrypt can already see it. It is safe to re-run and exits 2 (without
contacting Let's Encrypt) while any name is still pending, so no failed
validations are spent against the rate limit. Certbot edits
`/etc/nginx/sites-available/payminto.conf` in place: `nginx/payminto.conf` here
is the pre-certbot source and `nginx/payminto.conf.certbot` is the deployed
result. Reconcile both before ever re-installing from the repo.

Certbot's redirect blocks replace the plain-HTTP server bodies with
`return 404`, so `/.well-known/acme-challenge/` is only reachable on :80 via
the 301 to HTTPS. That is fine — HTTP-01 follows redirects to HTTPS, and the
challenge location is present in every TLS block (verified by fetching a probe
file through the redirect on all three vhosts).

The `payminto_rt` refresh cookie is `Secure` under `NODE_ENV=production`, so
dashboard sessions do not persist over plain HTTP — TLS is required for login,
not just advisable.

## Not yet production

`docs/OPERATIONS.md` lists what a production topology still requires
(versioned migrations, Postgres TLS and a restore drill, mainnet database,
two healthy RPC nodes per chain, SMTP, Sentry). This deployment is
`GATEWAY_ENVIRONMENT=test` on `testnet` with custody and CRE off; it is not
handling real money.
