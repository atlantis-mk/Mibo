## Why

Automatic source-first scanning currently relies too heavily on directory-shape inference to decide whether video content is a movie, a series episode, a season folder, or a mixed folder. This makes common messy libraries easy to misclassify, especially multi-file movie folders, flat episode folders, trailers, samples, and anime-style season directories.

## What Changes

- Replace directory-shape-first video classification with a fast, staged classifier that first identifies file role, then generates movie and episode candidates, then groups sibling files, and only then projects catalog items.
- Treat `movie_folder`, `season_folder`, `flat_episode_folder`, `mixed_folder`, and similar observations as evidence, not final content types.
- Add explicit classification decisions with candidate type, role, confidence, evidence, and review state so uncertain outcomes are explainable and recoverable.
- Use a fast path based only on path, filename, extension, sidecar names, and current-directory sibling listings; keep ffprobe, provider searches, hashing, and artwork work asynchronous.
- Add low-confidence and conflicting decisions to governance review instead of silently committing ambiguous movie-vs-episode choices.
- Add user-confirmed classification rules so corrections can be remembered for source-scoped paths and reused on future scans.

## Capabilities

### New Capabilities
- `fast-video-classification`: Fast staged video classification using file role detection, candidate generation, sibling grouping, confidence thresholds, reviewable evidence, and learned correction rules.

### Modified Capabilities
- `media-graph-scanner`: Scanner requirements change from directory-shape-first semantic projection to candidate-based, evidence-backed classification before catalog projection.
- `catalog-metadata-governance`: Governance requirements change to include classification decision review and correction flows for ambiguous movie-vs-episode outcomes.
- `source-first-auto-classification`: Source-first automatic classification requirements change to preserve fast feedback while using staged, candidate-based video classification and reviewable low-confidence decisions.

## Impact

- Backend scanner classification in `mibo-media-server/internal/library`, including filename parsing, directory snapshot use, catalog scan artifacts, and decision evidence.
- Catalog and inventory relationships for main assets, versions, trailers, extras, samples, subtitles, movies, series, seasons, and episodes.
- Governance APIs and review surfaces for ambiguous classification decisions and source-scoped correction rules.
- Frontend settings/governance surfaces that display classification evidence and let users resolve ambiguous groups.
- Tests for one-file movies, explicit episodes, flat numbered episode folders, multi-version movies, independent movies in one folder, trailers/extras/samples, and low-confidence review outcomes.
