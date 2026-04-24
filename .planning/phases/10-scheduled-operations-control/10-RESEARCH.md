# Phase 10: Scheduled Operations Control - Research

**Researched:** 2026-04-24
**Status:** Ready for planning

## Research Question

What does Mibo need so Phase 10 can ship product-native schedule management for recurring maintenance without breaking the existing `OpenList -> mibo-media-server -> web` boundary or the jobs/worker execution model?

## Key Findings

### 1. Current worker already proves the correct execution boundary, but not the product model
- `mibo-media-server/internal/jobs/service.go` already provides durable queue primitives: `Enqueue`, `EnqueueUnique`, `ClaimNext`, `Complete`, `Fail`, and job status values `queued/running/completed/failed`.
- `mibo-media-server/internal/worker/worker.go` already executes slow work from queued jobs and must remain the only execution path per D-10 and D-11.
- Gap: the current worker only has a special `triggerScheduledScans` ticker backed by scan settings. There is no persisted schedule model, no run history, no schedule-level latest result, and no generic schedule-to-job bridge.

### 2. Phase 10 needs a first-class schedule domain, not a jobs UI rename
- `mibo-media-server/internal/database/models.go` has `Job` and `SystemSetting`, but no `Schedule` or `ScheduleRun` persistence.
- `mibo-media-server/internal/httpapi/handlers_jobs.go` exposes only job list and retry, which cannot satisfy SJOB-07 / SJOB-08 because admins need schedule CRUD, enable/disable, run-now, next-run visibility, latest result, and history grouped by schedule.
- Therefore Phase 10 requires dedicated persistence and API surfaces centered on schedules, with jobs remaining the execution substrate.

### 3. Product-friendly frequency templates fit the locked scope better than cron exposure
- CONTEXT decisions D-04, D-05, and D-06 explicitly lock the UI to daily / weekly / monthly style templates instead of raw cron editing.
- Because the supported rules are bounded and product-shaped, Mibo does **not** need to expose cron strings in Phase 10.
- External option check: `robfig/cron` is suitable for future raw-cron parsing and timezone support, but using it now would encourage cron-shaped contracts that conflict with the locked UI scope. Phase 10 should store typed template fields and compute `next_run_at` from those fields directly.

### 4. The six required maintenance kinds split into two implementation families
- **Library-side maintenance:** scan, library cleanup, invalid-link check.
- **Metadata-side maintenance:** metadata refetch, trailer sync, artwork refresh.
- This split matches current package responsibilities:
  - `internal/library/*` already owns scans, missing-file cleanup, and library/provider traversal.
  - `internal/metadata/*` already owns TMDB-backed metadata, trailer selection, poster/backdrop refresh, and per-item refetch logic.
- Planning should preserve those ownership boundaries rather than routing all maintenance into `httpapi` or a monolithic “scheduler” package.

### 5. Not every required maintenance kind exists yet, so Phase 10 includes execution work too
- Existing code already supports:
  - full library scan via `library.QueueLibraryScan` / `RunSyncLibrary`
  - per-item metadata refetch via `library.QueueMediaItemMetadataRefetch` and `metadata.RefetchItem`
  - trailer persistence during metadata matching/refetch
  - cleanup of missing files/items as part of scan reconciliation
- Missing reusable batch operations still need to be added for the scheduled scope:
  - library-scoped metadata refetch
  - library-scoped trailer sync
  - library-scoped artwork refresh
  - explicit invalid-link check job
  - explicit cleanup job/product surface distinct from the old hidden scan setting

### 6. Target scope should reuse existing library ownership rules
- D-07 and D-08 lock the model to `maintenance kind + target scope`, where scope is at least `global` or `library`.
- `database.Library` already exists as the correct scope anchor, and `library.Service.ListActiveLibraries` already supports global fan-out for active libraries.
- Scope should therefore be modeled as either:
  - `global` → all active libraries
  - `library` → one specific `library_id`
- Phase 10 must not expand to item-level targeting.

### 7. Run history should be grouped by schedule but still link back to jobs
- D-13 through D-15 require a list view with `enabled`, `next run time`, and `latest result`, plus a detail-layer view of recent runs.
- The clean model is:
  - `Schedule` row stores definition + `next_run_at` + latest-result snapshot.
  - `ScheduleRun` row stores each trigger attempt, timestamps, status, error summary, and the spawned `job_id`.
