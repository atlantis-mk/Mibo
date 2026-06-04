## Why

The scan pipeline already classifies materialized directories as movie folders, movie version folders, series or season folders, collections, attachments, and ambiguous shapes, but the follow-up metadata stage still runs as a flat per-item match queue. This creates slow, noisy work: single movies, multi-version movies, and series folders can trigger repeated item-level matching even though the directory shape already identifies the semantic unit that should be resolved once.

Directory-scoped metadata resolution makes the metadata stage consume the same directory semantics produced by scan/materialization, reducing duplicate work and avoiding remote or speculative matching when local directory evidence is sufficient or the folder needs review.

## What Changes

- Introduce directory-scoped metadata resolution units derived from recognition materialization output and content-shape decisions.
- Resolve single-movie, multipart-movie, and movie-version folders once per movie work, then bind all playable resources, versions, parts, extras, and sidecar assets to that work.
- Resolve series, season, episode-pack, and flat episode folders once at the series scope, then create or bind season and episode metadata from local numbering, sidecars, and hierarchy evidence without independent per-episode search.
- Treat movie collection folders as multiple movie identities within one directory, using local sidecar/external-id evidence when available and generating local provisional movie metadata when no configured search provider exists.
- Suppress metadata matching for attachment-only, extras-only, ambiguous, mixed-conflict, and review-required folders until a user or later workflow supplies a clear target.
- Preserve existing manual apply, refetch, provider profile, local scan, governance, and catalog projection behavior while changing the automatic post-scan queueing model.

## Capabilities

### New Capabilities

- `directory-metadata-resolution`: Directory-scoped automatic metadata resolution after scan recognition materialization.

### Modified Capabilities

- None.

## Impact

- Backend workflow queueing in `mibo-media-server/internal/library/workflow.go` and `mibo-media-server/internal/library/materialize_support.go`.
- Recognition/materialization outputs in `mibo-media-server/internal/library/recognition_manifest.go`, `mibo-media-server/internal/recognition/materializer.go`, and content-shape support files.
- Metadata operation orchestration in `mibo-media-server/internal/metadata/*`, especially automatic match, local evidence application, and operation result recording.
- Catalog projection refresh inputs in `mibo-media-server/internal/catalog/*` remain required after directory-level resolution.
- No new external dependency or public HTTP API is required for the initial change.
