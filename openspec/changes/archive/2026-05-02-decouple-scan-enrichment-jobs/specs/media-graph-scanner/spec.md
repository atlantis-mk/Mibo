## ADDED Requirements

### Requirement: Scanner completes core synchronization before enrichment
The system SHALL complete library synchronization after storage refresh, catalog and inventory reconciliation, missing-file marking, availability updates, and projection refresh scheduling without requiring metadata matching or media probing to finish first.

#### Scenario: Manual scan encounters deleted files
- **WHEN** a manual `sync_library` job scans a library where previously indexed source files no longer appear in refreshed storage listings
- **THEN** the job MUST mark the missing inventory, asset, and catalog availability state before completing the scan job
- **AND** the job MUST NOT wait for metadata matching or media probing jobs before the synchronized state can be queried

#### Scenario: Scan creates new catalog and inventory records
- **WHEN** a scan discovers new supported video files and writes catalog and inventory rows
- **THEN** the scan job MUST be able to complete after those rows are reconciled and visible through catalog browse APIs
- **AND** metadata matching and media probing MUST be scheduled as follow-up enrichment work

### Requirement: Post-scan enrichment is scheduled as independent work
The system SHALL schedule catalog metadata matching and inventory media probing as independent post-scan enrichment jobs that can fail or retry separately from the completed scan.

#### Scenario: Metadata provider is unavailable after scan
- **WHEN** post-scan catalog metadata matching fails because a metadata provider is unavailable
- **THEN** the enrichment job MUST be marked failed or retryable independently
- **AND** the completed scan job MUST remain completed

#### Scenario: Media probing backlog exists
- **WHEN** a scan schedules media probing for many inventory files
- **THEN** the system MUST process probing as background enrichment without blocking future `sync_library` jobs from starting

### Requirement: Synchronization jobs have queue priority over enrichment jobs
The system SHALL prioritize synchronization jobs over metadata matching and media probing enrichment jobs when claiming available work from the job queue.

#### Scenario: A new scan is queued behind older probe work
- **WHEN** older `probe_inventory_file` or catalog matching work is queued and a new `sync_library` job is queued
- **THEN** the worker MUST claim the available `sync_library` job before lower-priority enrichment jobs

#### Scenario: No synchronization work is pending
- **WHEN** no available synchronization or projection work is queued
- **THEN** the worker MUST continue processing available enrichment jobs normally
