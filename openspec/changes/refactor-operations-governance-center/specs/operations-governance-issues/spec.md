## ADDED Requirements

### Requirement: Persistent Operations Issues
The system SHALL persist operations issues that group one or more low-level operational facts into a stable governance work item.

#### Scenario: Aggregates low-level facts into an issue
- **WHEN** multiple ingest conditions describe the same governance problem
- **THEN** the system creates or updates one operations issue with linked occurrences for each underlying condition

#### Scenario: Keeps stable identity across refreshes
- **WHEN** the same issue fingerprint is observed during a later aggregation run
- **THEN** the system updates the existing issue instead of creating a duplicate

### Requirement: Issue Lifecycle
The system SHALL track issue lifecycle separately from source fact lifecycle.

#### Scenario: Active issue is resolved
- **WHEN** a permitted governance action resolves all required targets for an issue
- **THEN** the issue status becomes `resolved` and records `resolved_at`, `resolved_by`, and a resolution event

#### Scenario: Resolved issue reopens
- **WHEN** a later aggregation run observes the same fingerprint with active source facts
- **THEN** the resolved issue becomes `reopened` or `active` and records a reopen event

### Requirement: Issue Evidence
The system SHALL retain evidence and affected targets for grouped issues.

#### Scenario: Issue has many affected files
- **WHEN** an issue includes more affected files than can fit in the list view
- **THEN** the issue summary includes counts and samples while the detail API exposes all linked targets and occurrences

### Requirement: Issue APIs
The system SHALL expose issue-oriented operations APIs.

#### Scenario: List active issues
- **WHEN** an authenticated user requests operations issues with status `active`
- **THEN** the API returns paginated issues sorted by lifecycle, severity, last seen time, and title

#### Scenario: Fetch issue detail
- **WHEN** an authenticated user requests a specific issue
- **THEN** the API returns issue metadata, targets, occurrences, recommended actions, and recent audit events

### Requirement: Task API Compatibility
The system SHALL preserve the existing operations task API during migration.

#### Scenario: Legacy task consumer requests tasks
- **WHEN** a client requests `/api/v1/operations/tasks`
- **THEN** the API returns task-shaped results derived from active issues or legacy fallback sources without requiring client changes
