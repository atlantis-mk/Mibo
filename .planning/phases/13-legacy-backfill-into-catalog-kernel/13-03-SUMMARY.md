---
phase: 13-legacy-backfill-into-catalog-kernel
plan: 03
subsystem: database
tags: [go, catalog, migration, inventory, movies]
requires:
  - phase: 13-legacy-backfill-into-catalog-kernel
    provides: durable backfill run/report contracts plus queued trigger and worker execution wiring
provides:
  - legacy movie rows mapped into catalog items, inventory files, media assets, and asset links
  - selected artwork plus provider identity/evidence persisted under catalog ownership
  - repeat-safe movie backfill coverage proving reruns reuse catalog and asset rows
affects: [phase-13-04, phase-13-05, catalog-backfill, migration-reporting]
tech-stack:
  added: []
  patterns: [library+source_path movie lookup, item/file-linked asset reuse, compact provider provenance payloads]
key-files:
  created:
    - mibo-media-server/internal/catalog/backfill_movies.go
    - mibo-media-server/internal/catalog/backfill_movies_test.go
  modified: []
key-decisions:
  - "Reuse legacy movies by library plus source_path and reuse assets by the item/file link tuple so reruns stay idempotent."
  - "Persist only compact provider provenance JSON for migrated movie metadata evidence instead of copying raw legacy blobs."
  - "Reuse selected item images and provider metadata rows on rerun so migrated movie artwork and evidence remain catalog-owned without duplication."
patterns-established:
  - "Movie backfill lives in internal/catalog and composes existing catalog/inventory service upserts instead of raw SQL writes."
  - "Each processed legacy movie row appends a success or skipped report entry while domain rows remain reusable across runs."
requirements-completed: [MIGR-01, MIGR-03]
duration: 10 min
completed: 2026-04-25
---

# Phase 13 Plan 03: Movie backfill into catalog kernel summary

**Legacy movie rows now backfill into reusable catalog items, inventory files, media assets, selected artwork, and provider evidence with rerun-safe identity reuse.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-04-25T07:41:13Z
- **Completed:** 2026-04-25T07:50:21Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Added RED coverage for movie backfill mapping, selected artwork/provider evidence persistence, and rerun idempotency.
- Implemented `backfillMovies` to migrate legacy movie rows into catalog items, inventory files, media assets, asset links, and per-run report entries.
- Reused movie/item/file identities across reruns so repeat backfills add new audit entries without duplicating catalog or asset domain rows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add failing movie backfill regression coverage**
   - `0ac4f20` `test(13-03): add failing movie backfill coverage`
2. **Task 2: Implement idempotent movie backfill mapping**
   - `7bdf151` `feat(13-03): map legacy movies into catalog kernel`

## Files Created/Modified

- `mibo-media-server/internal/catalog/backfill_movies.go` - Implements legacy movie querying, catalog/inventory/asset reuse, selected image migration, provider evidence persistence, and success/skipped report entries.
- `mibo-media-server/internal/catalog/backfill_movies_test.go` - Verifies movie backfill output shape, `source="legacy_backfill"` asset links, provider evidence payloads, and rerun idempotency.

## Decisions Made

- Used `(library_id, type=movie, source_path)` as the canonical movie lookup key so reruns reuse the same catalog row instead of creating duplicates.
- Reused media assets by the existing item/file link identity and kept asset-link writes on inventory service upserts to align with current uniqueness constraints.
- Treated artwork and provider evidence as catalog-owned rows that are updated/reused on rerun rather than appended every time.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Skip legacy movie rows that do not have a stable `source_path` key**
- **Found during:** Task 2 (Implement idempotent movie backfill mapping)
- **Issue:** The plan relied on `source_path` as the canonical movie identity, so blank paths would make movie reuse non-deterministic and could create bad catalog rows.
- **Fix:** Added a guard that records a `skipped` report entry instead of attempting to create a catalog item when a legacy movie row has no usable `source_path`.
- **Files modified:** `mibo-media-server/internal/catalog/backfill_movies.go`
- **Verification:** `cd mibo-media-server && go test ./internal/catalog -run 'TestLegacyBackfillMovies(Idempotent)?' -count=1`
- **Committed in:** `7bdf151`

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The guard preserves deterministic movie identity reuse and keeps malformed legacy rows observable without widening scope.

## Issues Encountered

- The main working tree already contained extensive unrelated dirty and untracked changes, so task commits staged only the movie-backfill files to avoid touching user work.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The series slice can reuse the same report-entry, inventory upsert, asset-link, image, and provider-evidence patterns established for movies.
- The progress/finalization slice can rely on movie backfill having stable catalog item, asset, and inventory identities available for later user-state migration.

## Self-Check: PASSED

- FOUND: `.planning/phases/13-legacy-backfill-into-catalog-kernel/13-03-SUMMARY.md`
- FOUND: `mibo-media-server/internal/catalog/backfill_movies.go`
- FOUND: `mibo-media-server/internal/catalog/backfill_movies_test.go`
- FOUND: `0ac4f20`
- FOUND: `7bdf151`
