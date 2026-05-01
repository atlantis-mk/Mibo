## 1. Baseline And Compatibility Tests

- [x] 1.1 Add focused tests documenting current automated TMDB movie match behavior before refactor
- [x] 1.2 Add focused tests documenting current automated TMDB series hierarchy match behavior before refactor
- [x] 1.3 Add focused tests documenting current MetaTube manual and automated movie behavior before refactor
- [x] 1.4 Add focused tests documenting current local sidecar evidence application and locked-field preservation
- [x] 1.5 Add focused tests documenting worker `match_catalog_item` skip behavior so the TMDB-only gate removal is explicit

## 2. Operation Contracts

- [x] 2.1 Define metadata operation request, operation type, result status, applied field, skipped field, warning, and affected-scope structs
- [x] 2.2 Define metadata execution plan and plan summary structs resolved from library metadata strategy
- [x] 2.3 Define provider attempt structs with stage, provider identity, outcome, error class, candidate count, and selected flag
- [x] 2.4 Define normalized metadata candidate, detail, image, people, external ID, and hierarchy structs
- [x] 2.5 Add compatibility adapters so existing match/refetch/manual APIs can return legacy result fields while carrying operation details internally

## 3. Operation Evidence Persistence

- [x] 3.1 Add database model and migration for operation-level metadata evidence or choose a documented source-backed interim representation
- [x] 3.2 Persist operation plan summary, provider attempts, selected candidate summary, applied fields, skipped fields, warnings, and status
- [x] 3.3 Link operation evidence to metadata sources created during the operation where practical
- [x] 3.4 Add tests for persisted attempt evidence on success, no-result, skipped provider, and provider failure outcomes

## 4. Strategy Resolution And Matchability

- [x] 4.1 Implement operation execution plan resolution from `library_metadata_strategies`
- [x] 4.2 Include provider capability, enabled state, configured state, availability, cooldown, and language overrides in plan resolution
- [x] 4.3 Implement strategy-based matchability checks for match, refetch, manual apply, and local apply operations
- [x] 4.4 Remove the worker's global TMDB API key gate and route `match_catalog_item` jobs to strategy-based operation execution
- [x] 4.5 Add tests for TMDB-only, MetaTube-only, local-evidence-only, no-provider, disabled-provider, and cooldown-provider strategies

## 5. Provider Execution Pipeline

- [x] 5.1 Implement stage executor that iterates providers in configured order and records attempt outcomes
- [x] 5.2 Implement TMDB search executor that returns normalized candidates and provider attempts
- [x] 5.3 Implement TMDB detail executor that returns normalized detail, images, people, external IDs, and TV hierarchy output
- [x] 5.4 Implement MetaTube search executor that returns normalized MetaTube candidates without TMDB identity semantics
- [x] 5.5 Implement MetaTube detail executor that returns normalized movie detail, images, people, and MetaTube external IDs
- [x] 5.6 Map provider HTTP/auth/rate-limit/timeout failures into provider availability updates and operation attempts
- [x] 5.7 Add fallback tests where primary provider is unavailable, returns no candidates, or fails retryably

## 6. Local Evidence Executor

- [x] 6.1 Implement scanner metadata source reader that extracts parsed sidecar hints, local image evidence, and sidecar external IDs
- [x] 6.2 Implement local evidence candidate seeding for sidecar-provided provider external IDs
- [x] 6.3 Implement local apply executor that applies supported sidecar fields through the shared operation contract
- [x] 6.4 Preserve `local_scan` provider-instance provenance while treating local evidence as a non-online executor
- [x] 6.5 Add tests for local-only strategy application, local evidence seeding remote detail, malformed sidecar evidence, and locked-field skip reporting

## 7. Field Application And Governance

- [x] 7.1 Implement field application policy for automated, manual, scanner/local, and system apply modes
- [x] 7.2 Route title, sort title, original title, overview, year, runtime, dates, ratings, and governance writes through the shared policy where applicable
- [x] 7.3 Record source attribution in field states for values applied from provider or local evidence sources
- [x] 7.4 Return skipped locked/manual fields in operation results without overwriting canonical values
- [x] 7.5 Standardize governance transitions for matched, needs-review, unmatched, manual, and skipped operations
- [x] 7.6 Add tests for high-confidence match, low-confidence review, no-candidate unmatched, manual apply, locked fields, and source attribution

## 8. Identity, Images, People, And Projections

- [x] 8.1 Implement explicit write rules for provider-facing external IDs versus stable catalog identities
- [x] 8.2 Move provider image application to normalized image outputs while preserving selected-image semantics
- [x] 8.3 Move provider people application to normalized people outputs while preserving cast/director roles and avatar URLs
- [x] 8.4 Collect affected item IDs during operations and refresh catalog projections once per operation scope
- [x] 8.5 Add tests for external ID ownership, identity ownership, image selection, people sync, and single-scope projection refresh

## 9. Operation Entry Point Migration

- [x] 9.1 Route `MatchCatalogItemWithResult` through the metadata operation orchestrator for movie items
- [x] 9.2 Route `RefetchCatalogItemWithResult` through the metadata operation orchestrator
- [x] 9.3 Route `ApplyCatalogCandidate` through the metadata operation orchestrator
- [x] 9.4 Route manual search candidate generation through normalized provider search executors where practical
- [x] 9.5 Preserve existing HTTP endpoint behavior with compatibility response mapping during migration
- [x] 9.6 Add integration tests for worker-triggered match, manual search/apply, and refetch API flows

## 10. TV Hierarchy Migration And Cleanup

- [x] 10.1 Move rooted series target resolution into the operation orchestrator while preserving episode/season origin result reporting
- [x] 10.2 Apply normalized TV hierarchy outputs to create or update season and episode descendants
- [x] 10.3 Preserve local asset links, missing/unaired availability states, and provider slot mismatch safeguards during hierarchy apply
- [x] 10.4 Record hierarchy-stage provider attempts and descendant-specific operation outcomes
- [x] 10.5 Remove or simplify obsolete TMDB-specific apply helpers after orchestrated paths cover equivalent behavior
- [x] 10.6 Run `go test ./...` from `mibo-media-server/` and document any unrelated pre-existing failures
