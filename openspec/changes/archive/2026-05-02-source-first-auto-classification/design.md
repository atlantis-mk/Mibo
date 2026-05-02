## Context

The current library flow exposes video semantic choices (`movies`, `shows`, `mixed`) during creation. Those choices flow through `database.Library.Type`, library APIs, the settings `LibraryForm`, scan scheduling, and the video classifier. Recent scanner work already separates inventory facts from catalog decisions and records resolver evidence, and mixed-content scanning is a partial version of automatic movie-vs-series classification.

This change intentionally treats the product as pre-release: existing library type data and compatibility behavior can be replaced instead of migrated. The desired model is source-first. Users add a storage source/path, Mibo performs a fast bounded probe to detect content classes, then background jobs build inventory, classify content semantics, enrich metadata, and surface low-confidence items for review.

## Goals / Non-Goals

**Goals:**
- Remove movie/show/mixed as required user choices in setup and library/source creation.
- Make source path onboarding fast by doing only bounded, lightweight probing synchronously.
- Represent detected content classes separately from semantic catalog item types.
- Classify video semantics automatically from file, directory, sidecar, and metadata evidence without relying on dedicated library type hints.
- Preserve asynchronous enrichment so source acceptance and initial browsing do not wait on ffprobe, hash calculation, provider metadata searches, or artwork downloads.
- Provide governance/review entry points for ambiguous or low-confidence classifier results.

**Non-Goals:**
- Migrating existing development data or preserving old movie/show/mixed creation payloads.
- Implementing full audio, text, image, or ebook metadata systems in this change.
- Requiring all classification to be perfect before catalog items become visible.
- Replacing existing storage providers or the existing inventory/catalog separation.

## Decisions

### Decision: Use source-first creation as the user-visible model

Users create content by selecting a media source and root path. The system derives the display name when possible and no longer asks whether the path is a movie, show, or mixed library.

Alternatives considered:
- Keep movie/show/mixed but default to mixed. This reduces friction but preserves the wrong mental model and still exposes internal video semantics.
- Ask for video/audio/text during creation. This is simpler than movie/show but remains a front-loaded decision that the system can infer cheaply.

Rationale: the user knows where files are; Mibo should own classification.

### Decision: Split content class from catalog semantic type

The source probe classifies physical files into broad content classes (`video`, `audio`, `text`, `image`, `other`). Catalog projection continues to use semantic item types (`movie`, `series`, `season`, `episode`, and future `album`, `track`, `book`, etc.).

Alternatives considered:
- Reuse `Library.Type` for both broad classes and semantic types. This would recreate the current confusion and block future non-video support.
- Store only semantic catalog types. This loses the cheap source-level distribution needed for fast onboarding and smart views.

Rationale: content class is a scan/source concern; semantic type is a catalog concern.

### Decision: Probe synchronously under strict budgets

Source probing must use only cheap storage listing, path names, file extensions, size/modtime if already returned by the provider, and shallow directory evidence. It must enforce maximum duration, object count, and depth budgets.

The probe must not run ffprobe, hash file contents, read large files, call TMDB/TVDB/MetaTube, download artwork, or block until a full recursive scan finishes.

Alternatives considered:
- Full deep scan before accepting the source. This improves initial accuracy but makes remote OpenList/NAS/WebDAV sources feel slow or unreliable.
- No probe at creation. This is fastest but gives the user no immediate confidence or summary.

Rationale: fast approximate feedback is better than slow precise feedback during onboarding.

### Decision: Use automatic video classification as the default resolver mode

Video classification should behave as an enhanced automatic resolver: one-file movie groups, season folders, flat episode folders, explicit SxxExx signals, sidecar metadata, extras, multi-version movie groups, and ambiguous evidence all produce resolver decisions with confidence and reasons. The old dedicated movie/show library mode is removed from the user-visible path.

Alternatives considered:
- Internally map all new sources to legacy `mixed`. This is a good transitional implementation detail but insufficient as the final design because current mixed rules over-classify multi-file folders as TV-like content.
- Preserve hidden movie/show override modes. This can be considered later as an advanced rule system, but it is not required for the default flow.

Rationale: automatic classification should be evidence-based and reviewable, not a one-time user declaration.

### Decision: Make low-confidence outcomes reviewable, not blocking

Ambiguous files or groups should still be inventoried and may produce provisional catalog projections, but classifier decisions below confidence thresholds must mark affected items or relationships for governance review with evidence explaining the ambiguity.

Alternatives considered:
- Hide ambiguous items until manually resolved. This avoids wrong catalog entries but makes files appear missing.
- Always choose the highest-confidence result. This maximizes automation but risks silent, hard-to-debug misclassification.

Rationale: post-scan exception handling is lower-friction than pre-scan selection and safer than silent guesses.

### Decision: Keep deep work asynchronous and independently retryable

Inventory discovery, classification, technical probing, metadata matching, artwork selection, projection rebuilds, and cleanup should remain separate job phases where possible. The source add flow enqueues background work after the quick probe returns.

Alternatives considered:
- Collapse probing and scanning into one synchronous API. Simpler API surface but poor responsiveness and harder recovery.

Rationale: this aligns with existing scan/enrichment job separation and preserves responsiveness.

## Risks / Trade-offs

- [Risk] Multi-file movie folders may still look like episode folders. → Mitigate with stronger evidence scoring for version labels, same-title similarity, movie sidecars, year-bearing parent folders, and low-confidence review instead of unconditional TV projection.
- [Risk] Remote sources can be slow even for listing. → Mitigate with probe budgets and partial-result summaries.
- [Risk] Removing library types touches many tests and UI assumptions. → Mitigate by updating the model consistently across API contracts, settings UI, scan scheduling, and helper fixtures in one change.
- [Risk] Future audio/text support may be implied before it is complete. → Mitigate by creating detected collections for broad classes while marking non-video deep enrichment as minimal or future-facing.
- [Risk] Existing development data becomes invalid. → Acceptable for this change because migration compatibility is explicitly out of scope.

## Migration Plan

No production migration is required. Implementation may replace or simplify library type columns, seed defaults, and development fixtures. Existing local development databases can be reset or recreated.

Rollback during development means reverting the change and resetting local data. No compatibility shim is required.

## Open Questions

- Should source-first objects still be named `Library` internally for now, or should the backend rename them to `SourceCollection`/`ContentSource` as part of the breaking rebuild?
- What initial probe budgets should be used for local storage versus OpenList-backed storage?
- Should low-confidence video groups be visible in the main catalog immediately, or grouped under a dedicated review/unknown view first?