- This lets the UI stay schedule-centric while still surfacing the real async job path.

### 8. The old scan interval setting is a migration anchor, not the final UX
- `settings.Service.GetScanSettings` and `worker.getRefreshInterval` currently drive the old all-library scan ticker.
- D-17 says this behavior may remain only as an implementation reference / migration anchor.
- Recommended treatment: preserve the legacy scan setting only as compatibility input while creating formal schedule rows; new admin-visible management must operate exclusively on the schedule model.

## Recommended Architecture

### Backend
1. Add a dedicated `internal/schedule` package owning:
   - schedule definitions
   - typed frequency templates
   - `next_run_at` calculation
   - schedule CRUD / toggle / run-now / history queries
   - schedule-run lifecycle helpers
2. Add persistent models for `Schedule` and `ScheduleRun` in `internal/database/models.go` and migrate them in `internal/database/database.go`.
3. Keep actual maintenance executors in their owning domains:
   - `internal/library` for scan / cleanup / invalid-link work
   - `internal/metadata` for metadata refetch / trailer sync / artwork refresh
4. Extend `worker.Runner` to:
   - enqueue due schedules onto the existing jobs queue
   - process schedule-managed jobs through the same handler switch
   - update `ScheduleRun` and schedule latest-result snapshots from worker outcomes

### Frontend
1. Add a dedicated admin workspace route following the Phase 7 metadata-governance pattern per D-01 / D-03.
2. Keep settings “通知与任务” as a lightweight summary / jump-in surface per D-02, not the main workspace.
3. Extend `web/src/lib/mibo-api.ts` and `web/src/lib/mibo-query.ts` with typed schedule contracts.
4. Show in the main workspace list:
   - kind
   - scope
   - frequency summary
   - enabled state
   - `next_run_at`
   - latest result summary
5. Put recent run history in a detail layer (drawer / dialog / side panel) instead of bloating the main list, per D-14.

## Data / Contract Recommendations

### Schedule definition
- `kind`: `scan | metadata_refetch | trailer_sync | library_cleanup | invalid_link_check | artwork_refresh`
- `scope_kind`: `global | library`
- `library_id`: nullable, required when `scope_kind=library`
- `frequency_kind`: `daily | weekly | monthly`
- `time_of_day`: `HH:MM`
- `weekday`: optional for weekly
- `day_of_month`: optional for monthly
- `timezone`: start with server/default timezone unless the codebase already defines a richer product timezone setting

### Schedule status snapshot
- `enabled`
- `next_run_at`
- `last_run_at`
- `latest_run_status`
- `latest_run_error`
- `latest_job_id`

### Schedule run history
- `schedule_id`
- `trigger_source` (`scheduler` or `manual`)
- `status` (`queued/running/completed/failed`)
- `job_id`
- `queued_at`, `started_at`, `finished_at`
- `error_message`

## Standard Stack

### Frontend
- React 19
- TanStack Router
- TanStack React Query
- existing shadcn/radix-nova UI primitives under `web/src/components/ui`
- `sonner` for success/error feedback

### Backend
- Go `net/http`
- GORM + SQLite/Postgres compatibility
- existing `jobs.Service`, `worker.Runner`, `library.Service`, `metadata.Service`, `settings.Service`

## Established Patterns To Reuse

### Reuse, do not hand-roll
- typed frontend API surface in `web/src/lib/mibo-api.ts`
- React Query invalidation keys from `web/src/lib/mibo-query.ts`
- authenticated JSON handlers in `internal/httpapi/*`
- strict request decoding via `decodeJSON`
- async execution through `jobs.Service` + `worker.Runner`
- dedicated admin workspace route pattern from Phase 7 metadata governance

### Do not hand-roll
- direct cron-text editing in the product UI
- synchronous long-running maintenance endpoints
- a second execution engine parallel to jobs/worker
- raw `fetch` calls inside feature components
- schedule logic inside `router.go` or settings page tabs

## Common Pitfalls

1. **Treating schedules as jobs with labels**
   - This would miss `next_run_at`, latest-result snapshotting, and grouped run history.

