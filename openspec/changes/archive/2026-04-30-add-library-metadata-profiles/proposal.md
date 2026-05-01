## Why

The current metadata flow is hard-wired around a single active TMDB configuration with partially reserved TVDB policy fields, which prevents different libraries from using different metadata acquisition methods. We need a library-scoped metadata strategy model so movie, TV, anime, documentary, and local-only libraries can select different provider instances, fallback behavior, and field sourcing without splitting metadata orchestration into a separate service.

## What Changes

- Introduce library-bound metadata profiles that define how a library searches, fetches, and applies metadata across provider instances and local sources.
- Introduce provider-instance configuration so the system can manage multiple TMDB tokens, future TVDB tokens, and custom provider entries as named runtime instances instead of a single global provider config.
- Replace the current TMDB-centric metadata execution path with a staged metadata pipeline that evaluates the bound library profile for search, detail, image, people, and hierarchy synchronization.
- Preserve the existing catalog governance model while recording profile-aware source evidence, provider instance provenance, and fallback outcomes.
- Extend metadata administration APIs and library policy APIs so operators can create provider instances, define reusable metadata profiles, and bind or override profiles per library.

## Capabilities

### New Capabilities
- `library-metadata-profiles`: Reusable library-scoped metadata profiles that compose provider instances, local metadata behavior, stage ordering, fallback rules, and field application policy.
- `metadata-provider-instances`: Named metadata provider instances for multiple TMDB tokens, future TVDB integration, and custom provider definitions with health and enablement state.

### Modified Capabilities
- `catalog-metadata-governance`: Metadata matching, refetch, and governance evidence must become profile-aware and retain provider-instance provenance for automated and fallback-driven writes.
- `tv-hierarchy-metadata-completion`: TV hierarchy synchronization must run through the selected library metadata profile while preserving rooted series matching and descendant identity behavior.

## Impact

- Affected backend packages include `internal/metadata`, `internal/settings`, `internal/library`, `internal/httpapi`, and catalog evidence/identity persistence paths.
- New persistence will be needed for provider instances, metadata profiles, and library-to-profile bindings or overrides.
- Existing metadata settings and library metadata policy APIs will need compatibility and migration handling as provider-instance/profile-based configuration becomes authoritative.
- Frontend or admin clients that manage metadata settings will need new endpoints or payloads to configure provider instances and bind profiles per library.
