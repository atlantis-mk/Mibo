## ADDED Requirements

### Requirement: Sidecars participate as resolver evidence
The scanner SHALL treat supported sidecar metadata as resolver evidence at group and file levels before catalog projection.

#### Scenario: TV folder has tvshow metadata
- **WHEN** a TV work directory contains `tvshow.nfo` or supported group metadata declaring a series title or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the series candidate while still preserving scanner identity for rescan reconciliation

#### Scenario: Movie folder has movie metadata
- **WHEN** a movie folder contains `movie.nfo` or supported group metadata declaring movie title, year, or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the movie candidate and metadata source without forcing one catalog item per video file

### Requirement: Sidecar hints respect field ownership
The scanner SHALL apply sidecar hints through catalog field ownership and governance rules.

#### Scenario: Manual field is locked
- **WHEN** a sidecar contains a title, year, overview, or provider ID for an item whose corresponding field has been locked or manually curated
- **THEN** the scanner MUST record sidecar evidence without overwriting the protected field
