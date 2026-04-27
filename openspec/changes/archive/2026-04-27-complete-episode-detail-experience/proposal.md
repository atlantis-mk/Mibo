## Why

Clicking an episode currently opens the generic catalog detail page, so the UI loses the series/season context and cannot present the Emby-style episode detail information shown in the design reference. Matching and refresh actions also operate through the series-root path without enough episode-level correction feedback, which makes wrong episode matches and asset links hard to diagnose or repair.

## What Changes

- Add an episode-specific detail presentation mode that shows the series title, `Sx:Ey` episode label, episode title, still artwork, series backdrop fallback, air date, runtime, rating, genres, and episode overview.
- Extend catalog detail and hierarchy APIs so episode detail responses include parent series/season context, sibling season episodes, progress-aware episode shelf data, external identities, and selected artwork needed by the frontend.
- Add detail-page media technical information for the selected playable asset: video summary, audio tracks, subtitle tracks, container, size, stream metadata, default/forced/external subtitle flags when available, and file/probe state.
- Make episode matching and refresh behavior explicit: resolve episode actions through the series root when provider hierarchy sync is needed, but surface descendant-specific identity, evidence, and actionable mismatch or missing identity states to the user.
- Add governance support for correcting episode asset links, multi-episode segment links, and incorrect season/episode numbers without overwriting unrelated metadata, artwork, or field locks.
- Populate episode people where provider data exists, including directors and guest/episode cast, with safe fallback to series-level people only when episode-level credits are absent.
- Keep series detail season shelves intact while adding single-episode same-season shelves with the current episode highlighted.

## Capabilities

### New Capabilities
- `episode-detail-experience`: Defines the episode-specific detail page behavior, parent context, same-season shelf, media stream presentation, and episode-level people display.

### Modified Capabilities
- `catalog-api-playback`: Item detail, series season, progress, and playback-adjacent responses must expose episode parent context, sibling episode shelf data, asset stream details, and progress-aware episode state needed by the episode detail page.
- `immersive-media-detail`: The immersive detail page must distinguish series, movie, and episode presentation rather than rendering every item through the same poster-first layout.
- `tv-hierarchy-metadata-completion`: TV hierarchy metadata sync must preserve enough descendant identity, artwork, evidence, people, and availability information for episode-level detail and repair flows.
- `catalog-governance-actions`: Governance actions must support episode-specific hierarchy and asset-link correction from the detail and metadata management flows.
- `media-card-progress-badges`: Episode rails and cards must surface watched/in-progress state and current-episode highlighting consistently with existing progress badge semantics.

## Impact

- Frontend: `web/src/features/media/*`, `web/src/lib/media-presentation.ts`, `web/src/lib/mibo-api.ts`, `web/src/lib/mibo-query.ts`, route search handling, playback entry wiring, and episode rail/card components.
- Backend: catalog detail DTOs and query loaders under `mibo-media-server/internal/catalog`, TV metadata sync under `internal/metadata`, playback/media stream DTOs under `internal/playback`, governance asset-link handlers under `internal/httpapi`, and inventory/probe stream projection if additional fields are exposed.
- API: additive catalog response fields for episode context, sibling shelves, progress-aware episode rails, and media stream details. No breaking removal of existing fields is intended.
- Data: uses existing catalog parent-child hierarchy, external IDs, metadata sources, selected images, item_people, asset_items, asset_files, inventory files, and media_streams. New persistence should be avoided unless an inspected stream attribute is not currently stored.
- Tests: backend catalog/query/metadata/governance/playback tests, frontend typecheck/build, and focused UI behavior coverage where practical.
