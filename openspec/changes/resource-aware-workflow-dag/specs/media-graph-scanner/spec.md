## ADDED Requirements

### Requirement: Scanner Work Is Workflow Backed
The system SHALL execute library scanner synchronization through workflow tasks rather than requiring one global job to complete all scan and post-scan work serially.

#### Scenario: Scan starts while another library is scanning
- **WHEN** a scan workflow is running for library A and a scan is requested for library B
- **THEN** the scanner MUST allow library B scan tasks to run concurrently with library A when dependencies, library safety, and resource budgets permit

### Requirement: Scanner Uses Bounded Batches
The system SHALL split large scan, materialization, projection, probing, and metadata follow-up work into bounded workflow tasks that can be fairly scheduled with work from other libraries.

#### Scenario: Large library creates many batches
- **WHEN** a scan discovers more files than a configured batch size
- **THEN** the scanner MUST create multiple bounded workflow tasks instead of one unbounded task
- **AND** tasks from other workflow runs MUST remain eligible for scheduling between those batches

## MODIFIED Requirements

### Requirement: Scanner completes core synchronization before enrichment
The system SHALL complete library synchronization after storage refresh, inventory reconciliation, skeleton visibility publication for newly discovered supported videos, missing-file marking or scheduling, availability updates, and projection refresh scheduling without requiring final catalog classification, metadata matching, media probing, artwork processing, or sidecar metadata parsing to finish first. The synchronization lifecycle MUST be represented by workflow run and task status, and completion of core synchronization MUST NOT require unrelated libraries' workflow tasks to finish.

#### Scenario: Manual scan encounters deleted files
- **WHEN** a manual scan workflow scans a library where previously indexed source files no longer appear in refreshed storage listings
- **THEN** the workflow MUST mark or schedule reconciliation of the missing inventory, asset, and catalog availability state before core synchronization is complete
- **AND** the workflow MUST NOT wait for metadata matching or media probing tasks before the synchronized state can be queried

#### Scenario: Scan creates new inventory-backed skeleton records
- **WHEN** a scan workflow discovers new supported video files that pass scan policy and exclusion filters
- **THEN** the workflow MUST be able to complete core synchronization after file inventory facts and visible skeleton entries are reconciled or published
- **AND** final catalog projection, metadata matching, media probing, sidecar parsing, and artwork enrichment MUST be scheduled or processed as dependent workflow tasks

#### Scenario: Scan creates final catalog records on the fast path
- **WHEN** a scan workflow can confidently create or reuse catalog and inventory rows without delaying skeleton ingest
- **THEN** the system MAY publish the final catalog-backed entry immediately
- **AND** metadata matching and media probing MUST still be scheduled as dependent workflow tasks

### Requirement: Post-scan enrichment is scheduled as independent work
The system SHALL schedule catalog classification refinement, catalog metadata matching, inventory media probing, sidecar metadata parsing, artwork processing, and projection refresh as independent workflow work that can fail or retry separately from completed core scan synchronization.

#### Scenario: Metadata provider is unavailable after scan
- **WHEN** post-scan catalog metadata matching fails because a metadata provider is unavailable
- **THEN** the enrichment task MUST be marked failed or retryable independently
- **AND** the completed core scan synchronization MUST remain completed

#### Scenario: Media probing backlog exists
- **WHEN** a scan workflow schedules media probing for many inventory files
- **THEN** the system MUST process probing as background enrichment without blocking future scan workflows for other libraries from starting

#### Scenario: Classification refinement fails after skeleton ingest
- **WHEN** asynchronous classification or catalog materialization fails for an inventory-backed discovered entry
- **THEN** the discovered entry MUST remain visible with a failure or review-required maturity state
- **AND** the failed refinement MUST NOT invalidate the persisted inventory facts

### Requirement: Synchronization jobs have queue priority over enrichment jobs
The system SHALL prioritize synchronization workflow tasks over metadata matching and media probing enrichment tasks when claiming available work, while still applying resource budgets and per-run fairness.

#### Scenario: A new scan is queued behind older probe work
- **WHEN** older probing or catalog matching work is queued and a new scan discovery task is queued
- **THEN** the scheduler MUST claim eligible scan synchronization work before lower-priority enrichment work when required resources are available

#### Scenario: No synchronization work is pending
- **WHEN** no available synchronization or projection workflow work is queued
- **THEN** the scheduler MUST continue processing available enrichment tasks normally
