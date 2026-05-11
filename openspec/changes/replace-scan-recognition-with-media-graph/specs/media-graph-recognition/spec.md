## ADDED Requirements

### Requirement: Scanner builds media graph groups before recognition decisions
The system SHALL build media graph groups from inventory files, directory structure, filename signals, sidecar evidence, and local rules before making movie, series, season, episode, version, collection, or supplemental decisions.

#### Scenario: Season folder contains leading numeric files
- **WHEN** the scanner processes `Show/Season 1/01.mkv` and `Show/Season 1/02.mkv`
- **THEN** the system MUST create a series group, a season group, and episode slots for episodes 1 and 2 before any catalog metadata is materialized
- **AND** the system MUST NOT create movie work candidates from the bare numeric filenames

#### Scenario: Single movie folder is scanned
- **WHEN** the scanner processes a folder with one non-extra main video and no stronger TV evidence
- **THEN** the system MUST create one movie package group with the video as its main playable resource candidate

#### Scenario: Independent movies share a folder
- **WHEN** the scanner processes a folder with multiple main videos that have distinct title or year evidence and no episode-sequence evidence
- **THEN** the system MUST create separate movie package groups instead of forcing one series group or one multi-version movie group

### Requirement: Graph decisions classify groups with explicit evidence
The system SHALL classify media graph groups using explicit evidence, hard conflicts, confidence, alternatives, and reason text rather than using a single file-level type guess.

#### Scenario: Strong TV evidence wins over weak movie evidence
- **WHEN** a video filename is weakly movie-like but its directory and sibling evidence identify a season or episode run
- **THEN** the graph decision MUST classify the group as TV content and include the directory and sibling evidence in the decision

#### Scenario: Conflicting type evidence is detected
- **WHEN** one group has high-confidence movie evidence and high-confidence episode evidence that cannot both be true
- **THEN** the graph decision MUST mark the group as review-required instead of materializing both a movie and an episode for the same file

#### Scenario: Supplemental video is detected
- **WHEN** a file is classified as trailer, sample, extra, featurette, interview, deleted scene, or behind-the-scenes content
- **THEN** the graph decision MUST record a supplemental role and link the resource candidate to the likely parent group without counting it as a main movie or episode

### Requirement: Recognition materialization consumes only accepted graph decisions
The system SHALL materialize `MetadataItem`, `Resource`, `ResourceMetadataLink`, and projection inputs only from accepted or explicitly provisional graph decisions.

#### Scenario: Accepted TV decision materializes hierarchy
- **WHEN** a graph decision accepts a series with season and episode slots
- **THEN** the materializer MUST create or reuse one series item, one season item per season number, one episode item per episode slot, and resource links from playable files to the matching episode items

#### Scenario: Accepted movie decision materializes movie resources
- **WHEN** a graph decision accepts a movie package with multiple version resources
- **THEN** the materializer MUST create or reuse one movie item and link each version resource to that movie without creating duplicate movie metadata items

#### Scenario: Review-required decision is preserved
- **WHEN** a graph decision is ambiguous or blocked by a hard conflict
- **THEN** the materializer MUST preserve inventory-backed visibility and review state without creating incorrect movie, series, season, or episode metadata

### Requirement: Old scan recognition paths are removed or demoted to evidence providers
The system SHALL remove or rewrite old scan-time final decision paths so filename parsing, sidecar parsing, content-shape analysis, path-tree grouping, and sibling matching cannot directly create or link final catalog metadata.

#### Scenario: Filename parser runs
- **WHEN** filename parsing detects title, year, season, episode, quality, source, codec, language, or release-group tokens
- **THEN** the parser MUST emit graph evidence and MUST NOT directly create movie or episode metadata

#### Scenario: Content-shape or path-tree analysis runs
- **WHEN** directory analysis detects series, season, movie collection, movie version, or ambiguous structure
- **THEN** the analysis MUST emit graph groups or evidence and MUST NOT independently materialize catalog rows outside graph decisions

#### Scenario: Sibling matching finds related media
- **WHEN** sibling matching finds likely same-work, version, supplemental, or collection relationships
- **THEN** the result MUST become graph evidence or graph edges and MUST NOT bypass the graph resolver

### Requirement: Graph recognition is idempotent and reset-friendly
The system SHALL persist stable graph group keys, evidence versions, decision state, and materialization keys so scans can be rerun after parser changes, sidecar changes, provider matches, or manual corrections without duplicating metadata.

#### Scenario: Scanner reruns unchanged library
- **WHEN** a library is scanned again without storage changes
- **THEN** the system MUST reuse existing graph groups, decisions, metadata items, resources, and resource links instead of creating duplicate catalog rows

#### Scenario: Parser version changes
- **WHEN** filename or directory parsing logic changes
- **THEN** the system MUST recompute affected graph evidence and decisions from inventory facts while preserving stable resource identity where the underlying file identity did not change

#### Scenario: Development database is reset
- **WHEN** local development data is cleared and the library is rescanned
- **THEN** the replacement pipeline MUST recreate catalog visibility from inventory and graph decisions without requiring legacy recognition tables or legacy catalog read models
