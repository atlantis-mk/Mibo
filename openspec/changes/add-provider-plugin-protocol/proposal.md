## Why

Mibo already has internal metadata providers and remote media sources, but each capability is compiled into the core application with provider-specific configuration and execution paths. A unified provider plugin protocol will let third parties provide metadata and storage capabilities without changing Mibo core, while preserving a manageable product experience for both self-hosted services and local companion plugins.

## What Changes

- Introduce a provider plugin protocol where plugins expose a manifest, health check, configuration schema, and typed capability endpoints.
- Support two deployment shapes for the same protocol: remote plugin services and locally managed companion processes.
- Add a core plugin registry and provider instance model that treats both deployment shapes as callable endpoints with declared capabilities.
- Allow metadata profiles and media source setup to select plugin-backed provider instances by capability rather than hard-coded provider type.
- Define the first supported capability contracts for metadata search/detail and storage browse/resolve/link operations.
- Keep existing built-in TMDB, MetaTube, local scan, local storage, and OpenList behavior available through compatibility adapters during migration.
- Defer plugin marketplace, automatic plugin upgrades, full UI extension points, and broad signing infrastructure to later changes.

## Capabilities

### New Capabilities
- `provider-plugin-protocol`: Defines plugin discovery, manifest semantics, deployment shapes, provider instances, capability routing, health checks, and protocol-level operational behavior.
- `plugin-metadata-providers`: Defines plugin-backed metadata provider behavior for candidate search, detail retrieval, fallback routing, and normalized metadata results.
- `plugin-storage-providers`: Defines plugin-backed storage provider behavior for browsing, resolving storage objects, and creating playable links.

### Modified Capabilities

None.

## Impact

- Backend provider registry, settings, metadata, library, storage, and HTTP API layers will need new abstractions for plugin manifests, instances, capability dispatch, health, and configuration schema handling.
- Frontend settings screens for metadata providers, metadata profiles, and media sources will need schema-driven provider configuration rather than provider-specific forms only.
- Existing provider records and media source records may require additive database fields for plugin identity, source kind, endpoint, manifest snapshot, capabilities, health, and configuration payloads.
- Tests should cover manifest validation, instance registration, capability dispatch, built-in compatibility adapters, remote endpoint behavior, local companion endpoint handling, and profile/source selection.
