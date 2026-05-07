## Context

Mibo currently stores catalog identity, library membership, metadata state, availability, and playback/user anchors through `CatalogItem` and related item-scoped tables. This made sense during catalog cutover, but it now blocks the desired product model: one global metadata identity can have many resource versions across one or more libraries, and a library is a resource view rather than the owner of the work.

The project is still in development, so this change intentionally favors a clean model over compatibility shims. Development data can be reset and rescanned. Storage adapters, workflow execution, provider fetchers, filename signals, content-shape classification, and probe/playback low-level helpers should be reused where they still fit, but unused old catalog architecture code must be removed instead of left dormant.

## Goals / Non-Goals

**Goals:**

- Replace library-owned metadata rows with global metadata identities.
- Model resources, files, library membership, and metadata links as separate concepts.
- Support same-title and normalized-title matches as versions when identity evidence is strong enough.
- Support movie versions, episode versions, multi-part files, multi-episode files, trailers, extras, sidecars, and subtitles through resource links.
- Make library browsing/search/home data read from library projections over global metadata and resources.
- Move metadata source, external ID, field, image, people, and tag ownership from catalog items to metadata items.
- Split user data into metadata-level state and resource-level playback state.
- Remove retired old catalog code paths during implementation.

**Non-Goals:**

- Preserve old development database contents by default.
- Keep legacy `/media-items` or catalog compatibility shims alive once the new paths are wired.
- Automatically merge weak same-name matches without review evidence.
- Implement full library-specific metadata field overrides in the first pass.
- Change storage provider APIs unless required by the new resource graph.

## Decisions

### Decision: Make MetadataItem global and library-free

`MetadataItem` replaces `CatalogItem` as the canonical work identity. It has `item_type`, `content_form`, hierarchy links, canonical fields, governance status, external IDs, sources, images, people, and tags. It MUST NOT contain `library_id`.

Alternative considered: keep `CatalogItem` and add a global identity table beside it. That preserves compatibility but leaves two competing truths and keeps the old architecture alive. The implementation should instead migrate behavior to the new model and delete unused old code.

### Decision: Make Resource the library-owned media entity

`Resource` represents a playable or related media resource. Files attach to resources through `resource_files`. Library membership attaches through `resource_library_links`. This lets one resource appear in multiple libraries and lets multiple resources represent versions of one metadata item.

Alternative considered: keep `MediaAsset.library_id` as the only resource entity. The current asset model can be reused conceptually, but it lacks explicit file grouping, cross-library membership, and metadata-link governance. A dedicated resource graph is clearer.

### Decision: Use ResourceMetadataLink for recognition results

Resource-to-metadata relationships carry role, confidence, evidence, segment index, optional time bounds, source, and review state. This is the durable place for versioning, multi-episode files, trailers, extras, and corrections.

Alternative considered: write the resolved metadata ID directly on resource. That cannot represent multi-episode files or multiple roles without ad hoc fields.

### Decision: Add LibraryMetadataProjection as the browsing contract

Library pages, home rails, library-scoped search, availability, latest-added, and hierarchy rollups read from projection rows keyed by `library_id + metadata_item_id`. Projections are rebuildable derived state.

Alternative considered: query resources and metadata live for every library view. That increases endpoint complexity and makes sorting/search/rollups expensive. Projections keep product reads fast and isolate joins.

### Decision: Keep library metadata strategy as fetch/display context

Libraries do not own metadata, but a library can still provide preferred provider profile, language, image language, and country context for metadata fetches and display selection. Metadata sources record the triggering profile/provider/language context.

Alternative considered: make all metadata settings global. That would simplify storage but remove valid per-library language/provider behavior.

### Decision: Split user metadata data from user resource data

Favorites, watched state, and aggregate progress attach to `metadata_item_id`. Version-specific playback position and resource preference attach to `resource_id + metadata_item_id`. Playback can inherit metadata-level progress when a resource has no specific state.

Alternative considered: keep all user state on resource. That loses favorites and watched state when a file moves or a user switches versions.

### Decision: Reset development data by default

Because this is a development-stage architectural replacement, implementation should prefer dropping/recreating old catalog metadata tables and rescanning rather than writing a comprehensive production migration.

Alternative considered: full migration from `CatalogItem` to metadata/resource/projection tables. This is more work than value unless production data compatibility becomes a requirement.

## Risks / Trade-offs

- Global bad merges affect multiple libraries → use strong/medium/weak identity tiers, link review state, and split/relink governance actions.
- Library visibility becomes less direct than `catalog_items.library_id` → use resource library links and projections as the read model.
- Query complexity increases → maintain projection/search document builders and focused indexes.
- Multi-language metadata can conflict → store source language and field locale; display chooses fields using library context.
- Playback context can be ambiguous when a resource belongs to multiple libraries → playback requests should accept optional library context and otherwise select by user/resource history.
- Removing old code can temporarily break broad areas → implement by vertical slices and delete old paths only after equivalent tests pass.
- Development reset can lose local data → document reset expectation and keep demo media rescannable.

## Migration Plan

1. Add new schema and constants for metadata items, resources, links, projections, search docs, and split user data.
2. Stop creating new old catalog rows in scanner/materialization paths; create inventory files, resources, library links, metadata candidates, and resource metadata links instead.
3. Retarget metadata operations to metadata item IDs and write metadata-scoped sources/fields/external IDs/images/people/tags.
4. Build projections from resource library links and resource metadata links.
5. Switch library/home/search/detail/playback/favorites APIs to projection/resource semantics while preserving endpoint intent where practical.
6. Remove unused old catalog models, handlers, query code, compatibility routes, and tests that only validate retired architecture.
7. Reset development data and rescan sample libraries.

Rollback is not expected for development data. During implementation, rollback means reverting the change branch before data reset or recloning/resetting local data.

## Open Questions

- Should resource IDs replace asset IDs in all frontend contracts immediately, or should response aliases keep `asset_id` temporarily during UI migration?
- Should first-pass metadata fields support locale-specific values only, or also library-specific display overrides?
- Should all resources support multiple libraries from day one, or should the schema support it while UI only exposes single-library resources initially?
- Should adult/anime/documentary content forms be automatically inferred during scan in the first implementation or only stored when evidence exists?
