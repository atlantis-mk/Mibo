## MODIFIED Requirements

### Requirement: Use metadata sidecar hints safely
The scanner SHALL parse supported `.nfo` and `.json` sidecars for high-confidence local metadata hints, including supported external identity fields, and SHALL provide them to the recognition resolver as evidence before metadata/resource graph materialization.

#### Scenario: JSON metadata improves movie classification
- **WHEN** `Movie A.mkv` has a matching JSON sidecar with title and year fields
- **THEN** the scanner SHALL add those fields as local resolver evidence for the movie work candidate

#### Scenario: NFO metadata improves episode classification
- **WHEN** an episode video has a matching NFO sidecar with series title, season number, and episode number
- **THEN** the scanner SHALL add those fields as local resolver evidence for the series, season, episode, and resource candidates

#### Scenario: Sidecar external identity seeds metadata enrichment
- **WHEN** a matching metadata sidecar contains a supported external identity such as a TMDB or MetaTube identifier
- **THEN** the resolver MUST persist that identity on the accepted metadata item with scanner provenance so later metadata enrichment can fetch detail without first performing a remote search

#### Scenario: Sidecar identity records local evidence source
- **WHEN** a metadata operation uses a sidecar-provided external identity to fetch provider detail
- **THEN** the operation evidence MUST identify the scanner metadata source as the seed and the provider detail source as the applied metadata source

#### Scenario: Curated metadata is preserved
- **WHEN** a metadata item is locked, manual, matched, or needs review
- **THEN** sidecar metadata hints SHALL NOT overwrite preserved descriptive fields for that item

### Requirement: Sidecars participate as resolver evidence
The scanner SHALL treat supported sidecar metadata as resolver evidence at group and file levels before metadata/resource graph materialization.

#### Scenario: TV folder has tvshow metadata
- **WHEN** a TV work directory contains `tvshow.nfo` or supported group metadata declaring a series title or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the series candidate while still preserving resolver identity for rescan reconciliation

#### Scenario: Movie folder has movie metadata
- **WHEN** a movie folder contains `movie.nfo` or supported group metadata declaring movie title, year, or provider identity
- **THEN** the scanner MUST use that sidecar as evidence for the movie candidate and metadata source without forcing one metadata item per video file

## ADDED Requirements

### Requirement: Sidecar evidence cannot bypass resolver conflicts
Sidecar metadata SHALL be high-priority local evidence, but it MUST NOT bypass resolver conflict gates or manual split rules.

#### Scenario: Sidecar external ID conflicts with existing hash-linked identity
- **WHEN** sidecar evidence points to one provider identity and file/hash or existing resource evidence points to a different metadata identity
- **THEN** the resolver MUST create a conflict decision and MUST NOT automatically merge or overwrite the existing identity

#### Scenario: Manual split rule conflicts with sidecar grouping
- **WHEN** a sidecar suggests grouping files that a scoped manual resolver rule has split
- **THEN** the resolver MUST honor the manual rule and preserve the sidecar as non-applied evidence
