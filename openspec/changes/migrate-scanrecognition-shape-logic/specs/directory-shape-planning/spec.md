## ADDED Requirements

### Requirement: Content shape is the authoritative directory planner
The system SHALL use `content_shape` profiles, plans, and assignments as the sole scan-time authority for directory shape decisions that drive recognition units and materialization.

#### Scenario: Directory materialization consumes content shape
- **WHEN** a scan materializes discovered media files
- **THEN** the materialization inputs MUST be derived from `content_shape` plans and assignments, not from the legacy tree classifier

#### Scenario: Stale plans are invalidated
- **WHEN** classifier version, inventory facts, file signals, scan policy, exclusion rules, or relevant sidecar evidence change
- **THEN** the system MUST regenerate affected `content_shape` plans before materialization

### Requirement: Large multi-work directories become movie collections
The system SHALL classify large directories with many primary videos, high title uniqueness, and low episode continuity as movie collection folders when no stronger episodic or conflict evidence exists.

#### Scenario: Large unique-title folder
- **WHEN** a directory contains many primary videos with mostly unique titles and sparse sequence continuity
- **THEN** the `content_shape` plan MUST be `movie_collection_folder` with a per-file work key strategy

#### Scenario: Episodic evidence wins over collection evidence
- **WHEN** a large directory has strong season or episode evidence
- **THEN** the system MUST classify it as an episodic shape or require review instead of forcing `movie_collection_folder`

### Requirement: Catalog identifier collections are recognized
The system SHALL extract catalog-style identifiers from filenames and use dense, unique catalog evidence to classify multi-work movie collection folders.

#### Scenario: JAV-style catalog identifiers
- **WHEN** a directory contains multiple primary videos with distinct identifiers such as `ABP-123`, `HEYZO-2087`, or provider-number variants
- **THEN** the `content_shape` plan MUST be `movie_collection_folder` with a per-catalog-id work key strategy

#### Scenario: Low catalog uniqueness does not force catalog grouping
- **WHEN** catalog identifiers are sparse or heavily duplicated across a directory
- **THEN** the system MUST use other shape evidence or require review instead of grouping primarily by catalog identifier

### Requirement: Movie version folders are recognized
The system SHALL classify folders as movie version folders when multiple primary videos share the same normalized movie identity and differ only by quality, edition, release, codec, or version noise.

#### Scenario: Same movie with quality variants
- **WHEN** a directory contains multiple primary videos for the same movie identity with variants such as `1080p`, `2160p`, `Directors Cut`, or `REMUX`
- **THEN** the `content_shape` plan MUST be `movie_versions_folder`

#### Scenario: Same title with distinct work evidence
- **WHEN** a directory contains the same base title but evidence indicates distinct works rather than versions
- **THEN** the system MUST not collapse the files into a single movie versions folder without sufficient same-work evidence

### Requirement: Multipart movie folders are recognized
The system SHALL classify folders as multipart movie folders when primary videos form a continuous part, CD, disc, or disk sequence for the same normalized movie identity.

#### Scenario: Continuous multipart sequence
- **WHEN** a directory contains primary videos named as continuous parts such as `Part1` and `Part2` or `CD1` and `CD2` for the same movie
- **THEN** the `content_shape` plan MUST be `multipart_movie_folder` and assignments MUST preserve part ordering for materialization

#### Scenario: Broken multipart sequence requires review
- **WHEN** multipart evidence has missing, duplicated, or conflicting part numbers
- **THEN** the system MUST require review or use another safer shape instead of materializing an invalid multipart movie

### Requirement: Token consensus informs directory shape
The system SHALL use filename token consensus across sibling primary videos to distinguish episode groups, movie versions, and multipart movies when direct filename parsing alone is insufficient.

#### Scenario: Version residual tokens
- **WHEN** sibling videos share a common movie identity and their residual tokens are only version or release tokens
- **THEN** the system MUST classify the directory as a movie versions folder

#### Scenario: Episode residual tokens
- **WHEN** sibling videos share common title tokens and each residual token contains a consistent season-episode marker
- **THEN** the system MUST classify the directory as a season or episode group according to folder season evidence

#### Scenario: Multipart residual tokens
- **WHEN** sibling videos share common movie tokens and residual tokens form a continuous part sequence
- **THEN** the system MUST classify the directory as a multipart movie folder

### Requirement: Sidecar conflicts require review
The system SHALL use NFO and sidecar shape hints as evidence, and contradictory high-confidence hints MUST produce a review-required plan.

#### Scenario: Movie sidecar conflicts with episode evidence
- **WHEN** a directory has movie NFO evidence but filenames or folder structure strongly indicate a season or episode group
- **THEN** the `content_shape` plan MUST be review-required with conflict evidence

#### Scenario: Episode sidecar conflicts with movie evidence
- **WHEN** a directory has episode NFO evidence but filenames strongly indicate a single movie or movie versions folder
- **THEN** the `content_shape` plan MUST be review-required with conflict evidence

#### Scenario: Sidecar reinforces weak evidence
- **WHEN** filename evidence is weak but sidecar evidence consistently identifies a movie, season, or episode group
- **THEN** the system MUST use the sidecar evidence to select that shape when no conflict exists

### Requirement: Primary videos drive shape decisions
The system SHALL prefer primary videos over trailers, samples, previews, featurettes, extras, and behind-the-scenes files when computing directory shape.

#### Scenario: Extras do not dominate folder shape
- **WHEN** a directory contains one or more primary videos plus supplemental videos
- **THEN** the system MUST compute the main directory shape from primary videos and assign supplemental videos as attachments where possible

#### Scenario: Extras-only folder
- **WHEN** a directory contains only supplemental videos or has explicit extras/trailers/sample path hints
- **THEN** the system MUST classify it as an attachment group or review-required shape rather than a normal movie collection

### Requirement: Season-only child directories form a series parent
The system SHALL infer a series parent when a directory contains only child directories that are confidently planned as season folders.

#### Scenario: Parent with season children
- **WHEN** every meaningful child directory under a parent has a season-folder plan
- **THEN** the parent directory MUST be planned as a series folder

#### Scenario: Mixed child shapes block series inference
- **WHEN** a parent has a mix of season folders and movie or unknown child shapes
- **THEN** the parent directory MUST not be automatically planned as a series folder

### Requirement: Legacy tree classifier is removed
The system SHALL remove the legacy tree-classification materialization path after equivalent `content_shape` behavior is implemented and tested.

#### Scenario: No legacy classifier authority remains
- **WHEN** backend tests compile and scan materialization runs
- **THEN** no production path MUST call the legacy tree classifier to decide directory shape or create recognition candidates

#### Scenario: Reusable parsing remains available
- **WHEN** low-level filename or folder parsing helpers are still needed
- **THEN** those helpers MUST live outside the removed tree-classification path or be owned directly by the active library pipeline
