---
phase: 13-legacy-backfill-into-catalog-kernel
plan: 04
subsystem: api
tags: [go, gorm, catalog, migration, inventory, tv]

# Dependency graph
requires:
  - phase: 13-01
    provides: durable legacy backfill run and report contracts
  - phase: 13-03
    provides: movie-slice asset and evidence mapping patterns reused by series backfill
provides:
  - repeat-safe series, season, and episode hierarchy backfill from legacy episode rows
  - duplicate slot, missing identity, and orphan file reporting for unsafe TV migrations
  - canonical episode asset linkage for every playable legacy episode candidate
affects: [Phase 13 Plan 05, Phase 14, Phase 15, Phase 16]

# Tech tracking
tech-stack:
  added: []
  patterns: [provider-id-first series grouping, canonical episode slot reuse, orphan-file migration auditing]

key-files:
  created: [mibo-media-server/internal/catalog/backfill_series.go]
  modified: [mibo-media-server/internal/catalog/backfill_series_test.go]

key-decisions:
  - "Provider-backed TV identity wins over title fallback even when fallback-only rows are encountered first."
  - "Series-level provider evidence is stored on the series item while assets stay attached to canonical episode items."

patterns-established:
  - "Series grouping: stable TV IDs alias same-title fallback rows into one canonical series group."
  - "Duplicate episode handling: record audit entries for every non-canonical slot claimant while still linking playable files."

requirements-completed: [MIGR-01, MIGR-02, MIGR-03]

# Metrics
duration: 7 min
completed: 2026-04-25
---

# Phase 13 Plan 04: Legacy series hierarchy backfill Summary

**Repeat-safe series/season/episode backfill with provider-first grouping, duplicate-slot audit entries, and orphan-file reporting.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-04-25T07:59:07Z
- **Completed:** 2026-04-25T08:07:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added RED coverage for provider-backed hierarchy creation, duplicate slot auditing, title fallback grouping, and orphan-file reporting.
- Implemented conservative legacy episode grouping that builds reusable series, season, and canonical episode rows.
- Mapped every playable legacy episode file to catalog assets while surfacing duplicate candidates and unsafe rows in the migration report.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing hierarchy and conflict-report regressions** - `be7cdc6` (test)
2. **Task 2: Implement conservative series hierarchy backfill** - `a373d04` (feat)

**Plan metadata:** pending final docs commit

## Files Created/Modified
- `mibo-media-server/internal/catalog/backfill_series.go` - Implements provider-first series grouping, canonical hierarchy creation, asset linking, and orphan-file reporting.
- `mibo-media-server/internal/catalog/backfill_series_test.go` - Covers hierarchy creation, duplicate-slot conflict auditing, title fallback grouping, and orphan-file regressions.

## Decisions Made
- Provider-backed `tv:`/`series:` IDs now absorb same-title fallback rows so canonical grouping does not depend on legacy row order.
- Series-level external IDs and metadata evidence are written to the canonical series row, while playable assets remain attached at the episode level.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Made provider-first grouping order-independent**
- **Found during:** Task 2 (Implement conservative series hierarchy backfill)
- **Issue:** Fallback-only rows could form a separate series group if they were seen before a later provider-backed row for the same show.
- **Fix:** Added fallback-to-provider alias merging so stable TV IDs always become the canonical group key regardless of row order.
- **Files modified:** `mibo-media-server/internal/catalog/backfill_series.go`, `mibo-media-server/internal/catalog/backfill_series_test.go`
- **Verification:** `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillSeries(Conflicts)?' -count=1`
- **Committed in:** `a373d04`

**2. [Rule 1 - Bug] Reported duplicate candidates against the chosen canonical row**
- **Found during:** Task 2 (Implement conservative series hierarchy backfill)
- **Issue:** Duplicate-slot reporting could tag the representative legacy row as a duplicate when metadata quality changed the canonical pick.
- **Fix:** Duplicate audit entries now skip the chosen canonical legacy episode and only report non-canonical slot claimants.
- **Files modified:** `mibo-media-server/internal/catalog/backfill_series.go`, `mibo-media-server/internal/catalog/backfill_series_test.go`
- **Verification:** `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillSeries(Conflicts)?' -count=1`
- **Committed in:** `a373d04`

---

**Total deviations:** 2 auto-fixed (2 bug)
**Impact on plan:** Both fixes were required to keep provider-first grouping conservative and to make duplicate-slot audit output trustworthy. No scope creep.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Ready for 13-05 progress migration, projection refresh, and backfill finalization work.
- Series hierarchy backfill now exposes the report signals Phase 13 needs before catalog read cutover.

## Self-Check: PASSED

- FOUND: `.planning/phases/13-legacy-backfill-into-catalog-kernel/13-04-SUMMARY.md`
- FOUND: `be7cdc6`
- FOUND: `a373d04`
