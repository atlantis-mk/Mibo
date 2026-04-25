---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: ready_to_plan
stopped_at: Completed 13-05-PLAN.md
last_updated: "2026-04-25T08:28:03.071Z"
last_activity: 2026-04-25
progress:
  total_phases: 9
  completed_phases: 3
  total_plans: 39
  completed_plans: 12
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-25)

**Core value:** 无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。
**Current focus:** Phase 13 — Legacy Backfill Into Catalog Kernel

## Current Position

Phase: 14
Plan: Not started
Status: Ready to plan
Last activity: 2026-04-25

Progress: [███░░░░░░░] 31%

## Performance Metrics

**Velocity:**

- Total plans completed: 45
- Average duration: n/a
- Total execution time: n/a

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1-6 | 13 complete | n/a | n/a |
| 7 | 3 complete | n/a | n/a |
| 08 | 4 complete | n/a | n/a |
| 09 | 3 complete | n/a | n/a |
| 10 | 7 complete | n/a | n/a |
| Phase 10 P07 | unknown | 7 tasks | 16 files |
| Phase 11-event-driven-refresh-hardening P01 | 3min | 2 tasks | 2 files |
| Phase 11-event-driven-refresh-hardening P03 | 13min | 2 tasks | 2 files |
| Phase 11-event-driven-refresh-hardening P04 | 6min | 2 tasks | 3 files |
| Phase 11-event-driven-refresh-hardening P05 | 9min | 2 tasks | 5 files |
| 11 | 5 | - | - |
| Phase 13 P01 | 8 min | 2 tasks | 4 files |
| Phase 13 P02 | 12 min | 2 tasks | 7 files |
| Phase 13 P03 | 10 min | 2 tasks | 2 files |
| Phase 13-legacy-backfill-into-catalog-kernel P05 | 7 min | 2 tasks | 5 files |
| 13 | 5 | - | - |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Keep the storage provider/OpenList → `mibo-media-server` → JSON API boundary intact.
- Keep search and filters product-native and backed by app-owned data, not external middleware.
- Layer scheduled jobs on the existing jobs/worker model rather than a parallel scheduler.
- Use scan listeners only to enqueue targeted refresh and reconciliation work, never direct canonical row mutation.
- Treat metadata governance as the quality foundation for search, filters, and trailers.
- Keep trailer discovery metadata-driven: the frontend consumes one persisted trailer result from GET /api/v1/media-items/{id}.
- Use SpecsSection as the formal trailer entry point and remove the hero placeholder from the primary interaction path.
- Play trailers inside a detail-page dialog so closing playback always returns users to the same detail context.
- Persist schedules as first-class rows with schedule-centric run history instead of deriving them from global job logs.
- Route all scheduled maintenance through the existing jobs/worker queue so manual and recurring execution share one lifecycle.
- Keep schedule management in a dedicated `/schedules` workspace and let settings act only as a summary bridge.
- Use database.Job.available_at as the durable 15-second listener debounce window for Phase 11.
- Seed one listener_reconcile intent per active library on the six-hour default cadence.
- Unsafe listener normalization falls back to sync_library intent rather than guessing a wider targeted root.
- Listener worker branches delegate to the listener service and only enqueue existing scan jobs.
- Reconciliation coverage is seeded before normal job claiming so active libraries keep a future fallback scan intent.
- Special-case normalized non-local / roots in Mibo HTTP validation while preserving local filepath.Rel and non-root prefix checks.
- Use JobActiveIntent.IntentKey for active listener uniqueness instead of making database.Job.JobKey globally unique.
- Keep listener refresh and reconcile guards keyed at library scope so completed historical jobs can remain in the jobs table.
- Use the Phase A catalog kernel as the target architecture for v3 instead of patching legacy `MediaItem` semantics further.
- Migrate catalog governance in reversible steps: contract and backfill first, scanner and metadata writes next, API/playback/frontend reads after, legacy cleanup last.
- Keep legacy backfill run creation inside catalog service helpers that require a non-zero triggered_by_user_id.
- Derive run counters from persisted CatalogMigrationEntry rows during finalization instead of trusting caller-supplied totals.
- Sort run detail entries by entry_type, library_id, legacy IDs, and id so report output stays deterministic across reruns.
- Reuse queued or running backfill jobs by job_key before creating a duplicate operator-visible run.
- Return typed LegacyBackfillRun DTOs from queue/list/detail APIs instead of exposing raw jobs rows to operators.
- Validate library-scoped worker payloads against the persisted run scope before advancing run lifecycle state.
- Reuse legacy movies by library plus source_path and reuse assets by the item/file link tuple so reruns stay idempotent.
- Persist only compact provider provenance JSON for migrated movie metadata evidence instead of copying raw legacy blobs.
- Reuse selected item images and provider metadata rows on rerun so migrated movie artwork and evidence remain catalog-owned without duplication.
- Backfill runs now execute movie, series, and progress slices before finalizing the persisted run status.
- Legacy playback progress resolves asset ownership through migrated catalog item plus asset/file links before upserting user_item_data.
- Worker success updates only catalog_backfill_completed_at and preserves catalog_read_enabled plus legacy_cleanup_completed_at from current settings.

### Pending Todos

None yet.

### Blockers/Concerns

- None currently.

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 260424-stv | 扫描入库时应该使用ffmpeg获取背景图和封面图，这样如果没有元数据时还能显示这两张图片 | 2026-04-24 | uncommitted | | [260424-stv-ffmpeg](./quick/260424-stv-ffmpeg/) |
| 260425-tvg | 基于已完成的 Phase A 新 catalog kernel，生成剩余剧集元数据治理实现计划 | 2026-04-25 | uncommitted | Verified | [260425-tvg-catalog-kernel-remaining-plan](./quick/260425-tvg-catalog-kernel-remaining-plan/) |

## Deferred Items

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-04-25T08:28:03.066Z
Stopped at: Completed 13-05-PLAN.md
Resume file: None

**Completed Phase:** 12 (Catalog Kernel Contracts & Migration Guards) — 6 plans — verified 2026-04-25

**Planned Phase:** 18 (Frontend Catalog Item Migration) — 4 plans — 2026-04-25T08:23:08.921Z
