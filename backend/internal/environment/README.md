# environment

Core module: live and test money never share a process, a database, a key or a ledger account.

## Owns

- `Environment` (`test` | `live`), the request context tag (`WithContext` / `FromContext`) and `Resolve`.
- `Guard`: every money-moving module calls `guard.Require(ctx, env)` before writing; a mismatch is a `*MismatchError` (`errors.Is(err, ErrMismatch)`).
- `CheckBoot`: the boot gate. The wiring layer gathers `BootFacts` from configuration; this package never reads env vars.
- API key prefixes: `sk_test_`, `sk_live_`, `pk_test_`, `pk_live_` (`KeyPrefix`, `KeyEnvironment`).

## Process model

One environment per process, `GATEWAY_ENVIRONMENT=test|live` (default `test`). The dashboard reads `GET /v2/environment`.

## Boot refusals

Live refuses: the test database (by name or `_test` suffix), a development keystore or local vault master key (`AES_KEY`, `DEV_KEYSTORE`), a vault in dev mode, any `*_PROVIDER=mock`, `SERVER` outside staging/production, `POSTGRES_SSL_MODE` other than `verify-full` (except the explicit loopback opt-out the config already models), and a non-mainnet network.
Test refuses a database that is neither `*_test` nor on a loopback host, and a mainnet network.

## Tables

No tables of its own. It adds `environment` to `api_keys` and `ledger_accounts` (migration `2026100705_environment_isolation`); existing rows default to `test`.

## Config keys

`GATEWAY_ENVIRONMENT`, `POSTGRES_TEST_DATABASE` (default `payminto_test`), `DEV_KEYSTORE`, and the slot providers `CUSTODY_PROVIDER`, `CONNECTORS_PROVIDER`, `CONVERSION_PROVIDER`, `PAYOUT_PROVIDER`, `KYC_PROVIDER`, `FRAUD_PROVIDER`, `BRIDGE_PROVIDER`.

## Wiring

`backend/internal/modules/environment.go` (`WireEnvironment`), route `backend/internal/api/routes_environment.go`.
