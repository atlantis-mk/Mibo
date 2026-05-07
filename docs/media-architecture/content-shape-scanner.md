# Content Shape Scanner

The content shape scanner is the directory-first classifier for source-first video scans. It profiles already-listed directory entries, compiles reusable plans, and materializes catalog rows from content-shape or path-tree assignments.

## Classifier Versioning

- `ContentShapeClassifierVersion` invalidates persisted profiles and plans when the classifier logic changes.
- Persisted shape profiles and plans remain reusable as long as their fingerprint, classifier version, scan policy, and exclusion-rule inputs match.

## Fallback And Review

- High-confidence episode, season, movie-version, and attachment plans can supply catalog assignments directly.
- Low-confidence or conflicting plans create review-required directory decisions and guarded local placeholders instead of silently accepting weak movie or episode rows.
- Files without a usable content-shape/path-tree assignment are skipped as `materialization_skipped_unplanned`.

## Path-Tree Work Groups

- A bounded parent-level work-group pass runs over already indexed file signals and already listed scan snapshots; it does not call TMDB, ffprobe, hashing, artwork download, or media-content reads.
- Sibling one-file movie release folders are grouped as one movie with multiple version assets when their normalized title/year work key matches and differences are release hints such as quality, source, codec, audio, edition, HDR, or release group.
- Movie collection directories split into one movie work group per distinct title/year key, while duplicate keys inside the collection remain version assets of the same movie.
- Sibling season-like directories such as `Season 1`/`Season 2`, `Show S01`/`Show S02`, and noisy Chinese season folders are normalized under one series root when season and series title evidence agree.
- High-confidence path-tree assignments override directory-local plans for covered files so sibling release folders do not create duplicate movie items.
- Ambiguous parent groups persist review-required `path_tree_work_group` decisions with affected file paths, evidence, candidate alternatives, and confidence instead of silently accepting a weak merge or split.

## Persistence And Queueing

- Parent work groups reuse the existing `content_shape_profiles`, `content_shape_plans`, and `content_shape_assignments` tables; no separate `recognition_work_groups` table is currently required.
- Work-group fingerprints include parent path, assignment inputs, classifier version, scan policy, and exclusion rules so unchanged groups reuse the same persisted plan and conflicting children recompile to a new fingerprint.
- Metadata matching is queued by recognized work group: movie version groups queue the movie item once, series groups queue only the series root, and movie collections queue once per independent movie group.

## Corrections

- Directory-level confirmations are stored as source/path-scoped `content_shape_directory` classification rules.
- Scoped rules are intended for shapes such as absolute episode packs, season folders, movie versions, and movie collections.
- Rules do not apply outside their configured library/source/path scope.
