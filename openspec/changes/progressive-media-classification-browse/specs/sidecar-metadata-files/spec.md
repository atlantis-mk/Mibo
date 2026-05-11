## MODIFIED Requirements

### Requirement: Sidecars participate as resolver evidence
The scanner SHALL treat supported sidecar metadata as resolver evidence at group and file levels before catalog projection and before final movie-or-series collapse decisions are made.

#### Scenario: TV folder has tvshow metadata
- **WHEN** a TV work directory contains `tvshow.nfo` or supported group metadata declaring a series title or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the series candidate while still preserving scanner identity for rescan reconciliation

#### Scenario: Movie folder has movie metadata
- **WHEN** a movie folder contains `movie.nfo` or supported group metadata declaring movie title, year, or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the movie candidate and metadata source without forcing one catalog item per video file

#### Scenario: Sidecar type evidence beats weak fallback
- **WHEN** sidecar metadata declares a series hierarchy or movie provider identity that conflicts with a weaker filename-only fallback
- **THEN** the scanner MUST use the stronger sidecar evidence to drive the work-group classification or mark the result review-required
- **AND** it MUST NOT ignore that sidecar and collapse the file into final movie metadata using weak fallback alone
