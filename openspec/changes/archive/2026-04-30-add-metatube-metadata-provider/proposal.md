## Why

Mibo can already model named metadata provider instances, but only TMDB and local scan can currently participate in execution. Operators who run a MetaTube server need to use it as a selectable HTTP metadata source per provider instance and per library strategy, without importing upstream SDK code into Mibo.

## What Changes

- Add a `metatube` metadata provider type backed by HTTP calls to a configured MetaTube server.
- Allow operators to create and edit MetaTube provider instances with base URL, optional bearer token, default upstream provider filter, fallback behavior, and timeout settings.
- Allow MetaTube provider instances to be selected in metadata templates and library metadata strategies for supported stages.
- Execute MetaTube search and detail flows through Mibo's metadata runtime, normalize returned movie metadata into catalog fields, and record provider-instance provenance.
- Preserve existing TMDB and `local_scan` behavior; no breaking changes to existing provider instances are intended.

## Capabilities

### New Capabilities
- `metatube-metadata-provider`: Defines MetaTube HTTP provider instance configuration, supported metadata stages, runtime execution behavior, identity/provenance recording, and UI/API availability.

### Modified Capabilities
- `library-source-policies`: Library metadata strategies can reference the new `metatube` provider type for supported stages.
- `catalog-metadata-governance`: Metadata evidence and external identities must preserve MetaTube provider-instance provenance and provider-specific external IDs.

## Impact

- Backend settings persistence and validation in `mibo-media-server/internal/settings` and `internal/database`.
- Metadata runtime orchestration and provider normalization in `mibo-media-server/internal/metadata`.
- HTTP admin APIs for metadata provider instances and library metadata strategies in `mibo-media-server/internal/httpapi`.
- Frontend metadata settings UI and API types in `web/src/features/settings` and `web/src/lib/mibo-api.ts`.
- Tests for provider configuration, stage validation, MetaTube HTTP client behavior, catalog matching, refetch, and provenance.
