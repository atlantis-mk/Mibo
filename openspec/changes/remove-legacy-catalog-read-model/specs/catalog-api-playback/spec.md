## ADDED Requirements

### Requirement: Playback uses metadata and resource identities only
The system SHALL resolve normal media playback from a metadata item and optional resource identifier, and SHALL NOT accept asset identifiers for product playback selection.

#### Scenario: User selects a playback version
- **WHEN** the detail page presents multiple playable versions
- **THEN** each choice MUST send the selected resource ID to playback and MUST NOT send an asset ID

#### Scenario: Playback request has no explicit resource
- **WHEN** a client requests playback for a metadata item without a resource ID
- **THEN** the backend MUST select a playable resource using resource-first selection policy and return metadata/resource source context

### Requirement: Progress uses metadata and resource user data only
The system SHALL persist and read normal playback progress through metadata/resource user-data records and SHALL NOT fall back to `UserItemData` for product continue-watching behavior.

#### Scenario: Playback progress is saved
- **WHEN** a user watches a selected resource
- **THEN** the backend MUST write resource-specific progress and aggregate metadata progress without writing catalog item or asset progress state

#### Scenario: Continue watching loads
- **WHEN** a user opens continue-watching after progress has been saved
- **THEN** the response MUST resolve display and playback context from metadata/resource/projection state

## REMOVED Requirements

### Requirement: Search and progress use catalog item and asset identities
**Reason**: Progress now uses metadata item and resource identities.
**Migration**: Use metadata/resource progress APIs and response fields.

### Requirement: Playback resolves an item into the best available asset and file
**Reason**: Playback now resolves a metadata item into the best available resource and file.
**Migration**: Request `/api/v1/items/{metadata_item_id}/playback` with optional `resource_id`.

### Requirement: Playback and detail asset selection remain consistent
**Reason**: Detail version selection now exposes resource choices instead of asset choices.
**Migration**: Use `/api/v1/items/{metadata_item_id}/resources` and pass selected `resource_id` to playback.
