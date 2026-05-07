## ADDED Requirements

### Requirement: Source-first classification uses directory shape without user semantics
The system SHALL classify source-first video content from directory shape profiles and plans without requiring users to choose movie, show, mixed, or broad media semantics for the library or source.

#### Scenario: Source contains mixed movies and episode packs
- **WHEN** a source contains both movie folders and large episode pack directories
- **THEN** the system SHALL classify each directory from content shape evidence and SHALL NOT require a source-level movie, show, or mixed selection

#### Scenario: Source shape is uncertain
- **WHEN** a source directory has ambiguous shape evidence
- **THEN** the system SHALL keep the source accepted, preserve inventory facts, and surface the directory or assignments as review-required after scanning

### Requirement: Directory-level review surfaces evidence
The system SHALL expose reviewable directory-level classification outcomes with shape, confidence, affected files, plan assignments, candidate alternatives, and supporting evidence when automatic classification is uncertain or conflicting.

#### Scenario: Ambiguous large directory needs review
- **WHEN** a large directory has both episode sequence evidence and independent movie collection evidence within a conflicting threshold range
- **THEN** the system SHALL create a review-required directory decision that includes the shape alternatives, confidence values, affected files, and proposed correction actions

#### Scenario: User correction creates scoped rule
- **WHEN** a user confirms a directory-level correction such as absolute episode pack, season folder, movie versions, or movie collection
- **THEN** the system SHALL store a source-scoped or path-scoped rule that can be used as evidence for future matching directories without applying outside its configured scope
