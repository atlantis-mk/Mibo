## Why

The current catalog model makes `CatalogItem` both the library-scoped display row and the metadata identity, which causes duplicate metadata across libraries, external ID ownership conflicts, repeated metadata fetching, and awkward version handling for same-title resources. Mibo is still in active development, so replacing the core model now is lower risk than layering compatibility behavior onto the wrong abstraction.

## What Changes

- **BREAKING** Replace the library-owned catalog identity model with a resource-first metadata graph: libraries organize resources, resources link to global metadata identities, and library views are derived projections.
- **BREAKING** Stop treating `CatalogItem.library_id` as metadata ownership; remove or retire old catalog/metadata code paths that are no longer used by the new architecture.
- Add global metadata identities for movies, series, seasons, episodes, collections, people, and related content forms without direct library ownership.
- Add resource and resource-file modeling so same-title files, renamed files, multi-part files, multi-episode files, trailers, extras, subtitles, and versions can be represented independently from metadata.
- Add resource-to-metadata links with roles, confidence, evidence, segments, and review state so multiple resources can point at the same metadata item and one resource can point at multiple metadata items.
- Add library metadata projections for browsing, searching, availability, rollups, and latest-added state within each library.
- Move metadata sources, external IDs, fields, images, people, and tags to target global metadata identities instead of library catalog rows.
- Split user data into metadata-level state and resource-level playback state so favorites and watched status can survive library/version changes while playback can remain version-aware.
- Update scan, materialization, metadata matching, playback, search, home/library browsing, and governance flows to use the new model.

## Capabilities

### New Capabilities

- `metadata-resource-graph`: Defines global metadata identities, resources, files, resource-to-metadata links, content forms, resource shapes, and version/multi-episode relationships.
- `library-metadata-projections`: Defines library-scoped projections over global metadata and resources for browsing, search, availability, hierarchy rollups, and latest-added views.
- `resource-aware-user-data`: Defines metadata-level favorites/watched state and resource-level playback progress/version selection.

### Modified Capabilities

- `library-detail-browsing`: Library detail pages browse projection rows instead of metadata identities owned by a library.
- `homepage-media-library-dashboard`: Home sections aggregate library projections and resource availability instead of library-owned catalog items.
- `catalog-api-playback`: Playback selects a resource for a metadata item instead of selecting an asset under a library-owned catalog item.
- `favorites-browsing`: Favorites bind to global metadata identities and resolve visible library/resource context at read time.
- `metadata-operation-pipeline`: Metadata operations target global metadata identities and record source/profile context separately from library ownership.
- `catalog-metadata-governance`: Governance actions operate on metadata identities and resource links, including split, merge, relink, hide/show projection, and field locks.
- `tv-hierarchy-metadata-completion`: TV hierarchy completion creates global series/season/episode metadata identities and projects only relevant library views.
- `sidecar-metadata-files`: Sidecar evidence attaches to resources/files and can create or enrich global metadata identities without making metadata library-owned.
- `library-metadata-profiles`: Library metadata strategy remains library-scoped but becomes fetch/display context rather than metadata ownership.
- `catalog-data-cutover`: Legacy catalog cutover requirements are superseded by the resource-first model and old unused catalog code must be removed rather than kept as compatibility shims.

## Impact

- Backend data model: replaces or retires `CatalogItem`-centered catalog metadata tables and adds metadata/resource/projection/user-data tables.
- Backend services: scan materialization, metadata matching, catalog queries, projection builders, playback selection, search indexing, governance, favorites, and user progress require changes.
- API contracts: existing item/library/playback/favorites endpoints keep product intent where practical but response IDs and internal semantics shift from catalog item IDs to metadata item/resource IDs.
- Frontend: library/home/detail/playback/favorites/governance views must consume metadata IDs, projection state, and resource version lists.
- Tests: old catalog ownership tests must be removed or rewritten around metadata identities, resources, links, and projections.
- Operations: development data should be reset and rescanned; complex compatibility migration is out of scope unless explicitly requested later.
