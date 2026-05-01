# storage-change-indexing Specification

## Purpose
TBD - created by archiving change add-storage-index-diff-planner. Update Purpose after archive.
## Requirements
### Requirement: Persistent Storage Observation Index
The system SHALL persist the last observed storage state for active library paths so storage changes can be detected independently of transient file-system events.

#### Scenario: Index records observed provider objects
- **WHEN** an observer lists a local or OpenList library path
- **THEN** the system records each observed path with library id, storage provider, normalized storage path, directory flag, size, modified time when available, stable identity when available, provider/hash evidence when available, observation timestamp, and present status

#### Scenario: Index updates existing path observations
- **WHEN** a later observation reports a previously indexed path with changed size, modified time, stable identity, or provider evidence
- **THEN** the system updates the existing storage index row instead of creating a duplicate row for the same library, provider, and normalized path

#### Scenario: Missing paths remain visible for planning
- **WHEN** a previously indexed path is not present in a reconciliation observation
- **THEN** the system marks that indexed path missing or stale without immediately deleting the index row

### Requirement: Provider Change Sources Feed a Shared Planner
The system SHALL normalize local watcher events, OpenList polling observations, and external storage-event hints into a shared storage change planning workflow.

#### Scenario: Local file-system event becomes a planning hint
- **WHEN** a local library emits a create, write, remove, or rename file-system event
- **THEN** the system records or refreshes storage observations for the affected path and submits the normalized change to the shared planner

#### Scenario: OpenList polling produces observations
- **WHEN** an OpenList-backed library is due for polling
- **THEN** the system lists configured library paths through the OpenList storage provider and submits observed additions, removals, and metadata changes to the shared planner

#### Scenario: External storage events remain supported
- **WHEN** a client posts a valid storage event to `POST /api/v1/storage-events`
- **THEN** the system treats the event as a planning hint that follows the same debounce, indexing, and refresh planning path as internal observers

### Requirement: Diff Planner Produces Safe Refresh Plans
The system SHALL compare current observations with the storage index and produce refresh plans that are bounded to the smallest safe library scope.

#### Scenario: New media path plans parent refresh
- **WHEN** the planner detects a new media file path under a library
- **THEN** it enqueues or updates refresh work for the file's parent directory rather than scanning the entire library

#### Scenario: Updated media path plans parent refresh
- **WHEN** the planner detects changed size, modified time, hash evidence, or provider metadata for an indexed media file
- **THEN** it enqueues or updates refresh work for the file's parent directory after the path has satisfied file-stability rules

#### Scenario: Deleted file plans availability refresh
- **WHEN** the planner detects that an indexed media file is missing from provider observations
- **THEN** it enqueues or updates refresh work for the closest existing parent scope so existing scanner cleanup marks the inventory, asset, and catalog availability consistently

#### Scenario: Deleted directory plans existing ancestor refresh
- **WHEN** the planner detects that an indexed directory and its descendants are missing
- **THEN** it plans refresh work at the closest existing ancestor or falls back to the library root if no narrower existing scope is safe

#### Scenario: Dispersed or ambiguous changes fall back safely
- **WHEN** pending changes for a library are too dispersed, ambiguous, or cannot be scoped to an existing path
- **THEN** the planner schedules a full library scan for that library

### Requirement: Move and Rename Detection Preserves Identity When Confident
The system SHALL detect moves and renames using stable identity when available and SHALL avoid unsafe merges when identity confidence is low.

#### Scenario: Stable identity move updates refresh scope
- **WHEN** a missing indexed path and a new observed path share the same stable identity within the same library and provider
- **THEN** the planner treats the change as a move or rename and schedules refresh work covering the relevant old and new parent scopes

#### Scenario: Local inode identity supports rename detection
- **WHEN** local storage can provide stable device and inode identity for a file
- **THEN** the system uses that identity as stable evidence for detecting moves and renames within the same local filesystem

#### Scenario: Low confidence movement does not merge records
- **WHEN** a potential move lacks stable identity and only weak heuristics match the old and new paths
- **THEN** the planner treats the change as a delete plus create unless confidence thresholds are explicitly satisfied

### Requirement: Scanner Remains the Catalog and Inventory Writer
The system SHALL route planned storage changes through existing scan, probe, metadata, and projection workflows rather than directly mutating catalog state from raw events.

#### Scenario: Created file is ingested by scanner
- **WHEN** a refresh plan for a new media file is processed
- **THEN** the existing scanner writes or updates inventory files, media assets, catalog items, probe jobs, metadata matching jobs, and catalog projections

#### Scenario: Removed file is marked missing by scanner semantics
- **WHEN** a refresh plan covers a path whose media file has disappeared
- **THEN** scanner-owned cleanup semantics mark affected inventory files, media assets, and catalog items unavailable or missing without deleting their historical rows

#### Scenario: Renamed stable file updates playback path
- **WHEN** a refresh plan covers a renamed file with stable identity evidence
- **THEN** scanner-owned identity reuse updates the stored inventory path so playback resolves through the new storage path

### Requirement: Event Coalescing and File Stability
The system SHALL coalesce noisy storage changes and delay refresh work for files that are still changing.

#### Scenario: Burst events are coalesced
- **WHEN** multiple changes occur under the same library within the merge window
- **THEN** the system coalesces them into one or a small number of refresh plans by common ancestor scope

#### Scenario: Actively written file is delayed
- **WHEN** repeated observations show that a file's size or modified time is still changing
- **THEN** the system defers scanning that file until a quiet period or stability rule is satisfied

### Requirement: Reconciliation Repairs Missed Events
The system SHALL periodically reconcile each active library against provider state to repair missed local events, OpenList polling gaps, and storage index drift.

#### Scenario: Reconcile discovers missed creation
- **WHEN** periodic reconciliation observes a media file that is not present in the storage index
- **THEN** the system indexes the path and schedules refresh work as a new path

#### Scenario: Reconcile discovers missed deletion
- **WHEN** periodic reconciliation no longer observes an indexed media path
- **THEN** the system marks the index path missing and schedules refresh work to update catalog availability

#### Scenario: Reconcile continues after provider errors
- **WHEN** a provider listing fails for part of a library during reconciliation
- **THEN** the system records the failure and avoids marking unobserved paths missing solely because of that failed listing

### Requirement: Storage Change Diagnostics
The system SHALL expose enough storage-change status for administrators to understand whether automatic detection is healthy.

#### Scenario: Library observer status is available
- **WHEN** an administrator requests scan or listener diagnostics
- **THEN** the system reports observer mode, last successful observation, last reconcile time, pending plan count, and recent failure summary per library

#### Scenario: Disabled observer falls back to scans
- **WHEN** automatic observation is disabled or unavailable for a library
- **THEN** the system reports the disabled state and continues to support manual and scheduled scans for that library

