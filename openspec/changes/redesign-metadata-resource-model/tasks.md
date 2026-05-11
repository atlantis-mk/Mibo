## 1. Schema And Model Foundation

- [x] 1.1 Add backend database models for `MetadataItem`, metadata external IDs, metadata sources, metadata field states, metadata images, metadata people, and metadata tags without `library_id` ownership.
- [x] 1.2 Add backend database models for `Resource`, `ResourceFile`, `ResourceLibraryLink`, and `ResourceMetadataLink` with roles, segment fields, confidence, evidence JSON, and review state.
- [x] 1.3 Add backend database models for `LibraryMetadataProjection`, metadata search documents, and library search documents.
- [x] 1.4 Add backend database models for `UserMetadataData` and `UserResourceData`.
- [x] 1.5 Add constants and normalization helpers for metadata item type, content form, resource type, resource shape, link role, projection availability, and review state.
- [x] 1.6 Update database migration/AutoMigrate registration for the new model and remove migration registration for retired catalog tables when no longer referenced.
- [x] 1.7 Add focused database tests for unique indexes, hierarchy fields, metadata external ID uniqueness, resource file grouping, library membership, and projection uniqueness.

## 2. Resource Graph Services

- [x] 2.1 Implement service methods to upsert inventory files by media source/provider/path without metadata ownership assumptions.
- [x] 2.2 Implement service methods to create/update resources and attach source/subtitle/sidecar/image files through `ResourceFile`.
- [x] 2.3 Implement service methods to attach resources to libraries through `ResourceLibraryLink` and update first/last seen and missing state.
- [x] 2.4 Implement service methods to link/unlink resources to metadata identities through `ResourceMetadataLink`.
- [x] 2.5 Implement strong/medium/weak candidate classification for external ID, sidecar, series/season/episode, normalized title/year, and weak same-name matches.
- [x] 2.6 Add tests for same path rescan, same basename different paths, cross-library same resource membership, version links, weak match review, multi-part resources, and multi-episode resources.

## 3. Scanner And Materialization Rewrite

- [x] 3.1 Replace catalog item creation in library materialization with inventory file, resource, resource file, and resource library link writes.
- [x] 3.2 Reuse filename signal and content-shape logic to produce resource shape and metadata candidate evidence.
- [x] 3.3 Implement movie resource linking so external ID or normalized title/year matches create version links to one metadata identity.
- [x] 3.4 Implement series/season/episode resource linking using series identity plus season and episode numbers.
- [x] 3.5 Implement trailer, extra, sample, sidecar, and subtitle classification as resource/file/link roles rather than standalone catalog metadata items.
- [x] 3.6 Implement multi-episode resource linking with segment indexes.
- [x] 3.7 Update scan ingest events to reference resources, metadata identities, and projections instead of retired catalog item ownership.
- [x] 3.8 Add end-to-end scanner tests covering movie versions, episode versions, multi-episode files, extras/trailers, same-title different-year movies, and sidecar evidence.

## 4. Metadata Operation Pipeline

- [x] 4.1 Retarget metadata match/refetch/manual apply/local apply requests to metadata item IDs.
- [x] 4.2 Move metadata source, external ID, field state, image, people, and tag writes to metadata item ownership.
- [x] 4.3 Preserve triggering library metadata profile, provider instance, language, and fallback context on metadata sources and operation records.
- [x] 4.4 Implement fetch deduplication keys by metadata item, stage, language, and provider/profile context.
- [x] 4.5 Update existing identity shortcut logic to load external IDs from metadata identities and detail-fetch without search when confidence is high.
- [x] 4.6 Update TV hierarchy completion to create global series/season/episode metadata identities and links.
- [x] 4.7 Update local sidecar evidence application to attach evidence to resources/files and enrich metadata identities.
- [x] 4.8 Add metadata pipeline tests for cross-library duplicate fetch prevention, existing external ID detail fetch, localized sources, TV hierarchy identities, and local sidecar enrichment.

## 5. Projection And Search Read Models

- [x] 5.1 Implement projection builder for one library/metadata identity from resource library links and resource metadata links.
- [x] 5.2 Implement hierarchy rollup projection rebuilds for series, seasons, and episodes.
- [x] 5.3 Implement projection rebuild triggers for resource status, library membership, metadata link, metadata field, and user data changes.
- [x] 5.4 Implement metadata search document builder for global metadata search fields.
- [x] 5.5 Implement library search document builder for library-scoped projected metadata fields and resource text.
- [x] 5.6 Add projection tests for same metadata in multiple libraries, deleting one library resource, missing/available transitions, latest-added ordering, and TV rollups.
- [x] 5.7 Add search tests for global metadata search and library-scoped projection search.