2. **Letting the scheduler own business work**
   - The schedule package should decide *when* and *what* to trigger, then hand off to existing domain-owned maintenance code.

3. **Reusing the old scan ticker as the final product surface**
   - D-17 forbids shipping the old hidden interval setting as the final answer.

4. **Collapsing trailer sync into generic metadata UI copy**
   - D-09 locks trailer sync in the formal maintenance set; the schedule type must be visible and intentional even if implementation shares metadata primitives.

5. **Putting full history in the main list**
   - D-14 requires detail-layer expansion for recent runs, not a giant table.

6. **Accidentally introducing item-level targeting**
   - D-08 explicitly blocks it.

## Security / Trust Boundaries

1. **Admin browser -> schedule mutation APIs**
   - Treat all payloads as untrusted.
   - Validate kind/scope/frequency combinations at route entry.

2. **Schedule service -> jobs queue**
   - Build job payloads server-side from validated schedule rows.
   - Never accept arbitrary payload JSON from the client.

3. **Worker -> storage / TMDB / OpenList**
   - Scheduled runs may fan out across many items; error summaries must be captured without leaking secrets into history rows.

4. **Schedule history -> admin UI**
   - Return actionable error summaries, not raw stack traces or secrets.

## Architectural Responsibility Map

| Concern | Owns It |
|---|---|
| Schedule definitions, next-run math, run-history records | `mibo-media-server/internal/schedule` |
| Library scan / cleanup / invalid-link maintenance | `mibo-media-server/internal/library` |
| Metadata refetch / trailer sync / artwork refresh maintenance | `mibo-media-server/internal/metadata` |
| Queue persistence and worker lifecycle | `mibo-media-server/internal/jobs` + `internal/worker` |
| Route registration / request validation | `mibo-media-server/internal/httpapi` |
| Typed browser contract | `web/src/lib/mibo-api.ts` |
| Admin workspace and settings summary | `web/src/features/schedules` + `web/src/features/settings` |

## Testing Guidance

### Existing coverage worth extending
- `mibo-media-server/internal/httpapi/router_test.go` for authenticated CRUD / mutation route coverage.
- `mibo-media-server/internal/worker/worker_test.go` for queued job execution and status transitions.
- Domain-package tests in `internal/library` and `internal/metadata` for maintenance semantics.

### Best Phase 10 verification strategy
- Backend first: add domain and API tests for schedule math, maintenance executors, and worker lifecycle.
- Frontend: rely on `pnpm typecheck` + `pnpm build`, then human-verify create/edit/toggle/run-now/history flows.
- End-to-end checks should prove:
  - all six maintenance kinds are creatable
  - schedule enable/disable changes `next_run_at`
  - manual run creates a history row and job-backed feedback
  - latest result updates after worker completion/failure
  - settings page remains an auxiliary summary, not the main workspace

## Validation Architecture

### Quick feedback loop
- Backend quick: `cd /root/Mibo/mibo-media-server && go test ./internal/schedule ./internal/library ./internal/metadata -run 'Test.*(Schedule|Maintenance|InvalidLink|Trailer|Artwork|Refetch)'`
- API quick: `cd /root/Mibo/mibo-media-server && go test ./internal/httpapi ./internal/worker -run 'Test.*Schedule'`
- Frontend quick: `cd /root/Mibo/web && pnpm typecheck`

### Full phase validation loop
- Backend full: `cd /root/Mibo/mibo-media-server && go test ./...`
- Frontend full: `cd /root/Mibo/web && pnpm build`

### Wave 0 needs
- No new test framework is required; existing Go test infrastructure and frontend type/build checks are sufficient.

## Planning Constraints

1. Honor D-01 through D-18 exactly.
2. Exclude deferred items:
   - item-level schedules
   - raw cron editing
   - external notifications / retention policy
   - extra maintenance kinds beyond the locked six
3. Keep work inside `web/` and `mibo-media-server/` only.
4. Preserve the `OpenList -> mibo-media-server -> client` boundary.
5. Keep schedules on the existing jobs/worker model; do not add a parallel scheduler service.

## Recommendation

Phase 10 should be planned as a backend-first schedule platform plus a dedicated admin workspace. The safest decomposition is: domain foundation -> maintenance executors -> schedule API/orchestration -> worker lifecycle -> frontend workspace -> regression closeout.
