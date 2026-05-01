## 1. Persistence And Migration

- [x] 1.1 Add database models and migrations for metadata provider instances, metadata profiles, and library profile bindings or overrides.
- [x] 1.2 Add migration/backfill logic that synthesizes a default TMDB-backed provider instance and migrated default metadata profiles from existing metadata settings.
- [x] 1.3 Backfill existing libraries to an effective migrated metadata profile that preserves current behavior.

## 2. Provider And Profile Runtime

- [x] 2.1 Introduce provider-instance loading and validation in the metadata/settings layer, including enablement and operational availability state.
- [x] 2.2 Define profile resolution and override handling so a library can resolve its effective metadata profile before any match or refetch operation.
- [x] 2.3 Refactor TMDB-specific metadata logic behind provider-neutral interfaces that can be addressed by named provider instances.

## 3. Metadata Pipeline Execution

- [x] 3.1 Implement a staged metadata pipeline that executes profile-configured search, detail, image, people, and hierarchy stages.
- [x] 3.2 Update match and refetch flows to use the effective library profile, including local-only profiles and configured fallback behavior.
- [x] 3.3 Preserve rooted TV synchronization semantics while routing series, season, and episode metadata operations through the resolved library profile.

## 4. Governance Evidence And APIs

- [x] 4.1 Extend metadata source evidence and related persistence paths to record effective profile provenance, selected provider instance, and fallback summaries.
- [x] 4.2 Update metadata administration and library policy APIs to manage provider instances, metadata profiles, and library profile bindings while keeping compatibility behavior for existing settings consumers.
- [x] 4.3 Update catalog governance and metadata operation responses so clients can inspect profile-aware provenance during match and refetch review.

## 5. Validation And Compatibility

- [x] 5.1 Add focused backend tests for provider-instance persistence, profile resolution, fallback behavior, and migrated default-library bindings.
- [x] 5.2 Add metadata service tests covering profile-aware movie matching, local-only execution, and TV hierarchy synchronization through profile-selected providers.
- [x] 5.3 Verify compatibility by exercising legacy metadata settings reads, migrated profile-backed execution, and governance lock preservation under refetch.
