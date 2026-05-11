## 1. Schema And Models

- [x] 1.1 Add media graph tables and Go models for graph nodes, graph edges, graph evidence, graph decisions, and decision-to-inventory mappings.
- [x] 1.2 Add stable keys and indexes for library scope, source scope, graph node type, graph group key, decision status, affected inventory file, and parser/evidence version.
- [x] 1.3 Add repository methods for upserting graph groups, replacing evidence for affected inventory files, recording decisions, and loading graph state for a library.
- [x] 1.4 Add development reset or rebuild support that can clear old graph decisions and old recognition-derived projections for one library.

## 2. Evidence Providers

- [x] 2.1 Convert existing filename signal extraction into media graph evidence without allowing it to create final movie or episode metadata directly.
- [x] 2.2 Convert sidecar parsing into media graph evidence and supplemental/resource role hints.
- [x] 2.3 Convert content-shape and path-tree directory analysis into media graph group proposals and evidence.
- [x] 2.4 Convert sibling matching, same-title matching, and variant detection outputs into graph edges or evidence instead of final metadata links.
- [x] 2.5 Add bounded supplemental detection for trailer, sample, extra, featurette, interview, deleted-scene, and behind-the-scenes files and folders.

## 3. Graph Construction

- [ ] 3.1 Build a graph constructor that loads inventory files for a library and emits directory, file, work group, resource candidate, and sidecar nodes.
- [ ] 3.2 Group `Show/Season 1/01.mkv` style folders into one series group, one season group, and ordered episode slots.
- [ ] 3.3 Group explicit episode filenames such as `S01E01`, `1x02`, `EP03`, and `第04集` into series and episode-run groups.
- [x] 3.4 Group single main-video folders into movie package groups when no stronger TV evidence exists.
- [x] 3.5 Group same-work multi-quality or edition files into movie version/resource candidates.
- [x] 3.6 Group folders with multiple distinct title/year videos into independent movie package groups or movie collection parent groups.
- [x] 3.7 Preserve ambiguous groups with evidence and alternatives instead of forcing movie or TV classification.

## 4. Graph Classifier And Decisions

- [ ] 4.1 Implement TV acceptance gates for explicit episode slots, season folders with numeric files, repeated sibling episode slots, TV sidecars, and manual rules.
- [x] 4.2 Implement movie acceptance gates for one-main-video groups, movie sidecars, title/year evidence, and manual rules.
- [x] 4.3 Implement version, collection, supplemental, and multi-episode decision logic.
- [x] 4.4 Implement hard conflicts for incompatible movie-vs-episode evidence, external ID disagreement, episode slot disagreement, and competing high-confidence parents.
- [x] 4.5 Persist accepted, provisional, review-required, and blocked decisions with evidence summaries, confidence, alternatives, and reason text.

## 5. Materialization And Projection

- [ ] 5.1 Implement graph materialization for movies, movie resources, movie versions, and supplemental resources.
- [ ] 5.2 Implement graph materialization for `series -> season -> episode -> resource` TV hierarchy.
- [ ] 5.3 Implement multi-episode resource links from one playable file to multiple episode slots with segment or ordering metadata.
- [ ] 5.4 Ensure materialization is idempotent across repeated scans, parser version changes, and sidecar updates.
- [x] 5.5 Refresh projections for affected movie items and full TV ancestor scopes after graph materialization.
- [ ] 5.6 Preserve inventory-backed review visibility for ambiguous groups without publishing normal movie or orphan episode items.

## 6. Workflow Integration

- [ ] 6.1 Wire library scan workflow to collect inventory, build media graph, classify graph groups, materialize accepted decisions, and refresh projections.
- [ ] 6.2 Stop current scan paths from directly calling old movie/episode metadata materialization or resource linking helpers.
- [x] 6.3 Queue automated metadata match only for graph-materialized `movie` and `series` roots.
- [x] 6.4 Skip unsupported season and episode items if they reach metadata match batches without failing the workflow.
- [ ] 6.5 Add targeted library rebuild behavior for development data that removes old orphan episode rows, movie fallback links, and stale recognition decisions before rescanning.

## 7. Old Code Cleanup

- [ ] 7.1 Delete or demote old scan classification helpers that depend on persisted library type or isolated per-file final type.
- [ ] 7.2 Delete or rewrite sidecar paths that mutate final item type after external IDs or links have already been created.
- [ ] 7.3 Delete or rewrite content-shape/path-tree code that performs final catalog materialization outside graph decisions.
- [ ] 7.4 Delete or rewrite sibling-matching fallbacks that directly merge or link metadata outside graph decisions.
- [ ] 7.5 Remove tests that only assert old final-decision behavior and replace them with graph fixture tests.

## 8. Verification

- [x] 8.1 Add graph fixture tests for `Show/Season 1/01.mkv` and `Show/Season 1/02.mkv` producing one series, one season, two episodes, and no movie candidates.
- [x] 8.2 Add graph fixture tests for single movie folders, movie versions, independent movie folders, movie collections, extras, samples, and trailers.
- [ ] 8.3 Add workflow tests proving scan completion is independent from metadata matching and probing enrichment.
- [x] 8.4 Add metadata match tests proving only movie and series roots are queued and unsupported episode/season batch inputs are skipped.
- [x] 8.5 Run `go test ./internal/library ./internal/recognition ./internal/catalog ./internal/metadata`.
- [ ] 8.6 Reset or rebuild local development media data, rescan the TV library, and verify nonzero series, season, episode, resource links, projections, and successful workflow status.
