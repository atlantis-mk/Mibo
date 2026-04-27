## ADDED Requirements

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
