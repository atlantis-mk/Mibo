## ADDED Requirements

### Requirement: Directory shape profiles are persisted
The system SHALL persist content shape profiles for scanned video directories using cheap directory and filename evidence, including directory path, provider, library scope, classifier version, fingerprint, video counts, attachment counts, episode marker coverage, numeric sequence coverage, year density, title uniqueness, shared title evidence, season hints, sidecar hints, confidence, and review state.

#### Scenario: Large mixed-naming episode directory is profiled
- **WHEN** a scanned directory contains hundreds of files named with mixed patterns such as `01.mkv`, `第002集.mkv`, `S01E003.mkv`, and `04.2160p.HD国语中字[网站].mkv`
- **THEN** the system SHALL persist one directory shape profile that records aggregate episode sequence evidence without running full per-file movie-versus-episode classification for every file

#### Scenario: Movie collection is profiled
- **WHEN** a scanned directory contains many files with distinct title and year evidence and weak episode sequence evidence
- **THEN** the system SHALL persist a directory shape profile that exposes movie collection evidence instead of treating the directory size alone as series evidence

#### Scenario: Profile uses fast evidence only
- **WHEN** the system builds a directory shape profile
- **THEN** it MUST use already listed storage objects, paths, filenames, sidecar names, and object metadata only
- **AND** it MUST NOT run ffprobe, media-content reads, content hashing, artwork downloads, or external metadata provider lookups

### Requirement: Directory fingerprints control plan reuse
The system SHALL compute and persist a directory fingerprint that changes when relevant directory contents, scan policy inputs, exclusion inputs, or classifier version inputs change, and SHALL use that fingerprint to decide whether an existing shape profile and plan can be reused.

#### Scenario: Directory is unchanged on rescan
- **WHEN** a rescan observes the same directory fingerprint for a previously profiled directory
- **THEN** the system SHALL reuse the existing shape profile and compiled plan without rebuilding full directory shape evidence

#### Scenario: Classifier version changes
- **WHEN** the directory contents are unchanged but the content shape classifier version changes
- **THEN** the system SHALL treat the persisted profile and plan as stale and SHALL rebuild or revalidate them before materialization

#### Scenario: Exclusion policy changes
- **WHEN** scan exclusion rules or scan policy values that affect visible video files change for a directory
- **THEN** the directory fingerprint SHALL change so the directory shape profile and plan are recomputed with the new visible file set

### Requirement: Directory plans classify content shapes
The system SHALL compile directory shape profiles into directory plans that classify supported content shapes, including `episode_pack`, `absolute_episode_pack`, `season_folder`, `flat_episode_folder`, `series_folder`, `movie_folder`, `movie_versions_folder`, `movie_collection_folder`, `attachment_group`, and `unknown_review`.

#### Scenario: High-confidence absolute episode pack is planned
- **WHEN** a directory has high numeric sequence coverage, low independent movie evidence, and no season subdirectory context
- **THEN** the system SHALL compile an `absolute_episode_pack` plan with a series title, default or inferred season context, absolute episode numbers, confidence, evidence, and file assignment rules

#### Scenario: Season folder is planned
- **WHEN** a directory path or directory name indicates a season and the visible main videos map to episode slots
- **THEN** the system SHALL compile a `season_folder` plan with the inferred series title, season number, episode numbering mode, confidence, evidence, and assignment rules

#### Scenario: Movie version folder is planned
- **WHEN** a directory has multiple main videos sharing a normalized work stem and differing primarily by quality, edition, source, language, container, or release tokens
- **THEN** the system SHALL compile a `movie_versions_folder` plan that maps those files to one movie work with multiple asset or version assignments

#### Scenario: Movie collection folder is planned
- **WHEN** a directory has distinct title or year evidence across main videos and weak episode sequence evidence
- **THEN** the system SHALL compile a `movie_collection_folder` plan that preserves separate movie work assignments or marks the group review-required instead of forcing one series or one movie work

### Requirement: Plan assignments are reusable and reviewable
The system SHALL produce file assignments from directory plans and SHALL persist enough assignment metadata or compact plan rules with exceptions to reuse assignments across materialization batches and future rescans.

#### Scenario: File covered by high-confidence episode plan
- **WHEN** a supported video file is covered by a high-confidence `episode_pack`, `absolute_episode_pack`, `season_folder`, or `flat_episode_folder` plan
- **THEN** the system SHALL assign the file to the planned series, season, and episode slot without running full file-first movie-versus-episode classification for that file

#### Scenario: New episode is added to an existing plan
- **WHEN** a rescan finds one new file matching the persisted rule for an existing high-confidence episode plan
- **THEN** the system SHALL assign only the new file from the existing plan rule and SHALL NOT recompile the entire directory unless the delta invalidates the plan confidence

#### Scenario: Assignment is ambiguous
- **WHEN** a file or directory cannot be assigned with sufficient confidence from a directory plan
- **THEN** the system SHALL preserve the inventory facts and create a review-required assignment with evidence and alternatives instead of silently creating unrelated movie or episode catalog rows

### Requirement: Shape index lifecycle follows library and source lifecycle
The system SHALL manage shape profile, plan, and assignment records as source-scoped scanner state tied to libraries, library paths, media sources, and classifier versions.

#### Scenario: Library path is deleted
- **WHEN** a library path or media source is deleted
- **THEN** associated directory shape profiles, plans, and assignments SHALL no longer participate in future scans or materialization for that deleted scope

#### Scenario: Shape index is disabled
- **WHEN** plan-based materialization is disabled by rollout configuration
- **THEN** the system SHALL ignore persisted shape profiles and plans and SHALL fall back to inventory-backed file-first classification without deleting persisted inventory facts
