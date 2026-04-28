## Why

OpenList can expose a provider-supplied `thumb` value on `/api/fs/list` and `/api/fs/get` responses. Mibo currently ignores that field, so video items without TMDB artwork and without sibling artwork can fall all the way through to ffmpeg frame extraction even when the upstream storage provider already has a usable video thumbnail.

Using OpenList thumbnails as an intermediate artwork source improves first-scan artwork coverage for remote libraries, reduces unnecessary ffmpeg work, and keeps the existing local fallback path intact.

## What Changes

- Capture OpenList `thumb` metadata in the storage adapter and expose it through the storage provider object contract.
- Add OpenList `thumb` as a catalog artwork candidate after TMDB/remote metadata and sibling artwork, but before ffmpeg frame extraction.
- Treat OpenList `thumb` as a poster candidate only by default, because it has generic thumbnail semantics rather than explicit backdrop/poster semantics.
- Keep sibling `backdrop`, `background`, and `fanart` images ahead of generated background extraction for true backdrop selection.
- Preserve TMDB and other non-generated selected images as highest priority so trusted metadata is not overwritten.
- Add tests covering OpenList thumbnail capture, priority ordering, non-overwrite behavior, and ffmpeg fallback when no better source exists.

## Capabilities

### New Capabilities

- `openlist-thumbnail-artwork`: Defines how OpenList `thumb` metadata participates in Mibo catalog artwork selection and fallback ordering.

### Modified Capabilities

- Catalog artwork fallback generation.
- Storage provider object metadata contract.

## Impact

- Affects backend OpenList storage adapter in `mibo-media-server/internal/storage/openlist/adapter.go`.
- Affects storage object contract in `mibo-media-server/internal/storage/provider.go`.
- Affects catalog fallback artwork selection in `mibo-media-server/internal/probe/artwork.go` and its caller path in `internal/probe/service.go`.
- Adds or updates backend tests in `internal/storage/openlist` and `internal/probe`.
- No frontend contract change is expected because selected catalog images already flow through existing `selected_images` fields.
