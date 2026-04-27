## ADDED Requirements

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
