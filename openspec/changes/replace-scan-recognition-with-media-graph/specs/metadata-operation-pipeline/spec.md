## ADDED Requirements

### Requirement: Automated metadata matching targets graph materialized works
The system SHALL queue and execute automated metadata match operations only for graph-materialized work targets that the metadata operation pipeline supports.

#### Scenario: Scan materializes movie and series roots
- **WHEN** media graph materialization creates or reuses movie, series, season, and episode metadata items
- **THEN** automated metadata match jobs MUST be queued only for movie and series items
- **AND** season and episode items MUST NOT be queued as independent match targets

#### Scenario: Episode triggers enrichment
- **WHEN** a newly scanned episode requires metadata enrichment
- **THEN** the enrichment workflow MUST target the series root for metadata matching and preserve descendant status for the originating episode

#### Scenario: Unsupported metadata item reaches match batch
- **WHEN** an unsupported item type such as season or episode appears in a metadata match batch
- **THEN** the batch MUST skip that item without failing the workflow
