## Legacy Reference Inventory

### Parser/helper usage to preserve or relocate

- `internal/library/scan_filename_signals.go` uses filename analysis, folder parsing, title cleanup, season/title path hints, and weak episode parsing.
- `internal/library/movie_work_key.go` and `folder_title_decision.go` use title cleanup, folder parsing, generic-name detection, and directory-kind constants for display-title helpers.
- `internal/library/content_shape_profile.go`, `content_shape_plan.go`, and `content_shape_assignment.go` use folder parsing, relative path helpers, title cleanup, video role signals, and weak episode parsing.
- `internal/library/scan_run.go`, `scan_noise_exclusion.go`, and `scan_catalog.go` use folder parsing, generic-name detection, and path-derived series/season helpers.
- `internal/recognition/materializer.go` uses folder parsing and generic-title cleanup helpers.

### Legacy tree-classifier authority to remove

- `internal/library/recognition_scan_adapter.go` builds a `scanrecognition.Tree`, calls `scanrecognition.ClassifyTree`, converts old directory-node kinds into recognition candidates, and builds `scanrecognition_outcome` decisions.
- `internal/library/recognition_manifest.go` still calls `buildScanRecognitionManifestOutput`, then uses the old adapter output to lock movie/episode kind before adding `content_shape` context evidence.
- `internal/library/recognition_local_adapter_test.go` tests the old adapter candidate behavior directly.
- `internal/scanrecognition/builder.go`, `classifier.go`, and `types.go` are the obsolete tree model and classifier once parity exists.

## Legacy Fixture Mapping

- Split/combined season folders, flat episode groups, episode token consensus, expected-count folders, single-file episode packs, and episode word patterns map to `content_shape` profile and plan tests for `season_folder`, `episode_pack`, `flat_episode_folder`, and `absolute_episode_pack`, plus recognition-unit tests for episode grouping.
- Movie folders map to `content_shape` `movie_folder` plan and assignment tests.
- Movie versions, edition noise, and directory token-consensus version fixtures map to `movie_versions_folder` plan, assignment, recognition-unit, and materialization tests that keep one movie work with multiple variant resources.
- Multipart movie fixtures, multipart token consensus, high-coverage multipart siblings, and non-part sibling handling map to `multipart_movie_folder` plan, assignment, recognition-unit, and materialization tests that preserve part order in one playable resource.
- Broken or duplicate multipart sequences map to review-required plan tests with conflict diagnostics.
- Large sampled movie collections, normal movie collections, and catalog identifier collections map to `movie_collection_folder` profile, plan, assignment, recognition-unit, and materialization tests using per-file-title, per-title-year, or per-catalog-id keys.
- Single SxxExx movie-like files, same-title different-year cases, and episodic evidence competing with collection evidence map to review/precedence tests that prevent unsafe grouping.
- Movie/episode NFO sidecar support maps to sidecar-shape hint profile evidence and conflict-first plan tests. Conflicting movie sidecar versus episode evidence and episode sidecar versus movie evidence map to review-required diagnostics.
- Trailer/sample/extra noise fixtures map to primary-video-first planning and attachment assignment tests.
- Parent directories whose meaningful children are all season folders map to child-shape summary and `series_folder` plan tests; mixed child shapes map to negative coverage.
