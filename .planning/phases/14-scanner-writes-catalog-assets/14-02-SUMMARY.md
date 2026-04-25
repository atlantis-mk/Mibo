---
phase: 14-scanner-writes-catalog-assets
plan: 02
subsystem: database
tags: [scanner, catalog, inventory, go]

# Dependency graph
requires:
  - phase: 13-legacy-backfill-into-catalog-kernel
    provides: existing catalog kernel rows, assets, and migration-safe baseline data
  - phase: 14-scanner-writes-catalog-assets
    provides: plan 14-01 catalog-first scan writer helpers and contracts
provides:
  - scanner traversal now writes catalog items, inventory files, media assets, asset files, and asset-item links directly
  - multi-episode ranges classify into ordered episode slots and scan into canonical episode hierarchy rows
  - duplicate episode-slot scans reuse one episode row and create version assets instead of duplicate items
affects: [phase-14-plan-03, phase-15-series-level-metadata-governance-engine, phase-16-catalog-api-search-progress-cutover, phase-17-playback-item-to-asset-cutover]

# Tech tracking
tech-stack:
  added: []
  patterns: [catalog-first scan traversal, canonical season/episode hierarchy paths, version assets for duplicate slots, inventory-file probe queuing]

key-files:
  created: [mibo-media-server/internal/library/scan_catalog.go]
  modified:
    - mibo-media-server/internal/library/scan.go
    - mibo-media-server/internal/library/scan_classify.go
    - mibo-media-server/internal/library/scan_run.go
    - mibo-media-server/internal/library/enrichment.go
    - mibo-media-server/internal/library/service.go
    - mibo-media-server/internal/library/scan_classify_test.go
    - mibo-media-server/internal/library/scan_catalog_test.go
    - mibo-media-server/internal/library/scan_identity_test.go

key-decisions:
  - "Canonical episode rows now use series-slug/season-XX/episode-XXXX paths so one logical slot survives multi-version scans."
  - "Duplicate-slot files become version assets while multi-episode files share one asset linked by multi_episode_part segment order."
  - "Fresh scan files enqueue inventory-file probe jobs instead of legacy media-file probe jobs."

patterns-established:
  - "Scan traversal writes only through catalog-first helpers and leaves legacy media tables uncreated for new content."
  - "Stable identity evidence reuses inventory files across renames so catalog scan rows do not duplicate on path changes."

requirements-completed: [SCAN-01, SCAN-02, SCAN-03]

# Metrics
duration: 10 min
completed: 2026-04-25
---

# Phase 14 Plan 02: Switch scan traversal to catalog writes with multi-episode and version assets Summary

**Scanner traversal now writes movies and episodes straight into catalog/inventory rows, maps multi-episode files onto ordered episode links, and converts duplicate episode-slot files into version assets.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-25T09:27:30Z
- **Completed:** 2026-04-25T09:37:37Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Added RED regression coverage for multi-episode classification, direct catalog writes, and duplicate-slot version assets.
- Cut `RunSyncLibrary` and `RunTargetedRefresh` over from legacy media-row upserts to catalog/inventory writes.
- Preserved inventory identity continuity on rescans and queued new inventory-file probe work for freshly discovered files.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing direct-scan integration coverage for multi-episode and version writes** - `4431ea3` (test)
2. **Task 2: Route scans through catalog-first writes and stop legacy row creation** - `b9daaa8` (feat)

## Files Created/Modified
- `mibo-media-server/internal/library/scan_classify_test.go` - RED coverage for multi-episode range parsing.
- `mibo-media-server/internal/library/scan_catalog_test.go` - integration checks for direct catalog writes, legacy-table non-creation, and version assets.
- `mibo-media-server/internal/library/scan_classify.go` - multi-episode parsing and episode-slot classification output.
- `mibo-media-server/internal/library/scan_run.go` - scan-loop cutover to catalog/inventory writer and inventory-file probe queueing.
- `mibo-media-server/internal/library/scan_catalog.go` - movie/episode persistence, canonical episode paths, version-asset decisions, and stable-identity inventory reuse.
- `mibo-media-server/internal/library/enrichment.go` - inventory-file probe queue helper.
- `mibo-media-server/internal/library/service.go` - inventory-file probe job kind constant.
- `mibo-media-server/internal/library/scan_identity_test.go` - catalog/inventory identity regression coverage after the cutover.

