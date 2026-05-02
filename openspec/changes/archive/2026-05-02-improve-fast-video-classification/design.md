## Context

Mibo is moving toward source-first onboarding where users add paths and the scanner owns semantic classification. The current automatic video resolver still carries directory-shape concepts such as `movie_folder`, `season_folder`, `flat_episode_folder`, and `mixed_folder` into semantic decisions. Those concepts are useful observations, but they are not stable content facts: a folder can contain one movie plus trailers, a season can be represented by numbered files without `SxxExx`, and a folder can hold multiple independent movies.

The scanner already separates inventory facts from catalog decisions and records resolver evidence. This change tightens that model around fast, staged classification: cheap evidence produces candidates immediately, expensive validation runs later, and uncertain decisions remain reviewable instead of being silently committed.

## Goals / Non-Goals

**Goals:**
- Classify video files quickly enough for source-first scanning without reading media contents, running ffprobe, hashing files, or calling metadata providers in the fast path.
- Make the first classification question file-centric: whether a video is a main playable asset, trailer, extra, sample, or unknown attachment.
- Generate movie and episode candidates before choosing a final catalog projection.
- Use current-directory sibling evidence to distinguish episode sequences, movie versions, independent movies, and attachment files.
- Preserve confidence, evidence, alternatives, and review state for ambiguous outcomes.
- Allow user corrections to become source-scoped classification rules for future scans.

**Non-Goals:**
- Guarantee perfect classification before catalog items become visible.
- Replace TMDB, MetaTube, ffprobe, artwork, or technical enrichment pipelines.
- Implement a machine-learning or LLM classifier.
- Build deep audio, book, or photo classification as part of this change.
- Remove catalog item types such as movie, series, season, and episode from persisted catalog projection.

## Decisions

### Decision: Classify file role before semantic work type

The fast classifier first identifies whether each supported video file is likely `main`, `trailer`, `extra`, `sample`, or `unknown_attachment` from filename, path segments, extension, sidecar hints, and existing exclusion rules. Only likely main files participate in movie-vs-episode decisions.

Alternatives considered:
- Decide movie-vs-episode first and classify extras later. This keeps the current shape but lets trailers, PVs, featurettes, and samples distort multi-file grouping.
- Ignore attachments until metadata enrichment. This is fast but causes false episode groups and duplicate movie items.

Rationale: removing non-main files early makes the remaining classification smaller, faster, and less error-prone.

### Decision: Generate candidates instead of writing a single immediate answer

Each main video can produce one or more classification candidates, such as movie candidate, explicit episode candidate, sorted-order episode candidate, movie-version candidate, or independent-movie candidate. Candidates carry confidence and evidence and are resolved into catalog projection only when the confidence threshold is met.

Alternatives considered:
- Hard-code a single winner in the scanner. This is simpler but makes wrong decisions difficult to explain or recover.
- Push every file to review. This is safe but defeats automatic scanning.

Rationale: candidates let fast evidence, sibling grouping, provider validation, and user review progressively converge without losing alternatives.

### Decision: Use sibling grouping, not directory type, as the fast context step

The classifier lists only the current directory when needed and groups sibling videos by role, normalized title stem, numbering sequence, year evidence, quality/version tokens, and episode markers. It classifies relationships between files, not the directory itself.

Alternatives considered:
- Keep directory shape as the primary resolver input. This is fast but overfits to folder layouts and mislabels mixed real-world directories.
- Recursively analyze the full source before classification. This can improve accuracy but is too slow for OpenList/NAS-backed sources and source-first feedback.

Rationale: current-directory sibling evidence is the smallest useful context that distinguishes common failure cases while preserving performance.

### Decision: Keep fast path strictly cheap and deterministic

Fast classification uses path strings, filenames, extensions, already-listed object metadata, sidecar filenames, and one current-directory listing. It must not run ffprobe, hash content, read media files, request external metadata, or download artwork.

Alternatives considered:
- Run ffprobe for all files before classification. This helps with duration-based guesses but is expensive and brittle for remote storage.
- Call provider search while scanning. This can improve correctness but makes scanning network-dependent and rate-limit-sensitive.

Rationale: classification must keep source acceptance and initial scan responsive; slow validation belongs in independently retryable jobs.

### Decision: Treat provider and ffprobe results as asynchronous validators

Slow jobs can validate provisional candidates with duration, stream metadata, TMDB/MetaTube movie searches, and TV series/season/episode searches. Validation may confirm, revise, or mark decisions for review, but it must preserve inventory facts and decision history.

Alternatives considered:
- Never revisit fast decisions. This is simple but locks in avoidable false positives.
- Block catalog projection until slow validation completes. This improves correctness but makes sources appear empty or stalled.

Rationale: provisional projection gives fast value while asynchronous validation improves accuracy over time.

### Decision: Make low-confidence outcomes reviewable and learnable

Decisions below confidence thresholds, or decisions with close movie/episode alternatives, are marked for governance review. User confirmations create source-scoped rules, such as path prefix plus series title, season number, episode numbering source, or movie-version grouping behavior.

Alternatives considered:
- Keep user corrections as one-off catalog edits. This fixes the current item but repeats work on rescan.
- Ask users to choose library or directory type up front. This contradicts source-first onboarding and exposes internal semantics.

Rationale: corrections should improve future classification without forcing users into pre-scan configuration.

## Risks / Trade-offs

- [Risk] Candidate storage and decision history increase model complexity. -> Mitigate by keeping inventory facts, media assets, catalog items, and classification decisions clearly separated.
- [Risk] Fast rules may still produce wrong high-confidence decisions for unusual names. -> Mitigate with conservative thresholds, alternative candidates, and asynchronous validation.
- [Risk] Learned rules can over-apply if scoped too broadly. -> Mitigate by requiring source-scoped path prefixes or patterns and showing rule evidence in review.
- [Risk] Sibling listing can be slow on remote providers. -> Mitigate by reusing directory snapshots from scan traversal and enforcing one-directory listing budgets.
- [Risk] Frontend review UI can become too technical. -> Mitigate by presenting concrete choices such as movie version group, episode sequence, independent movies, or attachment rather than internal shape labels.

## Migration Plan

No production data migration is required for the proposal itself. During implementation, existing scan decision records may be extended or replaced in development data. Rollback during development means reverting the change and rebuilding local scan/catalog data from inventory or resetting the local database.

## Open Questions

- What exact confidence thresholds should separate fast-confirmed, provisional, and review-required decisions?
- Should learned rules be stored as a new classification-rules table or reuse existing scan exclusion/governance rule infrastructure?
- Should season remain a persisted catalog item for all projected TV hierarchy, or can some fast-path review surfaces present season as derived organization only?
