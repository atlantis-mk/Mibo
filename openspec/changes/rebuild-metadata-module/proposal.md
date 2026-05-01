## Why

The current metadata subsystem has accumulated scanner evidence, provider configuration, catalog governance, manual matching, TV hierarchy completion, and projection refresh behavior without a single executable model. This makes provider settings hard to reason about, creates TMDB-specific gates in supposedly provider-agnostic flows, and leaves field ownership, fallback, and provenance semantics scattered across scan, metadata, settings, and catalog code.

This change rebuilds metadata around an explicit operation pipeline so automated match, refetch, manual apply, local evidence apply, image sync, people sync, and TV hierarchy sync all share one strategy resolution, provider execution, decision, and field-application model.

## What Changes

- Introduce a unified metadata operation pipeline with explicit operation types, execution plans, provider attempts, normalized candidates, decisions, and apply results.
- Replace ad hoc provider selection with stage-aware execution that honors ordered provider instances, operational availability, fallback attempts, and provider capabilities.
- Remove the global TMDB-only automatic matching gate and determine matchability from the target library's executable metadata strategy.
- Treat scanner/sidecar evidence as a first-class local evidence source that can seed provider identity, produce local candidates, or apply local fields without masquerading as an online search provider.
- Standardize metadata writes through field ownership rules so automated providers update unlocked fields, preserve locked/manual fields, and record source attribution for applied values.
- Clarify the boundary between stable catalog identities and provider-facing external IDs.
- Make automated match, refetch, manual candidate application, and local evidence application use the same operation result contract for API responses and governance visibility.
- Preserve rooted TV hierarchy behavior while moving season and episode completion into the shared pipeline and provider-stage semantics.
- Consolidate projection refresh so metadata operations refresh affected catalog read models once per operation scope rather than repeatedly during intermediate writes.
- **BREAKING**: Internal metadata service APIs and job execution semantics will change; callers that depend on TMDB-specific matching behavior, implicit local_scan detail fallback, or per-method result shapes must be migrated to the new operation contract.

## Capabilities

### New Capabilities
- `metadata-operation-pipeline`: Defines the unified metadata operation lifecycle, operation types, execution plans, provider attempts, normalized candidates, decisions, apply results, and projection refresh semantics.

### Modified Capabilities
- `metadata-provider-runtime-model`: Provider execution changes from first-provider selection to ordered stage execution with real fallback and local evidence participation.
- `metadata-provider-instances`: Provider availability and capability behavior becomes part of the operation execution contract rather than only configuration validation.
- `library-metadata-profiles`: Library strategies become the sole matchability gate and execution source of truth for automated and manual metadata operations.
- `catalog-metadata-governance`: Field ownership, provenance, governance transitions, and operation evidence are standardized across match, refetch, manual apply, and local apply.
- `sidecar-metadata-files`: Sidecar evidence is separated into scanner evidence capture, candidate seeding, and local apply behavior under the shared metadata pipeline.
- `tv-hierarchy-metadata-completion`: Rooted TV hierarchy completion is executed through normalized hierarchy stage outputs and shared operation results.
- `metatube-metadata-provider`: MetaTube participates in the unified operation pipeline without requiring TMDB configuration for automated movie matching.

## Impact

- Backend packages affected: `internal/metadata`, `internal/settings`, `internal/catalog`, `internal/library`, `internal/worker`, and related HTTP API handlers.
- Database impact: existing tables should be reused where possible, but new operation/attempt evidence tables or JSON evidence fields may be added to preserve execution history and applied-field attribution.
- API impact: metadata match/refetch/manual apply/governance endpoints will return a unified operation result; legacy response shapes may need bounded compatibility adapters.
- Job impact: `match_catalog_item` execution will resolve strategy per item/library and will no longer use a global TMDB-key gate.
- Migration impact: existing provider instances, profiles, library strategies, metadata sources, field states, external IDs, and identities must be migrated or interpreted without data loss.
- Test impact: provider strategy, fallback, TMDB-only, MetaTube-only, local-evidence-only, locked field, manual field, and TV hierarchy scenarios require focused regression coverage.
