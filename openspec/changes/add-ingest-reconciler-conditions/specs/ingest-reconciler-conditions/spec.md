## ADDED Requirements

### Requirement: Ingest reconciliation is dirty driven
The system SHALL reconcile ingest state by processing dirty inventory files, dirty library or root scopes, and retry-due failed conditions instead of periodically scanning every inventory file during normal operation.

#### Scenario: Scanner discovers a video file
- **WHEN** a scan records or refreshes an available supported video inventory file
- **THEN** the system MUST mark the file or its containing scope dirty for ingest reconciliation
- **AND** the scan job MUST NOT wait for materialization, probing, metadata matching, or projection reconciliation to finish

#### Scenario: Reconciler claims dirty work
- **WHEN** dirty ingest work is available
- **THEN** the reconciler MUST claim a bounded batch of dirty units or scopes
- **AND** it MUST derive required follow-up work from current database facts before dispatching any stage executor

#### Scenario: No dirty work exists
- **WHEN** no dirty units, dirty scopes, or retry-due conditions are available
- **THEN** the reconciler MUST avoid full-library fact scans during normal worker polling

### Requirement: Conditions represent current organizing state
The system SHALL maintain condition snapshots for ingest units that summarize current organizing state derived from inventory, catalog, asset, probe, metadata, and projection facts.

#### Scenario: File is discovered but not materialized
- **WHEN** an available video inventory file has no catalog asset/item linkage yet
- **THEN** its conditions MUST indicate that it is visible as discovered or organizing media
- **AND** its materialization condition MUST indicate pending, running, failed, or review-required state as appropriate

#### Scenario: File is playable but probe failed
- **WHEN** a file has catalog and asset links but media probing failed
- **THEN** its conditions MUST preserve playable/materialized status separately from the failed probe condition
- **AND** user-facing state MUST NOT imply that the item is unusable solely because probe failed

#### Scenario: Facts change after a condition was written
- **WHEN** reconciliation observes that condition snapshots conflict with current database facts
- **THEN** it MUST update the conditions to match current facts rather than trusting the previous condition as authoritative

### Requirement: User organizing summaries are derived from conditions
The system SHALL derive concise user-facing organizing summaries from ingest conditions for discovered and catalog-backed media.

#### Scenario: Media is still being organized
- **WHEN** one or more required ingest conditions are pending or running
- **THEN** the user-facing summary MUST report an organizing state and a concise message describing the highest-priority active stage

#### Scenario: Media needs manual review
- **WHEN** a classification, metadata, or governance-related condition is review-required
- **THEN** the user-facing summary MUST report a review-required state
- **AND** it MUST include a non-technical reason suitable for media card or library detail display

#### Scenario: Media is ready after skipped optional work
- **WHEN** required conditions are complete and optional metadata or probe work is skipped by policy or configuration
- **THEN** the user-facing summary MUST report ready or partially-ready state without treating the skipped optional stage as a failure

### Requirement: Administrator diagnostics expose stage-level status
The system SHALL provide administrator diagnostics that list failed, stale, running, pending, and review-required ingest stages with affected file, library, catalog item, and job references when available.

#### Scenario: Probe fails for a file
- **WHEN** media probing fails for an inventory file
- **THEN** administrator diagnostics MUST identify the affected file, library, probe condition, failure reason, attempt count, and retry eligibility

#### Scenario: Metadata matching finds no candidate
- **WHEN** metadata matching completes with no acceptable candidate for a movie or series
- **THEN** administrator diagnostics MUST identify the affected catalog target and condition reason without requiring the administrator to inspect raw job payload JSON

#### Scenario: Running stage becomes stale
- **WHEN** a condition remains running beyond the configured stale threshold without a matching active job
- **THEN** administrator diagnostics MUST surface the stage as stale and eligible for reconciliation or retry

### Requirement: Stage retry is condition scoped
The system SHALL allow administrators to retry eligible failed, stale, skipped, or review-required ingest stages without rerunning unrelated successful stages.

#### Scenario: Administrator retries failed probe
- **WHEN** an administrator retries a failed probe condition
- **THEN** the system MUST mark the affected unit dirty or dispatch probe work for that unit
- **AND** it MUST NOT rerun metadata matching or catalog materialization solely because probe was retried

#### Scenario: Administrator retries a running stage
- **WHEN** an administrator requests retry for a stage that is currently running with an active job
- **THEN** the system MUST avoid creating duplicate concurrent work
- **AND** it MUST return the current running state or mark a follow-up retry as pending

### Requirement: Ingest events record meaningful transitions
The system SHALL append ingest events for meaningful stage transitions, failures, retries, review-required outcomes, and administrator actions while using conditions for current state.

#### Scenario: Stage fails
- **WHEN** an ingest stage fails
- **THEN** the system MUST update the relevant condition and append an event with stage, reason, message, affected references, and timestamp

#### Scenario: Administrator retries a stage
- **WHEN** an administrator retries an ingest stage
- **THEN** the system MUST append an event identifying the action, stage, affected unit, and user when available

#### Scenario: Event retention runs
- **WHEN** event retention or compaction runs
- **THEN** the system MUST preserve current conditions and enough recent events for diagnostics while removing or compacting events outside the retention policy

### Requirement: Projection freshness is reconciled as a dirty scope
The system SHALL represent projection freshness as reconciled dirty scopes and conditions instead of relying only on ad-hoc projection jobs from individual executors.

#### Scenario: Metadata changes a catalog item
- **WHEN** metadata matching or manual governance changes fields that affect browse or detail projections
- **THEN** the system MUST mark the affected item or library projection scope dirty
- **AND** reconciliation MUST refresh projection work in bounded batches or queued jobs

#### Scenario: Projection refresh completes
- **WHEN** projection refresh succeeds for an affected scope
- **THEN** the relevant projection condition MUST become current
- **AND** user-facing organizing state MUST no longer report waiting on projection for that scope
