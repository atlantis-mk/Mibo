## Context

Mibo's current metadata implementation is centered on a single resolved TMDB configuration plus partially reserved TVDB settings. Library policy persistence already hints at a multi-provider future with `tmdb_enabled`, `tvdb_enabled`, and `provider_priority_json`, but runtime execution in `internal/metadata` still directly calls TMDB-specific search and detail code. This creates three immediate problems: different libraries cannot choose different metadata methods, multiple TMDB tokens cannot be modeled as independently managed runtime instances, and future custom providers would need to be wired into TMDB-shaped code paths instead of a reusable library-scoped orchestration model.

The change is cross-cutting because it touches settings persistence, library policy resolution, metadata orchestration, catalog governance evidence, and admin APIs. The repo also has an important constraint from earlier discussions: metadata selection must remain library-aware inside `mibo-media-server`, rather than being delegated to an external aggregator service.

## Goals / Non-Goals

**Goals:**
- Introduce named metadata provider instances so multiple TMDB tokens and future provider definitions can coexist with separate enablement and health state.
- Introduce reusable metadata profiles that encode library-scoped metadata acquisition behavior, including stage ordering, fallback, local-only flows, and field application policy.
- Replace the current TMDB-specific execution path with an internal staged pipeline that resolves the effective library profile before searching, fetching, and applying metadata.
- Preserve the current catalog governance model by extending evidence, identity, and refetch behavior with provider-instance and profile provenance instead of replacing governance semantics.
- Provide an incremental migration path from global TMDB settings and library metadata policy flags to profile-backed configuration.

**Non-Goals:**
- Full TVDB feature parity in the same change; this change prepares the instance/profile model and keeps TVDB integration structurally possible.
- A remote plugin protocol or separate metadata service.
- A generic user-authored scripting engine for arbitrary metadata transformations.
- Reworking catalog governance UX beyond the data needed to expose profile-aware evidence and profile bindings.

## Decisions

### 1. Model provider configuration as named provider instances

The system will add first-class metadata provider instance persistence rather than extending `system_settings` with more singleton keys.

Rationale:
- Multiple TMDB tokens are naturally represented as separate instances instead of a hidden token pool inside one provider config.
- Instance-level enablement, health, and cooldown state can be persisted and surfaced to operators.
- Future custom providers can use the same storage model without schema branching per provider type.

Alternatives considered:
- Keep `system_settings` and store arrays of TMDB tokens in a JSON blob: rejected because it hides instance identity, complicates health state, and makes library targeting opaque.
- Store a TMDB-only token pool first and defer the general model: rejected because it would need a second migration when TVDB/custom providers arrive.

### 2. Bind libraries to reusable metadata profiles instead of embedding complex JSON directly in library policy rows

Libraries will reference a metadata profile, with optional per-library override data for narrow cases. Profiles define which provider instances and local-source stages are used for search, detail, image, people, hierarchy, and field application behavior.

Rationale:
- Different libraries need distinct metadata methods; profiles make those methods reusable and explicit.
- Operators can reason about named strategies like `movie-default`, `anime-custom`, or `local-only` instead of inspecting raw provider ordering blobs per library.
- The library policy layer remains concise while still allowing targeted overrides.

Alternatives considered:
- Replace `provider_priority_json` with a larger ad hoc library JSON field: rejected because it becomes unstructured policy sprawl and makes reuse across libraries poor.
- Resolve everything globally based on library type: rejected because multiple movie or TV libraries may still require different tokens, languages, or fallback behavior.

### 3. Implement metadata execution as a staged internal pipeline

`internal/metadata` will be refactored around provider-neutral pipeline stages such as candidate search, candidate ranking, detail fetch, field apply, image sync, people sync, and TV hierarchy sync. Each stage consults the effective profile.

Rationale:
- It matches the current product need: a library-specific metadata method, not just a provider list.
- It keeps orchestration inside the service while isolating provider-specific code behind interfaces.
- It allows local-only or mixed-source profiles to skip online stages cleanly.

