# 26 CRE frontend: attestation badges, runs and records, public verification page

Status: ready-for-agent
Owner: Frontend engineer (Opus) with UI/UX
Blocked by: 20, 21

## Goal
Per `docs/cre/SPEC.md` section 12.
- `frontend/components/attestation-badge.tsx` with states attested, pending, stale, failed, mock, simulated; absent when provider is `none`.
- Badges on settlement rows and detail (deposit finality), conversion rows (reference and deviation), custody and treasury overview (solvency shown as the two numbers plus ratio, never a ratio alone).
- Settings page additions: last ten runs per kind with CRE execution ids, force-run for owner role, accept-new-workflow-id flow with audit entry.
- `/verify/[attestation_id]`: public, server-rendered, re-checks the log at render time, shows what is and is not proven. Returns 404 when `CRE_PUBLIC_VERIFY_ENABLED` is false or provider is `none`.
- Copy in `frontend/lib/copy/attestations.ts`; design per `docs/design/DESIGN.md`.

## Acceptance
Screenshots at 390 and 1440, light and dark, for every badge state and the public page in mock and simulated modes; no figure rendered that is not a stored attestation field; `npx tsc --noEmit`, lint and tests pass.
