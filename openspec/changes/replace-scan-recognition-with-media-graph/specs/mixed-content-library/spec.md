## MODIFIED Requirements

### Requirement: Mixed library scanning classifies multi-file groups as TV content
The scanner SHALL classify automatic video groups with more than one non-extra supported video file using media graph evidence-based movie, series, season, episode, version, and review semantics rather than unconditionally treating every multi-file group as TV-like content.

#### Scenario: Multiple episode-like videos create series hierarchy
- **WHEN** automatic video classification encounters a media graph group with multiple non-extra videos and episode or season evidence
- **THEN** the scanner MUST create or update a catalog series hierarchy for that group using graph materialization

#### Scenario: Multi-file group lacks explicit episode numbers
- **WHEN** automatic video classification encounters a media graph group without explicit episode numbers and with sufficient series-folder and sibling-order evidence
- **THEN** the scanner MUST assign deterministic episode ordering from the sorted non-extra files and record the inferred slots as graph evidence

#### Scenario: Multi-file movie versions are detected
- **WHEN** automatic video classification encounters a media graph group whose files appear to be versions of the same movie work
- **THEN** the scanner MUST create or update one movie item with multiple resources instead of creating a series hierarchy

#### Scenario: Multi-file group remains ambiguous
- **WHEN** automatic video classification cannot confidently distinguish series episodes from movie versions or unrelated videos
- **THEN** the scanner MUST mark the graph decision for governance review with evidence and confidence instead of silently choosing TV-like content

## ADDED Requirements

### Requirement: Mixed classification is graph-first
The system SHALL perform movie-vs-series classification at the media graph group level and SHALL NOT depend on a persisted library type or isolated per-file final type to decide catalog materialization.

#### Scenario: Bare numeric files appear under a season directory
- **WHEN** a source-first library contains `Series/Season 1/01.mkv` and `Series/Season 1/02.mkv`
- **THEN** graph-first classification MUST classify the files as episode resources under one series hierarchy
- **AND** it MUST NOT classify the files as independent movie items because their filenames are bare numbers

#### Scenario: One-file group has strong TV evidence
- **WHEN** a group has exactly one non-extra video but filename, sidecar, or directory evidence identifies a season and episode slot
- **THEN** graph-first classification MUST classify the group as TV content instead of applying the default one-file movie rule
