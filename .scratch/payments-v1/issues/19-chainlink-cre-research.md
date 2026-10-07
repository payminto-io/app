# 19 Chainlink CRE: research, market fit and module spec

Status: done
Owner: Chainlink engineer (Fable)
Blocked by: -

## Goal
Decide, with evidence, what Chainlink CRE (Chainlink Runtime Environment) does inside this gateway that a merchant or deployer would pay for, and specify it as an optional module. The earlier validation report (`../reports/Agent native crypto fiat gateway validation.md`) judged CCIP/CRE "keep optional, defer"; the owner has decided to build it now as an optional module, so this ticket must find the use cases with real market pull and say plainly where demand is thin.

## Deliverables
- `docs/cre/RESEARCH.md`: what CRE is today (capabilities, triggers, consensus, chains, pricing and access, maturity), who uses it in payments and stablecoins, competing approaches, candidate gateway use cases ranked by demand evidence and feasibility, sources linked.
- `docs/cre/SPEC.md`: the module spec. Optional at install (`CRE_ENABLED=false` default; the gateway runs fully without it), provider-shaped per `docs/architecture/MODULES.md`, the workflows to build, data in and out, trust model (what CRE attests, what the gateway still decides), failure modes, and how a deployer turns it on.
- Tickets 20+ for implementation, each with owner role and blocking order.

## Comments

Done on branch `cre`.
Commits: `6030a64` (docs/cre/RESEARCH.md), `1005644` (docs/cre/SPEC.md), `4e045b6` (tickets 20 to 28, spec table rows, Chainlink role).
Ranked outcome: solvency attestation (build first, real buyers in the adjacent PoR market), deposit finality attestation (build second, policy-gated, fails open by default), conversion reference stamp (survives as audit evidence only); CCIP orchestration, x402 verification and ACE are cut with reasons in RESEARCH.md section 4.
Verified locally: `cre` CLI v1.37.0 scaffolds and simulates a TypeScript workflow with no login (`.superpowers/cre-scratch/demo`), which is what makes the mock-plus-simulate demo path possible for deployers without Chainlink access.
