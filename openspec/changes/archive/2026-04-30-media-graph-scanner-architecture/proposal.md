## Why

Current scanning classifies each video independently, so files in the same TV directory can resolve to different series when filenames contain different title signals. We need a stable, extensible scanning architecture that groups media by directory/work identity, supports movie and TV semantics first, and can later expand to music and documents without rewriting the scanner.

## What Changes

- Introduce a Media Graph scanning architecture that separates storage facts, candidate graph construction, resolver decisions, catalog projection, and API DTO mapping.
- Add stable scanner/provider/manual identity reconciliation so title changes or filename noise do not create duplicate catalog items.
- Change video scanning behavior so TV directories are grouped into one series before episode files are classified.
- Change movie scanning behavior so a movie folder can produce one movie with multiple media sources, versions, and extras.
- Add an Emby-like media DTO layer for Movie, Series, Season, Episode, MediaSource, and MediaStream output without replacing Mibo's internal catalog contracts.
- Preserve existing Catalog/Inventory models where possible and evolve them with identity/evidence support instead of replacing them wholesale.

## Capabilities

### New Capabilities
- `media-graph-scanner`: Defines graph-based scan facts, grouping, resolver decisions, stable identities, and catalog projection for extensible media scanning.
- `emby-like-media-dto`: Defines Emby-like media item output for movies, series, seasons, episodes, media sources, and streams.

### Modified Capabilities
- `tv-hierarchy-metadata-completion`: TV hierarchy creation and metadata completion must operate on a stable series root produced by directory/work identity rather than per-file title inference.
- `sidecar-metadata-files`: Sidecar metadata must participate as resolver evidence at group and file levels without overriding manual/provider-owned fields incorrectly.
- `detailed-video-technical-specs`: Technical stream data must map cleanly into media source DTO output while remaining backed by Inventory/MediaStream data.

## Impact

- Affects backend scanning code in `mibo-media-server/internal/library`.
- Affects metadata matching in `mibo-media-server/internal/metadata` for root-object matching and descendant season/episode sync.
- Affects catalog query/contract mapping in `mibo-media-server/internal/catalog` for new Emby-like DTO output.
- May add database support for catalog identities and scanner decision evidence.
- Adds or updates tests for TV directory grouping, movie folder grouping, multi-version movies, multi-episode assets, sidecar evidence, and DTO mapping.
