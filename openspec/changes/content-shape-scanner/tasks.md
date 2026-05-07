## 1. Data Model And Rollout Controls

- [x] 1.1 Add additive database models and migration coverage for directory shape profiles, directory plans, assignment metadata or plan-rule exceptions, classifier version, fingerprint, confidence, evidence JSON, review state, and deleted-scope handling.
- [x] 1.2 Add service-level constants/configuration for the content shape classifier version, plan reuse thresholds, high-confidence thresholds, medium-confidence review thresholds, large-directory thresholds, and a rollout flag that can disable plan-based materialization.
- [x] 1.3 Add lifecycle cleanup or ignore behavior so profiles, plans, and assignments tied to deleted media sources, libraries, or library paths do not participate in future scans.
- [x] 1.4 Add repository/service helpers for loading, saving, invalidating, and reusing shape profiles and plans by library, provider, root path, directory path, fingerprint, and classifier version.

## 2. Filename Token Profiles

- [x] 2.1 Implement cheap filename token profile extraction for explicit SxE markers, Chinese episode markers, EP tokens, leading numeric episode candidates, release hints, codec/audio/source/subtitle hints, website noise, title-year evidence, and attachment role hints.
- [x] 2.2 Ensure release, codec, audio, quality, website, and attachment tokens suppress weak false positive episode numbers and title words during token profiling.
- [x] 2.3 Add scan/materialization-run caching for token profiles keyed by storage path so directory profiling, plan compilation, and fallback classification reuse parsed signals.
- [x] 2.4 Add focused tests for `01.mkv`, `第001集.mkv`, `S01E001.mkv`, `01.2160p.HD国语中字[网站].mkv`, codec/audio false positives, and movie title-year filenames.

## 3. Directory Profiles

- [x] 3.1 Implement directory profile construction from already-listed scan snapshots and token profiles without additional storage listing, ffprobe, hashing, media reads, artwork downloads, or external metadata provider calls.
- [x] 3.2 Compute aggregate profile evidence for video counts, non-extra counts, attachment counts, explicit episode coverage, leading numeric coverage, sequence coverage, sequence gaps, year density, title uniqueness, common title stem, version evidence, season directory hints, sidecar hints, and category/path hints.
- [x] 3.3 Implement directory fingerprints that include provider, library/path scope, directory path, classifier version, relevant scan policy and exclusion inputs, child names, stable identities when available, modified/size evidence, and visible video counts.
- [x] 3.4 Persist profiles and reuse unchanged profiles when fingerprints and classifier versions match.
- [x] 3.5 Add tests for unchanged directory reuse, classifier version invalidation, scan exclusion invalidation, and profile persistence for large episode directories and movie collections.

## 4. Directory Plan Compiler

- [x] 4.1 Implement plan compilation for `episode_pack`, `absolute_episode_pack`, `season_folder`, `flat_episode_folder`, `series_folder`, `movie_folder`, `movie_versions_folder`, `movie_collection_folder`, `attachment_group`, and `unknown_review`.
- [x] 4.2 Implement scoring and thresholds that distinguish episode packs from movie collections using sequence coverage, title uniqueness, year density, common title evidence, season/path hints, and independent movie evidence.
- [x] 4.3 Generate plan-level evidence, confidence, shape, series title, season number, numbering mode, movie work grouping, attachment mapping, review state, and alternatives considered.
- [x] 4.4 Persist compact plan rules and exceptions where possible, including absolute episode numbering, season-folder numbering, sorted-order fallback, movie-version grouping, movie-collection grouping, and attachment assignments.
- [x] 4.5 Add tests for mixed-naming high-confidence episode packs, Season 1/Season 2 folders, absolute 001-1000 episode packs, movie version folders, movie collection folders, and ambiguous conflicting directories.

## 5. Plan Assignments And Incremental Reuse

- [x] 5.1 Implement file assignment generation from directory plans for episode slots, absolute numbers, movie work IDs/keys, movie versions, attachments, and review-required files.
- [x] 5.2 Reuse persisted plan rules for unchanged directories and small deltas so a newly added episode can be assigned without recompiling the full directory.
- [x] 5.3 Detect deltas that invalidate plan confidence and trigger recompilation or review-required states.
- [x] 5.4 Persist enough assignment metadata or rule exceptions for materialization batches to reuse assignments consistently across batch boundaries.
- [x] 5.5 Add tests for adding one episode to an existing absolute pack, deleting an episode, adding a conflicting movie-like file, and preserving assignment stability across multiple materialization batches.

## 6. Plan-Based Materialization

- [x] 6.1 Integrate directory plan compilation/reuse into scan traversal and `RunCatalogMaterializeBatch` so directory plans are compiled once per directory and reused across catalog write batches.
- [x] 6.2 Materialize high-confidence episode and season assignments into existing series, season, episode, asset, and asset-item rows without running independent full file-first classification for covered files.
- [x] 6.3 Materialize movie version assignments into one movie item with multiple assets or version links using existing catalog and inventory semantics.
- [x] 6.4 Materialize attachment assignments as trailer, sample, preview, featurette, behind-the-scenes, or extra asset roles attached to the planned parent work.
- [x] 6.5 Preserve sidecar subtitle binding, artwork preselection, metadata source recording, scanner identity reconciliation, dirty projection marking, inventory probing, and post-materialize enrichment scheduling for plan-based outputs.
- [x] 6.6 Add integration tests proving a 1000-file episode directory compiles one plan and materializes through multiple DB write batches without recompiling full directory classification per batch.

## 7. Fallback, Review, And Governance

- [x] 7.1 Keep existing file-first classification as fallback for files and directories without usable high-confidence plan assignments.
- [x] 7.2 Create review-required directory decisions and assignment decisions when plan evidence is low confidence or conflicting.
- [x] 7.3 Preserve candidate alternatives and evidence showing whether a decision came from directory profile, directory plan, filename token profile, sidecar hint, user/scoped rule, or fallback classifier.
- [x] 7.4 Add source-scoped or path-scoped correction rule support for confirmed directory-level corrections such as absolute episode pack, season folder, movie versions, and movie collection.
- [x] 7.5 Add tests for ambiguous large directories, plan/file evidence conflicts, review-required outcomes that do not pollute movie/episode catalog rows, and scoped correction rule application.

## 8. Performance And Regression Validation

- [x] 8.1 Add instrumentation or test hooks to count token profile parses, directory profile builds, plan compiles, plan reuses, fallback classifications, and materialization batches.
- [x] 8.2 Add performance-oriented regression tests or benchmarks comparing large episode directory behavior before and after plan-based classification.
- [x] 8.3 Verify source-first scan completion still does not wait for ffprobe, metadata matching, artwork downloads, or external providers.
- [x] 8.4 Run focused backend tests for library scanning, materialization, probe scheduling, listener updates, missing cleanup, scan exclusions, metadata governance, and catalog projection behavior.
- [x] 8.5 Run full backend test suite with `go test ./...` from `mibo-media-server/` after implementation.

## 9. Cleanup And Documentation

- [x] 9.1 Remove or bypass redundant directory summary and repeated file-first classification paths for directories covered by high-confidence plans while retaining fallback paths.
- [x] 9.2 Update scanner evidence payloads and any admin/debug outputs to include shape profile, plan, assignment, and reuse information where relevant.
- [x] 9.3 Document rollout behavior, fallback switch, classifier version invalidation, and how directory-level review/correction rules affect future scans.
- [x] 9.4 Confirm existing frontend typecheck still passes if any review or diagnostics contracts are exposed to the web UI.