Alternatives considered:
- A single provider orchestrator with one search-order list and one detail-order list: rejected because stage-specific behavior and field-specific sourcing would become condition-heavy and brittle.
- A separate metadata microservice: rejected because library-aware selection must remain inside this service boundary.

### 4. Keep catalog governance as the canonical write model and extend evidence provenance

The change will continue writing canonical values through catalog field-state and evidence APIs. `metadata_sources` provenance will be extended to identify provider instances and the effective profile used for the operation, while refetch prefers the original provider identity when available.

Rationale:
- Existing governance semantics for locks, needs-review, unmatched, and manual states are already valuable and should not be bypassed.
- Profile-driven execution still needs auditable evidence explaining why a field was chosen and which fallback path was used.
- Reusing existing apply paths lowers migration risk.

Alternatives considered:
- Create a new metadata result store separate from `metadata_sources` and `metadata_field_states`: rejected because it duplicates governance persistence and fragments debugging.

### 5. Migrate in compatibility phases instead of cutover in one release

The system will preserve current settings inputs while introducing profile-backed configuration. A default migrated provider instance and default migrated metadata profile will be synthesized from current TMDB settings and existing library metadata flags during rollout.

Rationale:
- The current app already depends on metadata settings and library policy endpoints.
- A phased migration reduces operator breakage and lets existing libraries keep matching behavior until explicitly reassigned.

Alternatives considered:
- Immediate removal of old settings and direct mandatory profile assignment: rejected because it makes rollout fragile and complicates upgrades for already initialized installs.

## Risks / Trade-offs

- [Profile model becomes too expressive too early] → Start with stage ordering, instance selection, fallback toggles, and limited field-policy controls; defer free-form rule engines.
- [Migration creates mismatched defaults between old library flags and new profile behavior] → Generate deterministic default instances/profiles from existing settings and expose migrated bindings in admin reads before enforcing writes through the new model.
- [Runtime complexity increases in metadata matching flows] → Isolate provider-neutral pipeline interfaces, keep TMDB as the first concrete provider implementation, and add focused tests around profile resolution and fallback decisions.
- [Evidence becomes harder to read if every fallback writes noisy payloads] → Standardize provenance fields and summarize only the selected provider instance, attempted fallback chain, and final confidence in source evidence.
- [TV hierarchy sync may drift if provider-specific season/episode semantics differ] → Keep rooted series matching as the invariant and require provider adapters to return normalized season/episode hierarchy structures before catalog writes.

## Migration Plan

1. Add persistence for provider instances, metadata profiles, and library profile bindings/overrides.
2. Introduce a migration that creates a default TMDB-backed provider instance from the existing resolved TMDB settings when configuration exists.
3. Introduce one or more default metadata profiles that mirror current behavior, such as a TMDB-first default profile and a local-only profile.
4. Backfill each library to a migrated default profile based on existing metadata policy state so current behavior remains stable.
5. Update metadata execution to resolve the effective profile first, but keep compatibility reads/writes for legacy settings until all API consumers can use the new model.
6. Shift admin APIs so new writes manage provider instances and metadata profiles, while old singleton metadata settings APIs become compatibility views or wrappers.
7. After validation, stop treating legacy provider flags as authoritative and use them only for migration or backward-compatible reads.

Rollback strategy:
- Keep legacy settings resolution code available during the migration window.
- Preserve migrated default profiles so libraries can continue to operate even if richer profile editing is disabled.
- If the new pipeline causes regressions, route libraries back to the migrated default profile and restore TMDB-first behavior without losing newly written evidence.

## Open Questions

- Should provider-instance health and cooldown state be persisted in the primary database, or is in-memory runtime state with periodic admin refresh sufficient for the first release?
- How much per-library override is needed in the first iteration beyond profile binding, language override, and local-only toggles?
- Should `catalog_external_ids` carry explicit provider-instance provenance, or is instance-level provenance in `metadata_sources` sufficient while external identity remains provider-scoped?
