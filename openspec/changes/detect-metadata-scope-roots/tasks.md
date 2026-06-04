## 1. Baseline Audit

- [x] 1.1 Inventory current scan-to-materialization paths that treat `content_shape` directory plans as final metadata roots.
- [x] 1.2 Inventory current uses of `recognition_directory_reduction.go` and document which behaviors must be ported before cleanup.
- [x] 1.3 Record current classifier/version constants and derived-table fingerprints that must change when leaf or scope logic changes.
- [x] 1.4 Add failing fixture notes for `Spider-Noir (2026)/1080p彩版|4K彩版|4K黑白版/S01E01-S01E08` and expected scope outcome.

## 2. Leaf Classification Model

- [x] 2.1 Define leaf summary runtime types for shape, dominant identity, title evidence, season set, episode set, part set, version signature, attachment roles, confidence, review state, covered files, and residual-token evidence.
- [x] 2.2 Add persistence model/repository or extend existing content-shape persistence for leaf summaries with classifier version and fingerprint fields.
- [x] 2.3 Refactor `contentShapeTokenConsensus` residual/cancellation behavior into explicit leaf-classification helpers with focused tests.
- [x] 2.4 Ensure leaf classification only consumes direct primary video children and does not recursively classify nested child videos.
- [x] 2.5 Add primary/supplemental video filtering for trailers, samples, extras, featurettes, behind-the-scenes, interviews, deleted scenes, NCOP/NCED, and similar roles.
- [x] 2.6 Add conservative handling for ambiguous `SP`, `OVA`, `Specials`, and `番外` labels so they require episode/sidecar/duration evidence before becoming episode content.
- [x] 2.7 Persist leaf residual evidence and alternatives for review-required conflicts.

## 3. Leaf Classification Tests

- [x] 3.1 Add tests for token residual episode-pack detection from shared-title sibling filenames.
- [x] 3.2 Add tests for token residual movie-version detection from quality/source/codec/edition residuals.
- [x] 3.3 Add tests for token residual multipart detection with continuous and broken part sequences.
- [x] 3.4 Add tests for movie collection detection from distinct title/year residuals.
- [x] 3.5 Add tests proving attachment videos do not dominate a main movie or episode leaf shape.
- [x] 3.6 Add tests for extras-only leaf directories and ambiguous specials review behavior.
- [x] 3.7 Add reuse/invalidation tests for unchanged and changed leaf fingerprints.

## 4. Scope Decision Model

- [x] 4.1 Define scope decision runtime types for `scope_path`, `root_kind`, `layout`, `identity_key`, confidence, evidence, child roles, attachment roles, and covered files.
- [x] 4.2 Add persistence model/repository for metadata scope decisions and scope claims, including version and fingerprint fields.
- [x] 4.3 Include child leaf summaries, relevant directory snapshots, file signals, sidecar evidence, scan policy, and classifier versions in scope fingerprints.
- [x] 4.4 Add diagnostics helpers that expose why a scope was selected, rejected, or marked review-required.

## 5. Upward Scope Reducer

- [x] 5.1 Implement bounded ancestor traversal from changed leaf summaries to library root or configured maximum depth.
- [x] 5.2 Load sibling leaf summaries and attachment summaries from persisted snapshots instead of requiring a completed full-library scan.
- [x] 5.3 Implement identity-purity scoring for movie and series identities.
- [x] 5.4 Implement coverage-gain scoring for versions, seasons, parts, complementary episode ranges, and attachments.
- [x] 5.5 Implement layout explainability for `series/versioned_episode_packs`, `series/season_directories`, `series/split_episode_packs`, `movie/version_directories`, `movie/multipart_parts`, `movie_collection`, and review-required mixed layouts.
- [x] 5.6 Implement boundary evidence using directory-title identity match, parent category/source/share/library-root hints, and parent mixed-identity detection.
- [x] 5.7 Implement attachment neutrality so supplemental child folders are excluded from main identity purity but included in the accepted scope.
- [x] 5.8 Select the highest candidate that remains complete, pure, explainable, and bounded; otherwise create a review-required scope decision.

