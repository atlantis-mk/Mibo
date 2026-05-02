## 1. Data Model And Contracts

- [x] 1.1 Decide and document whether existing backend names remain `Library` internally or are renamed to source-first concepts during this rebuild
- [x] 1.2 Update backend request/response contracts so creating content requires media source and root path, not movie/show/mixed type
- [x] 1.3 Add source probe result fields for content-class counts, dominant class, uncertainty, budget-limited status, and sampled object totals
- [x] 1.4 Remove or neutralize user-selected movie/show/mixed type requirements from library/source persistence and fixtures
- [x] 1.5 Update API types in the frontend client to match the source-first contracts

## 2. Quick Source Probe

- [x] 2.1 Implement a bounded source probe service that samples provider listings by max duration, max object count, and max depth
- [x] 2.2 Classify sampled entries into `video`, `audio`, `text`, `image`, and `other` using extension and path evidence only
- [x] 2.3 Ensure probing never runs ffprobe, file-content hashing, external metadata searches, or artwork downloads
- [x] 2.4 Return partial probe summaries when budgets are exhausted or source traversal is incomplete
- [x] 2.5 Add tests for local and provider-backed probe budgets, content-class counts, and invalid path handling

## 3. Source Creation Flow

- [x] 3.1 Update backend source/library creation to run quick probing synchronously after path validation
- [x] 3.2 Enqueue asynchronous inventory and classification jobs after accepting the source
- [x] 3.3 Preserve existing scan, metadata, playback, subtitle, and exclusion policy defaults under source-first creation
- [x] 3.4 Update setup flow defaults so first-run content creation uses source-first semantics
- [x] 3.5 Add backend tests covering source acceptance, probe feedback, and background job enqueueing

## 4. Automatic Video Classification

- [x] 4.1 Refactor video classification entry points so they do not depend on user-selected movie or show library type hints
- [x] 4.2 Treat current mixed-content behavior as the baseline automatic resolver while removing dedicated type branching from user-visible flows
- [x] 4.3 Improve multi-file group scoring to distinguish episode groups, movie versions, extras, and ambiguous groups using evidence and confidence
- [x] 4.4 Ensure one-file non-extra video groups create movie items unless stronger TV evidence exists
- [x] 4.5 Ensure explicit season/episode and season-folder evidence creates stable series, season, and episode hierarchy
- [x] 4.6 Ensure multi-version movie folders project one movie item with multiple assets
- [x] 4.7 Add or update scanner tests for one-file movies, flat episodes, season folders, multi-episode assets, extras, movie versions, and ambiguous groups

## 5. Inventory, Collections, And Review

- [x] 5.1 Preserve inventory facts and content-class evidence for detected non-video files even when deep catalog projection is not implemented
- [x] 5.2 Expose source-scoped content collections or views derived from detected content classes
- [x] 5.3 Mark low-confidence or conflicting video resolver decisions for governance review with evidence, confidence, and proposed action
- [x] 5.4 Add API responses needed for clients to show probe summaries, classification progress, and reviewable decisions
- [x] 5.5 Add tests for ambiguous classification review state and non-video inventory visibility or summaries

## 6. Frontend Experience

- [x] 6.1 Remove movie/show/mixed selection from setup and settings content creation forms
- [x] 6.2 Present source-first creation with media source, root path, derived name, and clear submit state
- [x] 6.3 Show quick probe feedback including content-class distribution, partial/budget-limited status, and background scan progress
- [x] 6.4 Display generated content collections or views without requiring users to create separate semantic libraries
- [x] 6.5 Add a review surface or entry point for low-confidence classification decisions with evidence and correction actions
- [x] 6.6 Update frontend tests or type checks affected by removed library type fields

## 7. Cleanup And Verification

- [x] 7.1 Remove obsolete movie/show/mixed user-facing labels, form state, validation, and helper copy
- [x] 7.2 Update health/admin/settings references that still assume user-selected library type semantics
- [x] 7.3 Run backend tests with `go test ./...` from `mibo-media-server/`
- [x] 7.4 Run frontend typecheck with `pnpm typecheck` from `web/`
- [ ] 7.5 Manually verify local source creation with demo media and confirm first feedback appears before deep enrichment completes
