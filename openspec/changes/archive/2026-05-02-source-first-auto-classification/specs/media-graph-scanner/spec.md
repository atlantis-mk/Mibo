## MODIFIED Requirements

### Requirement: Scanner separates inventory facts from media decisions
The system SHALL collect storage facts independently from catalog content-class and semantic media classification decisions.

#### Scenario: Video file is scanned
- **WHEN** the scanner encounters a supported video file from a source-first path
- **THEN** the system MUST record or refresh inventory facts including storage path, provider, stable identity, size, modified time, hash evidence when available, and container without requiring final movie or episode classification first

#### Scenario: Resolver classification changes after rescan
- **WHEN** resolver logic changes how a scanned file should be classified
- **THEN** the system MUST be able to recompute catalog projection from inventory facts and resolver evidence without losing the physical file record

#### Scenario: Non-video file is discovered
- **WHEN** the scanner encounters supported audio, text, image, or other recognized file classes
- **THEN** the system MUST preserve inventory facts and content-class evidence even when no deep catalog projection exists yet

### Requirement: Scanner builds media graph candidates before catalog writes
The system SHALL group scanned directories, files, and sidecars into media graph candidates before writing catalog items and SHALL NOT require a user-selected movie or TV library type to build those candidates.

#### Scenario: Directory contains multiple episode-like videos
- **WHEN** a source-first directory contains multiple files that resolve to episode slots
- **THEN** the scanner MUST create a single series candidate for that directory before projecting episode catalog descendants

#### Scenario: Movie folder contains multiple main-like files
- **WHEN** a source-first directory contains multiple plausible main video files for the same work
- **THEN** the scanner MUST create one movie candidate with multiple asset candidates instead of creating one movie per file

### Requirement: Resolver decisions expose evidence and confidence
The system SHALL represent scanner grouping and classification as resolver decisions with evidence, confidence, and reason text.

#### Scenario: Series candidate is inferred from a flat episode folder
- **WHEN** a resolver groups a flat source-first folder into a series candidate
- **THEN** the decision MUST include the target series identity, confidence, evidence references, and a reason explaining the grouping

#### Scenario: Classification is ambiguous
- **WHEN** the scanner cannot confidently distinguish movie, TV, version, or extra semantics
- **THEN** the resolver decision MUST preserve the candidate evidence and mark the projected catalog item or relationship for governance review instead of silently creating unrelated works

### Requirement: Catalog items use durable scanner identities
The system SHALL reconcile scanner-created catalog items using durable identities that are independent of display title and independent of user-selected library type.

#### Scenario: TV file names contain different title signals
- **WHEN** files in the same TV series directory include different title prefixes or languages
- **THEN** the scanner MUST reconcile them to the same series identity derived from the directory/work group rather than creating separate series by title

#### Scenario: Display title changes after metadata match
- **WHEN** provider metadata or manual correction changes a movie or series title
- **THEN** subsequent scans MUST keep matching the same catalog item through scanner or provider identity rather than creating a duplicate under the new title

### Requirement: TV scanner projects a stable series-season-episode hierarchy
The system SHALL project source-first TV media graph decisions into durable catalog series, season, and episode hierarchy rows.

#### Scenario: Standard TV folder is scanned
- **WHEN** the scanner processes `Series/Season 1/Series - S01E01.mkv` from a source-first path
- **THEN** the system MUST create or reuse one series item, one season item for season 1, one episode item for episode 1, and a playable asset linked to that episode

#### Scenario: Flat TV season folder is scanned
- **WHEN** the scanner processes a source-first directory containing `S02E01`, `S02E02`, and `第03集` style files for the same work
- **THEN** the system MUST create or reuse one series item, infer the season context when available, and create episode descendants under that same series

### Requirement: Movie scanner projects one work with multiple assets
The system SHALL project source-first movie graph decisions into one movie catalog item with separate assets for main files, versions, and extras.

#### Scenario: Movie has two quality versions
- **WHEN** a source-first movie folder contains `Movie.1080p.mkv` and `Movie.2160p.mkv`
- **THEN** the scanner MUST create or reuse one movie item and expose both files as separate media assets or media sources for that movie

#### Scenario: Movie folder has trailer and featurette files
- **WHEN** a source-first movie folder contains a main movie file plus trailer or behind-the-scenes videos
- **THEN** the scanner MUST keep those extra files associated with the movie without treating them as separate movie works

### Requirement: Multi-episode assets link to all episode slots
The system SHALL represent one file containing multiple episodes as one asset linked to each episode slot.

#### Scenario: Multi-episode file is scanned
- **WHEN** the scanner processes `Show.S01E01-E02.mkv` from a source-first path
- **THEN** the system MUST create or reuse episode 1 and episode 2 and link the same media asset to both with segment ordering information

### Requirement: Future media types plug into the graph model
The system SHALL keep media graph, source probing, content-class evidence, resolver decisions, and catalog projection concepts generic enough to support future audio, document, and photo media types.

#### Scenario: New media type is introduced later
- **WHEN** a future scanner adds music or document resolvers
- **THEN** it MUST be able to reuse inventory facts, identity reconciliation, resolver decisions, and catalog projection patterns without changing video-specific resolver behavior
