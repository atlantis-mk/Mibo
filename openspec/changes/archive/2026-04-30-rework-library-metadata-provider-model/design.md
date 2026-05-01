## Context

The current metadata runtime is split across three overlapping persistence layers. `metadata_provider_instances` contains executable providers but currently only supports TMDB. `metadata_profiles` stores stage-by-stage provider lists and also carries `local_only` and `fallback_enabled`, which makes one profile act like a runtime mode switch instead of a reusable preset. `library_metadata_profile_bindings` and `library_metadata_policies` then add another layer of library-specific overrides and legacy booleans such as `tmdb_enabled`, `local_metadata_enabled`, and `provider_priority_json`.

At the same time, local sidecar metadata already exists as scanner-discovered evidence in catalog scanner payloads. That evidence is useful, but it is not represented as a first-class metadata provider. The result is a split-brain model where remote metadata is chosen through provider instances while local metadata is activated through synthetic profiles and compatibility flags.

## Goals / Non-Goals

**Goals:**
- Make provider instances the only executable metadata sources, including a built-in local scan provider.
- Replace profile-plus-policy runtime resolution with a single per-library metadata strategy source of truth.
- Preserve sidecar metadata discovery while making local metadata refresh flow through the same provider execution model as remote metadata.
- Keep reusable metadata presets available, but make them optional templates instead of runtime sentinels.
- Provide a staged migration path that can absorb existing `migrated-default-online`, `migrated-default-local-only`, and legacy library policy rows without losing library behavior.

**Non-Goals:**
- Adding new remote metadata vendors beyond the existing TMDB runtime instances.
- Redesigning catalog governance UX beyond the metadata configuration surfaces touched by this change.
- Replacing the existing sidecar parsing rules or scanner evidence format except where needed to support local provider reuse.

## Decisions

### Decision: Represent local metadata as a system-managed provider instance

The system will introduce a built-in metadata provider type, `local_scan`, and guarantee that one locked system instance exists without manual configuration. This removes the need for `migrated-default-local-only`, `local_only`, and other special-case runtime shortcuts.

Alternative considered: keep `local_only` as a profile or library flag and continue treating local metadata as a mode. This was rejected because it preserves duplicate runtime semantics and keeps local metadata outside the provider execution contract.

### Decision: Introduce `library_metadata_strategies` as the runtime source of truth

Executable provider ordering and library-level metadata overrides will move into a dedicated per-library strategy record. The strategy will persist ordered provider instance IDs for `search`, `detail`, `image`, `people`, and `hierarchy` stages plus language overrides. Runtime reads in `internal/metadata` will resolve directly from this table.

Existing `metadata_profiles` will become reusable templates rather than runtime dependencies. Existing `library_metadata_profile_bindings` and `library_metadata_policies` will be treated as migration inputs and compatibility shims until the rollout is complete.

Alternative considered: repurpose `library_metadata_profile_bindings` to hold the full executable strategy. This was rejected because it retains profile-centric naming and keeps migration logic entangled with legacy binding semantics.

### Decision: Execute local metadata from scanner evidence instead of direct storage reads

The scanner will continue to discover and parse `.nfo` and `.json` sidecars, record parsed evidence, and apply safe scan-time hints. The new `local_scan` provider will consume that recorded evidence for strategy-driven detail refresh rather than re-reading storage directly from the metadata service.

This keeps sidecar parsing in one place, works consistently for both local and OpenList-backed libraries, and avoids duplicating storage access logic across scanner and metadata runtime layers.

Alternative considered: teach the metadata service to read sidecar files directly through storage providers. This was rejected because it duplicates scanner behavior, complicates OpenList access paths, and weakens the current evidence-first governance model.

### Decision: Enforce provider capabilities by provider type

Provider stage compatibility will be type-driven. In the first version, `tmdb` remains valid for `search`, `detail`, `image`, `people`, and `hierarchy`, while `local_scan` is valid for `detail` only. Libraries that want local-only metadata execution can leave search empty and use `local_scan` as the sole detail-stage provider.

Alternative considered: allow any provider type in any stage and fail later during execution. This was rejected because it moves configuration errors into runtime behavior and makes strategy validation unpredictable for users.

### Decision: Template application copies values into library strategy

Templates will remain useful as reusable presets, but applying a template to a library will copy provider ordering and language defaults into that library's executable strategy. Libraries will not retain a hard runtime dependency on a template row.

Alternative considered: keep libraries bound to a template at runtime and layer library overrides on top. This was rejected because it reintroduces the current indirection problem and makes it harder to understand the exact runtime strategy for a library.

### Decision: Roll out through a compatibility-first migration

The change will create the built-in `local_scan` provider, migrate every library to a concrete strategy row, repurpose profile APIs toward templates, and then remove legacy flags in phases. During the migration window, reads can backfill from old rows only when a strategy row is missing.

This keeps startup migrations deterministic and limits rollback risk to a narrow compatibility layer.

## Risks / Trade-offs

- [Risk] Library metadata behavior could drift during migration if legacy booleans and profile-derived behavior are translated incorrectly. -> Mitigation: write a deterministic migration that resolves each library's effective current behavior first, then persists an equivalent strategy row and verifies it through focused integration tests.
- [Risk] Local-only libraries will lose current manual search affordances because `local_scan` is detail-only in the first version. -> Mitigation: make unsupported actions explicit in API and UI, and defer any local candidate-search design to a follow-up change.
- [Risk] Repurposing `metadata_profiles` into templates changes operator mental models and could confuse existing settings flows. -> Mitigation: rename labels and descriptions in UI/API responses, mark system-managed local provider behavior clearly, and remove local-only toggles in the same rollout.
- [Risk] Scanner evidence may be incomplete for some items, which would make `local_scan` refresh a no-op. -> Mitigation: define no-op behavior clearly, preserve existing scan-time hint application, and surface provider provenance so operators can tell when local evidence was or was not used.

## Migration Plan

1. Add the new runtime persistence for library metadata strategies and extend metadata provider instances to support a locked system-managed `local_scan` type.
2. Bootstrap the built-in `local_scan` provider instance on startup if it does not already exist.
3. Migrate each library by resolving its effective current metadata behavior from profile binding, profile contents, and remaining legacy metadata policy fields, then writing an equivalent strategy row.
4. Repurpose profile management flows into template management flows and update library management APIs/UI to read and write strategies directly.
5. Update metadata execution to resolve provider order from library strategy rows and to execute local detail refresh through scanner evidence when `local_scan` is selected.
6. Remove long-term compatibility fields and startup normalization for `migrated-default-local-only`, `force_local_only`, `local_only`, `tmdb_enabled`, `tvdb_enabled`, `local_metadata_enabled`, and `provider_priority_json` after all runtime reads depend on strategies.

Rollback strategy: keep compatibility reads from legacy rows until strategy writes are proven stable. If rollout must be reversed before cleanup, runtime can continue resolving from old rows while ignoring partially migrated strategy rows.

## Open Questions

- Should `metadata_country_code` remain part of the library strategy if no current runtime provider consumes it, or should it be dropped as part of the same cleanup?
- Should metadata source provenance gain an explicit strategy or template snapshot field, or is provider-instance provenance plus payload evidence sufficient for this change?
- Should the storage name `metadata_profiles` be renamed in the database during this change, or should table renaming wait until the runtime migration has fully settled?
