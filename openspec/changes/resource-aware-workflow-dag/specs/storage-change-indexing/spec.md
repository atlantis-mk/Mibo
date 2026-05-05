## MODIFIED Requirements

### Requirement: Provider Change Sources Feed a Shared Planner
The system SHALL normalize local watcher events, OpenList polling observations, and external storage-event hints into a shared storage change planning workflow. Planned refresh work MUST create or update library-scoped workflow runs or workflow tasks instead of blocking unrelated library work in a global queue.

#### Scenario: Local file-system event becomes a planning hint
- **WHEN** a local library emits a create, write, remove, or rename file-system event
- **THEN** the system records or refreshes storage observations for the affected path and submits the normalized change to the shared planner

#### Scenario: OpenList polling produces observations
- **WHEN** an OpenList-backed library is due for polling
- **THEN** the system lists configured library paths through the OpenList storage provider and submits observed additions, removals, and metadata changes to the shared planner

#### Scenario: External storage events remain supported
- **WHEN** a client posts a valid storage event to `POST /api/v1/storage-events`
- **THEN** the system treats the event as a planning hint that follows the same debounce, indexing, workflow scheduling, and refresh planning path as internal observers

### Requirement: Scanner Remains the Catalog and Inventory Writer
The system SHALL route planned storage changes through scanner-owned workflow tasks rather than directly mutating catalog state from raw events.

#### Scenario: Created file is ingested by scanner
- **WHEN** a refresh plan for a new media file is processed
- **THEN** scanner workflow tasks MUST write or update inventory files, media assets, catalog items, probe work, metadata matching work, and catalog projections

#### Scenario: Removed file is marked missing by scanner semantics
- **WHEN** a refresh plan covers a path whose media file has disappeared
- **THEN** scanner-owned workflow tasks MUST mark affected inventory files, media assets, and catalog items unavailable or missing without deleting their historical rows

#### Scenario: Renamed stable file updates playback path
- **WHEN** a refresh plan covers a renamed file with stable identity evidence
- **THEN** scanner-owned identity reuse MUST update the stored inventory path so playback resolves through the new storage path
