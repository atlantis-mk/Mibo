## ADDED Requirements

### Requirement: Issue Inbox
The frontend SHALL present operations issues as a grouped governance inbox.

#### Scenario: Shows grouped issue row
- **WHEN** an issue affects multiple episodes or files
- **THEN** the row displays the issue title, scope, affected counts, severity, lifecycle status, and sample targets

#### Scenario: Filters issue list
- **WHEN** a user filters by status, kind, action type, library, or search query
- **THEN** the workbench updates the issue list using server-side filters and preserves pagination

### Requirement: Issue Detail Workspace
The frontend SHALL provide issue detail surfaces for grouped evidence and remediation.

#### Scenario: Open episodic issue detail
- **WHEN** a user opens a series or season issue
- **THEN** the UI shows grouped season/episode targets, source files, evidence, and available governance actions

#### Scenario: Open movie issue detail
- **WHEN** a user opens a movie-scoped issue
- **THEN** the UI shows current metadata, linked resources, candidate search, and available remediation actions

### Requirement: Action Execution UX
The frontend SHALL execute issue actions with clear target counts, progress, and result feedback.

#### Scenario: Execute grouped action
- **WHEN** an admin runs a grouped action
- **THEN** the UI disables duplicate execution, displays progress, and refreshes the issue after completion

#### Scenario: Show partial failures
- **WHEN** an action completes with partial failures
- **THEN** the UI shows which targets succeeded or failed and keeps unresolved issue state visible

### Requirement: Legacy Compatibility UX
The frontend SHALL tolerate legacy task-shaped data while migrating to issues.

#### Scenario: Issue API unavailable
- **WHEN** the issue API fails but the legacy task API is available
- **THEN** the operations center can show a compatibility fallback instead of a blank page
