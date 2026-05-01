# detailed-video-technical-specs Specification

## Purpose
Define detailed video technical metadata capture, catalog-scoped exposure, and user-readable MediaInfo-style presentation.
## Requirements
### Requirement: Detailed video stream attributes are captured
The system SHALL capture detailed technical attributes for catalog video streams from probe data when those attributes are available.

#### Scenario: Probe returns detailed video stream metadata
- **WHEN** an inventory file probe returns video stream fields including codec, profile, level, dimensions, frame rate, field order, stream bitrate, color space, bit depth or pixel format, and reference frames
- **THEN** the system MUST persist those values on the corresponding media stream record without losing existing stream identity, language, title, duration, or dimensions

#### Scenario: Probe omits optional technical fields
- **WHEN** an inventory file probe returns a video stream without one or more detailed technical attributes
- **THEN** the system MUST persist the available values and leave missing detailed attributes empty without failing the probe

### Requirement: Detailed technical specs remain catalog scoped
The system SHALL associate detailed video technical specifications with catalog asset stream summaries rather than playback URL resolution.

#### Scenario: Catalog asset has probed video streams
- **WHEN** a catalog item detail includes an asset linked to inventory files with probed video streams
- **THEN** the item detail stream summaries MUST include the detailed technical attributes for those video streams as optional fields

#### Scenario: Catalog asset has sparse or legacy stream rows
- **WHEN** a catalog item detail includes stream rows created before detailed probing or from incomplete probe data
- **THEN** the item detail response MUST remain valid and include only the available stream fields

### Requirement: Video technical specs use user-readable formatting
The system SHALL present detailed video stream attributes in a user-readable technical specification format.

#### Scenario: User views a detailed video stream
- **WHEN** a user opens a media detail page for an item whose primary asset has detailed video stream attributes
- **THEN** the video information area MUST present fields for title, codec, profile, level, resolution, aspect ratio, interlace state, frame rate, bitrate, color space, bit depth, pixel format, and reference frames when available

#### Scenario: Technical value is unavailable
- **WHEN** a detailed video stream value is unavailable
- **THEN** the video information area MUST omit the unavailable optional value or show a clearly unknown value without implying the file is invalid

### Requirement: Technical streams map to media source DTO streams
The system SHALL map probed media stream records into Emby-like media source stream entries.

#### Scenario: DTO includes probed video, audio, and subtitle streams
- **WHEN** a media item DTO includes a media source backed by inventory files with probed streams
- **THEN** the DTO MUST expose stream entries with index, type, codec, language, dimensions, bitrate, frame rate, profile, level, channels, sample rate, and external subtitle path fields when available

### Requirement: Sparse technical data remains valid in DTO output
The system SHALL tolerate missing optional technical fields when building media DTOs.

#### Scenario: Stream row lacks optional probe fields
- **WHEN** an inventory file has sparse or legacy stream rows
- **THEN** the media source DTO MUST remain valid and include only available stream fields without failing the item response

