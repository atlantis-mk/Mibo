---
gsd_state_version: 1.0
milestone: v3
milestone_name: 剧集元数据治理 catalog kernel 迁移
status: ready_to_plan
stopped_at: milestone v3 started; Phase 12 not planned yet
last_updated: "2026-04-25T04:19:16Z"
last_activity: 2026-04-25
progress:
  total_phases: 9
  completed_phases: 0
  total_plans: 0
  completed_plans: 0
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-25)

**Core value:** 无论底层媒体文件来自本地磁盘、NAS 还是云盘，用户都能稳定地完成媒体库接入、内容浏览、播放和进度同步。
**Current focus:** v3 剧集元数据治理 catalog kernel 迁移

## Current Position

Phase: 12 - Catalog Kernel Contracts & Migration Guards
Plan: Not started
Status: Ready to plan Phase 12
Last activity: 2026-04-25 - Started milestone v3: 剧集元数据治理 catalog kernel 迁移

Progress: [----------] 0%

## Performance Metrics

**Velocity:**

- Total plans completed: 40
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

Last session: 2026-04-24T11:06:26.818Z
Stopped at: milestone v3 started; Phase 12 not planned yet
Resume file: None

**Planned Phase:** 12 (catalog-kernel-contracts-migration-guards) — 0 plans — 2026-04-25T04:11:27Z
