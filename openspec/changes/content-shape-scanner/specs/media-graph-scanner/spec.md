## ADDED Requirements

### Requirement: Scanner compiles directory plans before high-confidence catalog projection
The system SHALL compile or reuse directory content shape plans before projecting high-confidence directory groups into catalog rows, and SHALL materialize covered files from plan assignments rather than from repeated independent file-first classification.

#### Scenario: Large episode directory is scanned
- **WHEN** a source-first scan encounters a large directory whose profile compiles to a high-confidence episode plan
- **THEN** the scanner SHALL create or reuse the planned series, season, episode items, assets, and asset links from the directory plan assignments
- **AND** the scanner SHALL NOT rebuild full movie-versus-episode classification for each covered file

#### Scenario: Directory plan and database batching differ
- **WHEN** a directory contains more files than a catalog materialization write batch can handle
- **THEN** the scanner SHALL compile or reuse the directory plan once and SHALL allow catalog writes to proceed in batches without recompiling the directory plan for each batch

#### Scenario: Directory plan cannot classify safely
- **WHEN** a directory plan has low confidence or conflicting movie, episode, version, and collection evidence
- **THEN** the scanner SHALL preserve inventory facts and scanner decisions for review instead of projecting unrelated catalog items from weak per-file guesses

### Requirement: Plan materialization preserves existing media graph semantics
The system SHALL materialize directory plan assignments using existing catalog item and inventory asset concepts for series, seasons, episodes, movies, versions, attachments, and extras.

#### Scenario: Episode plan is materialized
- **WHEN** a directory plan assigns files to series, season, and episode slots
- **THEN** the scanner SHALL create or reuse one series item, season items, episode items, media assets, and asset-item links consistent with existing TV hierarchy semantics

#### Scenario: Movie version plan is materialized
- **WHEN** a directory plan assigns multiple files as versions of one movie work
- **THEN** the scanner SHALL create or reuse one movie item and separate assets or version links for each planned main file

#### Scenario: Attachment assignment is materialized
- **WHEN** a directory plan assigns a trailer, sample, preview, featurette, or behind-the-scenes file to a parent work
- **THEN** the scanner SHALL link the file as an attachment asset role and SHALL NOT create it as an independent movie or episode

### Requirement: Incremental scans reuse directory plan rules
The system SHALL reuse persisted directory plan rules for unchanged directories and small deltas when the plan confidence remains valid.

#### Scenario: One file is added to an absolute episode pack
- **WHEN** an existing `absolute_episode_pack` directory gains a new file matching the persisted numbering rule
- **THEN** the scanner SHALL assign and materialize that new file without reprocessing all previously assigned files in the directory

#### Scenario: Directory changes invalidate the plan
- **WHEN** a directory delta introduces enough conflicting evidence to lower plan confidence below the reuse threshold
- **THEN** the scanner SHALL recompile the directory plan and SHALL mark affected assignments provisional or review-required when needed
