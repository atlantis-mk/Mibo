## ADDED Requirements

### Requirement: Descendant TV catalog items retain durable provider identity and evidence
The system SHALL persist season-level and episode-level provider identities, evidence snapshots, and artwork candidates for catalog descendants that are generated or updated from a series-root metadata match.

#### Scenario: Series match populates season and episode descendants
- **WHEN** a series catalog item is matched or refreshed from a TV metadata provider
- **THEN** the system MUST ensure the corresponding season and episode catalog items retain durable descendant identities, source evidence, and artwork candidates needed by governance and descendant APIs

### Requirement: TV hierarchy queries are served from explicit catalog hierarchy reads
The system SHALL expose season, episode, and child-list reads from explicit catalog hierarchy queries instead of relying on legacy grouping behavior or compatibility-only presentation logic.

#### Scenario: Client requests a series hierarchy view
- **WHEN** a client requests children, seasons, or episodes for a catalog-backed series
- **THEN** the system MUST serve the hierarchy from durable catalog parent-child relationships and descendant state rather than reconstructing the hierarchy from legacy media rows

### Requirement: Missing and unaired TV descendants remain explicit catalog states
The system SHALL preserve missing and unaired episodes as durable catalog descendants with explicit availability semantics even when no local playable asset exists.

#### Scenario: Provider-known episode has no local asset
- **WHEN** provider metadata includes an episode that has not been scanned locally or has a future air date
- **THEN** the corresponding catalog episode MUST remain queryable with an explicit missing or unaired availability state instead of disappearing from the hierarchy

### Requirement: TV convenience reads expose hierarchy-specific operational views
The system SHALL provide convenience reads for TV-specific views such as missing episodes and next-up calculations where those views are not yet available from the current cutover.

#### Scenario: Client requests a missing or next-up series view
- **WHEN** a client requests a TV convenience view for a matched series
- **THEN** the system MUST derive the response from catalog hierarchy, descendant availability, and user progress data without depending on legacy media-item semantics
