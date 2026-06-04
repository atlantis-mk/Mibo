## ADDED Requirements

### Requirement: Enrichment consumes materialization outputs
The system SHALL schedule metadata matching, media probing, and projection refresh from persisted materialization results instead of recomputing affected metadata, resources, or files from manifests.

#### Scenario: Metadata match is scheduled from unit result
- **WHEN** a recognition unit materializes metadata IDs and the unit is eligible for remote metadata search
- **THEN** the system schedules metadata match work for the primary materialized metadata IDs recorded by the unit result

#### Scenario: Probe is scheduled from unit result
- **WHEN** a recognition unit materializes or updates playable resources and inventory files and probing is enabled for the library
- **THEN** the system schedules probe work for the file IDs recorded by the unit result

### Requirement: Enrichment eligibility is determined once
The system SHALL determine remote search eligibility and probe eligibility from directory shape, review state, materialization output, and library policy once per unit.

#### Scenario: Review-required directory skips remote match
- **WHEN** a materialized unit has review state `review_required`
- **THEN** the system records the skip reason and MUST NOT schedule remote metadata matching for that unit

#### Scenario: Attachment-only unit skips primary remote match
- **WHEN** a materialized unit is derived from an attachment group or supplemental-only assignment
- **THEN** the system records the skip reason and MUST NOT schedule primary remote metadata matching for that unit

#### Scenario: Movie collection unit requires provider availability
- **WHEN** a materialized unit from a movie collection requires remote search
- **THEN** the system verifies that the library has an operational remote search provider before scheduling metadata match work

### Requirement: Workflow tasks are coalesced by stable scope
The system SHALL coalesce workflow tasks by scan run, directory path, recognition unit key, enrichment target, and projection scope to avoid duplicate work.

#### Scenario: Duplicate task is requested
- **WHEN** multiple changed directories or units request the same enrichment target in one workflow run
- **THEN** the system creates one task for the normalized target and records all contributing unit IDs in the task payload or lineage

#### Scenario: Projection is requested by multiple units
- **WHEN** multiple recognition units in a scan run materialize metadata or resources for the same library scope
- **THEN** the system schedules a single projection refresh task for that affected scope after materialization and enrichment planning complete

### Requirement: Existing scan APIs preserve behavior
The system SHALL preserve the behavior of library creation, manual scan, scheduled scan, targeted refresh, and storage refresh APIs while routing new runs through the staged pipeline.

#### Scenario: Library is created
- **WHEN** a user creates a library with a media source and root path
- **THEN** the system creates the library and library path records, queues a staged ingest workflow for the root path, and eventually exposes materialized catalog items as before

#### Scenario: Targeted refresh is requested
- **WHEN** a user or storage listener requests refresh for a path inside a library path
- **THEN** the system scans and invalidates only snapshots, units, enrichment, and projection scopes affected by that target path

### Requirement: Pipeline lineage is inspectable
The system SHALL expose internal query helpers or diagnostics that connect a directory snapshot to its inventory files, signals, content shape plan, recognition units, materialization results, enrichment tasks, and projection refresh.

#### Scenario: Directory diagnosis is requested
- **WHEN** an operator inspects a directory after scan
- **THEN** the system can report the latest snapshot fingerprint, inventory status, signal status, shape plan, recognition unit status, materialization status, enrichment status, and skip reasons for that directory
