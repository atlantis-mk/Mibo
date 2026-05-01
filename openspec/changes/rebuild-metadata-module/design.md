## Context

The backend currently stores metadata-related state across catalog, scanner, settings, and metadata packages. The data model already contains useful primitives such as provider instances, library metadata strategies, metadata sources, field states, external IDs, stable identities, image candidates, and TV hierarchy descendants. The problem is that execution is not centralized: scans write evidence and enqueue matching, worker matching has a TMDB-specific configured gate, metadata methods choose only one provider per stage, local sidecar evidence is modeled as a detail provider fallback, and field ownership rules are inconsistently applied.

The rebuild should keep the durable catalog model and existing migration investments where possible, but replace the ad hoc execution flow with a single metadata operation pipeline. The pipeline becomes the boundary between scanner evidence, provider runtime configuration, governance, and catalog write models.

The major stakeholders are automated scan enrichment, manual governance actions, library metadata settings, TV hierarchy completion, and catalog read/projection consistency.

## Goals / Non-Goals

**Goals:**
- Define one operation model for automatic match, refetch, manual candidate apply, and local evidence apply.
- Make library metadata strategy the only source of runtime provider ordering and matchability.
- Execute provider stages in configured order with real fallback and recorded attempt outcomes.
- Separate scanner evidence capture from local evidence application.
- Apply catalog fields through a single field ownership path that respects locked/manual fields and records source attribution.
- Preserve the distinction between stable catalog identities and provider-facing external IDs.
- Keep rooted TV hierarchy synchronization, but make hierarchy execution use normalized provider outputs and operation evidence.
- Refresh catalog projections once per operation scope.
- Support TMDB, MetaTube, and local scanner evidence without making TMDB configuration a global gate.

**Non-Goals:**
- Replacing the catalog kernel or scanner classification model wholesale.
- Implementing full TVDB execution support beyond preserving the provider capability boundary.
- Building a frontend redesign for governance UI, except for API result shape compatibility needed by existing screens.
- Removing existing metadata tables unless a migration proves they are redundant and safely replaceable.
- Supporting arbitrary third-party provider plugins in this change.

## Decisions

### Decision: Introduce `MetadataOperation` as the execution boundary

All metadata entrypoints should call one orchestrator with an operation request:

```text
MetadataOperationRequest
  operation: match | refetch | manual_apply | local_apply
  origin_item_id
  target_item_id resolved by operation
  optional manual candidate / external id / provider preference
  force flags from user action
```

The operation returns a unified result:

```text
MetadataOperationResult
  operation
  origin_item_id
  target_item_id
  target_type
  status: applied | no_candidate | needs_review | skipped | failed
  governance_status
  plan summary
  provider attempts
  selected candidate
  applied fields
  skipped fields
  affected item ids
  warnings
```

Rationale: the current methods `MatchCatalogItem`, `RefetchCatalogItem`, `SearchCatalogCandidates`, and `ApplyCatalogCandidate` duplicate target resolution, provider selection, detail fetching, apply behavior, and result reporting. A single operation model makes behavior testable and visible.

Alternative considered: keep existing methods and gradually share helpers. This preserves less churn but leaves the user-visible mental model fragmented and does not solve fallback/provenance consistently.

### Decision: Resolve an immutable execution plan per operation

Each operation resolves a `MetadataExecutionPlan` from the target library strategy at start:

```text
MetadataExecutionPlan
  library_id
  strategy_id
  preferred metadata/image language
  search providers[]
  detail providers[]
  image providers[]
  people providers[]
  hierarchy providers[]
  local evidence policy
```

The plan is immutable for the operation and is written into operation evidence. Runtime should not consult legacy global provider enablement or TMDB settings except through resolved provider instance configuration.

Rationale: library strategies already exist and should be the executable source of truth. Resolving once avoids mid-operation behavior changes and gives governance a stable explanation.

Alternative considered: resolve provider lists lazily per stage. This is simpler initially but can produce inconsistent behavior if settings change mid-operation and makes debugging harder.

### Decision: Replace `CatalogMatchingConfigured` with strategy matchability

Worker execution should not skip `match_catalog_item` based on global TMDB API key. Instead, the operation should inspect the target library plan:

