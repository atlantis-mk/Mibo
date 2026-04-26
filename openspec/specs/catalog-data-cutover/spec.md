# catalog-data-cutover Specification

## Purpose
TBD - created by syncing change tvg-catalog-kernel-remaining. Update Purpose after archive.

## Requirements

### Requirement: Catalog contracts define canonical item and asset semantics
The system SHALL treat `series`, `season`, `episode`, `movie`, and `extra` as the canonical catalog item types, and SHALL represent playable versions and extras through `media_assets`, `asset_items`, `asset_files`, and `inventory_files` rather than legacy `MediaItem` and `MediaFile` semantics.

#### Scenario: Catalog detail is built from canonical item and asset semantics
- **WHEN** backend code constructs catalog list, detail, season, episode, or governance DTOs
- **THEN** the payload MUST expose catalog item types, selected images, external identities, field states, source evidence, child summaries, and linked assets without relying on legacy-only fields such as `series_title` or `source_path`

### Requirement: Legacy backfill is idempotent and reports migration exceptions
The system SHALL provide a repeatable legacy backfill flow that maps legacy media rows into catalog items, media assets, inventory files, asset links, item images, external identities, metadata sources, and progress records without creating duplicate catalog entities on repeated runs.

#### Scenario: Re-running backfill does not duplicate catalog entities
- **WHEN** a backfill run is executed more than once against the same legacy movie, series episode, or media file records
- **THEN** the system MUST reuse or upsert the existing catalog items, media assets, asset-item links, and inventory files and MUST record conflicts, orphan files, and duplicate-episode candidates in the migration report instead of creating duplicate rows

### Requirement: New scan writes target the catalog kernel
The system SHALL write scan results into `inventory_files`, `media_assets`, `asset_files`, and catalog item hierarchies, and SHALL stop creating new legacy `media_items` and `media_files` during normal scan execution once catalog scan cutover is enabled.

#### Scenario: Scanning an episodic file produces catalog hierarchy and asset links
- **WHEN** the scanner processes a TV episode file with library, title, season, episode, and path evidence
- **THEN** the system MUST upsert the inventory file, create or reuse the series, season, and episode catalog items, create or reuse a media asset, link the asset to the episode through `asset_items`, and avoid creating new legacy media rows for that file

### Requirement: Catalog projections refresh after catalog mutations
The system SHALL refresh catalog rollups and search-document projections after catalog item, asset, metadata, or progress mutations that can change availability, hierarchy summaries, or searchability.

#### Scenario: Mutations refresh affected projections
- **WHEN** a backfill, scan, metadata update, or progress update changes a catalog item subtree or linked asset state
- **THEN** the system MUST update the affected `item_rollups` and `catalog_search_documents` before the item is served by catalog-backed list, detail, or search APIs
