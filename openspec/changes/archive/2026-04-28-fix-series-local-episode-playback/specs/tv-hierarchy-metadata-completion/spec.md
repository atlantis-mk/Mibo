## ADDED Requirements

### Requirement: TV hierarchy distinguishes consumer-local and operational-complete views
The system SHALL preserve complete provider-known TV hierarchy state while allowing consumer detail views to present only local playable episode descendants.

#### Scenario: Provider sync creates missing descendants
- **WHEN** provider metadata includes episodes that have no local file or have not aired yet
- **THEN** the catalog MUST keep those missing or unaired episode descendants with explicit availability state for governance and operational reads

#### Scenario: Consumer detail reads a series hierarchy
- **WHEN** the default consumer series detail view reads seasons and episodes for display
- **THEN** the hierarchy used by that view MUST exclude missing and unaired episode descendants that do not have local playable files

#### Scenario: Operational view reads complete hierarchy state
- **WHEN** a missing-episode, metadata governance, or explicit availability query reads TV hierarchy state
- **THEN** the system MUST expose the complete matching set of provider-known descendants including missing and unaired episodes
