## Why

Current media detail pages only show a compact stream summary: codec, resolution, aspect ratio, bitrate, and basic audio/subtitle information. Users need a richer MediaInfo-style technical view for each video stream so they can verify encode profile, frame cadence, color characteristics, bit depth, and other playback-relevant attributes without leaving Mibo.

## What Changes

- Extend catalog media stream data to include detailed video technical attributes from `ffprobe`, including profile, level, frame rate, interlace state, color space, bit depth, pixel format, and reference frame count when available.
- Expose the new stream attributes through the catalog item detail API as optional fields so existing responses remain valid when probes or legacy rows lack the data.
- Update the immersive media detail technical section to render video streams in a MediaInfo-style label/value format matching fields such as title, codec, profile, level, resolution, aspect ratio, interlaced, frame rate, bitrate, color space, bit depth, pixel format, and reference frames.
- Keep existing audio, subtitle, and file summaries available, while making the video card more precise and readable for detailed inspection.
- Ensure re-probing can populate the new fields for newly scanned or refreshed catalog inventory files.

## Capabilities

### New Capabilities
- `detailed-video-technical-specs`: Catalog-backed video streams expose and render detailed technical specifications in a user-readable MediaInfo-style format.

### Modified Capabilities
- `catalog-api-playback`: Catalog item detail responses include optional detailed stream technical attributes for catalog assets.
- `immersive-media-detail`: The media detail information section presents detailed video technical specifications when available.

## Impact

- Backend probe service: expand `ffprobe` stream parsing and inventory stream persistence.
- Database models/migrations: add nullable media stream columns for detailed video attributes.
- Catalog contracts and query mapping: include optional detailed stream attributes in asset stream summaries.
- Frontend API types: add optional stream fields to `CatalogMediaStreamSummary`.
- Frontend detail UI: enhance `SpecsSection` video display and formatting helpers.
- Tests: update backend probe/catalog contract coverage and frontend typecheck/build validation.
