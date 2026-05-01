## MODIFIED Requirements

### Requirement: MetaTube search produces Mibo search candidates
The system SHALL execute movie search through MetaTube HTTP endpoints as part of the unified metadata operation pipeline and normalize returned MetaTube movie search results into Mibo catalog search candidates without pretending they are TMDB results.

#### Scenario: Automated search uses configured MetaTube instance
- **WHEN** a library strategy selects a MetaTube provider instance for the search stage and metadata matching runs for a movie catalog item
- **THEN** the system MUST call the configured MetaTube server search endpoint, convert successful results into provider `metatube` candidates, and record the MetaTube provider attempt in the operation result

#### Scenario: Manual search uses MetaTube provider filter
- **WHEN** a manual search runs through a MetaTube provider instance with a default upstream provider filter
- **THEN** the system MUST include that provider filter in the MetaTube search request unless the operation supplies a more specific supported filter

#### Scenario: MetaTube-only automated matching does not require TMDB
- **WHEN** a movie library strategy has an operational MetaTube search and detail provider but no TMDB API key is configured
- **THEN** automated metadata matching MUST execute through MetaTube rather than being skipped by a TMDB-specific configured check

### Requirement: MetaTube detail refresh applies normalized metadata
The system SHALL fetch MetaTube movie detail over HTTP, normalize the response, and apply metadata to catalog fields, image candidates, people, external identities, and source evidence according to shared metadata operation governance rules.

#### Scenario: Detail application records canonical fields
- **WHEN** a MetaTube detail response contains title, summary, release date, runtime, genres, director, actors, and image URLs
- **THEN** the system MUST normalize and apply supported values to unlocked catalog fields and related catalog records using the same governance protections as other metadata providers

#### Scenario: Detail application records MetaTube identity
- **WHEN** a MetaTube detail response is applied to a catalog item
- **THEN** the system MUST record an external identity with provider `metatube` and an identity key that preserves both the upstream MetaTube provider and upstream item ID

#### Scenario: Locked field is skipped during MetaTube apply
- **WHEN** MetaTube detail contains a value for a locked catalog field
- **THEN** the metadata operation MUST record the MetaTube source evidence but MUST NOT overwrite the locked field

### Requirement: MetaTube refetch reuses MetaTube identity
The system SHALL refetch catalog metadata from MetaTube through the unified metadata operation pipeline when the selected provider identity and library strategy identify a usable MetaTube provider instance.

#### Scenario: Refetch item previously matched through MetaTube
- **WHEN** a catalog item has a stored MetaTube identity and its library strategy still includes an operational MetaTube detail provider
- **THEN** refetch MUST call MetaTube detail for the stored upstream provider and ID, update metadata evidence through a metadata operation result, and avoid requiring a new search

#### Scenario: Refetch cannot find MetaTube identity
- **WHEN** a catalog item has no stored MetaTube identity and the selected detail provider is MetaTube
- **THEN** refetch MUST fail with a clear metadata operation result instead of using a TMDB identity or guessing an upstream ID
