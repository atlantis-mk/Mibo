## Context

Mibo already moved toward a global metadata/resource model, durable inventory facts, filename signal caching, content-shape planning, path-tree work grouping, and same-metadata sibling matching. Those pieces improved local cases, but the architecture still makes final identity decisions inside scan/materialization helpers that operate on one file or one local group at a time. That means the code now has several partially overlapping decision engines: content-shape assignments, path-tree assignments, catalog scan movie/episode linking, and sibling matching.

The desired product model is global and resource-first: a `MetadataItem` is the canonical work or episode identity, while multiple `Resource` records can represent playable copies, encodes, editions, extras, trailers, or multi-episode/multi-part media across one or more libraries. Recognition should therefore be a resolver problem over a complete set of candidates and evidence, not a side effect of whichever file happens to materialize first.

The project is still development-reset oriented, so this design intentionally replaces the old recognition architecture instead of adding compatibility shims. Existing parsers and evidence extractors may survive, but final identity decision paths that compete with the resolver must be removed or converted into evidence providers.

## Goals / Non-Goals

**Goals:**

- Make scanning order-independent by separating inventory collection, manifest construction, identity resolution, and materialization.
- Build a recognition manifest that captures work candidates, episode candidates, playable resource candidates, sidecar evidence, path evidence, filename signals, variant traits, hash evidence, conflicts, and alternatives.
- Use one deterministic resolver to create or reuse `MetadataItem`, `Resource`, `ResourceMetadataLink`, and projection inputs.
- Distinguish canonical identity from resource variant identity, edition/cut identity, duplicate binary evidence, and supplemental roles.
- Preserve the fast path by keeping remote metadata, ffprobe, artwork, and content hashing out of synchronous scan recognition.
- Make manual corrections first-class resolver rules that override automatic evidence and prevent repeated prompts.
- Delete or demote replaced content-shape/path-tree/sibling-matching final decision code so there is no long-term two-track maintenance.

**Non-Goals:**

- Preserve existing development database scan results or legacy duplicate metadata rows.
- Introduce a user-selected movie/show library type.
- Require remote provider matches before local browse visibility exists.
- Solve every ambiguous anthology, anime absolute-numbering, concert, sports, or extras-pack case automatically in the first pass.
- Keep old scan-link helpers as fallback final decision engines once the resolver is wired.

## Decisions

### Decision: Replace scan-time materialization with a four-stage pipeline

The scanner will run as:

```text
Inventory pass
  -> Recognition manifest build
  -> Identity resolver
  -> Idempotent materializer
```

The inventory pass records physical file facts, sidecar associations, and cheap file signals. The manifest builder groups those facts into candidate objects. The resolver decides accepted, provisional, review-required, or blocked outcomes. The materializer applies only accepted/provisional resolver output to the resource/metadata graph.

Alternative considered: keep current materialization and add a smarter `matchMovieSibling`. Rejected because it still lets a per-file write path create identities before the complete context is known.

### Decision: Recognition manifest is the canonical scanner output

The manifest is the scanner-owned representation of semantic intent. It contains stable candidate keys, candidate type, source scope, affected inventory files, sidecars, normalized work keys, episode slots, resource shapes, variant traits, edition traits, evidence, conflicts, alternatives, and resolver state.

It can be persisted in dedicated tables or in a compact JSON-plus-index model, but implementation must favor queryable decision state over opaque evidence blobs where governance or tests need direct access. Development reset allows replacing existing `classification_decisions` or content-shape state if that reduces duplication.

Alternative considered: use existing `content_shape_*` tables as the manifest. Rejected as the primary model because they are directory-plan oriented, not global identity resolver oriented. Their concepts may be reused if renamed and narrowed.

### Decision: Use candidate types that match product semantics

The resolver will model at least these candidate concepts:

- Work candidate: movie, series, season, episode, or future media identity.
- Playable resource candidate: single-file, multi-part, multi-episode, or related video resource.
- Variant candidate: encode/source/quality/codec/audio/subtitle/HDR/release-group/container traits.
- Edition candidate: theatrical, director cut, extended cut, unrated, remaster, or other cut-level identity.
- Supplemental candidate: trailer, sample, extra, featurette, preview, behind-the-scenes, or other non-main role.
- Duplicate binary evidence: identical hash/stable identity indicating the same file content across resources.

Alternative considered: keep only `primary` and `version` link roles. Rejected because copies, encodes, editions, and extras need different resolver rules and UI behavior.

### Decision: All recognizers become evidence providers

Filename parsing, sidecar parsing, directory shape analysis, path-tree grouping, existing rules, hashes, and external IDs will produce evidence into the manifest. None of them can directly create or link metadata except through resolver decisions.

The implementation should keep a single filename signal parser and any reliable sidecar parser, but old helpers that call `LinkResourceToMetadata` or create `MetadataItem` from local classifications must be removed from scan flow.

