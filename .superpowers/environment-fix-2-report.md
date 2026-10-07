# Ticket 13, fix round 2

Branch `environment`, worktree `gateway-wt-environment`, 2026-10-07. Re-review: `.superpowers/environment-rereview.md` (R1 important; N1-N6 minor). Step 0 merged `main` at `42bde82` (merged ledger with schema-qualified trigger functions, fee rules, analytics fixes); conflicts in the Makefile, `main.go`, `router.go`, `config.go`, the database README, migrations manifest and test, `startup.go`, `constraints.sql`, `model.go`, `service.go`, `modules/deps.go`, `registry.go` and `OPERATIONS.md`, all resolved by keeping both sides and re-applying the environment changes onto main's reshaped ledger (asset `varchar(32)`, `posting_started_at`, `replay()`, format()-installed functions). Two `main`-side router tests needed the environment module and an `AuthService` environment.

## Commits

- `42bde82` Merge branch 'main' into environment
- `1e84fdf` R1, N1, N2, N3, N5, N6: no process stamps a database that holds data; `adopt-live` accepts unstamped data; `adopt-test`; stamp is the last boot step; libpq escaping with pgx parse-back; `GATEWAY_TEST_DATABASE_NAME`; one slot list
- (this commit) Postgres tests for R1 and N5, docs, report

## R1: the remote pre-ticket database

- `EnvironmentModule.Stamp` now runs last (after `PrepareSchema`, `VerifySchema`, the registry and `EnforceModeMatch` in the server; after `ApplyMigrations`, `VerifySchema` and the new `config.CheckModeMatch` in `cmd/migrate`). With no stamp present it counts `ledger_accounts`, `api_keys` and `payment_requests`; any row refuses with `database "<name>" holds <table> rows but carries no environment stamp ... go run ./cmd/migrate adopt-live --confirm-adopt-live=<name> (or adopt-test --confirm-adopt-test=<name>)`. Only an empty database is stamped.
- `AdoptLive` accepts an unstamped database as well as one stamped `test` and never adopted; with no stamp it inserts the `live` row (`adopted_from = test`) inside the relabel transaction. The confirmation must equal `current_database()` (trimmed, case-insensitive) and the live name policy applies.
- `AdoptTest(ctx, db, confirm)` and `cmd/migrate adopt-test --confirm-adopt-test=<name>`: test-configured process, unstamped database, same confirmation, test name policy (loopback or `GATEWAY_TEST_DATABASE_NAME`), writes the `test` stamp; relabels nothing.
- `docs/OPERATIONS.md` "Environments" rewritten: the stamp is written last and only on an empty database; both adoption commands; the manual ownership step.

Tests: `modules/environment_test.go` (`TestStamp_RefusesAnUnstampedDatabaseThatHoldsData`, `TestAdoptLive_AcceptsAnUnstampedPopulatedDatabase`, `TestAdoptTest`); `modules/adopt_integration_test.go` `TestIntegration_RemotePreTicketDatabaseIsRefusedUntilAdoptedLive` on Postgres, reached through a non-loopback host in the DSN (the machine's LAN address when the published port answers there, else `localhost.`, which the policy treats as remote): pre-ticket rows and a legacy key, migration applied, test process refused, live `VerifyDatabase`/`VerifySchema` pass, live `Stamp` refuses naming `adopt-live`, legacy key still `test`, N5 refusal with a misowned function, adoption succeeds (2 accounts, 1 journal, 1 key), live boot accepted, balances visible to the live ledger only, test process still refused. `cmd/server/boot_integration_test.go` `TestIntegration_LiveBootRefusesUnstampedDataUntilAdopted`: the real server as a narrowed role exits with the adopt message, `AdoptLive`, then boots and answers the adopted legacy key with `environment: live`. No test inserts the stamp by hand any more except the pre-existing stamped-test adoption case, which models a database `adopt-test` already stamped.

## Minors

- N1: `dsnValue` doubles backslashes and backslashes quotes (the previous round's replacer had collapsed to backslash-to-backslash through a scripted edit; the re-review was right). `database.VerifiedDSN` parses the DSN back with `pgx.ParseConfig` and refuses when host, port, user, password or dbname differ or any runtime parameter appears; `Connect` uses it. Tests round-trip a trailing backslash, quotes and the review's crafted password, and prove an escaped name parses back as one value. Building from pgx config structs was not chosen because `sslmode`'s TLS semantics live in the string parser; parse-back verification gives the same guarantee without re-implementing them.
- N2: stamp after every other boot check, see R1; `cmd/migrate` checks the network-mode row (when present) before stamping, without writing it.
- N3: `GATEWAY_TEST_DATABASE_NAME` (`GatewayConfig.TestDatabaseName`, `BootFacts.TestDatabaseAllowName`, `DatabasePolicy.AllowName`); a test process may open exactly that remote database; the stamp rules still apply; the refusal message names the variable. Tests in `environment/database_test.go`, `config_test.go`, `modules` (`TestAdoptTest`).
- N4: `ledger_adopt_environment` is installed through the same `format()` block as the other ledger functions with `SET search_path = pg_catalog, pg_temp` and every table reference schema-qualified, in `constraints.sql` and the migration.
- N5: the migration gives the function to the owner of `ledger_accounts` when the migrator is a superuser or a member of that role; otherwise it drops the function and raises a NOTICE. `AdoptLive` checks `pg_proc.proowner = pg_class.relowner` first and refuses with the exact manual step (`ALTER FUNCTION ... OWNER TO ledger_owner; GRANT EXECUTE ... TO <migrator>`), proven on Postgres by misowning the function.
- N6: `config` reads `<SLOT>_PROVIDER` for `environment.KnownSlots`; the second list is gone.

## Checks

- `cd backend && go build ./... && go vet ./... && go test ./...`: exit 0.
- `make test-integration` (Docker testcontainers, `-p 4`, 45m timeout): every package `ok` on the first run except `cmd/server`, whose live happy-path test seeded keys into an unstamped database, which R1 now refuses (the correct outcome); the test lets the real first boot stamp the empty database through `EnvironmentModule.Stamp` before seeding, and the package passes (28.7s, all nine `TestIntegration_*` boot tests and seven subprocess refusals). No stamp is inserted by hand in any R1 test.