- `match` is runnable if search has at least one operational search provider, or if local evidence policy allows local candidate/application for the target.
- `refetch` is runnable if the item has a provider identity and an allowed operational detail provider for that identity, or local evidence can satisfy the refetch request.
- `manual_apply` is runnable if the selected candidate's provider has an allowed detail provider or the candidate carries enough detail payload.

Rationale: MetaTube-only and local-evidence-only strategies are valid designs and must not be blocked by TMDB configuration.

Alternative considered: broaden `CatalogMatchingConfigured` to check all provider instances globally. This still fails for library-specific strategies and would allow jobs to run for libraries that cannot actually match.

### Decision: Model provider execution as attempts, not selection

Provider stages should iterate ordered provider instances and record each attempt:

```text
ProviderAttempt
  stage
  provider_instance_id/name/type
  outcome: success | no_result | skipped_unavailable | skipped_unsupported | failed_retryable | failed_terminal
  status_code / error summary
  candidate_count
  selected
```

Search should collect normalized candidates across providers until policy decides enough candidates exist. Detail should try preferred provider provenance first for refetch, then configured fallbacks. Image, people, and hierarchy should use either selected detail payload when sufficient or stage providers when they become independently implemented.

Rationale: provider fallback is already promised by specs, but current code mostly selects the first provider. Attempts are the missing audit trail and execution primitive.

Alternative considered: only store final selected provider and fallback summary JSON. This is compact but cannot explain why primary providers were skipped or failed.

### Decision: Normalize provider outputs before applying

Provider-specific TMDB and MetaTube responses should be converted into internal normalized structs:

```text
MetadataCandidate
MetadataDetail
MetadataImages
MetadataPeople
MetadataHierarchy
```

Catalog apply code should consume normalized outputs, not TMDB or MetaTube response structs directly. Provider clients remain responsible for HTTP, auth, error mapping, and provider-specific response parsing.

Rationale: current application logic is TMDB-shaped, making MetaTube and future providers appear as special cases. Normalized outputs let governance and field application become provider-neutral.

Alternative considered: keep applying provider-specific detail responses with interface methods. This reduces conversion work but keeps provider semantics leaking into catalog field logic.

### Decision: Separate scanner evidence from local evidence application

Scanning remains responsible for detecting files, sidecars, artwork, and external IDs and recording scanner evidence. It should not be responsible for running provider match logic. A local evidence stage reads scanner evidence and can produce local candidates or apply supported fields under the same operation contract.

`local_scan` may remain as a system-managed provider instance for compatibility and provenance, but execution should treat it as a local evidence executor with explicit capabilities, not as an online provider.

Rationale: scanner evidence and metadata application are different responsibilities. Keeping them separate reduces scan-side mutations and makes local-only operation behavior explicit.

Alternative considered: remove `local_scan` provider entirely. That is conceptually cleaner, but existing specs, settings, and provenance already reference it; retaining it as a system-managed executor reduces migration risk.

### Decision: Apply fields through `FieldApplicationPolicy`

Metadata operations should build field changes and pass them through a single apply layer:

```text
FieldChange
  item_id
  field_key
  value
  source_id
  confidence
  apply_mode: automated | manual | scanner | system
```

The policy determines whether to apply, skip, lock, or require review based on existing `metadata_field_states`, item governance, field lock state, and operation type. Applied and skipped fields are returned in the operation result.

Rationale: field-level governance already exists but is not uniformly enforced. A policy object makes rules visible and testable.

Alternative considered: keep using `catalog.ApplyField` directly everywhere. This preserves existing behavior but does not centralize skipped-field reporting, source attribution, or governance transitions.

### Decision: Preserve both external IDs and identities with explicit ownership

The rebuilt module should use:

- `catalog_identities` for stable reconciliation and deduplication keys, including scanner, sidecar, manual, and provider identities.
- `catalog_external_ids` for provider-facing IDs used by refetch, display, and manual governance.

Provider apply should write both only when both semantics are present. Scanner sidecar IDs should seed provider-facing external IDs and may also create identities when they are strong enough to support reconciliation.

Rationale: the two tables serve different purposes but are currently written together without a clear rule. Keeping both avoids risky data loss while clarifying behavior.

Alternative considered: merge both into one identity table. This would simplify reads but requires a larger migration and risks confusing provider display IDs with scanner reconciliation keys.

### Decision: Make TV hierarchy a stage output

TV series matching remains rooted at the series item. Provider clients normalize season and episode data into `MetadataHierarchy`. The hierarchy apply layer then creates or updates season and episode catalog descendants, preserving local asset links and mismatch safeguards.

