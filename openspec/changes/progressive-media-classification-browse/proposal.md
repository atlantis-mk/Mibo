## Why

Current scans can expose media too late and can still collapse low-confidence files into the wrong movie or series metadata too early. We need the library to show inventory-backed results immediately while making movie-versus-series classification more conservative, group-aware, and easier to refine over time.

## What Changes

- Add a group-level media classification capability that classifies video files by work group or directory group before creating final metadata items.
- Tighten movie fallback rules so weak `title + year` evidence does not immediately create or reuse movie metadata when series evidence or conflicting directory signals exist.
- Promote sidecar metadata and external identities to first-class classification evidence before final movie or series collapse decisions are made.
- Extend browse behavior so unresolved inventory or resource-backed entries remain visible until confident metadata grouping is ready, then cleanly upgrade into catalog-backed entries.
- Preserve review-required and low-confidence results as inventory/resource-visible items instead of forcing premature metadata collapse.

## Capabilities

### New Capabilities
- `media-work-group-classification`: classify video content at work-group or directory-group scope, emit confidence-backed movie or series decisions, and defer final metadata creation when signals conflict.

### Modified Capabilities
- `source-first-auto-classification`: change automatic classification so strong episode evidence wins over movie fallback and low-confidence groups remain reviewable without forced metadata creation.
- `library-detail-browsing`: change browse behavior so unresolved inventory/resource entries stay visible until a final metadata-backed card is ready to replace them.
- `sidecar-metadata-files`: change sidecar handling so parsed local metadata and supported external identities participate in movie-versus-series classification before final metadata grouping.

## Impact

- Affected backend areas: `mibo-media-server/internal/library`, `internal/catalog`, `internal/ingest`, and related workflow/browse paths.
- Affected frontend/API behavior: library browse responses will more often include temporary organizing entries before final catalog collapse, with cleaner upgrade semantics.
- Affected data flow: classification, materialization, browse projection, and review-required handling will all use stronger confidence gates and group-level evidence.
