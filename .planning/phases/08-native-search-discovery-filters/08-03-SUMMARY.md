---
phase: 08-native-search-discovery-filters
plan: "03"
subsystem: search
tags: [go, worker, jobs, progress, metadata, discovery]
requires:
  - phase: 08-native-search-discovery-filters
    provides: SearchDocument projection and metadata-backed region/rating fields
provides:
  - explicit scan-driven search reindex jobs
  - metadata and progress refresh hooks for discovery freshness
affects: [phase-08, worker, progress, metadata, discovery]
tech-stack:
  added: []
  patterns: [job-backed discovery refresh, synchronous mutation-triggered reindex]
key-files:
  created: []
  modified:
    - mibo-media-server/internal/app/app.go
    - mibo-media-server/internal/library/service.go
    - mibo-media-server/internal/library/service_libraries.go
    - mibo-media-server/internal/library/scan_run.go
    - mibo-media-server/internal/metadata/service.go
    - mibo-media-server/internal/metadata/service_match.go
    - mibo-media-server/internal/progress/service.go
    - mibo-media-server/internal/search/service.go
    - mibo-media-server/internal/worker/worker.go
key-decisions:
  - Full and targeted scans enqueue search reindex work on the existing jobs/worker path instead of mutating discovery inline.
  - Metadata and progress mutations reindex the affected media item immediately so discovery freshness tracks user-visible changes.
patterns-established:
  - Discovery freshness is explicit and lifecycle-driven rather than inferred from later reads.
requirements-completed: [SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-07, SRCH-08, FLTR-05, FLTR-06]
duration: n/a
completed: 2026-04-24
---

# Phase 8 Plan 03: Discovery Freshness Wiring Summary

## Outcome

Wired the new discovery projection into scan, metadata, and progress lifecycles so search and browse stay synchronized after real catalog mutations.

## Accomplishments

- Added `reindex_search_document` and `reindex_library_search` job kinds plus worker dispatch.
- Queued library-scope reindex work after successful full and targeted scans.
- Injected `search.Service` into metadata and progress services so `applyDetail`, manual metadata edits, and progress updates refresh the affected discovery document.
- Kept watched-state semantics on the existing `unwatched / in_progress / watched` rules while making projection refresh explicit.

## Validation

- `go test ./internal/worker ./internal/library -run 'Test.*(Reindex|TargetedRefresh|SyncLibrary)'`
- `go test ./internal/metadata ./internal/progress ./internal/search -run 'Test.*(Reindex|Progress|Watched)'`

## Deviations From Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- The remaining Phase 8 gap can now be closed with regression tests instead of architectural changes.
