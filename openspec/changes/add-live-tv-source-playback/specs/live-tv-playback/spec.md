## ADDED Requirements

### Requirement: Users can request a Live TV playback source for an imported channel
The system SHALL provide an authenticated playback contract for imported Live TV channels without requiring them to exist as catalog items or inventory files.

#### Scenario: Resolve playback for an imported channel
- **WHEN** an authenticated user requests playback for an imported Live TV channel
- **THEN** the system returns a playback payload containing channel metadata and a backend-owned stream URL for that channel

#### Scenario: Reject playback for an unknown channel
- **WHEN** an authenticated user requests playback for a channel that does not exist
- **THEN** the system rejects the request with a not-found error

### Requirement: Live TV streaming is served through a backend-controlled stream path
The system SHALL serve imported Live TV channel streams through a backend-controlled path instead of exposing upstream provider URLs directly to the client.

#### Scenario: Proxy a reachable upstream stream
- **WHEN** an authenticated user requests the backend-owned stream URL for a reachable imported channel
- **THEN** the system proxies or relays the upstream stream response through the backend stream path

#### Scenario: Upstream stream failure is surfaced
- **WHEN** the backend cannot resolve or open the upstream stream for an imported channel
- **THEN** the system returns an error response indicating that the Live TV stream is unavailable

### Requirement: Live TV playback is isolated from deferred DVR and EPG features
The system SHALL support direct channel playback without requiring EPG, DVR, recording, or timeshift configuration.

#### Scenario: Playback succeeds without guide data
- **WHEN** an imported Live TV channel has no associated EPG or guide metadata
- **THEN** the system still allows playback if the channel stream URL is valid

#### Scenario: Playback contract excludes recording-only fields
- **WHEN** a client requests Live TV playback for an imported channel
- **THEN** the playback response omits DVR, recording schedule, and timeshift-specific requirements that are not implemented in this change
