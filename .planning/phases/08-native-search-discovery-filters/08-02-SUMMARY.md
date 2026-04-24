---
phase: 08-native-search-discovery-filters
plan: "02"
subsystem: search
tags: [go, sqlite, search, metadata, projection]
requires:
  - phase: 07-metadata-governance-matching
    provides: metadata-owned canonical media rows
provides:
  - discovery projection schema for shared search and browse filters
  - TMDB-backed region and rating persistence on canonical media rows
affects: [phase-08, search, discovery, metadata]
tech-stack:
  added: []
  patterns: [app-owned search projection, canonical-to-read-model reindex]
key-files:
  created: []
  modified:
    - mibo-media-server/internal/database/models.go
    - mibo-media-server/internal/database/database.go
    - mibo-media-server/internal/search/service.go
    - mibo-media-server/internal/metadata/service.go
    - mibo-media-server/internal/metadata/service_match.go
key-decisions:
  - Search and filter fields now flow through a dedicated Mibo-owned `SearchDocument` projection instead of live JSON substring reads.
  - Region and rating remain provider-derived metadata populated from TMDB detail responses and then mirrored into the discovery projection.
patterns-established:
  - Canonical `MediaItem` rows are the only source of truth for rebuilding discovery documents.
requirements-completed: [SRCH-01, SRCH-02, SRCH-03, SRCH-04, SRCH-06, FLTR-01, FLTR-02, FLTR-03, FLTR-04]
duration: n/a
completed: 2026-04-24
---

# Phase 8 Plan 02: Discovery Projection Foundation Summary

## Outcome

Introduced an app-owned discovery projection and persisted TMDB country/rating data so Phase 8 filters no longer depend on opportunistic live-row completeness.

## Accomplishments

- Added `SearchDocument` as the shared read model for query text, genre, region, year, type, and rating filters.
- Added `search.Service` reindex helpers for single-item and library-scope projection rebuilds.
- Extended TMDB detail decoding to persist `production_countries` into `regions_json` and `vote_average` into canonical `MediaItem` rows.
- Added metadata regression coverage proving country/rating persistence also refreshes the discovery projection.

## Validation

- `go test ./internal/search ./internal/database -run 'Test|^$'`
- `go test ./internal/metadata -run 'Test.*(Match|Apply|Refetch)'`

## Deviations From Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

- Phase 8 now has a stable projection to refresh from scan, metadata, and progress lifecycles.
- Region and minimum-rating filters have canonical backing data for end-to-end verification.
