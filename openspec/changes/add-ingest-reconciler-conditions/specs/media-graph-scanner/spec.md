## ADDED Requirements

### Requirement: Scanner marks ingest reconciliation dirty work
The system SHALL mark discovered, refreshed, missing, or restored inventory files and affected library/root scopes dirty for ingest reconciliation as part of scanner synchronization.

#### Scenario: New video inventory file is discovered
- **WHEN** the scanner records a new available supported video inventory file
- **THEN** it MUST mark that inventory file or its root scope dirty for ingest reconciliation
- **AND** the scan job MUST be able to complete before materialization, probe, metadata, and projection conditions reach terminal states

#### Scenario: Existing inventory file changes storage facts
- **WHEN** a rescan updates size, modified time, hash evidence, stable identity, status, or storage path for an existing inventory file
- **THEN** the scanner MUST mark the affected ingest unit dirty so reconciliation can recompute conditions and stale work

#### Scenario: Inventory file becomes missing
- **WHEN** scanner synchronization marks an inventory file missing
- **THEN** it MUST mark the affected ingest unit and projection scope dirty so conditions and browse visibility can converge with the missing state

### Requirement: Scanner does not treat enrichment queue payloads as the only continuation record
The system SHALL preserve post-scan continuation through dirty ingest state so enrichment work is recoverable even when individual queued job payloads are merged, superseded, failed, or removed.

#### Scenario: Enrichment job is lost or superseded
- **WHEN** an inventory file still lacks required materialization, probe, metadata, or projection facts after a scan
- **THEN** dirty-driven reconciliation MUST be able to rediscover the missing work from facts and conditions
- **AND** successful scan completion MUST NOT depend on a specific chained job payload remaining present forever
