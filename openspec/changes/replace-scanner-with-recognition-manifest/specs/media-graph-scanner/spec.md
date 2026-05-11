## MODIFIED Requirements

### Requirement: Scanner builds media graph candidates before catalog writes
The system SHALL group scanned files, current-directory siblings, sidecars, filename-derived signals, cached directory summaries, path context, hash evidence, and learned resolver rules into recognition manifest candidates before writing metadata/resource graph links, and SHALL treat directory shape as evidence rather than a final semantic type.

#### Scenario: Directory contains multiple episode-like videos
- **WHEN** a source directory contains multiple likely main videos that resolve to explicit or inferred episode slots
- **THEN** the scanner MUST create series, season, episode, and playable resource candidates in the recognition manifest before projecting episode metadata descendants

#### Scenario: Movie folder contains multiple main-like files
- **WHEN** a source directory contains multiple plausible main video files for the same movie work
- **THEN** the scanner MUST create one movie work candidate with multiple playable resource, variant, or edition candidates instead of creating one movie metadata item per file

#### Scenario: Directory contains independent movies
- **WHEN** a source directory contains multiple likely main videos with distinct movie-like title or year evidence and no episode-sequence evidence
- **THEN** the scanner MUST preserve separate movie work candidates instead of forcing the directory into one movie, one series, or one mixed semantic type

#### Scenario: Filename signals include release metadata
- **WHEN** scanned files include filename-derived release hints such as quality, source, codec, audio, subtitle, edition, or release group
- **THEN** the scanner MUST use those hints as candidate evidence for grouping and title cleanup without treating them as authoritative technical facts

### Requirement: Resolver decisions expose evidence and confidence
The system SHALL represent scanner grouping, classification, identity resolution, and materialization readiness as resolver decisions with candidate type, role, confidence, alternatives, filename-derived signal evidence, directory evidence, sidecar evidence, hash evidence, review state, conflict state, and reason text.

#### Scenario: Series candidate is inferred from a flat episode folder
- **WHEN** the resolver groups a flat source-first folder into a series candidate
- **THEN** the decision MUST include the target series identity, inferred season and episode slots when available, confidence, evidence references, alternatives considered, and a reason explaining the grouping

#### Scenario: Classification is ambiguous
- **WHEN** the scanner cannot confidently distinguish movie, episode, version, independent work, edition, or attachment semantics
- **THEN** the resolver decision MUST preserve candidate evidence and mark the projected graph relationship or candidate group for governance review instead of silently creating unrelated works

#### Scenario: Attachment is detected
- **WHEN** a video file is classified as trailer, extra, sample, preview, or another non-main role
- **THEN** the resolver decision MUST expose the supplemental role and evidence so materialization can link it to an accepted parent work without treating it as a standalone movie or episode

#### Scenario: Audio token prevents episode false positive
- **WHEN** the resolver rejects weak episode inference because a numeric-looking token is classified as filename-derived audio evidence
- **THEN** the decision MUST expose that anti-misclassification evidence in its reason or evidence summary

### Requirement: Scanner completes core synchronization before enrichment
The system SHALL complete library synchronization after storage refresh, inventory reconciliation, recognition manifest candidate persistence for newly discovered supported videos, missing-file marking or scheduling, availability updates, and resolver/projection scheduling without requiring final metadata enrichment, media probing, artwork processing, or remote metadata matching to finish first.

#### Scenario: Manual scan encounters deleted files
- **WHEN** a manual `sync_library` job scans a library where previously indexed source files no longer appear in refreshed storage listings
- **THEN** the job MUST mark or schedule reconciliation of the missing inventory, resource, metadata-link, and projection availability state before completing the scan job
- **AND** the job MUST NOT wait for metadata matching or media probing jobs before the synchronized state can be queried

#### Scenario: Scan creates new inventory-backed skeleton records
- **WHEN** a scan discovers new supported video files that pass scan policy and exclusion filters
- **THEN** the scan job MUST be able to complete after file inventory facts and recognition manifest candidates are reconciled or published
- **AND** final resolver materialization, metadata matching, media probing, sidecar parsing, and artwork enrichment MUST be scheduled or processed as follow-up work

#### Scenario: Scan creates final graph records on the fast path
- **WHEN** a scan has enough local evidence for resolver acceptance without delaying skeleton ingest
- **THEN** the system MAY publish final metadata/resource graph entries immediately through the resolver materializer
- **AND** metadata matching and media probing MUST still be scheduled as follow-up enrichment work

## REMOVED Requirements

### Requirement: Catalog items use durable scanner identities
**Reason**: Scanner identities are replaced by resolver candidate keys and global metadata/resource graph materialization. The scanner no longer owns catalog item identity directly.
**Migration**: Use `recognition-manifest-resolver` requirements for stable candidate keys, idempotent materialization, and resolver-owned metadata identity reuse.

### Requirement: TV scanner projects a stable series-season-episode hierarchy
**Reason**: TV hierarchy projection remains required, but it must be produced by resolver materialization from recognition manifest candidates rather than scanner-specific catalog item projection.
**Migration**: Represent series, season, episode, and playable resources as manifest candidates and materialize accepted decisions into `MetadataItem` hierarchy and `ResourceMetadataLink` records.

### Requirement: Movie scanner projects one work with multiple assets
**Reason**: Movie projection remains required, but the scanner-specific asset projection language is replaced by resolver-owned metadata/resource materialization with explicit variants, editions, and supplemental roles.
**Migration**: Use accepted movie work candidates plus playable resource candidates to create one movie metadata identity with multiple resource links.
