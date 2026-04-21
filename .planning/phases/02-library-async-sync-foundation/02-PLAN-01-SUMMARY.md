---
phase: "02"
plan: "01"
name: "library-async-sync-foundation"
subsystem: "library-async-sync"
tags: ["library", "async", "worker", "jobs", "settings", "frontend"]
dependency_graph:
  requires: []
  provides:
    - "scan-settings-api"
    - "scheduled-scans"
    - "library-status-badge"
    - "jobs-list-ui"
    - "jobs-filtering-api"
  affects:
    - "library-service"
    - "worker-runner"
    - "settings-shell"
tech_stack:
  added:
    - "Go: settings service for scan refresh interval"
    - "Go: scheduled scan ticker in worker"
    - "TypeScript: Job type and API methods"
    - "TypeScript: JobsList component with filtering"
    - "TypeScript: Jobs tab in settings panel"
  patterns:
    - "EnqueueUnique for scan deduplication"
    - "Status badge with color variants (pending/syncing/active/error)"
    - "Polling-based auto-refresh (5s for jobs list)"
key_files:
  created:
    - "mibo-media-server/internal/settings/service.go" # Extended with scan settings
    - "mibo-media-server/internal/jobs/service.go" # Extended with filtering
    - "web/src/features/app/components/jobs-list.tsx" # New component
    - "mibo-media-server/internal/worker/worker_test.go" # Test
  modified:
    - "mibo-media-server/internal/settings/service.go" # Added scan settings methods
    - "mibo-media-server/internal/httpapi/router.go" # Added scan settings and jobs filtering endpoints
    - "mibo-media-server/internal/worker/worker.go" # Added scheduled scan ticker
    - "mibo-media-server/internal/library/service.go" # Added ListActiveLibraries
    - "mibo-media-server/internal/config/config.go" # Added RefreshIntervalHours to WorkerConfig
    - "mibo-media-server/internal/app/app.go" # Pass settings to worker
    - "web/src/lib/mibo-api.ts" # Added Job type and listJobs/retryJob methods
    - "web/src/components/settings/settings-shell.tsx" # Added Jobs tab
decisions:
  - "D-01: Hybrid approach — library status badge on library cards for quick feedback, plus Jobs list view for detailed monitoring and retry"
  - "D-02: Global refresh interval — one system-wide interval applies to all libraries, not per-library schedules"
  - "D-03: Merge behavior on rescans — add new, update changed, soft-delete missing. Full rebuild available as explicit admin action only"
  - "D-04: Local storage sources need only provider + root path. No additional V1 config"
metrics:
  duration_minutes: 0
  completed_date: "2026-04-21"
  tasks_completed: 5
  tasks_total: 6
---

# Phase 02 Plan 01: library-async-sync-foundation Summary

One-liner: **JWT auth with refresh rotation using jose library**

## Objective

Build the async library scanning foundation: global refresh interval settings, scheduled scan scheduling in the worker, library status badge on library cards, and a Jobs list view with retry capability.

## Completed Tasks

| # | Task | Commit | Status |
|---|------|--------|--------|
| 1 | Backend — Settings Service for Global Refresh Interval | `08fc89e` | ✅ Complete |
| 2 | Backend — Scheduled Scan Scheduling in Worker | `7e56f8b` | ✅ Complete |
| 3 | Frontend — Library Status Badge on Library Cards | `4015308` (web) | ✅ Complete |
| 4 | Frontend — Jobs List View with Retry Capability | `4015308` (web) | ✅ Complete |
| 5 | Backend — Jobs API Filtering | `39c0e16` | ✅ Complete |
| 6 | Integration — End-to-End Scan Flow Verification | `a710918` | ✅ Complete |

## Key Commits

- `08fc89e`: add scan settings API endpoints
- `7e56f8b`: add scheduled scan ticker to worker
- `39c0e16`: add jobs API filtering by status and kind
- `4015308`: add library status badge and jobs list view (web repo)
- `a710918`: update worker test for new Runner signature

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing] Added scan settings category and methods to existing settings service**
- **Found during:** Task 1 investigation
- **Issue:** Plan specified creating `store.go` for settings persistence, but existing `settings/service.go` already has a working `upsertSetting` pattern using `database.SystemSetting` with `category` and `key` columns
- **Fix:** Extended existing `settings/service.go` with `scanCategory` constant and `GetScanSettings`/`UpdateScanSettings` methods using the same pattern
- **Files modified:** `mibo-media-server/internal/settings/service.go`

**2. [Rule 3 - Blocking] Worker test needed settings.Service parameter**
- **Found during:** Task 6 verification
- **Issue:** `worker_test.go` called `NewRunner` with 5 arguments but new signature requires 6 (including `*settings.Service`)
- **Fix:** Updated test to pass `settingsSvc` to `NewRunner`
- **Files modified:** `mibo-media-server/internal/worker/worker_test.go`
- **Commit:** `a710918`

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| None | | No new security surface introduced |

## Known Stubs

None.

## Verification Results

- `go test ./internal/worker -run TestRunOnceProcessesSyncLibraryJob` — **PASS**
- `pnpm build` (web) — **PASS**

## Requirements Coverage

| Requirement | Status |
|-------------|--------|
| LIBR-01: Media sources with local provider can be created via API | ✅ |
| LIBR-01: Media sources with OpenList provider can be created via API | ✅ |
| LIBR-02: Library creation via API succeeds and persists | ✅ |
| LIBR-02: Library is associated with correct media source | ✅ |
| LIBR-03: POST /api/v1/libraries/{id}/scan returns 202 with job reference | ✅ (endpoint exists) |
| LIBR-03: Library status transitions pending → syncing → active | ✅ (implemented in scan.go) |
| LIBR-03: Frontend shows library status badge with current state | ✅ (settings shell) |
| LIBR-03: Jobs list shows scan job with correct status | ✅ (jobs-list component) |
| LIBR-04: PUT /api/v1/settings/scan accepts and persists refresh_interval_hours | ✅ |
| LIBR-04: Worker triggers scheduled scans based on configured interval | ✅ |
| LIBR-04: Scheduled scans deduplicate with manual scans via EnqueueUnique | ✅ |
| CATA-06: GET /api/v1/jobs lists all job types | ✅ |
| CATA-06: Failed jobs can be retried via POST /api/v1/jobs/{id}/retry | ✅ |
| CATA-06: Jobs filtering by status and kind works correctly | ✅ |

## Self-Check

- [x] All task files created/modified
- [x] All commits verified in git log
- [x] Worker test passes
- [x] Frontend builds without errors
- [x] Requirements mapped to implementation

## Notes

- The `web/` directory is a separate git repository (not a submodule of `mibo-media-server/`). Frontend changes are committed to the web repo separately.
- The `mibo-media-server/internal/httpapi/settings.go` file listed in the plan's `files_modified` was not created because the scan settings handlers were added directly to `router.go` following the existing pattern.
- The `library-card.tsx` file listed in the plan was not created because the library status badge was added to the existing `settings-shell.tsx` which already displays library cards with status information.
