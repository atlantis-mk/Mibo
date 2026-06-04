## ADDED Requirements

### Requirement: Group Episode Failures By Season
The system SHALL group episode-level governance failures into one season-scoped issue when all affected episodes belong to the same series season.

#### Scenario: Same season has many metadata failures
- **WHEN** multiple episodes in the same season require metadata review for the same reason
- **THEN** the operations center shows one season-scoped issue with the affected episode count

#### Scenario: Season issue retains episode evidence
- **WHEN** a season-scoped issue is opened
- **THEN** the detail view exposes each affected episode, linked file, and underlying condition or decision

### Requirement: Group Multi-Season Failures By Series
The system SHALL group episode-level governance failures into one series-scoped issue when the same reason spans multiple seasons of one series.

#### Scenario: Multiple seasons share same failure reason
- **WHEN** episodes across multiple seasons of the same series fail with the same governance reason
- **THEN** the operations center shows one series-scoped issue with per-season counts

### Requirement: Preserve Movie And Standalone Behavior
The system SHALL avoid grouping unrelated movies or standalone files into episodic issues.

#### Scenario: Multiple movies have no candidates
- **WHEN** unrelated movie items each have no metadata candidate
- **THEN** the system creates separate movie-scoped issues unless they are explicitly part of the same collection governance problem

#### Scenario: Extras are not grouped as missing episodes
- **WHEN** trailer, sample, sidecar, or extra resources produce review facts
- **THEN** the system classifies them under resource or extra governance rather than series episode grouping

### Requirement: Fallback Scope
The system SHALL use the narrowest reliable fallback scope when series or season cannot be inferred.

#### Scenario: Episode-like files lack series metadata
- **WHEN** affected files look episodic but cannot be linked to a series or season
- **THEN** the system groups them by normalized folder scope and reason while surfacing that the series scope is unresolved
