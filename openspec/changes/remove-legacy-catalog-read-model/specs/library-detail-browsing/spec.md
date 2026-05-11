## ADDED Requirements

### Requirement: Library detail reads metadata projections
The system SHALL render library detail browse rows from library metadata projections and resource summaries rather than catalog item rows.

#### Scenario: User opens a library after fresh scan
- **WHEN** a user opens a scanned library
- **THEN** the page MUST list metadata projection rows with availability, artwork, progress, favorite state, and resource counts derived from resource-first data

### Requirement: Series hierarchy reads metadata hierarchy and projection state
The system SHALL render series, season, and episode shelves from metadata hierarchy plus resource/projection state rather than `CatalogItem` parent/child queries.

#### Scenario: User opens a series detail page
- **WHEN** the series has seasons and episodes in metadata hierarchy
- **THEN** the page MUST render season rails and playable episode rows using metadata item IDs and resource availability

#### Scenario: User opens an episode detail page
- **WHEN** the episode has parent series and season metadata
- **THEN** the page MUST show parent context and same-season siblings from metadata hierarchy without calling legacy series-season catalog endpoints

### Requirement: Organizing and file-level actions use inventory anchors
The system SHALL route file-level organizing, reprobe, and scan-exclusion actions through inventory-file or source-path anchors rather than catalog item or asset identities.

#### Scenario: User excludes a discovered file
- **WHEN** a user excludes or previews exclusion for a file-backed item
- **THEN** the request MUST identify the inventory file or source path and MUST NOT require a catalog item or asset ID
