## Why

Mibo already discovers same-folder `.srt` and `.ass` sidecar subtitles during scans, but it only records them as scanner evidence. Users can therefore see subtitle sidecars in metadata evidence while playback and item detail still report subtitles as unavailable.

## What Changes

- Bind discovered subtitle sidecars to the catalog asset for the matched video as playable external subtitle tracks.
- Preserve existing scanner evidence for sidecars while also making the subtitle available through normal item detail and playback subtitle track responses.
- Avoid reading subtitle dialogue or using subtitle contents for classification; sidecar subtitles remain non-authoritative media attachments.
- Keep missing, stale, unreadable, or unsupported sidecar subtitle files non-fatal during scanning and probing.
- Ensure OpenList-backed sidecars can be linked through existing storage link behavior without exposing signed provider internals in normal responses.

## Capabilities

### New Capabilities

- `external-sidecar-subtitles`: Defines how discovered sidecar subtitle files are bound to media assets and surfaced as playable external subtitle tracks.

### Modified Capabilities

- `sidecar-metadata-files`: Extend sidecar subtitle behavior from evidence-only discovery to asset/playback availability while keeping metadata-sidecar behavior unchanged.
- `catalog-api-playback`: Ensure playback responses include playable external subtitle tracks for bound sidecar subtitle files.

## Impact

- Affects backend scan/write paths in `mibo-media-server/internal/library`, especially sidecar matching and catalog asset persistence.
- Affects inventory/catalog asset linkage tables such as `inventory_files`, `asset_files`, and `media_streams` or equivalent subtitle-track representation.
- Affects catalog item detail and playback response construction in `internal/catalog` and `internal/playback`.
- Affects OpenList/local storage link handling for subtitle files when a playable URL is requested.
- Adds backend tests for sidecar subtitle binding, rescan cleanup/update behavior, catalog detail tracks, and playback tracks.
