## ADDED Requirements

### Requirement: Admin can manage Live TV playlist sources
The system SHALL allow authenticated administrators to create, list, update, and delete Live TV playlist sources backed by remote URLs.

#### Scenario: Create a remote playlist source
- **WHEN** an authenticated administrator submits a Live TV source with a name and remote playlist URL
- **THEN** the system creates the source with an enabled state and returns the canonical saved source record

#### Scenario: Reject an invalid source definition
- **WHEN** an authenticated administrator submits a Live TV source without a valid remote URL
- **THEN** the system rejects the request with a validation error and does not create the source

#### Scenario: Delete a source and its imported channels
- **WHEN** an authenticated administrator deletes an existing Live TV source
- **THEN** the system removes the source and its imported channel records from future source and channel listings

### Requirement: System imports channels from remote M3U and TXT playlists
The system SHALL support refreshing a Live TV source from remote `.m3u` and `.txt` playlist content and SHALL normalize imported entries into channel records.

#### Scenario: Refresh an M3U source
- **WHEN** an authenticated administrator triggers refresh for a reachable `.m3u` playlist source
- **THEN** the system parses channel entries, normalizes supported metadata fields, persists channel records, and records a successful refresh status

#### Scenario: Refresh a TXT source
- **WHEN** an authenticated administrator triggers refresh for a reachable `.txt` playlist source
- **THEN** the system parses supported channel definitions, normalizes channel records, and records a successful refresh status

#### Scenario: Refresh failure is observable
- **WHEN** the system cannot fetch or parse a remote playlist source during refresh
- **THEN** the system records the refresh failure status and a human-readable error for that source

### Requirement: Imported channels are browsable through a stable backend model
The system SHALL expose imported Live TV channels through a backend listing API that does not require the client to understand the original playlist format.

#### Scenario: List imported channels
- **WHEN** an authenticated user requests the Live TV channel list
- **THEN** the system returns normalized channel records including channel name and stream identity from the latest successful refresh

#### Scenario: Filter imported channels by source
- **WHEN** an authenticated user requests the Live TV channel list with a specific source filter
- **THEN** the system returns only channels imported from that source

#### Scenario: Preserve normalized metadata when available
- **WHEN** an imported playlist entry includes metadata such as group name, logo URL, or TVG identifiers
- **THEN** the system returns those normalized fields on the Live TV channel record
