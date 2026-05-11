## Context

`progressive-media-classification-browse` made movie-versus-series collapse more conservative, but it intentionally stopped short of strengthening same-metadata matching. The scanner can now leave weak or conflicting items unresolved, yet it still lacks a dedicated phase that answers whether sibling resources belong to one canonical work, one episode, one version group, or should remain isolated. The repo already uses a metadata/resource model with resource links, projection refresh, and review-required evidence, so this change should deepen that existing model instead of introducing a parallel ownership path.

The user also wants file `md5` to participate in matching so the same media discovered from different sources can be recognized quickly. That requirement adds a file-level identity signal that must help matching without regressing the source-first fast path.

Implementation reconnaissance notes:
- File hash data already has a durable landing point in `database.InventoryFile.HashesJSON`.
- Scan ingest already populates `catalogScanArtifact.HashesJSON` from storage `HashInfo`, so same-metadata matching can consume existing `md5` values without adding a new schema first.
- Current movie linking is centralized in `linkCatalogScanMovieResourceToMetadata(...)` and `findOrCreateCatalogScanMovieMetadataItem(...)`.
- Current episode linking is centralized in `linkCatalogScanEpisodeResourceToMetadata(...)` and already reuses episode metadata by series/season/episode tuple.
- Browse replacement behavior already exists once `resource_metadata_links` and projections are refreshed, so this change can likely ride the existing organizing-entry upgrade path.

## Goals / Non-Goals

**Goals:**
- Add a sibling-matching phase after work-group classification that decides whether resources belong to the same movie or episode metadata identity.
- Distinguish canonical work matching from version-trait matching so alternate encodes and cuts can share one metadata identity without collapsing unrelated works.
- Treat file `md5` as a strong file-level identity anchor for cross-source reuse and duplicate detection.
- Keep automatic linking conservative: strong same-metadata matches may auto-link, while weak or conflicting matches remain unresolved or review-required.
- Preserve progressive browse behavior so organizing entries upgrade cleanly when accepted sibling links become available.

**Non-Goals:**
- Rebuild metadata governance, merge/split APIs, or the broader workflow DAG.
- Require synchronous content hashing in the existing fast-path scan loop.
- Solve every complex pack, anthology, concert, or anime edge case in the first pass.
- Redefine how provider enrichment populates descriptive metadata fields.

## Decisions

### Decision: Add sibling matching as a post-classification phase
The implementation will keep `classifyWorkGroup(...)` focused on movie-versus-series decisions and add a separate same-metadata matching phase that consumes accepted movie or series outcomes. This new phase will decide whether a resource should link to an existing metadata identity as a primary/version resource, an episode version resource, a supplemental resource, or remain unresolved.

Alternatives considered:
- Extend the work-group classifier to also decide resource linking. Rejected because it would mix type classification with metadata identity matching and make the model harder to reason about.
- Continue relying on existing link-or-create movie helpers. Rejected because they do not model sibling conflict, version traits, or cross-source file identity clearly enough.

### Decision: Split canonical identity from version identity
Matching will first establish whether resources belong to the same canonical movie or episode identity, then separately infer whether they differ only by version traits such as quality, source, codec, or edition. Version evidence is not allowed to create canonical identity by itself.

Alternatives considered:
- Treat version traits as part of canonical matching. Rejected because this risks overfitting title strings and splitting obvious alternate encodes into duplicate metadata items.

### Decision: Use file `md5` as a strong but non-blocking identity anchor
`md5` will be treated as strong file-level evidence when it is already available, and missing hashes will not block initial classification or browse visibility. If `md5` is computed asynchronously later, the system should re-run sibling matching and projection refresh for affected resources.

Alternatives considered:
- Require hash computation before any sibling match. Rejected because it would slow the source-first pipeline and violate existing fast-path guarantees.
- Ignore content fingerprints and rely only on title/provider identity. Rejected because cross-source duplicates with inconsistent naming remain unnecessarily weak without a file-level anchor.

Concrete implementation direction:
- First pass should consume existing `md5` from `artifact.HashesJSON` and linked `inventory_files.hashes_json`.
- First pass does not require adding new async hashing infrastructure if storage providers already supply hashes.
- If a later follow-up adds asynchronous fingerprint completion, it should be additive and only trigger rematch for affected resources.

### Decision: Prefer strong local or provider identity over weak title-only reuse
Sidecar external IDs, supported provider IDs, explicit episode tuples, and file `md5` will outrank normalized title/year matching. Title-only or title-plus-year-only matches remain candidates unless reinforced by stronger evidence or an already-accepted work group.

Alternatives considered:
- Continue allowing title/year to auto-link same-metadata siblings. Rejected because it creates the same false-positive collapse risk this change is meant to reduce.

### Decision: Isolate supplemental media from primary/version auto-linking
Files classified as samples, trailers, previews, featurettes, or other extras will not be auto-linked as primary or version siblings of the main metadata identity. They may continue to exist as separate resource facts and can be governed later if needed.

Alternatives considered:
- Treat extras in the same folder as versions of the main title. Rejected because it pollutes primary resource sets and increases false automatic merges.

## Risks / Trade-offs

- [Incorrect auto-merge is more harmful than unresolved visibility] -> Mitigation: default to conservative outcomes and only auto-link with strong evidence.
- [Hash-aware rematch introduces asynchronous state changes] -> Mitigation: keep matching idempotent, store evidence on links, and refresh projections only for affected resources.
- [Version trait parsing could become too broad or too narrow] -> Mitigation: start with a small explicit trait set and expand through focused regression tests.
- [Cross-source duplicates may already exist as separate resources or metadata rows] -> Mitigation: prefer additive matching plus governance-safe review-required outcomes rather than destructive auto-merges.

## Migration Plan

1. Add sibling match outcomes and evidence handling without removing current unresolved browse behavior.
2. Teach the library scan/materialize path to consult sibling matching after accepted work-group classification.
3. Consume existing `md5` values from `artifact.HashesJSON` and `inventory_files.hashes_json` when available and leave missing hashes as non-blocking.
4. Trigger projection refresh when accepted sibling matches attach resources to existing metadata identities.
5. If asynchronous `md5` enrichment is added during implementation, re-run sibling matching only for affected resources.
6. Roll back by disabling the sibling auto-link gate and returning to current work-group-only behavior if duplicate merges or projection churn become unacceptable.

## Open Questions

- Current scan paths already persist `HashesJSON`; remaining question is whether target storage providers reliably populate `md5` often enough for the first pass to be valuable without a new async fingerprint job.
- Should identical `md5` across two resources prefer reusing the same resource row, or only reuse the same metadata identity while preserving separate resource-library links?
- Which edition/version traits should be treated as safe first-pass variants versus review-required conflicts?
