## 1. Work-Group Model And Inputs

- [x] 1.1 Define internal path-tree work-group structs for group shape, work key, confidence, review state, evidence, alternatives, and file assignments.
- [x] 1.2 Build parent directory summaries from existing directory snapshots, child snapshots, content-shape plans, and `inventory_file_signals` without additional storage listings.
- [x] 1.3 Implement normalized movie work-key helpers using indexed title/year signals and release-hint suppression.
- [x] 1.4 Add tests for work-key normalization across dot, hyphen, Chinese prefix, release-group, quality, audio, and codec variants.

## 2. Sibling Movie Version Grouping

- [x] 2.1 Detect sibling one-file child directories with matching title/year work keys and release-hint differences.
- [x] 2.2 Generate one movie work-group assignment with multiple asset/version file assignments for sibling release folders.
- [x] 2.3 Integrate sibling movie-version assignments into catalog materialization so one movie item receives multiple assets.
- [x] 2.4 Add regression tests for `3.Iron.2004...MiniHD` and `3-Iron.2004...TAGHD` style sibling folders.
- [x] 2.5 Add negative tests for sibling folders with materially different titles or years to ensure they remain independent movies.

## 3. Movie Collection Splitting

- [x] 3.1 Detect parent directories whose child files or child folders have high title/year uniqueness and low episode sequence evidence.
- [x] 3.2 Generate separate movie work groups for each distinct title/year key in movie collections.
- [x] 3.3 Preserve movie-version grouping inside collections when multiple children share one title/year key.
- [x] 3.4 Add tests for mixed file and one-file-folder movie collections.

## 4. Series Root Grouping

- [x] 4.1 Detect parent directories with sibling season folders, episode packs, or noisy season-like directories that share a series title.
- [x] 4.2 Generate one series work group with stable season and episode assignments across child directories.
- [x] 4.3 Preserve existing single-directory episode-pack and season-folder behavior by reusing content-shape assignments.
- [x] 4.4 Add tests for `Season 1`/`Season 2`, `Show S01`/`Show S02`, and noisy Chinese season directory variants.

## 5. Plan Persistence And Reuse

- [x] 5.1 Decide whether parent work groups can persist in existing content-shape tables or require an additive `recognition_work_groups` model.
- [x] 5.2 Persist work-group fingerprints including parent path, child directory fingerprints, file signal fingerprints, scan policy, exclusion rules, classifier version, and scoped correction rules.
- [x] 5.3 Reuse unchanged work-group plans and assignments across rescans and materialization batches.
- [x] 5.4 Invalidate or recompile work groups when new children conflict with an existing group rule.
- [x] 5.5 Add tests for unchanged reuse, new sibling version addition, conflicting child addition, and classifier-version invalidation.

## 6. Governance And Corrections

- [x] 6.1 Persist review-required work-group decisions with affected files, alternatives, evidence, confidence, and proposed correction actions.
- [x] 6.2 Extend scoped classification rules to express sibling movie versions, movie collections, independent sibling movies, and series root grouping.
- [x] 6.3 Apply scoped correction rules before automatic work-group scoring when they match source and path scope.
- [x] 6.4 Add tests for user-confirmed movie versions, movie collection split, independent movies, and series root corrections.

## 7. Metadata Match Queueing

- [x] 7.1 Ensure movie version work groups queue at most one metadata match for the movie item, not one per asset file.
- [x] 7.2 Ensure series work groups queue metadata matching at the series root, not per episode.
- [x] 7.3 Ensure movie collections queue one match per movie work group after materialization.
- [x] 7.4 Add tests for match job counts after movie-version, movie-collection, and series work-group scans.

## 8. Cleanup And Fallback Reduction

- [x] 8.1 Bypass redundant file-first classification for files covered by high-confidence work-group assignments.
- [x] 8.2 Keep guarded placeholders only for review-required outcomes that need catalog visibility.
- [x] 8.3 Remove or isolate obsolete fallback helpers once work-group coverage has equivalent tests.
- [x] 8.4 Add regression tests ensuring ambiguous groups do not silently create accepted wrong semantics.

## 9. Verification

- [x] 9.1 Run focused backend tests for content-shape, work-group recognition, catalog materialization, metadata queueing, and governance.
- [x] 9.2 Run `go test ./...` from `mibo-media-server/`.
- [x] 9.3 Document scanner behavior for sibling movie versions, movie collections, series root grouping, work-group reuse, and correction rules.