## Decisions Made
- Used canonical synthetic episode paths (`episode-%04d`) as the durable duplicate-slot key instead of file paths.
- Treated single-slot duplicates as `asset_type="version"` + `asset_items.role="version"`, while multi-episode files keep one asset linked by `multi_episode_part` segment order.
- Stopped running legacy cleanup/upsert paths inside scanner traversal so fresh scans only affect the catalog kernel during this cutover.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added inventory-file probe queueing for catalog-written scans**
- **Found during:** Task 2 (Route scans through catalog-first writes and stop legacy row creation)
- **Issue:** Removing legacy `QueueMediaFileProbe` calls would have stranded newly scanned catalog assets without any follow-up probe contract.
- **Fix:** Added `JobKindProbeInventoryFile` plus `QueueInventoryFileProbe`, then switched scan traversal to enqueue inventory-file probe jobs after catalog writes.
- **Files modified:** `mibo-media-server/internal/library/service.go`, `mibo-media-server/internal/library/enrichment.go`, `mibo-media-server/internal/library/scan_run.go`
- **Verification:** `go test ./internal/library -run 'Test(ClassifyMediaFileParsesMultiEpisodeRange|RunSyncLibraryWritesCatalogRowsWithoutLegacyMediaTables|RunSyncLibraryCreatesVersionAssetForDuplicateEpisodeSlot)' -count=1`
- **Committed in:** `b9daaa8`

**2. [Rule 2 - Missing Critical] Reused inventory files by stable identity across rescans**
- **Found during:** Task 2 (Route scans through catalog-first writes and stop legacy row creation)
- **Issue:** The new catalog-first path initially lost rename continuity and caused the existing identity regression suite to fail because inventory files were keyed only by current storage path.
- **Fix:** Reused existing `inventory_files` rows when stable identity evidence matched, refreshed the path/metadata in place, and updated identity regression tests to assert catalog/inventory semantics instead of legacy media rows.
- **Files modified:** `mibo-media-server/internal/library/scan_catalog.go`, `mibo-media-server/internal/library/scan_identity_test.go`
- **Verification:** `go test ./internal/library -count=1`
- **Committed in:** `b9daaa8`

---

**Total deviations:** 2 auto-fixed (2 missing critical)
**Impact on plan:** Both fixes were required to keep the cutover operational and non-duplicating under the new catalog kernel. No unrelated scope creep was introduced.

## Issues Encountered
- The first RED integration harness used a local-storage root outside the configured provider root, so the tests failed for fixture setup instead of scanner behavior. The harness was corrected before the RED commit so the failing tests reflected the intended missing catalog-write behavior.
- The initial catalog cutover broke legacy identity tests that still expected `MediaItem` / `MediaFile` rows. Those regressions were updated to the new catalog/inventory semantics, and stable-identity inventory reuse was added so rename continuity still holds.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Ready for `14-03-PLAN.md`: scan traversal now emits inventory-file probe jobs and catalog-first assets that the probe worker can consume.
- Legacy delete/availability semantics are still intentionally deferred to Phase 14 Plan 04.

## Self-Check
PASSED

- Found `.planning/phases/14-scanner-writes-catalog-assets/14-02-SUMMARY.md`
- Verified commit `4431ea3` exists in git history
- Verified commit `b9daaa8` exists in git history

---
*Phase: 14-scanner-writes-catalog-assets*
*Completed: 2026-04-25*
