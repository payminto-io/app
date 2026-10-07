# Payments V1 execution plan

Living document. Updated by the coordinator after every merge. Tickets and owners are in `spec.md`; this file is the order.

## Merged into main

01 ledger, 02 fee rules, 13 live/test environments (plus follow-ups), 18 dashboard honesty, 19 CRE research and spec, design system, brand, checkout UI, dashboard pages, analytics tenant-leak fix.

## Wave 1: finish and merge what is built (in this order)

Order matters because of migration numbers (03 switch, 04 custody, 06 links, 07 CRE, 08 Solana) and shared wiring.

1. 05 switch core: fix round 4 (live ledger guard, connector calls bounded by the claim lease), final check, merge.
2. 08 custody: re-review of round 3, merge.
3. 03 payment links, then 04 link form (based on links): re-reviews, merge.
4. 22 CRE consumer contract: re-audit, merge.
5. 20/21 CRE module: review, align verifier with the contract's final replay rules, merge.
6. 09 Solana USDC and USDT: finish fix round 1, re-review, merge. At merge, turn off the watcher's own deposit journal (switch posts attempt journals).

Every merge: `go build`, `go vet`, unit tests, then `make test-integration` on main before the next merge.

## Wave 2: complete the money path (starts as Wave 1 lands)

| Ticket | Needs | Notes |
| --- | --- | --- |
| 06 routing | 05 | Hyperswitch algorithms plus rail cost, finality and settlement preference |
| 07 card connector | 05 | Stripe through hosted fields only |
| 10 conversion at receipt | 01, 09 | Jupiter on Solana, executed rate as a ledger trade |
| 17 public checkout projection | 03, 05, 09 | fields the checkout already renders when present |
| 23 CRE solvency workflow | 21, 22 | in progress on branch `cre-solvency` |

## Wave 3: settlement and the attested money path

| Ticket | Needs | Notes |
| --- | --- | --- |
| 11 settlement policy and runs | 08, 10 | posts payout journals from custody confirmations; refunds of late and underpaid funds |
| 12 BitGo adapter | 08, 11 | conformance suite |
| 24 CRE deposit finality | 09, 21, 22 | gates settlement above a policy threshold |
| 21b CRE settlement gate | 11, 21, 24 | fail open by default |
| 25 CRE conversion reference | 10, 21, 22 | audit evidence only |
| 26 CRE frontend surfaces | 20, 21 | badges and public verification page |

## Wave 4: ship

| Ticket | Needs | Notes |
| --- | --- | --- |
| 14 hosted checkout wired end to end | 04, 07, 09, 17 | UI exists; wire real methods |
| 15 compose, CI, release | 05, 08 | one command demo, folds in the CRE profile |
| 27 CRE end-to-end demo | 23-26 | `cre workflow simulate` path for deployers without Chainlink access |
| 16 QA, security review, demo script | 11, 14, 15 | gate before any live merchant |
| 28 CRE Solana receiver | 09, 23 | stretch |

## Standing rules for agents

Each ticket: own worktree and branch, failing test first, review by a fresh agent, fix rounds until clean, final review against main, merge by the coordinator only. Never prune the shared Docker host.
