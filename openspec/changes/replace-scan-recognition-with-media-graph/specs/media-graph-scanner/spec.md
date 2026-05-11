## MODIFIED Requirements

### Requirement: Scanner separates inventory facts from media decisions
The system SHALL collect storage facts independently from catalog content-class and semantic media classification decisions, and SHALL treat media graph decisions as the only scanner-owned source for catalog materialization.

#### Scenario: Video file is scanned
- **WHEN** the scanner encounters a supported video file from a source-first path
- **THEN** the system MUST record or refresh inventory facts including storage path, provider, stable identity, size, modified time, hash evidence when available, and container without requiring final movie or episode classification first

#### Scenario: Resolver classification changes after rescan
- **WHEN** resolver logic changes how a scanned file should be classified
- **THEN** the system MUST be able to recompute catalog projection from inventory facts and graph resolver evidence without losing the physical file record

#### Scenario: Non-video file is discovered
- **WHEN** the scanner encounters supported audio, text, image, or other recognized file classes
- **THEN** the system MUST preserve inventory facts and content-class evidence even when no deep catalog projection exists yet

### Requirement: Scanner builds media graph candidates before catalog writes
The system SHALL group scanned files, current-directory siblings, sidecars, filename-derived signals, cached directory summaries, and learned classification rules into media graph candidates before writing catalog items, and SHALL treat directory shape as evidence rather than a final semantic type.

#### Scenario: Directory contains multiple episode-like videos
- **WHEN** a source directory contains multiple likely main videos that resolve to explicit or inferred episode slots
- **THEN** the scanner MUST create a single series candidate for that directory before projecting episode catalog descendants

#### Scenario: Movie folder contains multiple main-like files
- **WHEN** a source directory contains multiple plausible main video files for the same movie work
- **THEN** the scanner MUST create one movie candidate with multiple asset or version candidates instead of creating one movie per file

#### Scenario: Directory contains independent movies
- **WHEN** a source directory contains multiple likely main videos with distinct movie-like title or year evidence and no episode-sequence evidence
- **THEN** the scanner MUST preserve separate movie candidates instead of forcing the directory into one movie, one series, or one mixed semantic type

#### Scenario: Filename signals include release metadata
- **WHEN** scanned files include filename-derived release hints such as quality, source, codec, audio, subtitle, edition, or release group
- **THEN** the scanner MUST use those hints as candidate evidence for grouping and title cleanup without treating them as authoritative technical facts

#### Scenario: Catalog write is requested during scan
- **WHEN** scanner code is ready to create or link catalog metadata for discovered media
- **THEN** it MUST first persist or update the corresponding media graph group and graph decision
- **AND** final catalog writes MUST be performed by graph materialization, not by ad hoc scan helpers

## ADDED Requirements

### Requirement: Scanner cleans up replaced recognition state
The system SHALL remove or ignore replaced scan recognition state when the media graph scanner is enabled so old provisional decisions, orphan episodes, and movie fallback links cannot contaminate new projections.

#### Scenario: Existing library has old orphan episode metadata
- **WHEN** a development library is rebuilt under the media graph scanner
- **THEN** the rebuild MUST discard or supersede old orphan episode metadata, old movie fallback resource links, and old provisional recognition decisions for that library
- **AND** the resulting projections MUST come from current graph decisions only
