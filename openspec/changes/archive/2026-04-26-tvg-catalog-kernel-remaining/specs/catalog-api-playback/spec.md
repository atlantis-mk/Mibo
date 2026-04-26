## ADDED Requirements

### Requirement: Media discovery APIs read from catalog projections
The system SHALL serve library lists, item detail, series season hierarchies, and home or discovery-style media responses from catalog-backed DTOs instead of legacy `MediaItem` query shapes.

#### Scenario: Library items return catalog list semantics
- **WHEN** a client requests a library item list after backfill or catalog scanning has populated the new tables
- **THEN** the API MUST return catalog item DTOs rooted at movie and series items, with availability, governance, child summary, and selected-image data derived from catalog projections

### Requirement: Search and progress use catalog item and asset identities
The system SHALL index searchable media through catalog search documents and SHALL persist playback progress against catalog item and asset identifiers instead of legacy media-item and media-file identifiers.

#### Scenario: Saving progress records the selected catalog item and asset
- **WHEN** playback state is updated for a chosen catalog item and asset version
- **THEN** the system MUST write progress using the catalog item identity and, when available, the resolved asset identity so later continue-watching and playback requests can target the correct version

### Requirement: Playback resolves an item into the best available asset and file
The system SHALL accept playback requests rooted at a catalog item and resolve them to an available media asset and underlying file using explicit asset selection when provided and deterministic fallback ordering when it is not.

#### Scenario: Playback chooses a default asset when none is specified
- **WHEN** a client requests playback for an item that has multiple available asset versions and does not specify an asset identifier
- **THEN** the system MUST choose a default asset according to the configured availability and quality ordering, return the selected item, asset, and file context, and avoid returning a 500 when the item is known but currently unavailable

### Requirement: Migration-period compatibility is explicit
The system SHALL make migration-period compatibility behavior explicit for legacy media endpoints, either by bridging them to catalog data or by returning a clear deprecation outcome once catalog-backed clients are available.

#### Scenario: Legacy client calls a retired media-item endpoint after cutover
- **WHEN** a client calls a legacy media-item endpoint that is no longer the primary contract
- **THEN** the server MUST respond with an explicit compatibility or deprecation behavior defined by the migration phase rather than silently serving stale legacy-only data