## 6. Scope Reducer Tests

- [x] 6.1 Add tests where versioned sibling `episode_pack` leaves produce one `series` scope with `versioned_episode_packs` layout.
- [x] 6.2 Add tests where season child directories produce one `series` scope with `season_directories` layout.
- [x] 6.3 Add tests where complementary episode child ranges produce one `series` scope with `split_episode_packs` layout.
- [x] 6.4 Add tests where sibling movie version directories produce one `movie` scope with version child roles.
- [x] 6.5 Add tests where parent-level trailers/extras attach to the nearest compatible movie or series scope.
- [x] 6.6 Add tests where attachment-only orphan scopes become review-required and do not create speculative metadata.
- [x] 6.7 Add tests where a parent source/category directory with unrelated identities stops upward traversal.
- [x] 6.8 Add tests for partial refresh of one child version updating the parent scope decision and claim.

## 7. Pipeline Integration

- [x] 7.1 Integrate leaf summary creation into the scan stage after inventory and reusable file signals are available.
- [x] 7.2 Integrate scope decision generation after leaf summaries are saved and before recognition units are built.
- [x] 7.3 Add scope claim checks so covered leaf directories do not enqueue duplicate recognition or materialization tasks.
- [x] 7.4 Update recognition unit construction to prefer accepted scope decisions over treating leaf `content_shape` plans as final roots.
- [x] 7.5 Update materialization to create one hierarchy for versioned series scopes and bind matching episode resources as versions under the same episode metadata items.
- [x] 7.6 Update movie version scope materialization to resolve one movie work and bind all version resources to that work.
- [x] 7.7 Update directory metadata resolution payload creation to include scope root kind, layout, child roles, attachment roles, and covered resource IDs.
- [x] 7.8 Preserve review-required scope outcomes in operations/governance evidence without running automatic provider matching.

## 8. Cleanup and Migration

- [x] 8.1 Bump leaf and scope classifier versions and verify stale content shape plans, recognition units, and directory metadata resolution rows are regenerated.
- [x] 8.2 Remove or gate code paths that infer final metadata roots solely from single-directory `content_shape` plans when scope decisions exist.
- [x] 8.3 Port useful `recognition_directory_reduction.go` behaviors into leaf/scope reducer tests and disable it as a competing materialization authority.
- [x] 8.4 Delete obsolete residual grouping/materialization code after parity tests pass, keeping only diagnostic helpers if still useful.
- [x] 8.5 Update backend diagnostics and README/developer notes to explain leaf shape versus metadata scope root.

## 9. End-to-End Verification

- [x] 9.1 Add integration test for the Spider-Noir versioned series directory tree producing one series hierarchy with three resource versions per episode.
- [x] 9.2 Add integration test for movie root with trailer/extras child folders producing one movie work plus attachments.
- [x] 9.3 Add integration test for movie collection directories preserving separate movie identities.
- [x] 9.4 Add integration test for partial refresh adding/removing a version child directory and invalidating the parent scope claim.
- [x] 9.5 Run targeted library tests for content shape, recognition units, materialization, directory metadata resolution, and operations issue evidence.
- [x] 9.6 Run `go test ./...` in `mibo-media-server` and record any intentionally deferred frontend impact.

## 10. Automatic-First Scope Decisions

