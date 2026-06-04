## Why

Current hierarchical library browsing in Mibo derives display folders at browse time from resource paths and naming heuristics. That works for common movie and show layouts, but it remains fragile for mixed collections, category folders that temporarily contain one title, and libraries whose playable files live below season, edition, or split-part directories. We need scan-driven display paths so library browsing reads stable semantics produced by the scan and projection pipeline instead of guessing them during each browse request.

## What Changes

- Add scan-driven display path fields to the library metadata projection model so each library-specific metadata item stores the directory where it should appear in hierarchical browsing.
- Compute display paths during projection rebuild by using scan/link data from `inventory_files`, `resource_files`, `resource_metadata_links`, and ancestor metadata relationships instead of browse-time folder inference.
- Distinguish between media-root directories and structural subdirectories such as season folders, split-part folders, and edition/version folders when deriving the display path.
- Update hierarchical library browsing to build folder trees primarily from projection display paths, with browse-time inference kept only as a compatibility fallback for stale or incomplete projection rows.
- Keep existing playback, detail, visibility, and authorization behavior unchanged after the user reaches a metadata item.

## Capabilities

### New Capabilities
- `library-display-projection`: Persist library-specific display root paths and directory semantics in catalog projections so browse consumers can render stable folder hierarchies without recomputing them.
- `scan-driven-library-browser`: Build hierarchical library browsing from scan-driven projection paths, including direct item surfacing for single-title media directories and series roots derived from episode resources.

### Modified Capabilities
- None.

## Impact

- Backend projection code in `mibo-media-server/internal/catalog` that rebuilds `library_metadata_projections`.
- Database schema for `library_metadata_projections` to persist display path metadata.
- Hierarchical browse service and HTTP API in `mibo-media-server/internal/catalog` and `mibo-media-server/internal/httpapi`.
- Scan/projection refresh behavior in `mibo-media-server/internal/ingest` because projection rebuilds must now refresh display path semantics.
- Frontend library browser consumers in `frontend/src` indirectly benefit from more stable node layout without requiring route contract changes.
- New tests covering movies, multi-version folders, split-part folders, series directories derived from episode files, category folders, and fallback behavior when projection display paths are absent.
