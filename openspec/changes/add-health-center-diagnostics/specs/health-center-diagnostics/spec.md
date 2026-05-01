## ADDED Requirements

### Requirement: Structured Health Diagnostics
The system SHALL expose structured health diagnostics for active media source, library, job, and external dependency issues using stable reason codes, severity values, affected scopes, user-facing summaries, technical details, and recommended actions.

#### Scenario: Known storage authentication failure is classified
- **WHEN** a recent failed storage-related job contains an OpenList or PikPak captcha/authentication expiration error
- **THEN** the diagnostics response includes a blocking health issue with reason code `storage_auth_expired`, affected media source and library references, user-facing recovery guidance, and the raw job error available as technical details

#### Scenario: Unknown failure has fallback diagnosis
- **WHEN** a recent failed job cannot be matched to a known classifier
- **THEN** the diagnostics response includes a health issue with reason code `job_failed_unknown`, the associated job reference, affected scope where derivable, and the raw error available as technical details

#### Scenario: Healthy system returns no active issues
- **WHEN** there are no active blocking, error, warning, or informational conditions detected from current state and recent failures
- **THEN** the diagnostics response returns an empty active issues list and an overall status of healthy

### Requirement: Health Impact Modeling
The system SHALL distinguish severity from product impact so that callers can determine whether an issue blocks scans, metadata enrichment, playback, or home-page catalog visibility.

#### Scenario: Storage auth expiration blocks scans and home visibility
- **WHEN** a media source authentication issue affects active libraries whose catalog content is hidden by library availability rules
- **THEN** the health issue marks scan impact and home visibility impact while listing the affected libraries and affected catalog counts when available

#### Scenario: Non-blocking issue remains visible without hiding content
- **WHEN** an issue affects enrichment quality but does not make catalog content unavailable
- **THEN** the health issue uses a non-blocking severity or impact and does not require callers to hide otherwise displayable catalog items

### Requirement: Health Center Issue Listing
The system SHALL provide an authenticated Health Center surface that lists active issues grouped by severity and scope, explains impact in user-friendly language, and links to affected media sources, libraries, and jobs.

#### Scenario: User views active issues
- **WHEN** an authenticated user opens the Health Center with active blocking and warning issues
- **THEN** the UI presents blocking issues first, shows affected libraries or sources, summarizes user impact, and provides access to technical details without making raw errors the primary message

#### Scenario: User views technical details
- **WHEN** a user expands an issue's technical details
- **THEN** the UI displays the related job kind, job status, failure timestamp when available, payload context where safe, and raw error text

### Requirement: Global Health Indicators
The system SHALL surface active blocking or error health issues outside the Health Center through lightweight indicators in global navigation, media library navigation entries, and relevant settings cards.

#### Scenario: Sidebar shows affected library indicator
- **WHEN** a library has an active blocking or error health issue
- **THEN** the sidebar marks that library as needing attention and links the user toward the issue details or affected library context

#### Scenario: Settings card shows actionable health summary
- **WHEN** a media library or media source card represents an affected scope
- **THEN** the card displays a user-friendly health summary, the current impact, and a way to open the corresponding Health Center issue

### Requirement: Guided Recovery Actions
The system SHALL attach recovery action descriptors to health issues when the system can guide or perform a next step, including opening external administration, validating media source connectivity, re-scanning affected libraries, and viewing related jobs.

#### Scenario: Storage auth issue offers recovery flow
- **WHEN** a storage authentication issue affects an OpenList-backed media source
- **THEN** the issue offers actions to open the configured OpenList administration entry point, validate the media source after external repair, re-scan affected libraries, and inspect related failed jobs

#### Scenario: Validation clears recoverable issue
- **WHEN** the user completes external repair and Mibo validates that the affected media source can be accessed again
- **THEN** the health issue no longer appears as active once the underlying current state and related retry results are healthy

#### Scenario: Recovery action failure remains visible
- **WHEN** validation or re-scan fails during a recovery flow
- **THEN** the Health Center keeps or updates the active issue with the latest failure context and does not imply the problem is resolved
