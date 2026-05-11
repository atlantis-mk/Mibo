## ADDED Requirements

### Requirement: Catalog cutover removes legacy read-model dependency
The system SHALL complete catalog cutover by removing runtime dependencies on library-owned catalog item metadata and asset ownership tables after metadata/resource/projection replacements are in use.

#### Scenario: Catalog read-model helpers are removed
- **WHEN** the cutover cleanup is complete
- **THEN** backend product services MUST NOT use helpers whose only purpose is reading `CatalogItem.library_id` ownership, `MediaAsset` versions, or `AssetItem` item links

#### Scenario: Fresh scan populates canonical read data
- **WHEN** demo media is scanned into a fresh database
- **THEN** metadata items, resources, resource-library links, resource-metadata links, and library projections MUST be sufficient to serve browse, detail, search, and home responses

## REMOVED Requirements

### Requirement: Legacy backfill is idempotent and reports migration exceptions
**Reason**: The removal is development-reset oriented and does not support carrying old legacy media rows forward.
**Migration**: Reset local data or point the backend at a fresh SQLite database, then rescan media into the resource-first schema.

### Requirement: New scan writes target the catalog kernel
**Reason**: The catalog kernel tables are superseded by metadata/resource graph writes.
**Migration**: New scans write inventory files, resources, resource files, resource-library links, resource-metadata links, and library projections.

### Requirement: Catalog projections refresh after catalog mutations
**Reason**: Legacy catalog rollups/search documents are superseded by library metadata projections and metadata/library search documents.
**Migration**: Refresh `LibraryMetadataProjection`, metadata search documents, and library search documents after metadata/resource/user-data mutations.
