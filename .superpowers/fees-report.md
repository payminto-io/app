# Ticket 02 - versioned fee rules and fee preview: report

Branch `fees`, worktree `gateway-wt-fees`.
Commits (oldest first): `10ad400`, `7fa8d7e`, `6211723`, `30eb457`, `acce8ec`, plus the final commit that adds this report and closes the ticket.

## Files

New package `backend/internal/fees/` (core module, shape per `docs/architecture/MODULES.md`):

- `port.go` - `Port` interface, value types (`Scope`, `Pricing`, `RuleInput`, `Rule`, `Slab`, `Query`, `Breakdown`, `PreviewRequest`, `RuleFilter`, `PaymentFee`), enums, typed errors (`*ValidationError`, `*AmbiguousRuleError`, sentinels).
- `compute.go` - `Compute(Rule, amount) Breakdown`, `MinorUnits`, half-up rounding.
- `resolve.go` - pure specificity resolution with tie detection.
- `validate.go` - scope and pricing validation, `Policy`/`ParsePolicy` (surcharge), pure `preview`.
- `journal.go` - the fee journal builder posted through `ledger.PostIn`.
- `model.go` - GORM row, jsonb slabs type, embedded `schema.sql`, `Migrate` for dev/test.
- `service.go` - `Service` (Postgres implementation of `Port`), including `NewVersion` and `ApplyToPayment`.
- `schema.sql` - the DDL; `README.md` - tables, rounding, slab bounds, ledger lines, config, access.
- Tests: `compute_test.go`, `resolve_test.go`, `validate_test.go`, `preview_test.go`, `journal_test.go`, `schema_test.go` (unit) and `integration_test.go` (`//go:build integration`, testcontainer).

New package `backend/internal/modules/`:

- `deps.go` - `Deps{DB, Config, Ledger}`. Must not import `internal/service` (the registry imports `modules`).
- `fees.go` - `WireFees(Deps) (*FeesModule, error)`; `fees_test.go`.

HTTP: `backend/internal/api/routes_fees.go` (`RegisterFeesRoutes` + handler) and `routes_fees_test.go`.

Changed:

- `backend/internal/database/migrations/2026100702_fees_rules.up.sql` - byte-identical to `fees/schema.sql` (enforced by `schema_test.go`); registered in `migrations.go`.
- `backend/internal/database/startup.go` - `MigrateExpandSchema` also runs `fees.Migrate` (the same path the ledger used; `NewTestDB` and auto-migrate pick it up).
- `backend/internal/database/migrations_integration_test.go` - expects three applied migrations.
- `backend/internal/service/registry.go` - field, one `modules.WireFees` call, `FeesModule()` accessor.
- `backend/internal/api/router.go` - `RouterConfig.Fees` and one `RegisterFeesRoutes` call; `backend/cmd/server/main.go` passes `reg.FeesModule()`.
- `backend/internal/api/README.md` - one line.

The runner embeds only `*.up.sql` and no existing migration has a down file, so none was added.

## Schema

`fee_rules`: `id bigserial`, `lineage_id uuid`, `version int`, `method`, nullable `connector`/`card_type`/`region`, `currency`, `percent numeric(9,6)`, `flat numeric(38,18)`, `slabs jsonb` nullable, `min_fee`/`max_fee numeric(38,18)` nullable, `taxable`, `tax_percent numeric(9,6)`, `fee_bearer`, `effective_from`, `effective_to` nullable, `created_by`, `created_at`.
Constraints: unique `(lineage_id, version)`, unique `(id, version)`, CHECKs on enums, ranges, min <= max, tax consistency, window, and the scope ladder.
Trigger `fee_rules_append_only`: rejects DELETE, TRUNCATE, and any UPDATE other than moving `effective_to` earlier.

`payment_requests`: nullable `fee_rule_id bigint`, `fee_rule_version int`, composite FK to `fee_rules (id, version)`, both-or-neither CHECK, and trigger `payment_requests_fee_snapshot_fixed` so a snapshot cannot be changed once written.

## Decisions

