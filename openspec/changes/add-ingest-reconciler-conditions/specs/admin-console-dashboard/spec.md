## ADDED Requirements

### Requirement: Console summarizes ingest organizing health
The system SHALL include ingest organizing health in the admin console using counts for organizing, failed, stale, review-required, and retry-eligible ingest stages.

#### Scenario: Ingest diagnostics are available
- **WHEN** the admin console summary loads and ingest condition data exists
- **THEN** the console MUST show concise counts for organizing media, failed stages, stale stages, review-required media, and retry-eligible stages

#### Scenario: Ingest conditions are unavailable
- **WHEN** ingest condition data cannot be loaded
- **THEN** the console MUST mark the ingest health section as warning or unknown without presenting stale data as healthy

### Requirement: Console provides ingest diagnostics entry points
The system SHALL provide administrator navigation or actions from the console to inspect ingest diagnostics and retry eligible stages.

#### Scenario: Failed ingest stages exist
- **WHEN** one or more ingest stages are failed or stale
- **THEN** the console MUST provide an entry point to an ingest diagnostics view or panel filtered to those issues

#### Scenario: Administrator retries a stage from diagnostics
- **WHEN** an administrator selects retry for an eligible ingest stage
- **THEN** the system MUST invoke the stage-scoped retry action and show queued, running, or rejected feedback

#### Scenario: No ingest issues exist
- **WHEN** all ingest conditions are ready, skipped, or otherwise terminal without warning severity
- **THEN** the console MUST show ingest health as normal and avoid prompting unnecessary maintenance actions
