# environment

Core module: live and test money never share a process, a database, a key, a session or a ledger account.

## Owns

- `Environment` (`test` | `live`), the request context tag (`WithContext` / `FromContext`) and `Resolve`.
- `Guard`: every money-moving module calls `guard.Require(ctx, env)` before writing; a mismatch is a `*MismatchError` (`errors.Is(err, ErrMismatch)`). Every `Wire<Slot>` calls `guard.RequireProvider(slot, environment.ResolveProvider(env, configured))`; live refuses the mock and an unset provider (`ErrProvider`).
- `CheckBoot`: the configuration gate. The wiring layer gathers `BootFacts`; this package never reads env vars.
- `CheckDatabase`: the database policy, run on the configured name before connecting and on `current_database()` plus the stamp after.
- `StampRow` (`gateway_environment`): the one-row stamp a database carries; written on first boot, checked on every boot by the server, `cmd/migrate` and `cmd/devseed`.
- `DeriveKey` / `Audience`: HKDF binding of shared secrets (JWT access keys, refresh-token hashes) and the `payminto:<env>` audience every session must present.
- API key prefixes: `sk_test_`, `sk_live_`, `pk_test_`, `pk_live_` (`KeyPrefix`, `KeyEnvironment`).

## Process model

One environment per process, `GATEWAY_ENVIRONMENT=test|live` (default `test`), one database per environment. The dashboard reads `GET /v2/environment`.

## Boot refusals

Live refuses: the test database (by name, `_test` suffix, or stamp), a development keystore or local vault master key (`AES_KEY`, `DEV_KEYSTORE`), a vault in dev mode, any slot that resolves to `mock`, `SERVER` outside staging/production, `POSTGRES_SSL_MODE` other than `verify-full` (except the explicit loopback opt-out the config already models), a non-mainnet network, and a `JWT_SECRET` that is a shipped development default or shorter than 32 bytes.
Test refuses a database that is neither `*_test`, nor on a loopback host, nor named by `GATEWAY_TEST_DATABASE_NAME`; a database stamped live; and a mainnet network.
After schema preparation `VerifySchema` refuses missing `environment` columns and any key row whose visible prefix belongs to the other environment.

## Adoption

A process stamps only an empty database, as its last boot step. A pre-ticket database (data, no stamp) refuses every boot until `cmd/migrate adopt-live --confirm-adopt-live=<name>` or `adopt-test --confirm-adopt-test=<name>` decides what it is (docs/OPERATIONS.md, Environments).

## Tables

`gateway_environment`, plus `environment` on `api_keys`, `ledger_accounts` and `ledger_journals` (migration `2026100705_environment_isolation`); existing rows default to `test`. The ledger's four-column account index stays beside the five-column one until ticket 17 drops it.

## Config keys

`GATEWAY_ENVIRONMENT`, `GATEWAY_TEST_DATABASE_NAME`, `POSTGRES_TEST_DATABASE` (default `payminto_test`), `DEV_KEYSTORE`, and the slot providers (`KnownSlots` is the one list) `CUSTODY_PROVIDER`, `CONNECTORS_PROVIDER`, `CONVERSION_PROVIDER`, `PAYOUT_PROVIDER`, `KYC_PROVIDER`, `FRAUD_PROVIDER`, `BRIDGE_PROVIDER`.

## Wiring

`backend/internal/modules/environment.go` (`WireEnvironment`, `VerifyDatabase`, `VerifySchema`, `Stamp`, `AdoptLive`), route `backend/internal/api/routes_environment.go`.
