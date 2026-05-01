## ADDED Requirements

### Requirement: Media item DTO exposes Emby-like common fields
The system SHALL expose a media item DTO with Emby-like common fields for supported catalog item types.

#### Scenario: Client requests a media item DTO
- **WHEN** a client requests a supported catalog item through the media DTO API
- **THEN** the response MUST include stable `Id`, `Name`, `Type`, `Path`, provider IDs, image tags, people, and type-specific fields when available

### Requirement: Movie DTO includes media sources and streams
The system SHALL expose movie catalog items as Emby-like Movie DTOs with media source and stream details.

#### Scenario: Movie has a playable asset
- **WHEN** a movie item has an available media asset linked to an inventory file with probed streams
- **THEN** the Movie DTO MUST include `Type: Movie`, `MediaType: Video`, production metadata when available, and `MediaSources` containing file path, container, size, runtime ticks, and stream details

#### Scenario: Movie has multiple versions
- **WHEN** a movie item has multiple available version assets
- **THEN** the Movie DTO MUST expose each version as a separate media source or distinguishable source entry for client selection

### Requirement: Series DTO includes hierarchy summary fields
The system SHALL expose series catalog items as Emby-like Series DTOs with hierarchy summary fields.

#### Scenario: Series has seasons and local episodes
- **WHEN** a series item has season and episode descendants
- **THEN** the Series DTO MUST include `Type: Series`, `MediaType: Video`, provider IDs, selected image tags, `ChildCount`, and `RecursiveItemCount` derived from catalog hierarchy and rollup data

### Requirement: Season DTO includes series context
The system SHALL expose season catalog items as Emby-like Season DTOs with parent series context.

#### Scenario: Client requests a season DTO
- **WHEN** a season item has a parent series
- **THEN** the Season DTO MUST include `Type: Season`, `SeriesId`, `SeriesName`, `IndexNumber`, `Path`, provider IDs when available, image tags, and `ChildCount`

### Requirement: Episode DTO includes series, season, media source, and stream context
The system SHALL expose episode catalog items as Emby-like Episode DTOs with parent hierarchy and playback source context.

#### Scenario: Client requests an episode DTO
- **WHEN** an episode item has a parent season, root series, and an available media asset
- **THEN** the Episode DTO MUST include `Type: Episode`, `MediaType: Video`, `SeriesId`, `SeriesName`, `SeasonId`, `SeasonName`, `IndexNumber`, `ParentIndexNumber`, provider IDs, selected images, and media sources with streams

### Requirement: DTO runtime uses Emby tick units
The system SHALL convert duration seconds into Emby-compatible runtime ticks in media DTOs.

#### Scenario: Runtime seconds are available
- **WHEN** a catalog item or media asset has runtime or duration seconds
- **THEN** the DTO MUST expose `RunTimeTicks` equal to seconds multiplied by 10,000,000

### Requirement: DTO adapter does not replace internal catalog contracts
The system SHALL implement Emby-like media DTOs as an adapter over Mibo catalog and inventory data.

#### Scenario: Internal detail API remains available
- **WHEN** Emby-like DTO endpoints are added
- **THEN** existing catalog detail and governance APIs MUST remain available for internal workflows and MUST NOT be forced to use Emby field names