Alternative considered: keep path-tree work grouping as a high-priority materialization override. Rejected because override paths are exactly what make identity behavior hard to reason about.

### Decision: Resolver uses explicit gates, not accumulated vibes

Automatic acceptance requires no blocking conflict and one of the supported identity gates:

- Same supported external identity for the target metadata type.
- Same sidecar-provided canonical identity or provider identity.
- Same series/season/episode tuple with compatible series identity evidence.
- Same normalized movie title and year plus compatible work context and variant/edition evidence.
- Same binary hash only as duplicate binary evidence, not as proof of descriptive metadata if it conflicts with stronger identity evidence.
- User-approved resolver rule.

Conflicts block auto-merge when external IDs disagree, movie/episode types disagree, years disagree materially, episode tuples disagree, or two high-confidence candidates compete within the same file set.

Alternative considered: use a single weighted score threshold. Rejected as the only mechanism because some conflicts must be hard blockers regardless of score.

### Decision: Materialization is idempotent and resolver-owned

Materialization creates or reuses global `MetadataItem` rows and `Resource` rows from stable resolver keys. `ResourceMetadataLink` records carry role, confidence, evidence, review state, segment index, edition/variant context, and source. Projection refresh reads from the resulting graph.

The materializer must be safe to rerun after parser version changes, sidecar updates, async hash/probe completion, metadata enrichment, or user correction. It must update resolver-owned links and projections without duplicating metadata rows.

Alternative considered: keep scanner-owned metadata rows and reconcile after enrichment. Rejected because duplicate cleanup becomes a permanent subsystem.

### Decision: User corrections produce resolver rules

Manual merge, split, classify-as-versions, classify-as-independent, classify-as-series, and attachment corrections must persist source/path/file scoped resolver rules. Rules are evidence with highest local precedence and must be visible in governance responses.

Alternative considered: patch current links directly. Rejected because rescans would rediscover the same ambiguity and require repeated manual work.

### Decision: Remove replaced code as part of the change

This change must include cleanup, not just additive implementation. Code that remains must be categorized as one of:

- Inventory fact collection.
- Signal/sidecar/evidence extraction.
- Manifest building.
- Resolver decision logic.
- Materialization/projection.
- Governance/rule handling.

Anything outside those categories that performs old final recognition must be deleted or rewritten. Tests that only prove the old architecture are removed or replaced with resolver tests.

Alternative considered: leave old paths disabled by flags. Rejected because the user explicitly wants no legacy maintenance burden during development.

## Risks / Trade-offs

- [Risk] Large refactor can temporarily break scan visibility. Mitigation: implement vertical slices with inventory-only visibility first, then resolver materialization for movies, then episodes, then extras and advanced shapes.
- [Risk] Resolver manifest tables add schema complexity. Mitigation: make them development-reset oriented, index only stable keys and governance-critical fields, and keep evidence JSON for diagnostics.
- [Risk] Removing old fallback paths may reduce visibility for ambiguous files. Mitigation: publish inventory-backed skeleton/review states without creating wrong metadata.
- [Risk] Hard conflict rules can under-merge valid edge cases. Mitigation: user resolver rules can override scoped cases with audit evidence.
- [Risk] Async enrichment can churn projections. Mitigation: resolver reruns only affected candidates and projection refresh is keyed by affected metadata/resource IDs.
- [Risk] Existing OpenSpec specs describe old assets/catalog wording. Mitigation: delta specs in this change update behavior and tasks include cleanup of stale terminology in touched tests/code.

## Migration Plan

1. Add manifest/resolver schema and in-memory types behind the development reset assumption.
2. Create inventory-to-manifest builders using existing inventory files, file signals, sidecars, and path context.
3. Implement resolver gates, conflicts, rule precedence, and decision persistence.
4. Implement idempotent materialization into `MetadataItem`, `Resource`, `ResourceMetadataLink`, library links, and projection refresh inputs.
5. Wire scan jobs to stop direct metadata linking and instead build/resolve/materialize manifests.
6. Port movie version, movie collection, standard episode, multi-episode, and supplemental cases to resolver tests.
7. Remove obsolete final-decision code from content-shape/path-tree/sibling-matching/catalog scan helpers and delete obsolete tests.
8. Reset local development data and rescan demo media.

Rollback is not expected after local data reset. Before reset, rollback means reverting the change branch. During implementation, incomplete slices can keep inventory-only visibility while resolver materialization is finished.

## Open Questions

- Should the manifest be stored as normalized tables from day one, or as `recognition_candidates` plus typed JSON evidence with selected indexed columns?
- Should edition/cut be a separate table or fields on resource metadata links in the first implementation?
- What exact UI surface should expose resolver review groups first: existing governance detail, library settings, or a new recognition review view?
