## 1. Schema And Core Types

- [x] 1.1 Add recognition manifest, candidate, evidence, decision, conflict, and resolver rule database models or replace the existing scanner decision schema with equivalent resolver-owned tables.
- [x] 1.2 Add stable key types for work candidates, episode candidates, playable resource candidates, variant candidates, edition candidates, supplemental candidates, and duplicate binary evidence.
- [x] 1.3 Add repository helpers to create, load, update, supersede, and query manifests by library/source/root path, inventory file, candidate key, resolver state, and affected metadata/resource IDs.
- [x] 1.4 Add migration/reset documentation for clearing old development scan state before validating the new resolver architecture.

## 2. Manifest Builders

- [x] 2.1 Build an inventory-to-manifest pipeline that consumes `InventoryFile`, `ResourceFile` facts where relevant, indexed file signals, sidecar associations, scan exclusion state, and source scope.
- [x] 2.2 Convert filename signal extraction into manifest evidence for title, year, series title, season, episode slots, release hints, edition/cut hints, role hints, and anti-misclassification evidence.
- [x] 2.3 Convert sidecar metadata parsing into manifest evidence for local title/year, series/season/episode, provider identity, field hints, and sidecar parse failures.
- [x] 2.4 Convert directory content-shape and path-tree analysis into bounded context evidence providers without allowing them to create metadata or resource links directly.
- [x] 2.5 Build manifest candidate grouping for movie works, series/season/episode hierarchy, independent movie collections, sibling movie versions, multi-episode files, multi-part resources, and supplemental media.

## 3. Resolver Engine

- [x] 3.1 Implement resolver rule precedence with manual split/merge/classification rules applied before automatic heuristic evidence.
- [x] 3.2 Implement acceptance gates for supported external identity, sidecar identity, series-season-episode tuple, normalized movie title/year with compatible variant evidence, same-binary duplicate evidence, and manual resolver rules.
- [x] 3.3 Implement blocking conflicts for incompatible external IDs, incompatible years, incompatible media types, incompatible episode tuples, competing high-confidence candidates, and manual split rules.
- [x] 3.4 Implement resolver outcomes for accepted, provisional, review-required, blocked-conflict, unmatched, and superseded decisions with persisted evidence and alternatives.
- [x] 3.5 Add focused resolver tests for same-folder movie versions, sibling-folder movie versions, independent movie collections, standard TV, flat episode folders, multi-episode files, trailers/extras, same-hash duplicates, and external-ID conflicts.

## 4. Idempotent Materialization

- [x] 4.1 Implement resolver-owned materialization for movie, series, season, and episode `MetadataItem` records using stable candidate keys and accepted identity evidence.
- [x] 4.2 Implement resolver-owned `Resource`, `ResourceFile`, `ResourceLibraryLink`, and `ResourceMetadataLink` creation for single-file, multi-part, multi-episode, variant, edition, duplicate, and supplemental resource candidates.
- [x] 4.3 Preserve variant, edition, supplemental role, confidence, review state, segment index, and resolver evidence on resource metadata links or associated resolver records.
- [x] 4.4 Make materialization rerunnable after manifest rebuilds, classifier version changes, sidecar updates, hash/probe enrichment, and user correction without duplicating metadata/resource graph rows.
- [x] 4.5 Refresh metadata projections and search documents from affected resolver materialization outputs only.

## 5. Scan Pipeline Integration

- [x] 5.1 Change scan synchronization to persist inventory facts and recognition manifest candidates before any resolver materialization.
- [x] 5.2 Wire scan jobs to run resolver/materializer after manifest creation when local evidence is sufficient and to schedule follow-up resolver work when asynchronous evidence is pending.
- [x] 5.3 Preserve inventory-backed skeleton or review visibility for ambiguous files without creating weak fallback metadata items.
- [x] 5.4 Ensure fast scans still avoid ffprobe, content hashing, remote provider calls, artwork downloads, and media file reads in the synchronous classification path.
- [x] 5.5 Add integration tests proving scan order does not affect primary/version/resource grouping outcomes.

## 6. Governance And Corrections

- [x] 6.1 Update governance read APIs to expose resolver manifest candidates, conflicts, alternatives, evidence, linked resources, and proposed correction actions.
- [x] 6.2 Update merge, split, classify-as-versions, classify-as-independent, classify-as-series, and attachment correction flows to write resolver rules instead of patching legacy scanner decisions.
- [x] 6.3 Ensure resolver rules are source/path/evidence scoped and override automatic evidence on future scans.
- [x] 6.4 Update governance tests for review-required decisions, conflict blocking, manual merge, manual split, and rescan rule reuse.

## 7. Cleanup Of Replaced Architecture

- [x] 7.1 Remove direct scanner calls that create or link metadata from `writeCatalogScanMovie*`, `writeCatalogScanEpisode*`, and related catalog scan helpers, or rewrite them behind the resolver materializer.
- [x] 7.2 Delete or demote `same_metadata_sibling_matching` final decision logic so it no longer performs metadata lookup/link decisions outside the resolver.
- [x] 7.3 Delete or demote content-shape assignment and path-tree work-group code that acts as a final materialization override; keep only evidence extraction needed by manifest builders.
- [x] 7.4 Remove legacy weak title/year fallback metadata creation and any guarded placeholder path that creates durable metadata outside resolver decisions.
- [x] 7.5 Remove obsolete tests that assert old matcher-specific or scan-order-dependent final behavior and replace them with resolver manifest/materialization tests.
- [x] 7.6 Remove unused models, constants, repository helpers, routes, and response fields left behind by the retired recognition paths.

## 8. Verification

- [x] 8.1 Run backend unit tests for library scanning, resolver, inventory, governance, catalog projection, and metadata operations.
- [x] 8.2 Run `go test ./...` from `mibo-media-server/` and fix failures caused by the architecture replacement.
- [ ] 8.3 Reset local development data, rescan demo media with `MIBO_LOCAL_ROOT_PATH=/Users/atlan/Desktop/IdeaProjects/Mibo/demo-media`, and verify browse/detail/playback/progress/favorites/search/home still use resolver-materialized metadata/resource graph rows.
- [ ] 8.4 Validate same-folder versions, sibling-folder versions, cross-folder same external ID, same hash duplicates, conflicting external IDs, independent movie collections, TV hierarchy, multi-episode files, extras, sidecars, scan exclusion, and reprobe flows manually or through tests.
- [x] 8.5 Run `openspec validate replace-scanner-with-recognition-manifest --strict` and resolve any spec/task validation issues.
