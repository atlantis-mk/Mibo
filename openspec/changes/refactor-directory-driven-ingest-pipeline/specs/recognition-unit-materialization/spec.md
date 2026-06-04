## ADDED Requirements

### Requirement: Recognition units are derived from directory shape plans
The system SHALL create durable recognition units from content shape plans and assignments before building recognition manifests.

#### Scenario: Single movie directory creates one unit
- **WHEN** a directory shape plan identifies a `movie_folder`
- **THEN** the system creates one recognition unit for the directory using the assigned primary movie files, sidecars, file signals, and directory context

#### Scenario: Movie versions directory creates one movie unit
- **WHEN** a directory shape plan identifies a `movie_versions_folder`
- **THEN** the system creates one recognition unit whose files materialize to one movie work with multiple resource variants or editions

#### Scenario: Movie collection directory creates per-work units
- **WHEN** a directory shape plan identifies a `movie_collection_folder`
- **THEN** the system creates recognition units grouped by planned movie work identity rather than one unbounded manifest for the entire collection

#### Scenario: Episode directory creates episodic units
- **WHEN** a directory shape plan identifies a `season_folder`, `episode_pack`, `absolute_episode_pack`, or `flat_episode_folder`
- **THEN** the system creates recognition units that preserve series, season, episode, and multi-episode assignments from the directory plan

#### Scenario: Attachment directory does not create primary work unit
- **WHEN** a directory shape plan identifies an `attachment_group`
- **THEN** the system creates only supplemental or linkable resource work as appropriate and MUST NOT schedule remote metadata matching for a primary work from that directory alone

### Requirement: Recognition units are fingerprinted and idempotent
The system SHALL store a fingerprint for each recognition unit that includes its source shape plan, assignment version, file membership, file signals, sidecar evidence, and classifier version.

#### Scenario: Recognition unit is unchanged
- **WHEN** a recognition unit fingerprint matches the last successfully materialized fingerprint
- **THEN** the system skips manifest rebuild and materialization for that unit

#### Scenario: Recognition unit input changes
- **WHEN** file membership, assignment, signal, sidecar evidence, or classifier version changes for a recognition unit
- **THEN** the system marks the unit stale and schedules recognition manifest rebuild and materialization

### Requirement: Manifests are built from recognition units
The system SHALL build recognition manifests, candidates, evidence, media graph records, and decisions from recognition unit inputs.

#### Scenario: Unit materialization needs a manifest
- **WHEN** a stale recognition unit is ready to materialize
- **THEN** the system builds or updates the recognition manifest for that unit using persisted inventory facts, file signals, sidecar inputs, and directory assignments

#### Scenario: Existing recognition keys remain stable
- **WHEN** a recognition unit is materialized after the refactor
- **THEN** the system preserves canonical movie, series, season, episode, resource, variant, and edition keys compatible with existing metadata/resource records

### Requirement: Materialization consumes recognition unit results
The system SHALL materialize metadata, resources, resource-file links, and resource-metadata links from accepted recognition unit decisions and record the resulting IDs as the unit materialization output.

#### Scenario: Materialization succeeds
- **WHEN** a recognition unit has accepted work and playable resource decisions
- **THEN** the system upserts metadata items, resources, file links, metadata links, and stores the materialized metadata IDs, resource IDs, file IDs, projection IDs, and completed fingerprint for the unit

#### Scenario: Review required unit is not silently materialized
- **WHEN** a recognition unit is based on a review-required directory shape or has blocking recognition conflicts
- **THEN** the system records the review state and MUST NOT materialize accepted primary metadata/resources until an accepted decision or rule resolves the unit