Rationale: rooted TV behavior is already required and tested, but it is TMDB-specific today. Treating hierarchy as normalized stage output makes provider participation explicit.

Alternative considered: keep TV hierarchy in TMDB detail apply only. This is less work but prevents a coherent provider pipeline and keeps episode-triggered operations hard to explain.

### Decision: Add operation evidence without replacing metadata sources immediately

Use existing `metadata_sources` for provider/local payload evidence and add operation-level evidence either as a new table or as structured JSON attached to sources. Preferred shape:

```text
metadata_operations
  id, operation, origin_item_id, target_item_id, library_id,
  status, governance_status, plan_json, attempts_json,
  selected_candidate_json, applied_fields_json, warnings_json,
  started_at, finished_at
```

Existing `metadata_sources` remain the evidence records for payloads that produced applied fields. `metadata_operations` explains the full execution.

Rationale: metadata sources explain payload provenance but not complete operation attempts, skipped providers, or field decisions. A separate operation record improves governance and debugging.

Alternative considered: pack all operation evidence into `metadata_sources.fallback_summary_json`. This avoids a table but overloads a field already used for fallback summary and makes operation-level queries difficult.

### Decision: Refresh projections once per operation scope

The apply layer should collect affected item IDs and refresh projections once at the end. For TV hierarchy operations, the scope is the series root and affected descendants.

Rationale: current methods refresh projections inside `ApplyField`, `SetExternalID`, `RecordMetadataSource`, and again after detail apply. Operation-level refresh reduces duplicate work and makes partial failure behavior clearer.

Alternative considered: keep immediate refreshes for safety. This minimizes refactor risk but increases write amplification and makes operation transactions harder to reason about.

## Risks / Trade-offs

- **Risk: Scope is large and crosses many packages** -> Implement in phases, keep old method names as adapters until callers migrate, and add focused regression tests before deleting old code paths.
- **Risk: Operation evidence table adds schema complexity** -> Reuse existing metadata source tables for raw payloads and keep operation rows compact, JSON-backed, and append-only initially.
- **Risk: Field policy may change subtle behavior** -> Start by encoding current behavior as tests, especially locked/manual/matched/needs-review cases, then tighten rules intentionally.
- **Risk: Provider fallback can multiply HTTP calls** -> Cap candidate counts, stop on high-confidence matches where policy allows, and preserve per-provider timeouts.
- **Risk: TV hierarchy operations can affect many descendants** -> Keep rooted operations transactional where practical, track affected IDs, and test missing/unaired/local-mismatch cases.
- **Risk: Existing UI expects legacy result shapes** -> Provide compatibility adapters for current HTTP responses while exposing the richer operation result for governance.
- **Risk: MetaTube-only behavior may expose gaps hidden by TMDB gate** -> Add explicit MetaTube-only automated match tests and clear unsupported-stage errors.

## Migration Plan

1. Add new operation types, normalized structs, provider attempt model, and execution-plan resolver without changing existing behavior.
2. Add operation evidence persistence, or a temporary in-memory/result-only implementation if schema migration needs to be staged.
3. Route `MatchCatalogItemWithResult` through the new orchestrator for movie TMDB first, preserving current public result semantics.
4. Add strategy matchability and remove the worker's TMDB-only skip gate.
5. Move MetaTube movie match/detail into the new provider executor.
6. Move local sidecar evidence application into the local evidence executor.
7. Move refetch and manual apply into the orchestrator.
8. Move TV hierarchy sync into normalized hierarchy apply.
9. Convert field writes to operation-level field policy and source attribution.
10. Consolidate projection refreshes at operation end.
11. Remove obsolete helper paths only after tests cover migrated behavior.

Rollback should keep existing tables intact. If operation evidence tables are added, rollback can ignore them while compatibility adapters preserve old method signatures during the migration window.

## Open Questions

- Should operation evidence be persisted in a new `metadata_operations` table in the first implementation phase, or introduced after behavior is unified?
- Should search collect candidates from all providers, or stop after the first provider returns a high-confidence result?
- Should local sidecar evidence be allowed to mark an item `matched`, or should it produce a distinct `local_matched`/manual-equivalent status?
- How much legacy HTTP response compatibility is required for existing frontend screens during the transition?
- Should provider attempt history be retained indefinitely, or pruned by age/count per item?
