## Context

The current `/library` experience in the frontend uses the discovery-style browse flow and renders a flat list of media metadata items. On the backend, catalog browsing is driven by metadata projections and discovered inventory entries, which is useful for search-like exploration but does not preserve the folder hierarchy users intentionally create under a library root.

This change needs to span backend browse/query layers, HTTP API contracts, and frontend library navigation. The main constraint is that playback, metadata detail, visibility, and scan pipelines already work and should remain the source of truth once the user reaches a final metadata item. The new work should add a folder-aware navigation layer rather than replace item recognition or playback resolution.

## Goals / Non-Goals

**Goals:**
- Add a browse mode where the first level shows accessible libraries and deeper levels show filesystem-derived folders before metadata items.
- Reuse existing scanned inventory and metadata projection data so folder browse results reflect the current library contents without introducing a second recognition pipeline.
- Support stable folder navigation with breadcrumbs, parent navigation, pagination, and mixed result sets containing both folder nodes and metadata items.
- Keep existing item detail, playback authorization, and library visibility behavior intact once a metadata item is selected.

**Non-Goals:**
- Reworking the general discovery/search page into a folder browser for every surface in the app.
- Introducing write operations on folders such as rename, move, merge, or delete.
- Changing how metadata items are recognized, merged, or matched during scanning.
- Solving every special-case media layout in the first version, such as season-level virtual folders, box-set collapsing rules, or cross-library deduplication UX.

## Decisions

### 1. Add a dedicated hierarchical browse contract instead of overloading flat discovery responses

The backend will expose a dedicated hierarchical browse API for the library page. The response will return a `node_kind` for each entry (`library`, `folder`, or `item`) plus navigation context such as `node_id`, `parent_node_id`, `path_segments`, and breadcrumbs.

Rationale:
- The current discovery response is item-centric and assumes every card is directly playable or detail-addressable.
- Folder nodes need different affordances, counters, and navigation semantics than metadata items.
- A separate contract lets the frontend evolve the library page without forcing home, favorites, or search to understand synthetic folder nodes.

Alternatives considered:
- Extending the existing discovery payload with optional folder nodes. Rejected because it would complicate every existing consumer and blur the distinction between search results and navigation nodes.
- Building hierarchy entirely on the frontend from a flat item list. Rejected because pagination, visibility filtering, and mixed discovered/projected item states belong on the server.

### 2. Build folder nodes from scanned inventory paths and attach metadata items at the deepest relevant folder

The browse service will derive folder nodes from `inventory_files.storage_path` scoped to a library root. Metadata items will be associated with the folder path of their linked primary resources, while unrecognized/discovered files can still surface as final item nodes when they are the only leaf entries under a folder.

Rationale:
- Inventory paths already represent the canonical filesystem structure the user expects to browse.
- This keeps folder hierarchy aligned with the actual storage layout, including plugin-backed libraries whose files were normalized during scan.
- Reusing resource-to-metadata links avoids inventing a second hierarchy table before the first version proves out.

Alternatives considered:
- Storing a fully materialized folder tree in a dedicated table during every scan. Deferred because it adds migration and synchronization complexity before we validate the UX and query shape.
- Building hierarchy from metadata fields like region or genre. Rejected because the requested behavior is explicitly folder-driven, not taxonomy-driven.

### 3. Materialize a read-optimized browse node view keyed by library and relative folder path

The backend design will introduce a read model or query helper that groups inventory-backed resources by relative folder path under each library root and produces:
- child folder summaries for the current path
- direct metadata item leaves for the current path
- counts for descendant items where useful for UI badges

This may be implemented as a cached table refreshed during projection rebuilds or as a query-layer aggregation over inventory/resource/projection tables, but the contract should preserve a stable folder node identifier derived from `library_id + relative_path`.

Rationale:
- Hierarchical browsing needs consistent addressing for breadcrumbs and paging.
- Computing every folder level ad hoc from raw files on each request may become too expensive for large libraries.
- Tying refresh to existing scan/projection lifecycle keeps folder browse data coherent with inventory and metadata state.

Alternatives considered:
- Pure on-demand SQL aggregation for every request. Acceptable for a prototype but riskier for large libraries with deep nesting.
- Encoding folder state only in opaque frontend cursors. Rejected because shareable URLs and direct reloads would become fragile.

### 4. Keep metadata item cards and routes unchanged after leaf selection

When a browse result is an `item`, the frontend will continue navigating to the existing metadata detail/playback routes using the current metadata item identifier or inventory-file fallback identity. Folder and library nodes only affect traversal into the list, not what happens after the user selects playable content.

Rationale:
- Existing detail, favorites, progress, and playback flows are already integrated with current item identifiers.
- This sharply limits regression risk and keeps the scope focused on navigation rather than content presentation.

Alternatives considered:
- Introducing a new unified hierarchical route for both folders and item detail. Rejected because it would duplicate working media detail behavior.

## Risks / Trade-offs

- [Large libraries may create expensive folder aggregations] -> Mitigation: define the API around a read model or cacheable aggregation boundary and cap child-result page sizes from the start.
- [One metadata item may map to resources in multiple folders] -> Mitigation: define a deterministic primary browse path, such as the preferred primary resource path or the shallowest linked folder, and document duplicate suppression rules.
- [Recognized series structures may not align perfectly with raw filesystem folders] -> Mitigation: keep the first version folder-driven and treat richer virtual grouping as a future enhancement.
- [Folder nodes could reveal inaccessible content counts] -> Mitigation: apply library visibility filters before aggregation and only compute nodes from already-accessible libraries.
- [Mixed discovered and organized items may feel inconsistent] -> Mitigation: keep existing organizing state badges on leaf items and limit folder nodes to navigation responsibilities only.

## Migration Plan

1. Add the backend hierarchical browse contract and supporting query/read model behind new API endpoints.
2. Backfill or rebuild browse-node data from existing library inventory/projection state during startup or the next library projection refresh.
3. Update the frontend library route to use the hierarchical API and render library roots, folders, and breadcrumbs.
4. Keep the current flat discovery implementation available for other surfaces while validating the library page behavior.
5. Roll back by switching the frontend route back to the flat browse query and ignoring the new hierarchical endpoints if regressions appear.

## Open Questions

- Should empty folders be returned when scan results exist but all descendant files are currently filtered out or hidden?
- Should the first version mix folders and metadata items in one sorted list, or always render folders ahead of items for faster traversal?
- Do we want the library landing page to be exclusively hierarchical, or should there be a toggle between folder browse and the current flat filtered discovery mode?
