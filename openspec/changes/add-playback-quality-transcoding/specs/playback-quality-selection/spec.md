## ADDED Requirements

### Requirement: Playback variants are exposed to browser clients
The system SHALL include available playback variants in the playback source response for browser clients, including the original source and any compatible transcode options the backend can provide.

#### Scenario: Original quality is the default variant
- **WHEN** a browser client requests playback without specifying a variant
- **THEN** the system SHALL select the original variant by default

#### Scenario: Quality variants are available when FFmpeg can transcode
- **WHEN** FFmpeg transcoding is enabled and the selected media file is playable through backend access
- **THEN** the playback response SHALL include selectable target quality variants for 720P, 1080P, 2K, and 4K where they are applicable to the source media

#### Scenario: Upscaling variants are not offered by default
- **WHEN** a source media file has a known resolution lower than a target quality
- **THEN** the system SHALL omit or mark that higher target quality as unavailable unless the variant is needed for codec compatibility repair

### Requirement: Playback page provides quality and compatibility controls
The frontend SHALL provide a playback control that displays the currently selected variant and allows users to choose original quality, available quality transcodes, or audio compatibility repair.

#### Scenario: User opens the quality menu
- **WHEN** playback metadata includes more than one available variant
- **THEN** the playback page SHALL show a quality selector in the player controls with original quality selected initially

#### Scenario: User selects a quality variant
- **WHEN** a user selects 720P, 1080P, 2K, or 4K from the quality selector
- **THEN** the player SHALL request playback for that variant and switch to the returned stream while preserving the current playback position when possible

#### Scenario: User selects audio repair
- **WHEN** a user selects the audio compatibility repair option
- **THEN** the player SHALL request a variant that preserves original video when possible and converts unsupported audio to a browser-compatible format

### Requirement: Variant state is represented in playback information
The system SHALL identify the selected playback mode as original, audio repair, or quality transcode so the UI can display an accurate playback badge and troubleshooting state.

#### Scenario: Original source is selected
- **WHEN** the selected variant is original and direct playback is compatible
- **THEN** the playback response SHALL identify the mode as direct original playback

#### Scenario: Transcoded source is selected
- **WHEN** the selected variant requires FFmpeg processing
- **THEN** the playback response SHALL identify the mode as a fallback transcode and include the selected quality or repair label

#### Scenario: No compatible option exists
- **WHEN** original playback is incompatible and transcoding is unavailable or fails to initialize
- **THEN** the playback response SHALL report the media as unplayable with reasons that distinguish unsupported browser codecs from missing transcode capability
