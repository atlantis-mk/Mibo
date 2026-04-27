# catalog-api-playback Specification

## Purpose
TBD - created by syncing change tvg-catalog-kernel-remaining. Update Purpose after archive.

## Requirements

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

### Requirement: Catalog detail exposes immersive presentation metadata
The system SHALL expose catalog-backed media detail metadata needed by the immersive detail page as typed item detail fields.

#### Scenario: Client requests item detail for presentation
- **WHEN** a client requests a catalog item detail
- **THEN** the response MUST include available user-facing metadata such as community rating, official rating, year and end year or air-date range, series status, child summary, selected images, external identities, and displayable tags or genres

#### Scenario: Metadata is unavailable
- **WHEN** optional presentation metadata is unavailable for an item
- **THEN** the response MUST remain valid and omit or return empty values for optional fields without failing the detail request

### Requirement: Catalog detail exposes related media candidates
The system SHALL expose deterministic related media candidates for use by detail-page related shelves.

#### Scenario: Related media can be derived
- **WHEN** related items can be derived from catalog hierarchy, same-library relationships, shared tags, or other catalog-backed criteria
- **THEN** the response or companion catalog query MUST return candidates as catalog list items with selected images, year data, availability, and child summary fields

#### Scenario: No related media can be derived
- **WHEN** no related items are available for a detail item
- **THEN** the response or companion query MUST return an empty related list rather than synthetic placeholder items

### Requirement: Episode detail supports progress-aware shelves
The system SHALL make enough episode identity and user progress data available for detail episode shelves to display watched or in-progress state.

#### Scenario: Client renders a series episode shelf
- **WHEN** a client renders a series season hierarchy for a signed-in user
- **THEN** each playable episode MUST have stable catalog item identity and enough progress state or progress lookup support to render watched and in-progress indicators

### Requirement: Catalog item detail exposes episode parent context
The system SHALL expose explicit parent series and season context for catalog episode detail responses.

#### Scenario: Client requests episode item detail
- **WHEN** a client requests detail for a catalog episode
- **THEN** the response MUST include stable identifiers and display metadata for the parent series, containing season, season number, episode number, and available parent artwork needed to render an episode detail page

#### Scenario: Parent context is unavailable
- **WHEN** an episode item lacks valid parent hierarchy records
- **THEN** the response MUST remain valid and include enough item-level data for the client to show an incomplete hierarchy state

### Requirement: Catalog APIs provide same-season episode shelves
The system SHALL provide same-season sibling episode data for episode detail presentation.

#### Scenario: Client renders an episode detail page
- **WHEN** a signed-in client opens an episode that belongs to a catalog season
- **THEN** the API MUST make the containing season's episode list available with stable item IDs, labels, images, availability, runtime, overview, and progress or watched state when known

#### Scenario: Sibling progress is absent
- **WHEN** no user progress exists for sibling episodes
- **THEN** the episode shelf data MUST remain valid and omit progress values rather than failing the hierarchy request

### Requirement: Catalog asset detail exposes media stream summaries
The system SHALL expose probed media stream summaries for assets linked to catalog items.

#### Scenario: Asset has media streams
- **WHEN** an item detail response includes a linked asset whose source file has probed streams
- **THEN** the asset detail MUST include video, audio, and subtitle stream summaries sufficient for the client to render codec, resolution, language, title, channel, bitrate, duration, and known disposition metadata

#### Scenario: Asset has no probed streams
- **WHEN** an asset lacks stream rows or probe is pending
- **THEN** the asset detail MUST still include asset status, file IDs, probe status, and known file metadata without returning placeholder stream rows

### Requirement: Playback and detail asset selection remain consistent
The system SHALL use consistent asset identities between detail-page asset choices and playback requests.

#### Scenario: User selects an episode playback version
- **WHEN** the detail page presents multiple assets for an episode and the user chooses one
- **THEN** the playback request MUST target the same catalog item ID and asset ID shown in the detail asset list

#### Scenario: Requested asset is no longer linked
- **WHEN** a client requests playback for an asset that is no longer linked to the opened episode
- **THEN** the API MUST return a clear unplayable or unavailable decision instead of silently playing another episode's asset
