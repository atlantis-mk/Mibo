## ADDED Requirements

### Requirement: Scanner classifies video content at work-group scope before final metadata collapse
The system SHALL derive a work-group classification for scanned video content before creating or reusing final movie or series metadata, and SHALL base that classification on directory-level and grouped evidence rather than only a single file title fallback.

#### Scenario: Movie version directory forms one work group
- **WHEN** a directory contains multiple playable movie files that share one normalized movie identity and differ only by release hints such as quality, edition, or codec
- **THEN** the scanner MUST classify them as one movie work group with multiple version resources
- **AND** it MUST NOT create one final movie metadata item per file

#### Scenario: Episode directory forms one series hierarchy
- **WHEN** grouped evidence contains a series title plus season and episode structure from path-tree, content-shape, or filename signals
- **THEN** the scanner MUST classify the group as series content before any movie fallback is attempted

### Requirement: Work-group classification uses confidence gates before metadata creation
The system SHALL assign each work-group decision a confidence-backed outcome of accepted, guarded, or review-required and SHALL only collapse accepted groups directly into final metadata.

#### Scenario: High-confidence movie group collapses immediately
- **WHEN** a work group has strong movie evidence without conflicting episode or series signals
- **THEN** the scanner MUST create or reuse final movie metadata for that group during materialization

#### Scenario: Conflicting group stays unresolved
- **WHEN** a work group contains conflicting movie and episode evidence without a stronger resolving signal
- **THEN** the scanner MUST preserve the inventory and resource facts
- **AND** it MUST mark the group review-required instead of forcing final movie or series metadata collapse

### Requirement: Weak movie fallback is gated by absence of stronger series evidence
The system SHALL only use weak movie fallback evidence such as normalized `title + year` when stronger series signals are absent and group-level evidence does not indicate a structured series hierarchy.

#### Scenario: Weak movie title loses to episode evidence
- **WHEN** a file has a plausible movie title and year but also has explicit episode markers or season-directory evidence
- **THEN** the scanner MUST prefer series classification or review-required handling over direct movie fallback

#### Scenario: Weak fallback remains unresolved without group support
- **WHEN** a file has only weak movie title evidence and no confirming group-level movie shape
- **THEN** the scanner MUST keep the item unresolved and browse-visible instead of immediately creating final movie metadata
