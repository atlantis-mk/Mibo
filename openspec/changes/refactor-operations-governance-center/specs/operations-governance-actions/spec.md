## ADDED Requirements

### Requirement: Issue-Level Actions
The system SHALL execute remediation actions against operations issues rather than only single low-level conditions.

#### Scenario: Execute grouped action
- **WHEN** an admin executes an action on an issue with multiple affected targets
- **THEN** the system applies the action to all eligible targets and returns per-target results

#### Scenario: Partial action failure
- **WHEN** some targets fail during a grouped action
- **THEN** the system records successful and failed target results and keeps the issue unresolved unless completion criteria are met

### Requirement: Metadata Governance Actions
The system SHALL support issue actions for metadata candidate application and manual metadata confirmation.

#### Scenario: Apply candidate to metadata issue
- **WHEN** an admin applies a metadata candidate from an issue
- **THEN** the target metadata item or grouped items are updated, relevant conditions are refreshed, and the issue records an action event

#### Scenario: Mark metadata issue governed
- **WHEN** an admin marks a metadata issue as governed after manual edits
- **THEN** the linked metadata items are marked manual or locked as requested and the issue can resolve

### Requirement: Classification Governance Actions
The system SHALL support issue actions for classification review and correction.

#### Scenario: Accept grouped classification decisions
- **WHEN** an admin accepts a grouped classification issue
- **THEN** all linked review-required classification decisions in that issue scope are accepted and affected files leave review-required scan state

#### Scenario: Correct classification shape
- **WHEN** an admin changes the content shape for a classification issue
- **THEN** the system persists the correction, queues affected files for reprocessing, and records the action event

### Requirement: Resource Governance Actions
The system SHALL support issue actions for resource relinking, unlinking, merge, and split workflows.

#### Scenario: Relink resource
- **WHEN** an admin relinks an affected resource to a different metadata item
- **THEN** the system updates resource metadata links, queues affected projections, and records the previous and new targets

### Requirement: Retry And Exclusion Actions
The system SHALL support retry and exclusion actions with explicit governance audit.

#### Scenario: Retry affected issue scope
- **WHEN** an admin retries an issue
- **THEN** all eligible linked files, metadata items, or library scopes are marked dirty and a retry event is recorded

#### Scenario: Exclude affected files
- **WHEN** an admin excludes files from scanning through an issue action
- **THEN** the system records the exclusion rule, updates linked issue targets, and closes only targets covered by the exclusion