- [x] 10.1 Treat leaf summaries as the primary facts for parent decisions: parent reducers MUST consume persisted leaf summaries and MUST NOT reparse or recursively pull child files into a parent leaf classification.
- [x] 10.2 Require explicit version or edition evidence before merging sibling movie leaves into a single `movie/version_directories` scope.
- [x] 10.3 Default multiple movie leaves without version or multipart evidence to `movie_collection` instead of a single-work merge or review-required outcome.
- [x] 10.4 Attach `Trailers`, `Extras`, `Samples`, `Featurettes`, behind-the-scenes, deleted scenes, NCOP/NCED, and similar supplemental leaves to the nearest compatible main scope.
- [x] 10.5 Keep attachment-only leaves visible as attachment-orphan review evidence when no compatible main scope exists, without creating speculative movie/series metadata.
- [x] 10.6 Add reducer and workflow tests for mixed movie siblings, sibling versions, multipart children, extras-only children, movie-plus-extras roots, season siblings, and mixed movie/series parents.
- [x] 10.7 Update diagnostics so each automatic scope records whether it was selected by strong evidence, safe collection fallback, attachment attachment, or review fallback.

## 11. Remote Metadata Validation

- [x] 11.1 Define a validation input built from local candidate groups: root kind, layout, local title, year, episode/season sets, file roles, and candidate metadata item IDs.
- [x] 11.2 Add a provider-agnostic validation result that can confirm, reject, split, or mark uncertain a local scope decision.
- [x] 11.3 Reuse existing metadata provider search/detail execution for validation without applying metadata fields during validation.
- [x] 11.4 Validate movie candidates by title/year/external ID confidence before merging versions or applying remote detail.
- [x] 11.5 Validate series candidates by series title plus season/episode coverage before merging season or versioned episode scopes.
- [x] 11.6 Validate movie collections by confirming files resolve to distinct high-confidence movie candidates when providers are available.
- [x] 11.7 When remote validation contradicts local structure, prefer the safer split/collection result and record a governance issue instead of blocking ingestion.
- [x] 11.8 Add fake-provider tests for confirmed movie, confirmed series, split collection, provider unavailable, ambiguous provider result, and provider conflict.

## 12. Search Caching and Performance

- [x] 12.1 Add metadata validation/search cache keys based on provider instance, media type, normalized title, year, language, and query source.
- [x] 12.2 Deduplicate remote searches across files and directories in the same scan run; one candidate group should search once.
- [x] 12.3 Reuse cached validation results across incremental scans while provider config, profile language, query facts, and local fingerprints are unchanged.
- [x] 12.4 Make remote validation asynchronous or follow-up work when local confidence is high enough to materialize immediately.
- [x] 12.5 Keep the scan stage bounded to local facts, leaf summaries, and scope candidates; remote provider IO MUST NOT block directory walking.
- [x] 12.6 Add metrics/diagnostics for cache hit rate, validation attempts, validation outcomes, and provider latency.
- [x] 12.7 Add performance-focused tests proving parent scope candidate evaluation loads leaf summaries once per ancestor chain.
- [x] 12.8 Add performance-focused tests proving remote search executes once per unique candidate key.

## 13. Confidence, Governance, and Rollout

- [x] 13.1 Introduce confidence tiers for local and remote decisions: `high_confidence`, `medium_confidence`, `low_confidence`, and `conflict`.
- [x] 13.2 Automatically materialize high-confidence and medium-confidence outcomes; mark medium-confidence outcomes as governable rather than review-blocking.
- [x] 13.3 Route low-confidence outcomes to safe defaults when possible: single movie, movie collection, attachment orphan, or unmatched item.
- [x] 13.4 Reserve hard review-required outcomes for destructive ambiguity, provider conflict, broken multipart evidence, sidecar contradiction, or unattachable supplemental media.
- [x] 13.5 Persist confidence reason codes such as `remote_unverified`, `safe_collection_fallback`, `provider_conflict`, `structure_conflict`, and `title_ambiguous`.
- [x] 13.6 Surface these reason codes through operations/governance diagnostics so users can fix the minority of uncertain cases.
- [x] 13.7 Add a rollout flag or version gate so the automatic-first validator can be enabled per library or classifier version.
- [x] 13.8 Complete verification with focused library tests, metadata provider tests, workflow tests, and `go test ./...` for `mibo-media-server`.
