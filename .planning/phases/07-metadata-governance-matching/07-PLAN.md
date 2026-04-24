# Phase 7: Metadata Governance & Matching - Plan

**Planned:** 2026-04-24
**Status:** Executed
**Phase Goal:** Admins can correct, enrich, and refresh app-owned media metadata so later discovery features build on reliable Mibo-owned data.

## Planning Summary

Phase 7 is split into three execution plans so the work lands in vertical slices without mixing heavy implementation and human checkpoints in the same step. The ordering follows the locked decisions in `07-CONTEXT.md`: backend contracts first, then the dedicated governance UI, then integrated verification and finish work.

## Plan Inventory

| Plan | Name | Wave | Goal | Requirements |
|------|------|------|------|--------------|
| 01 | Backend Metadata Governance Contract | 1 | Add the missing write/read and async backend contract for manual governance, while preserving the jobs/worker model. | META-01, META-02, META-03, META-04 |
| 02 | Governance UI And Entry Flow | 2 | Ship the dedicated admin governance route, draft editing flow, candidate preview/apply UX, and dual entry points. | META-01, META-02, META-05, META-06 |
| 03 | Integrated Refresh, Verification, And Polish | 3 | Verify the full governance workflow end to end, close integration gaps, and finalize phase-level validation. | META-01, META-02, META-03, META-04, META-05, META-06 |

## Execution Order

### Plan 01: Backend Metadata Governance Contract

**Intent:** Fill the current backend gaps identified in `07-RESEARCH.md`: no manual metadata edit API, no dedicated governance payload, and no separate metadata refetch action.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 7-01-01 | Add validated governance mutation endpoints and typed request/response shapes for manual metadata save, preserving auth and input validation at route entry. | META-01, META-02 | T-7-01 | `cd mibo-media-server && go test ./internal/httpapi -run 'Test.*Metadata'` |
| 7-01-02 | Extend metadata services and async queue paths for semi-automatic metadata fields and a dedicated metadata refetch action that stays job-backed. | META-03, META-04 | T-7-02 | `cd mibo-media-server && go test ./internal/metadata -run 'Test(ListTV|ListSeason|Match)'` |

**Expected outputs:**
- Manual governance API under `/api/v1/media-items/{id}/...`
- Distinct metadata refetch queue endpoint separate from rematch
- Backend tests covering auth, payload validation, and service behavior

### Plan 02: Governance UI And Entry Flow

**Intent:** Introduce the dedicated management page and draft workflow required by D-01 through D-10 without overbuilding episodic editorial tools.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 7-02-01 | Add typed client APIs, a dedicated governance route/page, dual entry points, unified draft editing, and leave-guard behavior for unsaved changes. | META-01, META-02, META-05, META-06 | T-7-03 | `cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm typecheck` |
| 7-02-02 | Implement candidate preview/apply flow plus rematch/refetch async feedback states that invalidate and refresh detail/governance queries after completion. | META-05, META-06 | T-7-04 | `cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build` |

**Expected outputs:**
- Dedicated admin governance route in `web/`
- Detail-page and admin/global entry points into the governance workflow
- Candidate diff/preview before apply
- Queued/running/completed UX for rematch and metadata refetch

### Plan 03: Integrated Refresh, Verification, And Polish

**Intent:** Close the phase by proving that saved edits, candidate apply, rematch, and metadata refetch all refresh the same managed media surfaces correctly.

| Task ID | Description | Requirement | Threat Ref | Verification |
|---------|-------------|-------------|------------|--------------|
| 7-03-01 | Run integrated backend/frontend fixes and validation so the full governance flow persists edits, refreshes detail surfaces, and satisfies the phase success criteria. | META-01..META-06 | T-7-01 / T-7-04 | `cd mibo-media-server && go test ./... && cd /Users/atlan/Desktop/IdeaProjects/Mibo/web && pnpm build` |

**Expected outputs:**
- Full phase verification complete
- Manual-only checks executed for leave guard, preview-before-apply, async job clarity, and both entry paths
- Roadmap/state ready for execution tracking and later phase transition

## Constraints To Preserve During Execution

- Keep work inside `web/` and `mibo-media-server/` only.
- Preserve the `OpenList -> mibo-media-server -> client` boundary.
- Keep rematch and metadata refetch as separate job-backed actions.
- Do not turn Phase 7 into a bulk governance tool or deep editorial CMS.
- Prefer candidate-picked artwork over manual URL entry or uploads.

## Definition Of Planned

Phase 7 planning is complete when:

1. Each requirement is assigned to a concrete plan/task.
2. Validation commands exist for every task or phase checkpoint.
3. Execution can begin with Plan 01 without further discovery work.

---

*Phase: 07-metadata-governance-matching*
*Plan created: 2026-04-24*
