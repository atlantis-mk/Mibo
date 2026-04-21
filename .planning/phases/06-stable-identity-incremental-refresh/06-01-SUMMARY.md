---
phase: 06-stable-identity-incremental-refresh
plan: "01"
subsystem: api
tags: [go, gorm, scan, stable-identity, openlist]
requires:
  - phase: 05-playback-decision-intelligence
    provides: playback continuity anchored on media items and probe-backed media facts
provides:
  - stable-identity evidence fields on storage objects and media files
  - exact stable-id scan matching across storage path changes
  - provisional fallback candidate staging when path alone is insufficient
affects: [probe, worker, incremental-refresh, reconciliation]
tech-stack:
  added: []
  patterns: [stable-id-first media file matching, provisional fallback candidate staging]
key-files:
  created: [mibo-media-server/internal/library/scan_identity_test.go]
  modified:
    - mibo-media-server/internal/storage/provider.go
    - mibo-media-server/internal/storage/openlist/adapter.go
    - mibo-media-server/internal/database/models.go
    - mibo-media-server/internal/library/scan.go
key-decisions:
  - "Trust exact stable identity for scan-time continuity, but treat path as a locator only when the underlying object changes without stable identity."
  - "Keep deleted media-file candidates linked to their prior media item so later size+duration reconciliation can safely reclaim continuity."
patterns-established:
  - "Storage adapters surface raw identity evidence even when they cannot provide a trustworthy stable object ID."
  - "Fallback candidates stay provisional and review-pending instead of rebinding continuity on path-only evidence."
requirements-completed: [SYNC-01]
duration: 11min
completed: 2026-04-21
---

# Phase 6 Plan 01: Stable identity evidence contract and conservative scan staging Summary

**Stable-id-first scan ingestion now persists provider/hash evidence, survives exact-ID path moves, and stages same-path replacements as provisional fallback candidates.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-04-21T22:37:46Z
- **Completed:** 2026-04-21T22:48:26Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Extended `storage.Object` and the OpenList adapter to preserve provider and hash evidence for later reconciliation.
- Persisted stable-identity, review, and replacement bookkeeping on `media_files` and used exact stable IDs to reuse file identity across path changes.
- Changed scan ingestion to stage provisional fallback candidates when path-only evidence would otherwise rebind continuity.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend stable identity evidence contracts** - `58f1bfc` (test), `7402d8f` (feat)
2. **Task 2: Stage provisional fallback candidates instead of path-first rebinding** - `7649459` (test), `b15d577` (feat)

## Files Created/Modified
- `mibo-media-server/internal/storage/provider.go` - Added stable identity, provider, and hash evidence fields to storage objects.
- `mibo-media-server/internal/storage/openlist/adapter.go` - Preserved OpenList provider/hash metadata on list/get responses.
- `mibo-media-server/internal/database/models.go` - Added media-file identity source, status, review, and replacement fields.
- `mibo-media-server/internal/library/scan.go` - Implemented stable-id-first matching and provisional fallback staging.
- `mibo-media-server/internal/library/scan_identity_test.go` - Added regression coverage for exact stable-ID reuse and provisional fallback candidates.

## Decisions Made
- Exact stable identity is the only scan-time signal allowed to preserve continuity across path changes.
- Path-matched files with changed fingerprints become detached provisional candidates instead of reusing the old media-file row.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Preserved deleted candidate media-item links for later reconciliation**
- **Found during:** Task 2 (Stage provisional fallback candidates instead of path-first rebinding)
- **Issue:** The existing missing-file cleanup nulled `media_item_id`, which would have removed the target continuity needed by the later size+duration reconciliation plan.
- **Fix:** Stopped cleanup from clearing `media_item_id` and explicitly soft-deleted same-path superseded rows instead.
- **Files modified:** `mibo-media-server/internal/library/scan.go`
- **Verification:** `go test ./internal/library -run 'TestRunSyncLibrary(UsesStableIdentityEvidence|CreatesFallbackCandidateWithoutPathRebind)'`
- **Committed in:** `b15d577`

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The adjustment was required to keep fallback reconciliation data intact for the next plan without widening scope.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Media files now persist stable-id evidence and provisional review state needed for post-probe reconciliation.
- Deleted candidates retain the prior media-item link needed to recover playback continuity conservatively in Plan 06-02.

## Self-Check: PASSED
