## Why

Mibo's library model currently captures only a single root path and a small set of operational flags, while the catalog, inventory, and governance layers already need richer per-library behavior. Adding library paths and policies now gives users Emby-like control over source layout, scanning, metadata, playback, and subtitle behavior without weakening Mibo's existing media-source and governance architecture.

## What Changes

- Add first-class multi-path library support so one library can include multiple enabled roots, each scoped to a media source and path.
- Add library-level scan policy settings for refresh behavior, realtime/listener participation, hidden-file handling, extension ignores, sample-size ignores, and rule participation.
- Add library-level metadata policy settings for preferred language, country/region, provider enablement, local sidecar participation, and provider priority.
- Add library-level playback and subtitle policy settings for resume thresholds, external subtitle enablement, subtitle language preferences, strict matching, and safe subtitle handling.
- Update library creation, detail, listing, scan, and management APIs/UI to expose these settings while preserving existing single-root library creation as the default path.
- Keep scan exclusions, catalog governance, and metadata field locks as the authoritative safety mechanisms when policy-driven behavior conflicts with curated catalog state.

## Capabilities

### New Capabilities
- `library-source-policies`: Defines multi-path libraries and per-library scan, metadata, playback, and subtitle policies.

### Modified Capabilities
- None.

## Impact

- Backend database models and migrations for library paths and policy records.
- Library create/update/detail/list APIs and DTOs.
- Scanner, listener, scheduled scan, storage index, metadata, playback, and subtitle binding flows that need to resolve effective library policy.
- Settings UI for media sources/libraries and library detail surfaces that display or edit paths and policies.
- Tests for migration compatibility, policy defaults, multi-path scanning, path deletion/disable behavior, and effective policy enforcement.
