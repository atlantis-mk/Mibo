---
phase: 14-scanner-writes-catalog-assets
plan: 03
subsystem: api
tags: [scanner, ffprobe, inventory, worker, catalog]
requires:
  - phase: 14-scanner-writes-catalog-assets
    provides: catalog-first scan writes that enqueue inventory-file probe jobs
provides:
  - inventory-file probe queue reset behavior for linked media assets
  - worker dispatch for probe_inventory_file jobs keyed by inventory_file_id
  - ffprobe persistence into media_streams and linked catalog asset runtime data
affects: [catalog-api-search-progress-cutover, playback-item-to-asset-cutover, scanner]
tech-stack:
  added: []
  patterns: [queued inventory-file probing, compact technical summary persistence, media_stream rebuild by inventory file]
key-files:
  created:
    - mibo-media-server/internal/probe/service_inventory_test.go
    - mibo-media-server/internal/worker/worker_catalog_scan_test.go
  modified:
    - mibo-media-server/internal/library/enrichment.go
    - mibo-media-server/internal/probe/service.go
    - mibo-media-server/internal/worker/worker.go
key-decisions:
  - "Use inventory_file_id as the only probe job payload so worker dispatch no longer depends on legacy MediaFile IDs."
  - "Persist normalized media_stream rows and a compact asset technical summary instead of storing raw ffprobe blobs."
patterns-established:
  - "Queue helper reset pattern: forced inventory probes clear linked asset summary state before re-enqueue."
  - "Probe worker pattern: decode typed payloads first, then persist catalog-asset updates inside one transaction."
requirements-completed: [SCAN-01, SCAN-02]
duration: 13 min
completed: 2026-04-25
---

# Phase 14 Plan 03: Inventory Probe Pipeline Summary

**Inventory-file ffprobe jobs now rebuild media_streams and update linked catalog assets without relying on legacy MediaFile IDs.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-04-25T09:42:12Z
- **Completed:** 2026-04-25T09:55:58Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added RED-phase regression coverage for queued inventory probe jobs, worker dispatch, and asset/media_stream persistence.
- Implemented `QueueInventoryFileProbe` force-reset behavior plus worker dispatch for `probe_inventory_file` payloads.
- Added `ProbeInventoryFile` to persist normalized stream rows, compact technical summaries, and linked catalog item runtime updates.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing inventory-file probe queue and service coverage** - `b490a09` (test)
2. **Task 2: Implement inventory-file probe queue, worker dispatch, and asset updates** - `fcfc629` (feat)

**Plan metadata:** pending (created in the final docs commit)

## Files Created/Modified
- `mibo-media-server/internal/probe/service_inventory_test.go` - RED coverage for asset/runtime/media_stream persistence through the new inventory probe entrypoint.
- `mibo-media-server/internal/worker/worker_catalog_scan_test.go` - worker regression test for queue payload shape and end-to-end probe job completion.
- `mibo-media-server/internal/library/enrichment.go` - forced inventory probe requeue now resets linked asset probe state and summary data.
- `mibo-media-server/internal/probe/service.go` - inventory-file probe execution updates `media_streams`, linked assets, and leaf catalog runtime data.
- `mibo-media-server/internal/worker/worker.go` - worker dispatch now understands `library.JobKindProbeInventoryFile`.

## Decisions Made
- Use `inventory_file_id` as the probe contract boundary so catalog-first scans can stay detached from legacy `MediaFile` rows.
- Persist compact summary JSON plus normalized `media_streams` rows to satisfy the threat model without storing raw ffprobe output blobs.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `gsd-sdk query roadmap.update-plan-progress "14"` did not match the current ROADMAP checkbox format, so the Phase 14 roadmap entries were updated manually after the command reported no-op.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Catalog-first scan writes now retain runtime and stream enrichment through inventory-file probe jobs.
- Ready for the remaining scanner/catalog asset work in `14-04`.

## Self-Check: PASSED

- Found summary file on disk.
- Verified task commits `b490a09` and `fcfc629` in git history.
