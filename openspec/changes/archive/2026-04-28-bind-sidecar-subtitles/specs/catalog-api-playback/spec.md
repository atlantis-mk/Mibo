## ADDED Requirements

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
