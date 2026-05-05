## MODIFIED Requirements

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
