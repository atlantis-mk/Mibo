## ADDED Requirements

### Requirement: Series detail exposes a playable episode target
The system SHALL expose a nullable playback target for catalog series detail responses based on the user's progress and locally playable episode assets.

#### Scenario: User has an unfinished local episode
- **WHEN** an authenticated client requests detail for a series whose user has unfinished progress on a locally playable episode
- **THEN** the response MUST identify that episode as the series playback target with stable episode item identity, selected asset identity when available, display label, and a continue-playback selection reason

#### Scenario: User has no unfinished local episode
- **WHEN** an authenticated client requests detail for a series with locally playable episodes but no unfinished episode progress
- **THEN** the response MUST identify the earliest locally playable episode by season and episode ordering as the series playback target

#### Scenario: Series has no locally playable episodes
- **WHEN** a client requests detail for a series whose descendants are all missing, unaired, unavailable, or lack playable linked assets
- **THEN** the response MUST omit the series playback target instead of fabricating a playable series asset

### Requirement: Series playback requests resolve to the selected episode
The system SHALL resolve catalog playback requests for series items to the same locally playable episode target used by series detail.

#### Scenario: Client requests playback for a playable series
- **WHEN** an authenticated client requests playback for a catalog series that has a selected local episode target
- **THEN** the playback response MUST return source context for the resolved episode item and selected asset rather than returning a no-asset decision for the series item itself

#### Scenario: Client requests playback for a series without local episodes
- **WHEN** an authenticated client requests playback for a catalog series with no locally playable episode target
- **THEN** the playback response MUST be a clear unplayable decision and MUST NOT fail with a server error

### Requirement: Consumer series hierarchy can be scoped to local playable episodes
The system SHALL provide a consumer series hierarchy for detail-page shelves that contains only locally playable episode descendants.

#### Scenario: Series has mixed local and missing descendants
- **WHEN** a client requests the default consumer season hierarchy for a series containing available, missing, and unaired episodes
- **THEN** the response MUST include only seasons with locally playable episodes and MUST include only those local episodes in each season's episode list

#### Scenario: Client requests missing episode information explicitly
- **WHEN** a client requests the dedicated missing-episode series view or an explicit non-local availability view
- **THEN** the API MUST continue to return matching missing or unaired descendants instead of applying the consumer local-only shelf filter
