## 1. Classifier Model And Contracts

- [x] 1.1 Define fast classification decision structures for role, semantic candidates, confidence, alternatives, evidence, thresholds, and review state
- [x] 1.2 Add or extend persistence for classification decisions and source-scoped learned classification rules
- [x] 1.3 Update backend API contracts to expose reviewable classification decisions, alternatives, evidence, affected files, and correction actions
- [x] 1.4 Update frontend API types for classification decisions, candidate roles, candidate semantic types, confidence, and learned rule summaries

## 2. Fast File Role Detection

- [x] 2.1 Implement cheap filename/path role detection for main, trailer, extra, sample, preview/PV, featurette, and unknown attachment video files
- [x] 2.2 Ensure attachment-role videos are excluded from movie-vs-episode main-file grouping while retaining asset/link candidates
- [x] 2.3 Add tests for trailers, samples, PVs, featurettes, special/extra naming, and normal main files in movie and TV-like folders

## 3. Candidate Generation

- [x] 3.1 Generate explicit episode candidates from SxxExx, 1x02, EP tokens, numeric episode names, Chinese episode markers, and multi-episode ranges
- [x] 3.2 Generate movie candidates from title/year evidence, one-main-file evidence, sidecar evidence, and parent-directory title evidence
- [x] 3.3 Preserve competing movie and episode alternatives when evidence conflicts instead of selecting one hard winner immediately
- [x] 3.4 Add unit tests for movie-like titles with season-looking words, anime season directories, single-file movies, and ambiguous movie-vs-episode candidates

## 4. Bounded Sibling Grouping

- [x] 4.1 Reuse scan directory snapshots or bounded current-directory listings as sibling context for fast classification
- [x] 4.2 Implement grouping for consecutive episode sequences with shared title or path context
- [x] 4.3 Implement grouping for movie versions using normalized title stems, quality tokens, edition/cut tokens, language tokens, and release noise
- [x] 4.4 Detect independent movies in one directory and avoid merging them into one movie or one series group
- [x] 4.5 Add scanner tests for flat episode folders, multi-version movie folders, independent movies in one folder, and mixed attachment/main directories

## 5. Thresholds And Catalog Projection

- [x] 5.1 Add configurable confidence thresholds for confirmed-fast, provisional, and review-required classification outcomes
- [x] 5.2 Project high-confidence movie candidates to movie catalog items with main/version/attachment assets
- [x] 5.3 Project high-confidence episode candidates to stable series-season-episode hierarchy and asset links
- [x] 5.4 Preserve provisional decisions for asynchronous validation without blocking inventory persistence
- [x] 5.5 Mark low-confidence or conflicting decisions for governance review without silently committing final semantic type

## 6. Asynchronous Validation

- [x] 6.1 Queue slow validation for provisional or review-sensitive candidates without running ffprobe or provider searches in the fast path
- [x] 6.2 Use technical probe results such as duration and streams as additional evidence when available
- [x] 6.3 Compare provider movie and TV/episode validation results for competing candidates and update decisions without losing history
- [x] 6.4 Add tests proving fast scan works without provider/ffprobe and later validation can confirm or revise provisional outcomes

## 7. Governance Review And Learned Rules

- [x] 7.1 Add governance read endpoints or extend existing responses to show classification alternatives, evidence, confidence, affected files, and proposed actions
- [x] 7.2 Add correction actions for resolving a group as episode sequence, movie versions, independent movies, or attachments
- [x] 7.3 Persist source-scoped learned rules from user-confirmed corrections and apply them as evidence during future scans
- [x] 7.4 Add backend tests for review actions, rule scoping, rule application, and rescan behavior after correction

## 8. Frontend Review Experience

- [x] 8.1 Add or update review UI entry points for ambiguous classification decisions in the governance/settings surfaces
- [x] 8.2 Present user-facing choices using concrete outcomes instead of internal directory shape labels
- [x] 8.3 Show evidence, affected files, confidence, and expected catalog result before the user confirms a correction
- [x] 8.4 Add frontend typecheck coverage and focused component tests where existing test structure supports it

## 9. Verification

- [x] 9.1 Run `go test ./...` from `mibo-media-server/`
- [x] 9.2 Run `pnpm typecheck` from `web/`
- [x] 9.3 Manually verify demo-media source scanning covers one-file movie, explicit episode, flat numbered episodes, movie versions, trailers/extras, and ambiguous review outcomes
