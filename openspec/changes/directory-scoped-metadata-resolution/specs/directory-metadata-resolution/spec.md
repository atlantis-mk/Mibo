## ADDED Requirements

### Requirement: Directory-scoped automatic metadata resolution units
The system SHALL derive automatic post-scan metadata resolution work from directory recognition/materialization units instead of only from flat metadata item ID batches.

#### Scenario: Materialized directory queues one resolution unit
- **WHEN** a scan materializes metadata and resources for a recognized directory scope
- **THEN** the follow-up metadata workflow is queued with the directory scope, directory shape, materialized metadata IDs, resource IDs, and review state needed to resolve that directory as one semantic unit

#### Scenario: Unsupported metadata items are not independently queued
- **WHEN** a directory materializes episode metadata items beneath a series or season scope
- **THEN** the automatic metadata workflow does not create independent metadata search tasks for each episode item

### Requirement: Movie folder metadata resolves once per work
The system SHALL resolve a single movie, multipart movie, or multi-version movie directory once for the represented movie work and SHALL bind all recognized playable resources to that work.

#### Scenario: Single movie folder
- **WHEN** a directory is classified as a single movie folder with one main playable video
- **THEN** automatic metadata resolution targets the movie work once and links the playable resource to that resolved or local-provisional metadata item

#### Scenario: Multipart movie folder
- **WHEN** a directory is classified as one multipart movie with multiple part files
- **THEN** automatic metadata resolution targets one movie work and keeps the parts attached to one multipart playable resource

#### Scenario: Movie versions folder
- **WHEN** a directory is classified as multiple versions or editions of the same movie
- **THEN** automatic metadata resolution targets one movie work and binds each version resource to that same metadata item with version or edition role evidence

### Requirement: Series directory metadata resolves at series scope
The system SHALL resolve series, season, episode-pack, absolute episode-pack, and flat episode folders at the series scope and SHALL derive season and episode bindings from local hierarchy evidence.

#### Scenario: Season folder
- **WHEN** a directory is classified as a season folder with recognized episode numbers
- **THEN** automatic metadata resolution targets the parent series once and creates or updates season and episode metadata from local numbering, sidecars, and available hierarchy detail without per-episode search

#### Scenario: Flat episode folder
- **WHEN** a directory is classified as a flat episode folder without an explicit season marker
- **THEN** automatic metadata resolution targets one series scope and assigns episode metadata using recognized numbering mode and directory evidence

#### Scenario: Series remote detail provides hierarchy
- **WHEN** a configured detail provider returns series hierarchy during directory metadata resolution
- **THEN** the system applies the hierarchy to matching season and episode metadata produced by the directory unit

### Requirement: Movie collection directories resolve per movie identity without blind matching
The system SHALL treat movie collection directories as one directory scope containing multiple movie identities and SHALL avoid speculative provider matching when no local identity evidence or operational search provider exists.

#### Scenario: Collection with local sidecar evidence
- **WHEN** a movie collection directory contains multiple movie identities and local sidecar or external ID evidence for an identity
- **THEN** the system applies that local evidence or compatible detail lookup to the corresponding movie metadata item

#### Scenario: Collection without search provider
- **WHEN** a movie collection directory contains multiple movie identities but the library has no operational search provider and no local evidence for those identities
- **THEN** the system keeps locally generated movie metadata as provisional and does not record repeated no-candidate metadata match operations

#### Scenario: Collection with search provider
- **WHEN** a movie collection directory contains multiple clear movie identities and an operational search provider is configured
- **THEN** the system may resolve each clear movie identity while preserving the parent directory scope in operation evidence

### Requirement: Review-required directories suppress automatic matching
The system SHALL suppress automatic provider search and item-level matching for attachment-only, extras-only, ambiguous, mixed-conflict, or review-required directory shapes.

#### Scenario: Extras-only directory
- **WHEN** a directory is classified as attachment-only or extras-only
- **THEN** automatic metadata resolution does not search providers and only binds attachments to an existing parent if one is known

#### Scenario: Ambiguous directory
- **WHEN** directory recognition produces an ambiguous, mixed movie/episode conflict, or unknown review-required shape
- **THEN** the system records the review-required state and does not run automatic metadata search for that directory

### Requirement: Directory resolution preserves governance and projection behavior
The system SHALL record metadata operation evidence and refresh catalog projection after directory-scoped resolution completes, including no-op review-required outcomes.

#### Scenario: Successful directory resolution refreshes projection
- **WHEN** directory-scoped metadata resolution creates, updates, or links metadata items and resources
- **THEN** the catalog projection for the affected library scope is refreshed after the resolution task completes

#### Scenario: No-op directory resolution records outcome
- **WHEN** directory-scoped metadata resolution skips provider search because no search provider exists or the directory requires review
- **THEN** the system records an operation outcome that explains the skip reason and still allows later manual apply or refetch on individual metadata items
