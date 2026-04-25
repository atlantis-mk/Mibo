---
phase: 14-scanner-writes-catalog-assets
plan: 04
subsystem: database
tags: [scanner, catalog, inventory, availability, go]
requires:
  - phase: 14-scanner-writes-catalog-assets
    provides: [catalog-first scan writes, inventory-file probe pipeline]
provides:
  - missing-file cleanup that preserves governed catalog metadata
  - leaf availability recomputation from remaining available assets
  - stable-identity rename coverage for catalog-first rescans
affects: [15-series-level-metadata-governance-engine, 16-catalog-api-search-progress-cutover, 17-playback-item-to-asset-cutover]
tech-stack:
  added: []
  patterns: [catalog-first cleanup updates status and availability instead of deleting rows, stable identity remains the reuse key for file and asset continuity]
key-files:
  created: [mibo-media-server/internal/inventory/service.go]
  modified: [mibo-media-server/internal/library/scan_catalog.go, mibo-media-server/internal/library/scan_run.go, mibo-media-server/internal/library/scan_catalog_test.go]
key-decisions:
  - "Catalog-first missing-file handling flips inventory, asset, and leaf availability state instead of deleting governed catalog records."
  - "Leaf availability is recomputed from linked asset availability so multi-version episodes stay playable while any version remains available."
patterns-established:
  - "Scanner cleanup walks file -> asset -> item state transitions inside the scanned scope."
  - "Stable identity rename coverage protects inventory and asset row reuse across rescans."
requirements-completed: [SCAN-03]
duration: 10 min
completed: 2026-04-25
---

# Phase 14 Plan 04: Preserve governed catalog metadata across deletes and rescans Summary

**Catalog-first scanner cleanup now marks missing files and assets unavailable, keeps catalog metadata durable, and preserves stable file identity across rename rescans.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-25T10:09:30Z
- **Completed:** 2026-04-25T10:19:34Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added delete/rescan regression coverage for missing-file availability, surviving-version availability, and stable-identity rename reuse.
- Introduced catalog-first cleanup that marks `inventory_files` and `media_assets` as `missing` without deleting catalog rows or link rows.
- Recomputed leaf `catalog_items.availability_status` from linked asset availability so one missing version no longer makes a still-playable episode unavailable.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing rescan/delete regression coverage for catalog-first availability semantics** - `acfbde9` (test)
2. **Task 2: Implement availability-only cleanup and stable-identity reuse for catalog scans** - `56b88cd` (feat)

**Plan metadata:** pending docs commit

## Files Created/Modified
- `mibo-media-server/internal/library/scan_catalog_test.go` - Adds missing-file, surviving-version, and stable-identity rename regression coverage.
- `mibo-media-server/internal/library/scan_catalog.go` - Implements catalog-first cleanup helpers that mark missing files/assets and recompute leaf availability.
- `mibo-media-server/internal/library/scan_run.go` - Invokes catalog cleanup after traversal for catalog-first scans.
- `mibo-media-server/internal/inventory/service.go` - Defines explicit missing status constants for asset and file lifecycle state.

## Decisions Made
- Catalog cleanup updates availability and status only; governed `catalog_items`, `metadata_sources`, `asset_items`, and `asset_files` remain durable across file disappearance.
- Leaf availability is derived from remaining available assets so version loss does not incorrectly collapse episode availability.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 14 is ready to close with delete-safe catalog-first scan semantics in place.
- Later metadata, API, and playback cutover phases can now rely on governed catalog rows surviving file disappearance while availability remains truthful.

## Self-Check: PASSED
- FOUND: `.planning/phases/14-scanner-writes-catalog-assets/14-04-SUMMARY.md`
- FOUND: `mibo-media-server/internal/library/scan_catalog_test.go`
- FOUND: `mibo-media-server/internal/library/scan_catalog.go`
- FOUND: `mibo-media-server/internal/library/scan_run.go`
- FOUND: `mibo-media-server/internal/inventory/service.go`
- FOUND COMMIT: `acfbde9`
- FOUND COMMIT: `56b88cd`

---
*Phase: 14-scanner-writes-catalog-assets*
*Completed: 2026-04-25*
