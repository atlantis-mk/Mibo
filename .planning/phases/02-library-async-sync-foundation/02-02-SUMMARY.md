---
phase: 02-library-async-sync-foundation
plan: 02
subsystem: frontend-web
tags:
  - react
  - settings
  - jobs
  - scan-settings
requires:
  - phase: 02-01
    provides: scan settings endpoints and filtered jobs APIs
provides:
  - Minimal source and library creation flows for the admin settings shell
  - Library status badges plus a dedicated jobs monitoring tab
  - Global refresh interval editing through typed scan settings calls
affects:
  - settings-shell
  - async-sync-observability
  - library-admin-flow
tech-stack:
  added: []
  patterns:
    - Typed settings and jobs client calls via createMiboApi()
    - Minimal local-source setup with auto-generated names from root paths
    - Settings tab composition that combines quick status badges with detailed jobs monitoring
key-files:
  created: []
  modified:
    - web/src/lib/mibo-api.ts
    - web/src/features/app/components/source-drawer.tsx
    - web/src/features/app/components/library-drawer.tsx
    - web/src/features/app/components/jobs-list.tsx
    - web/src/components/settings/settings-shell.tsx
    - web/src/features/app/components/settings-app-shell.tsx
    - web/src/features/app/hooks/use-app-controller.ts
    - web/src/features/app/constants.ts
key-decisions:
  - Keep local source creation to provider plus root path by deriving names from the selected directory.
  - Surface async sync health in two layers: badge on the library card and detailed filtering/retry in the Jobs tab.
  - Put the global refresh interval editor inside Settings so persisted scan cadence changes stay next to job observability.
patterns-established:
  - Frontend settings data loads system info, metadata settings, and scan settings together for a coherent admin shell.
  - Job errors render as plain text content and retries stay behind authenticated typed API calls.
requirements-completed:
  - LIBR-01
  - LIBR-02
  - LIBR-03
  - LIBR-04
  - CATA-06
duration: 4 min
completed: 2026-04-21T18:59:00Z
---

# Phase 02 Plan 02: Admin Async-Sync Settings Summary

**Minimal source/library setup, typed scan settings, and job observability inside the web settings shell**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-21T18:52:02Z
- **Completed:** 2026-04-21T18:59:00Z
- **Tasks:** 3 completed
- **Files modified:** 8

## Accomplishments

- Kept local source creation minimal by auto-generating source names from the selected root path instead of collecting extra V1-only fields.
- Added typed scan settings support and wired the settings shell to load and save `refresh_interval_hours`.
- Added a detailed Jobs tab with filters, retry, and plain-text error rendering while keeping quick library scan status badges on cards.

## Task Commits

Each implementation task was committed atomically in the `web/` repo:

1. **Task 1: Keep source and library creation flows minimal and typed** - `6536229` (feat)
2. **Task 2: Add scan status, jobs monitoring, and refresh controls to settings** - `27d08de` (feat)

**Plan metadata:** Pending commit in root planning repo.

## Files Created/Modified

- `web/src/lib/mibo-api.ts` - Added typed scan settings endpoints alongside the jobs client contract.
- `web/src/features/app/components/source-drawer.tsx` - Reduced local source input surface to provider plus root path with an auto-name preview.
- `web/src/features/app/components/library-drawer.tsx` - Kept inline source creation minimal and library binding focused on source plus root path.
- `web/src/features/app/components/jobs-list.tsx` - Added filtered jobs fetching, retry handling, and plain-text job error rendering.
- `web/src/components/settings/settings-shell.tsx` - Added library status badge helper, refresh interval editor, and JobsList composition.
- `web/src/features/app/components/settings-app-shell.tsx` - Passed settings-shell scan settings and auth context props.
- `web/src/features/app/hooks/use-app-controller.ts` - Loaded scan settings, saved refresh cadence updates, and derived local source names for submissions.
- `web/src/features/app/constants.ts` - Shifted storage provider copy toward product-facing terms.

## Decisions Made

- Use root-path-derived naming for local sources so the local flow matches D-04 without backend schema changes.
- Pass authenticated `apiBaseUrl` and `token` into `JobsList` from the settings shell instead of relying on local storage lookups.
- Keep the refresh interval editor in the Settings → Jobs area so cadence management and async work monitoring live together.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `gsd-sdk` was not executable in this environment, so init/state automation from the executor workflow could not run. Per the prompt, STATE/ROADMAP updates were skipped and work continued with direct repo inspection.

## User Setup Required

None - no external service configuration required.

## Verification

- `cd web && pnpm build` -> PASS
- Manual verification -> PASS:
  - Settings → Media Source kept local source setup to storage type plus root path, with auto-generated naming.
  - Settings → Library dialog stayed focused on source selection, library type/name, and root path.
  - Jobs tab showed status updates, plain-text error text, and retry returned a failed job to queued.
  - `refresh_interval_hours` updated from 24 to 25 and persisted after page reload.

## Next Phase Readiness

- Implementation tasks are complete and the web build is green.
- Human verification approved the end-to-end admin async-sync flow.

## Self-Check: PASSED

- Summary file created at `.planning/phases/02-library-async-sync-foundation/02-02-SUMMARY.md`
- Task commit `6536229` exists in `web/`
- Task commit `27d08de` exists in `web/`
- `cd web && pnpm build` passed after each task

---
*Phase: 02-library-async-sync-foundation*
*Completed: 2026-04-21*
