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

### Requirement: Series detail exposes a playable episode target
The system SHALL expose a nullable playback target for catalog series detail responses based on the user's progress and locally playable episode assets.

#### Scenario: User has an unfinished local episode
- **WHEN** an authenticated client requests detail for a series whose user has unfinished progress on a locally playable episode
- **THEN** the response MUST identify that episode as the series playback target with stable episode item identity, selected asset identity when available, display label, and a continue-playback selection reason

#### Scenario: User has no unfinished local episode
- **WHEN** an authenticated client requests detail for a series with locally playable episodes but no unfinished episode progress
- **THEN** the response MUST identify the earliest locally playable episode by season and episode ordering as the series playback target

#### Scenario: Series has no locally playable episodes
- **WHEN** a client requests detail for a series whose descendants are all missing, unaired, unavailable, or lack playable linked assets
- **THEN** the response MUST omit the series playback target instead of fabricating a playable series asset

### Requirement: Series playback requests resolve to the selected episode
The system SHALL resolve catalog playback requests for series items to the same locally playable episode target used by series detail.

#### Scenario: Client requests playback for a playable series
- **WHEN** an authenticated client requests playback for a catalog series that has a selected local episode target
- **THEN** the playback response MUST return source context for the resolved episode item and selected asset rather than returning a no-asset decision for the series item itself

#### Scenario: Client requests playback for a series without local episodes
- **WHEN** an authenticated client requests playback for a catalog series with no locally playable episode target
- **THEN** the playback response MUST be a clear unplayable decision and MUST NOT fail with a server error

### Requirement: Consumer series hierarchy can be scoped to local playable episodes
The system SHALL provide a consumer series hierarchy for detail-page shelves that contains only locally playable episode descendants.

#### Scenario: Series has mixed local and missing descendants
- **WHEN** a client requests the default consumer season hierarchy for a series containing available, missing, and unaired episodes
- **THEN** the response MUST include only seasons with locally playable episodes and MUST include only those local episodes in each season's episode list

#### Scenario: Client requests missing episode information explicitly
- **WHEN** a client requests the dedicated missing-episode series view or an explicit non-local availability view
- **THEN** the API MUST continue to return matching missing or unaired descendants instead of applying the consumer local-only shelf filter

### Requirement: Catalog detail exposes detailed video stream technical attributes
The system SHALL expose detailed video stream technical attributes as optional fields in catalog asset stream summaries.

#### Scenario: Client requests item detail with detailed video metadata
- **WHEN** a client requests catalog item detail for an item whose asset streams have captured video technical attributes
- **THEN** the response MUST include optional stream fields for profile, level, frame rate, field order or interlace state, stream bitrate, color space, bit depth, pixel format, and reference frame count where available

#### Scenario: Client requests item detail for unprobed or partially probed media
- **WHEN** a client requests catalog item detail for an item whose streams lack some detailed video technical attributes
- **THEN** the response MUST remain successful and omit unavailable optional fields instead of returning placeholder data as authoritative metadata

### Requirement: Detailed stream attributes preserve existing catalog detail compatibility
The system SHALL add detailed video stream attributes without changing existing catalog item, asset, file, or playback identity semantics.

#### Scenario: Existing client consumes compact stream fields
- **WHEN** a client only reads existing stream fields such as stream type, codec, dimensions, channels, language, title, bitrate, duration, and disposition flags
- **THEN** the response MUST continue to provide those fields with the same meaning after detailed video attributes are added

#### Scenario: Playback endpoint resolves an item
- **WHEN** a client requests playback for a catalog item or selected asset
- **THEN** detailed video technical fields MUST NOT be required for playback resolution or direct-play eligibility checks

### Requirement: Playback exposes bound external subtitle tracks
The system SHALL include bound external sidecar subtitles in catalog playback responses alongside embedded subtitle stream summaries.

#### Scenario: Playback item has a bound sidecar subtitle
- **WHEN** a client requests playback for a catalog item whose selected asset has a bound sidecar subtitle
- **THEN** the playback response MUST include a subtitle track with enough display and fetch information for the client to offer that subtitle during playback

#### Scenario: Playback item has embedded and sidecar subtitles
- **WHEN** a selected asset has both embedded subtitle streams and bound sidecar subtitle tracks
- **THEN** the playback response MUST preserve both kinds of subtitles and mark sidecar subtitles as external

### Requirement: Catalog detail exposes bound external subtitle tracks
The system SHALL include bound external sidecar subtitles in catalog asset detail stream summaries so clients can show subtitle availability before playback starts.

#### Scenario: Detail item has a sidecar subtitle
- **WHEN** a client requests item detail for a catalog item whose asset has a bound sidecar subtitle
- **THEN** the asset detail MUST include a subtitle stream summary for the external sidecar subtitle

#### Scenario: Detail item has no subtitles
- **WHEN** a client requests item detail for a catalog item with no embedded or bound external subtitles
- **THEN** the asset detail MUST remain valid and omit subtitle stream rows rather than fabricating unavailable subtitle placeholders

### Requirement: Catalog APIs expose mixed-library results as standard media items
The system SHALL expose items scanned from mixed content libraries through existing catalog movie and series response semantics without requiring clients to handle a new catalog item type.

#### Scenario: Client browses a mixed content library
- **WHEN** a client requests the item list for a mixed content library after scanning has produced movie and series catalog items
- **THEN** the API SHALL return standard catalog list items for those movies and series with the same identity, availability, artwork, and child summary fields used by dedicated movie and show libraries

#### Scenario: Client opens a mixed-library item detail
- **WHEN** a client requests detail or playback for a movie or series produced from a mixed content library
- **THEN** the API SHALL use the existing movie or series detail and playback contracts for that item type

