## Why

Mibo currently reads files through several ad hoc paths that each decide differently whether to use provider metadata, direct links, local filesystem paths, or HTTP proxying. That inconsistency makes security policy hard to enforce, duplicates fallback logic across playback, artwork, sidecar, and probe flows, and breaks down for OpenList because its access URLs are ephemeral.

## What Changes

- Introduce a unified access layer that resolves stable file locators and issues purpose-specific, short-lived access grants for every file read.
- Treat both local and OpenList resources as expiring external-facing links so that copied playback and artwork URLs cannot be reused indefinitely.
- Centralize provider-specific access resolution, including OpenList real-time URL refresh and local file serving through signed temporary access routes.
- Refactor playback, artwork, sidecar hydration, and probe flows to consume the unified access layer instead of mixing `Get`, `Link`, `RawURL`, and direct filesystem assumptions.
- Persist only stable resource location data in storage-facing records and treat provider URLs and thumbnails as volatile runtime access data.

## Capabilities

### New Capabilities
- `secure-file-access`: Unified short-lived access grants for file, artwork, metadata, and probe reads across all storage providers.
- `provider-runtime-access`: Provider runtime access resolution rules for local storage and OpenList, including volatile URL refresh and proxy/direct serving policy.

### Modified Capabilities

- None.

## Impact

- Affected backend packages include `internal/storage`, `internal/playback`, `internal/httpapi`, `internal/library`, `internal/probe`, and provider registry wiring.
- New signed access endpoints and verification middleware will affect playback and artwork URL generation.
- Existing assumptions around `storage.Object.RawURL` will need to be replaced or narrowed so local paths and remote URLs are no longer conflated.
- OpenList and local media source behavior will become consistent from a security and lifecycle perspective, with short-lived external URLs for both.
