## Why

Mibo now has durable inventory facts, content-shape planning, and a persistent file signal index, but the remaining recognition gaps are at the path-tree and work-group level: sibling directories containing the same movie release are still treated as separate works, movie collections can require better splitting, and low-confidence decisions need directory-level governance without user-heavy workflows. Completing the automatic recognition pipeline will let users add sources without media-type choices while keeping scans fast, mostly automatic, and more accurate for real-world mixed folders.

## What Changes

- Add a path-tree recognition layer that groups directory snapshots and indexed file signals into work groups before catalog materialization.
- Detect sibling-directory movie versions, including cases where each release is in its own folder but shares the same cleaned title/year and differs mainly by release hints.
- Strengthen movie collection splitting so directories with multiple distinct movie title/year signals produce separate movie work groups instead of one merged work or an episode pack.
- Strengthen series root detection across season folders, flat episode packs, absolute-numbered packs, and sibling season-like directories.
- Reuse existing `inventory_files`, `inventory_file_signals`, `content_shape_profiles`, `content_shape_plans`, and `content_shape_assignments` where possible.
- Clean up or bypass old file-first fallback paths when a work-group plan covers the files.
- Keep remote metadata matching asynchronous and group-scoped; do not call TMDB per file during fast recognition.
- Preserve low-user-participation governance: high-confidence plans auto-materialize, uncertain plans keep guarded placeholders or review groups with evidence, and confirmed corrections become scoped rules.

## Capabilities

### New Capabilities
- `path-tree-work-grouping`: Path-tree recognition that groups directories and files into movie, movie-version, movie-collection, series, season, episode-pack, attachment, or review groups before catalog projection.

### Modified Capabilities
- `media-graph-scanner`: Scanner grouping must operate at work-group scope, including sibling directories and collection splitting, before catalog writes.
- `fast-video-classification`: Fast classification must use indexed signals and path-tree work groups, and bypass redundant file-first classification for covered files.
- `source-first-auto-classification`: Automatic classification must continue without media library type choices while surfacing directory/work-group review outcomes.
- `metadata-operation-pipeline`: Metadata matching must run from recognized work groups and avoid per-file remote lookups in fast scanning.
- `catalog-metadata-governance`: Reviewable recognition decisions and scoped corrections must cover directory/work-group outcomes with evidence and alternatives.

## Impact

- Backend scanner and materialization paths under `mibo-media-server/internal/library/`, especially content-shape planning, assignment generation, directory snapshot reuse, and catalog write selection.
- Database use of existing content-shape and classification-rule/decision tables; add fields or tables only if existing rule/evidence storage cannot represent work groups.
- Metadata match queueing and candidate generation for movie/series work groups.
- Regression tests for sibling-directory movie versions, movie collections, series/season grouping, ambiguous folders, incremental additions, and low-confidence review behavior.
- Existing playback, subtitles, artwork preselection, missing cleanup, projection refresh, and metadata enrichment must remain compatible.
