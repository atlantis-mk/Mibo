## ADDED Requirements

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