- Specificity ladder is enforced as data shape, so specificity is a total order: `card_type` requires `connector` and method `card`; `region` requires `card_type`.
- Ties at the winning specificity return `*AmbiguousRuleError` with sorted rule ids (HTTP 409 `ambiguous_fee_rule`); ties below the winner are ignored.
- Slabs: tier on the whole amount; slab i covers `(up_to[i-1], up_to[i]]`, lower exclusive, upper inclusive; last slab must be open; slabs and rule-level percent/flat are mutually exclusive.
- Order: base, clamp `[min, max]`, round half-up to the minor unit, then tax on the rounded fee, rounded the same way.
- Rounding: fiat 2 by default with ISO 4217 exceptions (0 and 3 place currencies); crypto by a per-asset map (USDC/USDT 6, BTC 8, SOL 9, ETH 18, ...); unknown codes are rejected.
- Percent is stored in percent (`2.9` = 2.9%).
- Versioning: `NewVersion` locks the head with `FOR UPDATE`, refuses non-head ids (`ErrStaleVersion`, 409), closes n at `max(n+1.from, n.from)` and inserts n+1 in one transaction; the unique index is the backstop and maps to `ErrStaleVersion`.
- `effective_from` defaults to now and may not be backdated; a version body is pricing only (strict JSON rejects scope fields).
- Surcharge: `FEES_SURCHARGE_FORBIDDEN_METHODS` (default `upi`) refuses `fee_bearer: customer` at rule creation, at versioning, and at preview (override or stored rule).
- Ledger: `ApplyToPayment` debits `member/<merchant>/<ccy>/liability` by fee + tax, credits `fees/platform/<ccy>/income` by fee and `fees/tax/<ccy>/liability` by tax, key `fee:payment_request:<id>`; zero fee posts nothing; same-rule replay is a no-op; a different rule is `ErrSnapshotConflict`.

## Deviations from the ticket text

- `min`/`max` are `min_fee`/`max_fee`; `tax_percent` added (a tax flag alone cannot compute tax).
- `RegisterFeesRoutes` takes a third argument, the admin permission guard, because `modules` cannot hold `*service.AuthService` without an import cycle (`service/registry.go` imports `modules`). The router passes `middleware.RequirePermission(MEPRoleSvc, "system.admin")` and mounts fees on a `JWTOrAPIKey` group. MODULES.md rule 5 may want to say this.
- Added `GET /admin/fee-rules/:id`.

## Tests

- Unit: specificity order, input-order independence, non-matching scopes skipped, ties (typed error and ids), tie below the winner, effective windows including inclusive start, exclusive end and a future rule, no match; percent plus flat; slab boundaries at, just below and just above each bound and the open slab; min and max clamps; tax; both bearers; rounding for USD, JPY, KWD, USDC, BTC; `MinorUnits`; validation of every field; surcharge forbidden at creation and preview; policy parsing; fee journal balance and accounts; migration equals schema.
- Handler: snake_case response with string decimals; malformed, unknown-field, trailing-data and non-numeric bodies; missing amount; every typed error to status/code (no internal detail leaked); create and version argument passing; scope field in a version body rejected; bad ids; operator gate; admin routes unmounted without a guard; guard applied; 401 without a member; real `NewRouter` puts all five routes behind auth.
- Integration (`-tags=integration`): new version closes the previous atomically and resolves by time; close rolled back when the insert fails (injected trigger); 8 concurrent editors produce exactly one v2; append-only trigger rejects update, reopen, extend, delete, truncate; tie reported over real rows; slabs and decimals round trip and preview; snapshot plus fee journal, idempotent replay, conflict, trigger blocks direct overwrite, unknown payment.

Checks: `go build ./...`, `go vet ./...`, `go test ./...` and `make test-integration` all pass.

## Follow-ups

- Nothing calls `ApplyToPayment` yet: the legacy `PaymentService.CreatePayment` has no method or connector to price, so the call belongs in the switch (ticket 05) when an attempt is created.
- `system.admin` is granted per platform to every owner; `FEES_OPERATOR_PLATFORM_ID` closes that for fee rules, but an operator-level role would be cleaner (ticket 13 or an RBAC ticket).
- Overlapping same-scope rules are allowed at write time and surface as an ambiguity at resolution, as specified; a write-time overlap warning in the admin UI would help.
- `modules/deps.go` is created here; parallel tickets (custody, environment, switch) will likely add the same file and need a merge.
- The asset precision map lives in code; if assets become data (`currencies.wallet_precision`), `MinorUnits` should read it.