## 6. Backend API Cutover

- [x] 6.1 Update library browse/detail endpoints to read `LibraryMetadataProjection` and return metadata identity IDs with resource summaries.
- [x] 6.2 Update item detail endpoints to return metadata identity fields plus optional library projection and visible resource context.
- [x] 6.3 Add or update resource/version list endpoints for a metadata identity with optional library filtering.
- [x] 6.4 Update home dashboard endpoints to aggregate libraries from projections and resource-backed ingest/projection state.
- [x] 6.5 Update favorites endpoints to store and read `UserMetadataData` while resolving library/resource context at read time.
- [x] 6.6 Update search endpoints to use metadata and library search documents.
- [x] 6.7 Remove retired catalog compatibility handlers and routes that are not used by the new frontend/backend flows.
- [x] 6.8 Add API contract tests for library browse, item detail, resources list, home sections, favorites, and search under the new semantics.

## 7. Playback And User State

- [x] 7.1 Update playback requests to resolve a metadata item and optional resource ID instead of a catalog item and asset ID.
- [x] 7.2 Implement resource selection using explicit resource, recent resource progress, preferred resource, primary role, availability, and quality policy.
- [x] 7.3 Implement multi-episode playback resolution using resource metadata link segment indexes and optional time bounds.
- [x] 7.4 Update subtitle selection to use the selected resource and provided library context.
- [x] 7.5 Write playback progress to `UserResourceData` and aggregate completion/progress to `UserMetadataData`.
- [x] 7.6 Implement progress inheritance from metadata-level progress when a resource has no resource-specific progress.
- [x] 7.7 Add playback tests for explicit resource playback, automatic version selection, cross-version progress inheritance, multi-episode resources, subtitles, and missing resources.

## 8. Governance And Manual Corrections

- [x] 8.1 Implement governance actions to link, unlink, and relink resources to metadata identities.
- [x] 8.2 Implement governance actions to change resource link roles and review state.
- [x] 8.3 Implement metadata identity merge and split operations that migrate resource links, projections, metadata sources, external IDs, and user metadata data.
- [x] 8.4 Implement library projection hide/show actions without deleting metadata identities or resources.
- [x] 8.5 Update field lock/manual edit actions to target metadata item field states.
- [x] 8.6 Add governance tests for relink, unlink, role change, hide/show projection, merge, split, and locked field protection.

## 9. Frontend Cutover

- [x] 9.1 Update TypeScript API types from catalog item/asset ownership semantics to metadata item/resource/projection semantics.
- [x] 9.2 Update library detail UI to render projection rows and resource/version summaries.
- [x] 9.3 Update media detail and episode detail UI to load metadata identity details and resource version lists.
- [x] 9.4 Update playback UI to pass resource IDs when selecting versions and metadata IDs for default playback.
- [x] 9.5 Update favorites UI to use metadata-level favorites and library/resource context.
- [x] 9.6 Update home dashboard and latest rails to consume projection-based responses.
- [x] 9.7 Update governance UI affordances for relink, merge/split, hide/show, and version role correction where existing UI supports manual review.
- [x] 9.8 Run `pnpm typecheck` and update affected frontend tests.

## 10. Remove Retired Architecture

- [ ] 10.1 Delete old catalog models, query helpers, and service methods that only support library-owned `CatalogItem` metadata after replacement code is in use.
- [ ] 10.2 Delete old metadata item-scoped tables or code paths that still target retired catalog item IDs.
- [ ] 10.3 Delete retired legacy routes and frontend API methods for old media item/file compatibility paths that are no longer used.
- [ ] 10.4 Delete or rewrite tests that assert old `CatalogItem.library_id` ownership rather than metadata/resource/projection behavior.
- [x] 10.5 Update AGENTS.md runtime notes if endpoint names, reset expectations, or catalog route guidance changes.

## 11. Verification And Reset

- [x] 11.1 Add a development reset note or script path for clearing old local catalog data before rescanning.
- [x] 11.2 Run backend focused tests for scanner/materialization, metadata pipeline, projection/query, playback, governance, and favorites.
- [x] 11.3 Run full backend `go test ./...` from `mibo-media-server/`.
- [x] 11.4 Run frontend `pnpm typecheck` from `web/`.
- [ ] 11.5 Manually rescan demo media and verify library browse, item detail, multi-version playback, favorites, search, and home dashboard.
