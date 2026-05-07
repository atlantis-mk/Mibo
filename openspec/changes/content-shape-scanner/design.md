## Context

Mibo's source-first scanner already separates storage inventory from semantic catalog projection, but the current video classifier is still primarily file-first. Directory shape decisions and summaries are cached within a scan or materialization batch, yet high-cardinality directories can still repeat expensive filename signal extraction and directory summary work across `catalog_materialize_batch` chunks. A 1000-episode directory can be split into many materialization batches while each batch rebuilds enough context to decide the same directory shape again.

The desired scanner should treat directories as content groups before projecting files. The same source may contain mixed content: movie folders, movie version folders, movie collections, flat episode packs, season folders, and absolute-numbered series packs. Users should not need to choose movie/show/mixed library semantics up front, and low-confidence decisions must remain reviewable.

Existing constraints:
- Inventory facts must remain durable and independent of catalog classification.
- Fast scan-time classification must use storage listings, paths, filenames, sidecar names, and already available object metadata only.
- Expensive evidence such as ffprobe, hashes, artwork downloads, and external metadata providers belongs in asynchronous follow-up work.
- Existing scan exclusions, subtitle sidecars, artwork preselection, probe scheduling, metadata matching, missing cleanup, and catalog projection refresh must remain compatible.

## Goals / Non-Goals

**Goals:**
- Build a shape-first scanner that classifies directory content forms before materializing individual files.
- Support high-confidence fast paths for large mixed-naming episode directories, including `01.mkv`, `第001集.mkv`, `S01E001.mkv`, `01.2160p.HD国语中字[网站].mkv`, `Season 1/Season 2`, and absolute `001-1000` episode packs.
- Avoid repeated full movie-vs-episode classification for files that are already covered by a high-confidence directory plan.
- Persist directory content shape profiles and plan rules so unchanged directories and small deltas can be processed incrementally.
- Distinguish episode packs from movie collections and movie version folders using directory-level evidence rather than count alone.
- Preserve review and governance behavior for ambiguous directories and assignments.
- Deliver the full architecture in phases without leaving a temporary-only optimization as the final state.

**Non-Goals:**
- Do not reintroduce user-selected movie/show/mixed library types.
- Do not use ffprobe, media reads, external metadata lookups, or artwork downloads for shape planning.
- Do not redesign playback, metadata provider matching, or frontend catalog browsing beyond what is required to expose reviewable scanner outcomes.
- Do not require every ambiguous directory to be automatically classified; safe review-required outcomes are acceptable.

## Decisions

### Decision: Add a Content Shape Index rather than only an in-memory DirectoryPlan

The implementation will introduce persistent directory shape profiles and plan rules in addition to in-memory scan caches.

Rationale:
- In-memory plans fix duplicate work inside one scan, but do not help later rescans or small deltas.
- Persistent fingerprints allow unchanged directories to reuse existing plans.
- Plan rules allow a new episode in an existing `absolute_episode_pack` or `season_folder` to be assigned without recompiling the entire directory.

Alternatives considered:
- Only add filename signal caching. This reduces duplicate parsing but keeps the file-first architecture and does not solve batch-level directory reanalysis.
- Only add temporary directory plans. This improves one scan but misses incremental rescans and does not provide a durable correction target.

### Decision: Split shape processing into Profile, Plan, Assignment, and Materialization

The scanner will process directories in four conceptual stages:

```text
Directory snapshot
  -> DirectoryProfile
  -> DirectoryPlan
  -> File assignments
  -> Catalog materialization
```

`DirectoryProfile` contains cheap aggregate evidence such as video counts, attachment counts, episode marker coverage, numeric sequence coverage, year density, title uniqueness, shared title stem evidence, season directory hints, and sidecar hints.

`DirectoryPlan` determines the content shape and stores plan-level fields such as shape, confidence, series title, season number, numbering mode, plan rule, review state, and evidence summary.

`File assignments` map files to episode slots, movie works, movie versions, attachments, or review-required outcomes.

Rationale:
- Keeps expensive catalog writes separate from classification planning.
- Allows plan compilation once and DB writes in batches.
- Creates explicit points for review and future user overrides.

### Decision: Use cheap filename token profiles for directory planning

Directory profiling will use a lightweight token scan rather than full `classifiedMedia` generation. Token profiles should identify:
- Explicit episode markers: `S01E001`, `1x02`, `EP02`, `第02集`.
- Weak episode markers: leading numeric names such as `01.mkv` or `01.2160p...mkv`.
- Release/noise hints: quality, codec, source, subtitle, website noise, release group, audio tokens.
- Movie-like hints: title-year evidence and distinct title stems.
- Attachment role hints: trailer, sample, PV, featurette, behind-the-scenes, preview.

Rationale:
- Large directories need one cheap pass over filenames, not repeated title normalization and candidate generation.
- The same token profile can feed profile aggregation, plan assignment, and evidence records.

