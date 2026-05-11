## Context

`redesign-metadata-resource-model` introduced global metadata items, resources, resource-library links, resource-metadata links, library projections, metadata/resource user data, and resource-first playback. The application now has resource-first paths for browse, detail, playback, governance, favorites, search, and home surfaces, but legacy `CatalogItem`, `MediaAsset`, `AssetItem`, and `UserItemData` read paths still exist as compatibility fallbacks and as implementation dependencies in series hierarchy, scan exclusion, manual restructure, and progress code.

This change removes those remaining read-model dependencies after the resource-first implementation is already present. The target state is development-reset oriented: old local SQLite data is discarded rather than migrated, and product flows operate only on metadata/resource/projection state.

## Goals / Non-Goals

**Goals:**
- Remove product read dependencies on library-owned `CatalogItem` metadata, `MediaAsset` playback selection, `AssetItem` item links, and `UserItemData` progress/favorite state.
- Replace remaining series hierarchy, playback, progress, favorites, scan exclusion, and governance/restructure callers with metadata/resource/projection equivalents.
- Delete backend routes, services, helpers, frontend API wrappers, and tests that only exist for legacy catalog read semantics.
- Keep the app runnable from a fresh development database and demo media rescan.

**Non-Goals:**
- No migration of old local development databases.
- No compatibility guarantees for retired `/api/v1/media-items/*`, `/api/v1/media-files/*`, asset-link governance, or asset-selected playback contracts.
- No OpenList upstream changes.
- No redesign of the UI visual language beyond replacing data sources and actions.

## Decisions

1. Retire by replacement, not bulk deletion first.

   Replace each live caller with metadata/resource/projection behavior before deleting its legacy helper. This avoids breaking still-active UI paths and keeps each removal verifiable by focused tests.

2. Use `MetadataItem` as the product item identity and `Resource` as the playable/version identity.

   The previous `CatalogItem`/`MediaAsset` split encoded library ownership in the canonical item. The new model keeps canonical metadata global and scopes visibility/availability through `LibraryMetadataProjection` and `ResourceLibraryLink`.

3. Use `InventoryFile` only for file-level operations.

   Scan exclusion and reprobe flows that need a physical file anchor should target inventory files or source paths, not catalog items or assets. Playback and progress should not use inventory files except for explicit admin/debug file playback.

4. Treat manual restructure as resource governance.

   Existing manual restructure payloads still use asset IDs. They should move to resource IDs and resource metadata links so movie-version, independent-movie, and episode-sequence corrections update the resource graph directly.

5. Delete legacy tests only after equivalent resource-first coverage exists.

   Tests asserting `CatalogItem.library_id`, `AssetItem`, or `UserItemData` behavior should be rewritten against metadata/resource/projection outcomes or removed when the behavior is retired.

## Risks / Trade-offs

- [Risk] Removing fallbacks can expose missing resource/projection data after scans. → Mitigation: run a fresh demo-media rescan and add contract tests for browse, detail, hierarchy, playback, favorites, search, and home.
- [Risk] Series hierarchy behavior may regress when replacing `ListSeriesSeasons`. → Mitigation: add metadata hierarchy query tests for series, seasons, episodes, local-only shelves, missing episode visibility, and progress states.
- [Risk] Manual governance corrections are complex and currently asset-oriented. → Mitigation: migrate one correction family at a time and keep operation evidence assertions for link, relink, merge, split, and projection visibility.
- [Risk] Full deletion may require model/AutoMigrate changes that invalidate old DB files. → Mitigation: document fresh database requirement and keep dev reset guidance in `AGENTS.md`.
