## MODIFIED Requirements

### Requirement: Sidecars participate as resolver evidence
The scanner SHALL treat supported sidecar metadata as resolver evidence at group and file levels before catalog projection and before final movie-or-series or same-metadata sibling collapse decisions are made.

#### Scenario: TV folder has tvshow metadata
- **WHEN** a TV work directory contains `tvshow.nfo` or supported group metadata declaring a series title or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the series candidate while still preserving scanner identity for rescan reconciliation

#### Scenario: Movie folder has movie metadata
- **WHEN** a movie folder contains `movie.nfo` or supported group metadata declaring movie title, year, or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the movie candidate and metadata source without forcing one catalog item per video file

#### Scenario: Sidecar identity reuses existing metadata match
- **WHEN** a sidecar declares a supported provider identity or explicit episode tuple that matches an existing metadata identity
- **THEN** the scanner MUST use that sidecar as strong same-metadata sibling evidence when deciding whether to reuse that metadata identity

#### Scenario: Sidecar conflict blocks weak sibling reuse
- **WHEN** sidecar identity evidence conflicts with a weaker filename-only or title-plus-year sibling candidate
- **THEN** the scanner MUST prefer the stronger sidecar evidence or mark the match review-required