### Decision: Prefer directory-level high-confidence fast paths with safe fallbacks

The scanner will use high-confidence plans when directory evidence is strong. Examples:
- `episode_pack` or `absolute_episode_pack` when non-extra videos have strong sequence coverage and low independent-movie evidence.
- `season_folder` when the directory name or parent path provides season context and files map to episode slots.
- `movie_versions_folder` when main files share a normalized work stem and differ mainly by quality, edition, source, language, or release tokens.
- `movie_collection_folder` when files have distinct movie-like titles/year evidence and weak episode sequence evidence.

If evidence is medium confidence, the system may materialize with provisional or needs-review governance. If evidence is low confidence or conflicting, the system should preserve inventory and scanner decisions without creating unrelated movie or episode pollution.

Rationale:
- Speed comes from confidently bypassing per-file full classification in common large episode directories.
- Accuracy comes from recognizing movie collections and review-required groups instead of treating every large directory as a series.

### Decision: Persist fingerprints and classifier versions

Each directory profile will include a fingerprint built from provider, library/path scope, directory path, relevant scan policy/exclusion state, classifier version, child names, child stable identities when available, child modified/size evidence, and counts. The classifier version will invalidate old plans when shape logic changes.

Rationale:
- Prevents stale plans after algorithm changes.
- Lets unchanged directories skip profile/plan recompilation.
- Supports incremental reassignment for small additions and deletions.

### Decision: Keep catalog projection compatible with existing graph semantics

Plan materialization will continue to create or reuse existing catalog item types:
- Series, season, and episode hierarchy for episode plans.
- Movie items with multiple assets for movie version plans.
- Separate movie items or reviewable groups for movie collection plans.
- Asset roles for trailers, samples, featurettes, and extras.

Rationale:
- Avoids a catalog API cutover.
- Preserves playback, metadata enrichment, sidecar subtitles, artwork, missing cleanup, and projection refresh behavior.

### Decision: Implement in phases but require the complete architecture

Implementation should land in ordered phases:
1. Lightweight token profiles and in-memory directory plan compilation.
2. Plan-based materialization for high-confidence episode packs and season folders.
3. Persistent content shape index, fingerprints, and plan rules.
4. Incremental plan reuse for unchanged directories and small deltas.
5. Movie versions, movie collections, attachments, ambiguity handling, and review integration.
6. Cleanup of redundant file-first paths once plan coverage is validated.

Rationale:
- Reduces risk while ensuring the end state is not merely a temporary cache optimization.

## Risks / Trade-offs

- [Risk] Large movie collections could be mistaken for episode packs. -> Mitigate with movie collection score, year density, title uniqueness, low sequence coverage checks, and review-required fallback.
- [Risk] Pure numeric episode directories can conflict with numeric movie titles or disc files. -> Mitigate with minimum directory size, sequence coverage thresholds, release-token suppression, parent/title evidence, and low-confidence review.
- [Risk] Persisted plans can become stale after file moves, exclusions, or classifier changes. -> Mitigate with fingerprints, scan policy inputs, exclusion state, and classifier version invalidation.
- [Risk] New database tables increase migration and cleanup complexity. -> Mitigate with additive migrations, soft rollout, and cleanup paths tied to library/source deletion.
- [Risk] Review states may expose more ambiguous scanner outcomes initially. -> Mitigate by keeping high-confidence fast paths automatic and limiting review to conflicting/low-confidence groups.
- [Risk] Phased delivery could stop after the temporary fast path. -> Mitigate by making persistent index and incremental reuse explicit required tasks and acceptance criteria.

## Migration Plan

1. Add schema for directory shape profiles, plan rules, assignment metadata, classifier version, fingerprint, confidence, and review state without changing existing catalog tables.
2. Start writing profiles/plans alongside existing classification, initially using the plan only for high-confidence large episode directories.
3. Enable plan-based materialization for high-confidence episode and season plans while keeping file-first fallback for unsupported shapes.
4. Enable persistent plan reuse for unchanged directories and small deltas.
5. Expand plan compiler to movie versions, movie collections, attachments, and ambiguous groups.
6. Remove redundant repeated directory summary/classification work after parity tests pass.

Rollback strategy:
- Disable plan-based materialization via configuration or feature flag and fall back to existing file-first classification using the persisted inventory facts.
- Leave shape profile tables intact; they are additive and can be ignored by older logic.
- Keep catalog items reconciled through existing scanner identities so a rollback does not require deleting inventory or catalog records.

## Open Questions

- What exact thresholds should define high-confidence `episode_pack`, `absolute_episode_pack`, `movie_versions_folder`, and `movie_collection_folder` after evaluating real fixture samples?
- Should assignment metadata be stored as per-file rows, compact plan rules with exceptions, or both?
- Which UI surface should show directory-level review groups first: existing governance views, settings/library diagnostics, or a dedicated scanner review panel?
- How aggressively should small deltas reuse old plan rules when the directory fingerprint changes only by additions?
