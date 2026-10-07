# Modules: how every capability plugs in

The gateway is a set of modules behind ports. Any agent can build, replace or test one module without reading the others. This document is the contract; a ticket that breaks it is not done.

## Shape of a module

```
backend/internal/<module>/
  port.go          the interface(s) other code depends on, and the module's errors and value types
  service.go       core logic that is not provider specific (policy, validation, state machines)
  <provider>/      one folder per implementation of a port (mock, direct, bitgo, stripe, solana, ...)
  conformance/     for slot modules: a test suite every provider must pass
  README.md        what it owns, its tables, its config keys, its events
```

- **Core modules** have one implementation and own data: `ledger`, `fees`, `links`, `switch`, `routing`, `settlement`, `environment`.
- **Slot modules** have many providers chosen by configuration: `connectors` (card, bank, chain), `custody`, `conversion`, `payout`, `kyc`, `fraud`, `bridge`.
- A provider is registered by code in its module's registry (same pattern as `blockchain.AdapterRegistry`). Selection is configuration, never an import change.

## Rules

1. A module imports another module only through that module's `port.go`. Never import a sibling's provider folder or internal types.
2. Only `ledger` writes balances. Every module that moves money posts a journal; nothing stores a balance.
3. Policy runs in core before any provider is called (settlement caps, allow-lists, approvals, environment match). Providers may enforce it too; core never relies on that.
4. Each module wires itself in its own file, `backend/internal/modules/<module>.go`, exposing `func Wire<Module>(deps Deps) (*<Module>Module, error)`. `service/registry.go` gains one line per module. This keeps parallel agents from colliding in the registry.
5. HTTP routes live in `backend/internal/api/routes_<module>.go` with `func Register<Module>Routes(rg *gin.RouterGroup, m *modules.<Module>Module)`. Request and response JSON is snake_case.
6. Configuration lives under one section per module, for example `CUSTODY_PROVIDER=mock|direct|bitgo` plus provider-scoped keys `CUSTODY_BITGO_*`. Missing optional provider config degrades to the mock in test environments and refuses to start in live. Every `Wire<Slot>` resolves its provider with `environment.ResolveProvider` and then calls `guard.RequireProvider(slot, resolved)` from the environment module before constructing it; the boot gate only sees the configured strings, the wiring sees what they resolved to.
7. Migrations are new files in `backend/internal/database/migrations/` named `<YYYYMMDD><NN>_<module>_<what>.up.sql`, additive only. The runner requires applied history to be a contiguous prefix of the manifest, so numbers follow merge order: a branch uses any placeholder number while it is open, and the coordinator renumbers it to the next number above main's highest when it merges. Never insert a migration below one that main already has.
8. Every slot module ships a `mock` provider usable by tests and by `docker compose up` with no keys.
9. Events between modules go through the existing event emitter with named, versioned event types; a module never calls another module's internals to notify it.
10. Tests: unit tests next to the code; integration tests behind `//go:build integration` using `database.NewTestDB`; slot modules run `conformance/` against every provider. Run `make test-integration`.
11. Enterprise providers (see the spec's open-core list) live under `backend/internal/ee/<module>/<provider>/` and register through the same registry. Core never imports `ee`.

## Picking up work as an agent

Read `CLAUDE.md`, `.scratch/payments-v1/spec.md`, your ticket, and this file. Claim the ticket (`Status: claimed`), work in your own git worktree and branch named after the ticket, keep to your module's folders plus your one wiring line, your routes file and your migration, and finish with `Status: done` and the commit hashes in the ticket's `## Comments`.
