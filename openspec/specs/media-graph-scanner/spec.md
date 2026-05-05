# media-graph-scanner Specification

## Purpose
Define source-first inventory collection, resolver evidence, durable identity reconciliation, and catalog projection for media graph scanning.

## Requirements
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

### Requirement: Resolver decisions expose evidence and confidence
The system SHALL represent scanner grouping and classification as resolver decisions with candidate type, role, confidence, alternatives, filename-derived signal evidence, directory summary evidence, review state, and reason text.

#### Scenario: Series candidate is inferred from a flat episode folder
- **WHEN** a resolver groups a flat source-first folder into a series candidate
- **THEN** the decision MUST include the target series identity, inferred season and episode slots when available, confidence, evidence references, alternatives considered, and a reason explaining the grouping

#### Scenario: Classification is ambiguous
- **WHEN** the scanner cannot confidently distinguish movie, episode, version, independent work, or attachment semantics
- **THEN** the resolver decision MUST preserve candidate evidence and mark the projected catalog item or relationship for governance review instead of silently creating unrelated works

#### Scenario: Attachment is detected
- **WHEN** a video file is classified as trailer, extra, sample, preview, or another non-main role
- **THEN** the resolver decision MUST expose the attachment role and evidence so catalog projection can link it to a likely parent work without treating it as a standalone movie or episode

#### Scenario: Audio token prevents episode false positive
- **WHEN** a resolver rejects weak episode inference because a numeric-looking token is classified as filename-derived audio evidence
- **THEN** the decision MUST expose that anti-misclassification evidence in its reason or evidence summary

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

### Requirement: Missing media is hard deleted after retention
The system SHALL hard delete media records that remain missing beyond the configured retention period when the missing cleanup job runs.

#### Scenario: Missing file exceeds retention
- **WHEN** an inventory file, its media asset, and its catalog item have remained missing beyond the configured retention window
- **THEN** the cleanup job MUST physically delete the eligible inventory, asset, catalog, and dependent rows from the database
- **AND** the deleted records MUST NOT remain as soft-deleted rows with `deleted_at` set

#### Scenario: Missing file has not exceeded retention
- **WHEN** a media graph is marked missing but its missing age is less than the configured retention window
- **THEN** the cleanup job MUST leave the missing rows in place

### Requirement: Hard cleanup deletes user and governance state
The system SHALL delete user-specific, metadata, and governance records linked to media that is hard deleted by missing cleanup.

#### Scenario: Missing media has user activity
- **WHEN** a hard-deleted catalog item has favorites, playback progress, or user item data
- **THEN** the cleanup job MUST delete those user records together with the catalog item

#### Scenario: Missing media has manual curation
- **WHEN** a hard-deleted catalog item has manual metadata matches, field states, external IDs, selected images, tags, people links, or governance decisions
- **THEN** the cleanup job MUST delete those curation records together with the catalog item
- **AND** those records MUST NOT prevent hard deletion

### Requirement: Hard cleanup preserves available graph branches
The system SHALL avoid deleting catalog parents or shared assets that still have available local media descendants or links.

#### Scenario: Series has one missing and one available episode
- **WHEN** cleanup processes a missing episode in a series that still has another available episode
- **THEN** the cleanup job MUST hard delete the missing episode graph
- **AND** it MUST keep the series, season, and available episode graph

#### Scenario: Movie version is missing but another version remains
- **WHEN** cleanup processes a missing asset for a movie that still has another available source asset
- **THEN** the cleanup job MUST delete only the missing asset and file graph
- **AND** it MUST keep the movie catalog item available

### Requirement: Hard cleanup refreshes projections
The system SHALL refresh catalog projections after hard deleting missing media records.

#### Scenario: Cleanup deletes catalog records
- **WHEN** the cleanup job hard deletes one or more catalog records for a library
- **THEN** the system MUST refresh or queue refresh of catalog rollups and search documents for the affected library scope
- **AND** subsequent browse and search queries MUST NOT return deleted records

### Requirement: Scanner completes core synchronization before enrichment
The system SHALL complete library synchronization after storage refresh, inventory reconciliation, skeleton visibility publication for newly discovered supported videos, missing-file marking or scheduling, availability updates, and projection refresh scheduling without requiring final catalog classification, metadata matching, media probing, artwork processing, or sidecar metadata parsing to finish first.

#### Scenario: Manual scan encounters deleted files
- **WHEN** a manual `sync_library` job scans a library where previously indexed source files no longer appear in refreshed storage listings
- **THEN** the job MUST mark or schedule reconciliation of the missing inventory, asset, and catalog availability state before completing the scan job
- **AND** the job MUST NOT wait for metadata matching or media probing jobs before the synchronized state can be queried

#### Scenario: Scan creates new inventory-backed skeleton records
- **WHEN** a scan discovers new supported video files that pass scan policy and exclusion filters
- **THEN** the scan job MUST be able to complete after file inventory facts and visible skeleton entries are reconciled or published
- **AND** final catalog projection, metadata matching, media probing, sidecar parsing, and artwork enrichment MUST be scheduled or processed as follow-up work

#### Scenario: Scan creates final catalog records on the fast path
- **WHEN** a scan can confidently create or reuse catalog and inventory rows without delaying skeleton ingest
- **THEN** the system MAY publish the final catalog-backed entry immediately
- **AND** metadata matching and media probing MUST still be scheduled as follow-up enrichment work

### Requirement: Post-scan enrichment is scheduled as independent work
The system SHALL schedule catalog classification refinement, catalog metadata matching, inventory media probing, sidecar metadata parsing, artwork processing, and projection refresh as independent post-scan work that can fail or retry separately from the completed scan.

#### Scenario: Metadata provider is unavailable after scan
- **WHEN** post-scan catalog metadata matching fails because a metadata provider is unavailable
- **THEN** the enrichment job MUST be marked failed or retryable independently
- **AND** the completed scan job MUST remain completed

#### Scenario: Media probing backlog exists
- **WHEN** a scan schedules media probing for many inventory files
- **THEN** the system MUST process probing as background enrichment without blocking future `sync_library` jobs from starting

#### Scenario: Classification refinement fails after skeleton ingest
- **WHEN** asynchronous classification or catalog materialization fails for an inventory-backed discovered entry
- **THEN** the discovered entry MUST remain visible with a failure or review-required maturity state
- **AND** the failed refinement MUST NOT invalidate the persisted inventory facts

### Requirement: Synchronization jobs have queue priority over enrichment jobs
The system SHALL prioritize synchronization jobs over metadata matching and media probing enrichment jobs when claiming available work from the job queue.

#### Scenario: A new scan is queued behind older probe work
- **WHEN** older `probe_inventory_file` or catalog matching work is queued and a new `sync_library` job is queued
- **THEN** the worker MUST claim the available `sync_library` job before lower-priority enrichment jobs

#### Scenario: No synchronization work is pending
- **WHEN** no available synchronization or projection work is queued
- **THEN** the worker MUST continue processing available enrichment jobs normally
